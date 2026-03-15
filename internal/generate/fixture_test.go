package generate

import (
	"os"
	"strings"
	"testing"

	"github.com/svyatov/oz/internal/config"
)

// fixtureTest defines expected parse results for a real tool's help output.
type fixtureTest struct {
	file     string // testdata filename.
	name     string // wizard name for emit.
	command  string // wizard command for emit.
	minFlags int    // minimum expected flags (allows growth as tools add flags).
	// spotChecks validate specific flags exist with correct properties.
	spotChecks []spotCheck
}

type spotCheck struct {
	long        string
	short       string // empty if no short flag expected.
	isBool      bool   // true if flag should have no placeholder.
	hasDefault  bool
	hasEnum     bool
	description string // substring match; empty to skip.
}

var fixtureTests = []fixtureTest{
	{
		file: "docker-run.txt", name: "docker-run", command: "docker run",
		minFlags: 90,
		spotChecks: []spotCheck{
			{long: "--detach", short: "-d", isBool: true},
			{long: "--env", short: "-e"},
			{long: "--name"},
			{long: "--publish", short: "-p"},
			{long: "--restart", hasDefault: true, description: "Restart policy"},
			{long: "--rm", isBool: true},
			{long: "--volume", short: "-v"},
		},
	},
	{
		file: "docker-compose-up.txt", name: "docker-compose-up", command: "docker compose up",
		minFlags: 25,
		spotChecks: []spotCheck{
			{long: "--build", isBool: true},
			{long: "--detach", short: "-d", isBool: true},
			{long: "--force-recreate", isBool: true},
			{long: "--scale"},
		},
	},
	{
		file: "kubectl-apply.txt", name: "kubectl-apply", command: "kubectl apply",
		minFlags: 15,
		spotChecks: []spotCheck{
			{long: "--filename", short: "-f"},
			{long: "--output", short: "-o"},
			{long: "--dry-run", hasDefault: true},
		},
	},
	{
		file: "cargo-new.txt", name: "cargo-new", command: "cargo new",
		minFlags: 5,
		spotChecks: []spotCheck{
			{long: "--vcs", hasEnum: true, description: "version control"},
			{long: "--edition", hasEnum: true},
			{long: "--bin", isBool: true},
			{long: "--lib", isBool: true},
			{long: "--name"},
		},
	},
	{
		file: "gh-repo-create.txt", name: "gh-repo-create", command: "gh repo create",
		minFlags: 10,
		spotChecks: []spotCheck{
			{long: "--clone", short: "-c", isBool: true},
			{long: "--public", isBool: true},
			{long: "--private", isBool: true},
			{long: "--description", short: "-d"},
		},
	},
	{
		file: "gh-pr-create.txt", name: "gh-pr-create", command: "gh pr create",
		minFlags: 12,
		spotChecks: []spotCheck{
			{long: "--title", short: "-t"},
			{long: "--body", short: "-b"},
			{long: "--draft", short: "-d", isBool: true},
			{long: "--base", short: "-B"},
		},
	},
	{
		file: "brew-install.txt", name: "brew-install", command: "brew install",
		minFlags: 30,
		spotChecks: []spotCheck{
			{long: "--debug", short: "-d", isBool: true},
			{long: "--force", isBool: true},
			{long: "--verbose", short: "-v", isBool: true},
			{long: "--cask", isBool: true},
		},
	},
	{
		file: "ansible-playbook.txt", name: "ansible-playbook", command: "ansible-playbook",
		minFlags: 15,
		spotChecks: []spotCheck{
			{long: "--flush-cache", isBool: true, description: "clear the fact cache"},
			{long: "--syntax-check", isBool: true},
			{long: "--check", short: "-C", isBool: true},
		},
	},
	{
		file: "rails-new.txt", name: "rails-new", command: "rails new",
		minFlags: 30,
		spotChecks: []spotCheck{
			{long: "--database", hasEnum: true, hasDefault: true},
			{long: "--javascript", short: "-j", hasEnum: true, hasDefault: true},
			{long: "--css", short: "-c", hasEnum: true},
			{long: "--main", isBool: true},
			{long: "--pretend", short: "-p", isBool: true, description: "Run but do not make any changes"},
			{long: "--api", isBool: true, description: "Preconfigure smaller stack for API only apps"},
			{long: "--skip-javascript", short: "-J", isBool: true},
			{long: "--skip-git", isBool: true},
			{long: "--skip-test", isBool: true},
			{long: "--name"},
			{long: "--skip-docker", isBool: true},
		},
	},
	{
		file: "just.txt", name: "just", command: "just",
		minFlags: 25,
		spotChecks: []spotCheck{
			{long: "--color", hasDefault: true, hasEnum: true, description: "Print colorful output"},
			{long: "--check", isBool: true},
			{long: "--justfile", short: "-f"},
			{long: "--yes", isBool: true},
		},
	},
	{
		file: "task.txt", name: "task", command: "task",
		minFlags: 20,
		spotChecks: []spotCheck{
			{long: "--color", short: "-c", hasDefault: true},
			{long: "--dry", short: "-n", isBool: true},
			{long: "--force", short: "-f", isBool: true},
			{long: "--verbose", short: "-v", isBool: true},
		},
	},
	{
		file: "yt-dlp.txt", name: "yt-dlp", command: "yt-dlp",
		minFlags: 200,
		spotChecks: []spotCheck{
			{long: "--update", isBool: true},
			{long: "--verbose", short: "-v", isBool: true},
			{long: "--format", short: "-f"},
		},
	},
	{
		file: "npm-init.txt", name: "npm-init", command: "npm init",
		minFlags: 8,
		spotChecks: []spotCheck{
			{long: "--init-author-name"},
			{long: "--init-license"},
			{long: "--scope"},
		},
	},
	{
		file: "pip-install.txt", name: "pip-install", command: "pip install",
		minFlags: 40,
		spotChecks: []spotCheck{
			{long: "--requirement", short: "-r"},
			{long: "--upgrade", short: "-U", isBool: true},
			{long: "--force-reinstall", isBool: true},
			{long: "--no-deps", isBool: true},
		},
	},
	{
		file: "pnpm-create.txt", name: "pnpm-create", command: "pnpm create",
		minFlags: 1,
	},
	{
		file: "bundle-gem.txt", name: "bundle-gem", command: "bundle gem",
		minFlags: 10,
		spotChecks: []spotCheck{
			{long: "--coc", isBool: true},
			{long: "--changelog", isBool: true},
			{long: "--exe", isBool: true},
		},
	},
	// Helm (Cobra-based Kubernetes package manager).
	{
		file: "helm-install.txt", name: "helm-install", command: "helm install",
		minFlags: 35,
		spotChecks: []spotCheck{
			{long: "--create-namespace", isBool: true},
			{long: "--dry-run", isBool: true},
			{long: "--namespace", short: "-n"},
			{long: "--values", short: "-f"},
			{long: "--set"},
			{long: "--wait", isBool: true},
		},
	},
	{
		file: "helm-upgrade.txt", name: "helm-upgrade", command: "helm upgrade",
		minFlags: 35,
		spotChecks: []spotCheck{
			{long: "--install", short: "-i", isBool: true},
			{long: "--namespace", short: "-n"},
			{long: "--reuse-values", isBool: true},
			{long: "--values", short: "-f"},
		},
	},
	// AWS CLI (man-page style output).
	{
		file: "aws-s3-cp.txt", name: "aws-s3-cp", command: "aws s3 cp",
		minFlags: 15,
		spotChecks: []spotCheck{
			{long: "--dryrun", isBool: true},
			{long: "--quiet", isBool: true},
			{long: "--storage-class"},
			{long: "--content-type"},
		},
	},
	// golangci-lint (Cobra-based Go linter).
	{
		file: "golangci-lint-run.txt", name: "golangci-lint-run",
		command: "golangci-lint run",
		minFlags: 30,
		spotChecks: []spotCheck{
			{long: "--config", short: "-c"},
			{long: "--no-config", isBool: true},
			{long: "--fix", isBool: true},
			{long: "--timeout"},
		},
	},

	// --- Ruby ---
	{
		file: "rspec.txt", name: "rspec", command: "rspec", minFlags: 20,
		spotChecks: []spotCheck{
			{long: "--format", short: "-f"},
			{long: "--force-color", isBool: true},
		},
	},
	{
		file: "rubocop.txt", name: "rubocop", command: "rubocop", minFlags: 30,
		spotChecks: []spotCheck{
			{long: "--fix-layout", short: "-x", isBool: true},
			{long: "--format", short: "-f"},
		},
	},
	{file: "gem-install.txt", name: "gem-install", command: "gem install", minFlags: 20},
	{file: "brakeman.txt", name: "brakeman", command: "brakeman", minFlags: 30},

	// --- Python ---
	{
		file: "ruff-check.txt", name: "ruff-check", command: "ruff check", minFlags: 10,
		spotChecks: []spotCheck{
			{long: "--fix", isBool: true},
		},
	},
	{file: "ruff-format.txt", name: "ruff-format", command: "ruff format", minFlags: 5},
	{
		file: "poetry-new.txt", name: "poetry-new", command: "poetry new", minFlags: 5,
		spotChecks: []spotCheck{{long: "--name"}},
	},
	{file: "poetry-install.txt", name: "poetry-install", command: "poetry install", minFlags: 10},
	{
		file: "black.txt", name: "black", command: "black", minFlags: 20,
		spotChecks: []spotCheck{
			{long: "--check", isBool: true},
			{long: "--diff", isBool: true},
			{long: "--line-length", short: "-l"},
		},
	},
	{
		file: "pytest.txt", name: "pytest", command: "pytest", minFlags: 50,
		spotChecks: []spotCheck{
			{long: "--verbose", short: "-v", isBool: true},
		},
	},
	{
		file: "uv-init.txt", name: "uv-init", command: "uv init", minFlags: 10,
		spotChecks: []spotCheck{{long: "--name"}},
	},

	// --- Node/Bun ---
	{file: "bun-init.txt", name: "bun-init", command: "bun init", minFlags: 3},
	{
		file: "bun-install.txt", name: "bun-install", command: "bun install", minFlags: 20,
		spotChecks: []spotCheck{
			{long: "--production", isBool: true},
			{long: "--frozen-lockfile", isBool: true},
		},
	},
	{file: "yarn.txt", name: "yarn", command: "yarn", minFlags: 30},
	{
		file: "eslint.txt", name: "eslint", command: "eslint", minFlags: 30,
		spotChecks: []spotCheck{
			{long: "--fix", isBool: true},
			{long: "--config", short: "-c"},
		},
	},
	{file: "prettier.txt", name: "prettier", command: "prettier", minFlags: 30},

	// --- Go ---
	{file: "air.txt", name: "air", command: "air", minFlags: 1},
	{
		file: "dlv-debug.txt", name: "dlv-debug", command: "dlv debug", minFlags: 3,
		spotChecks: []spotCheck{
			{long: "--continue", isBool: true},
		},
	},

	// --- Rust ---
	{
		file: "cargo-build.txt", name: "cargo-build", command: "cargo build", minFlags: 10,
		spotChecks: []spotCheck{
			{long: "--release", isBool: true},
		},
	},
	{
		file: "cargo-test.txt", name: "cargo-test", command: "cargo test", minFlags: 10,
		spotChecks: []spotCheck{
			{long: "--no-run", isBool: true},
		},
	},

	// --- Cloud services ---
	{
		file: "fly-launch.txt", name: "fly-launch", command: "fly launch", minFlags: 30,
		spotChecks: []spotCheck{
			{long: "--name"},
			{long: "--region"},
		},
	},
	{file: "fly-deploy.txt", name: "fly-deploy", command: "fly deploy", minFlags: 30},
	{file: "vercel.txt", name: "vercel", command: "vercel", minFlags: 5},
	{
		file: "doctl-droplet-create.txt", name: "doctl-droplet-create",
		command: "doctl compute droplet create", minFlags: 15,
		spotChecks: []spotCheck{
			{long: "--region"},
			{long: "--size"},
			{long: "--image"},
		},
	},
	{file: "supabase-init.txt", name: "supabase-init", command: "supabase init", minFlags: 5},
	{file: "firebase-deploy.txt", name: "firebase-deploy", command: "firebase deploy", minFlags: 3},
	{
		file: "gcloud-instances-create.txt", name: "gcloud-instances-create",
		command: "gcloud compute instances create", minFlags: 50,
		spotChecks: []spotCheck{
			{long: "--zone"},
			{long: "--machine-type"},
		},
	},
	{
		file: "az-vm-create.txt", name: "az-vm-create", command: "az vm create", minFlags: 50,
		spotChecks: []spotCheck{
			{long: "--resource-group"},
			{long: "--name"},
			{long: "--image"},
		},
	},
	{file: "heroku-apps-create.txt", name: "heroku-apps-create", command: "heroku apps:create", minFlags: 5},

	// --- Extras ---
	{
		file: "yq.txt", name: "yq", command: "yq", minFlags: 20,
		spotChecks: []spotCheck{
			{long: "--indent", short: "-I"},
		},
	},
	{
		file: "bat.txt", name: "bat", command: "bat", minFlags: 20,
		spotChecks: []spotCheck{
			{long: "--language", short: "-l"},
			{long: "--theme"},
		},
	},
	{
		file: "eza.txt", name: "eza", command: "eza", minFlags: 10,
		spotChecks: []spotCheck{
			{long: "--oneline", isBool: true},
			{long: "--long", short: "-l", isBool: true},
		},
	},
	{
		file: "xh.txt", name: "xh", command: "xh", minFlags: 30,
		spotChecks: []spotCheck{
			{long: "--json", short: "-j", isBool: true},
			{long: "--pretty", hasEnum: true, description: "Controls output processing"},
			{long: "--style", short: "-s", hasEnum: true},
		},
	},

	// --- More Ruby ---
	{
		file: "hanami-new.txt", name: "hanami-new", command: "hanami new", minFlags: 3,
		spotChecks: []spotCheck{{long: "--database"}},
	},

	// --- Search/CLI tools ---
	{
		file: "rg.txt", name: "rg", command: "rg", minFlags: 80,
		spotChecks: []spotCheck{
			{long: "--ignore-case", short: "-i", isBool: true},
			{long: "--fixed-strings", short: "-F", isBool: true},
		},
	},
	{
		file: "fd.txt", name: "fd", command: "fd", minFlags: 30,
		spotChecks: []spotCheck{
			{long: "--hidden", short: "-H", isBool: true},
			{long: "--type", short: "-t"},
			{long: "--extension", short: "-e"},
		},
	},
	{
		file: "git-clone.txt", name: "git-clone", command: "git clone", minFlags: 20,
		spotChecks: []spotCheck{
			{long: "--depth"},
			{long: "--bare", isBool: true},
		},
	},
}

func TestFixtures_Parse(t *testing.T) {
	for _, tt := range fixtureTests {
		t.Run(tt.name, func(t *testing.T) {
			data := readFixture(t, tt.file)
			flags := Parse(string(data))

			if len(flags) < tt.minFlags {
				t.Errorf("got %d flags, want at least %d", len(flags), tt.minFlags)
			}

			flagMap := indexFlags(flags)
			for _, sc := range tt.spotChecks {
				checkFlag(t, flagMap, sc)
			}
		})
	}
}

func TestFixtures_Emit_RoundTrip(t *testing.T) {
	for _, tt := range fixtureTests {
		t.Run(tt.name, func(t *testing.T) {
			data := readFixture(t, tt.file)
			flags := Parse(string(data))

			yamlStr := Emit(EmitConfig{Name: tt.name, Command: tt.command}, flags)

			w, err := config.ParseWizard([]byte(yamlStr))
			if err != nil {
				t.Fatalf("ParseWizard failed: %v", err)
			}

			if w.Name != tt.name {
				t.Errorf("Name = %q, want %q", w.Name, tt.name)
			}
			if w.Command != tt.command {
				t.Errorf("Command = %q, want %q", w.Command, tt.command)
			}

			for _, o := range w.Options {
				if o.Name == "" {
					t.Error("option has empty name")
				}
				if o.Label == "" {
					t.Errorf("option %q has empty label", o.Name)
				}
				if !o.Type.IsValid() {
					t.Errorf("option %q has invalid type %q", o.Name, o.Type)
				}
			}
		})
	}
}

func TestFixtures_Emit_Validates(t *testing.T) {
	for _, tt := range fixtureTests {
		t.Run(tt.name, func(t *testing.T) {
			data := readFixture(t, tt.file)
			flags := Parse(string(data))
			yamlStr := Emit(EmitConfig{Name: tt.name, Command: tt.command}, flags)

			w, err := config.ParseWizard([]byte(yamlStr))
			if err != nil {
				t.Fatalf("ParseWizard failed: %v", err)
			}

			errs := config.Validate(w)
			if len(errs) > 0 {
				t.Errorf("Validate returned %d errors:\n%s",
					len(errs), config.FormatErrors(errs))
			}
		})
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return data
}

func indexFlags(flags []Flag) map[string]*Flag {
	m := make(map[string]*Flag, len(flags))
	for i := range flags {
		if flags[i].Long != "" {
			m[flags[i].Long] = &flags[i]
		}
		if flags[i].Short != "" {
			m[flags[i].Short] = &flags[i]
		}
	}
	return m
}

func checkFlag(t *testing.T, flagMap map[string]*Flag, sc spotCheck) {
	t.Helper()

	f, ok := flagMap[sc.long]
	if !ok {
		t.Errorf("expected flag %s not found", sc.long)
		return
	}

	if sc.short != "" && f.Short != sc.short {
		t.Errorf("%s: Short = %q, want %q", sc.long, f.Short, sc.short)
	}

	if sc.isBool && f.Placeholder != "" {
		t.Errorf("%s: expected boolean (no placeholder), got %q", sc.long, f.Placeholder)
	}

	if sc.hasDefault && f.Default == "" {
		t.Errorf("%s: expected a default value", sc.long)
	}

	if sc.hasEnum && len(f.EnumValues) == 0 {
		t.Errorf("%s: expected enum values", sc.long)
	}

	if sc.description != "" {
		if !strings.Contains(f.Description, sc.description) {
			t.Errorf("%s: expected description containing %q, got %q", sc.long, sc.description, f.Description)
		}
	}
}
