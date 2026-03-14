package generate

import (
	"testing"
)

const dockerRunHelp = `Usage:  docker run [OPTIONS] IMAGE [COMMAND] [ARG...]

Create and run a new container

Aliases:
  docker container run, docker run

Options:
      --add-host list                  Add a custom host-to-IP mapping
                                       (host:ip)
  -d, --detach                         Run container in background and
                                       print container ID
  -e, --env list                       Set environment variables
      --name string                    Assign a name to the container
  -p, --publish list                   Publish a container's port(s) to
                                       the host
      --restart string                 Restart policy to apply when a
                                       container exits (default "no")
      --rm                             Automatically remove the container
                                       when it exits
  -v, --volume list                    Bind mount a volume
  -w, --workdir string                 Working directory inside the
                                       container
`

const argparseHelp = `usage: mytool [-h] [--verbose] [--output OUTPUT] [--format {json,csv,txt}] [--count COUNT]

My awesome tool

Options:
  -h, --help            show this help message and exit
  --verbose             enable verbose output
  --output OUTPUT       output file path
  --format {json,csv,txt}
                        output format (default: json)
  --count COUNT         number of iterations (default: 10)
`

const cobraHelp = `Build a Go binary

Usage:
  go build [packages] [flags]

Flags:
  -o, --output string    output file path
  -v, --verbose          verbose output during build
      --race             enable race detector
      --trimpath         remove all file system paths from the binary
      --ldflags string   arguments to pass to the linker (default "-s -w")
  -h, --help             help for build
`

const kubectlHelp = `Apply a configuration to a resource by file name or stdin.

Examples:
  kubectl apply -f ./pod.json

Options:
    --all=false:
        Select all resources in the namespace of the specified resource types.
    --dry-run='none':
        Must be "none", "server", or "client". If client strategy, only print
        the object that would be sent, without sending it.
    -f, --filename=[]:
        The files that contain the configurations to apply.
    --force=false:
        If true, immediately remove resources from API and bypass graceful
        deletion.
    -o, --output='':
        Output format. One of: (json, yaml, name, go-template).
`

func TestParse_Docker(t *testing.T) {
	flags := Parse(dockerRunHelp)

	expected := map[string]struct {
		short       string
		long        string
		placeholder string
		defVal      string
	}{
		"add-host": {long: "--add-host", placeholder: "list"},
		"detach":   {short: "-d", long: "--detach"},
		"env":      {short: "-e", long: "--env", placeholder: "list"},
		"name":     {long: "--name", placeholder: "string"},
		"publish":  {short: "-p", long: "--publish", placeholder: "list"},
		"restart":  {long: "--restart", placeholder: "string", defVal: "no"},
		"rm":       {long: "--rm"},
		"volume":   {short: "-v", long: "--volume", placeholder: "list"},
		"workdir":  {short: "-w", long: "--workdir", placeholder: "string"},
	}

	if len(flags) != len(expected) {
		t.Fatalf("got %d flags, want %d", len(flags), len(expected))
	}

	for _, f := range flags {
		name := f.Long
		if name == "" {
			name = f.Short
		}
		name = name[2:] // strip --
		exp, ok := expected[name]
		if !ok {
			t.Errorf("unexpected flag %q", name)
			continue
		}
		if f.Short != exp.short {
			t.Errorf("%s: Short = %q, want %q", name, f.Short, exp.short)
		}
		if f.Long != exp.long {
			t.Errorf("%s: Long = %q, want %q", name, f.Long, exp.long)
		}
		if f.Placeholder != exp.placeholder {
			t.Errorf("%s: Placeholder = %q, want %q", name, f.Placeholder, exp.placeholder)
		}
		if f.Default != exp.defVal {
			t.Errorf("%s: Default = %q, want %q", name, f.Default, exp.defVal)
		}
	}
}

func TestParse_Argparse(t *testing.T) {
	flags := Parse(argparseHelp)

	// --help should be filtered out.
	for _, f := range flags {
		if f.Long == "--help" {
			t.Error("--help should be filtered out")
		}
	}

	// Should find --verbose, --output, --format, --count.
	if len(flags) != 4 {
		t.Fatalf("got %d flags, want 4: %+v", len(flags), flags)
	}

	// Check enum detection on --format.
	var formatFlag *Flag
	for i := range flags {
		if flags[i].Long == "--format" {
			formatFlag = &flags[i]
			break
		}
	}
	if formatFlag == nil {
		t.Fatal("--format not found")
	}
	if len(formatFlag.EnumValues) != 3 {
		t.Errorf("--format EnumValues = %v, want [json csv txt]", formatFlag.EnumValues)
	}
	if formatFlag.Default != "json" {
		t.Errorf("--format Default = %q, want %q", formatFlag.Default, "json")
	}

	// Check default extraction on --count.
	var countFlag *Flag
	for i := range flags {
		if flags[i].Long == "--count" {
			countFlag = &flags[i]
			break
		}
	}
	if countFlag == nil {
		t.Fatal("--count not found")
	}
	if countFlag.Default != "10" {
		t.Errorf("--count Default = %q, want %q", countFlag.Default, "10")
	}
}

func TestParse_Cobra(t *testing.T) {
	flags := Parse(cobraHelp)

	// --help should be filtered out.
	for _, f := range flags {
		if f.Long == "--help" {
			t.Error("--help should be filtered out")
		}
	}

	if len(flags) != 5 {
		t.Fatalf("got %d flags, want 5: %+v", len(flags), flags)
	}

	// --race and --trimpath should have no placeholder (boolean).
	for _, f := range flags {
		if f.Long == "--race" || f.Long == "--trimpath" {
			if f.Placeholder != "" {
				t.Errorf("%s: Placeholder = %q, want empty", f.Long, f.Placeholder)
			}
		}
	}

	// --ldflags should have a default.
	for _, f := range flags {
		if f.Long == "--ldflags" && f.Default != "-s -w" {
			t.Errorf("--ldflags Default = %q, want %q", f.Default, "-s -w")
		}
	}
}

func TestParse_Kubectl(t *testing.T) {
	flags := Parse(kubectlHelp)

	expected := map[string]string{
		"--all":      "false",
		"--dry-run":  "none",
		"--filename": "[]",
		"--force":    "false",
		"--output":   "",
	}

	if len(flags) != len(expected) {
		t.Fatalf("got %d flags, want %d: %+v", len(flags), len(expected), flags)
	}

	for _, f := range flags {
		exp, ok := expected[f.Long]
		if !ok {
			t.Errorf("unexpected flag %q", f.Long)
			continue
		}
		if f.Default != exp {
			t.Errorf("%s: Default = %q, want %q", f.Long, f.Default, exp)
		}
		if f.Description == "" {
			t.Errorf("%s: Description should not be empty", f.Long)
		}
	}

	// Check short flag detection.
	for _, f := range flags {
		if f.Long == "--filename" && f.Short != "-f" {
			t.Errorf("--filename Short = %q, want %q", f.Short, "-f")
		}
		if f.Long == "--output" && f.Short != "-o" {
			t.Errorf("--output Short = %q, want %q", f.Short, "-o")
		}
	}
}

func TestParse_ANSIStripping(t *testing.T) {
	// Help text with ANSI escape codes.
	ansiHelp := "Options:\n  \x1b[1m--verbose\x1b[0m             enable verbose output\n"
	flags := Parse(ansiHelp)

	if len(flags) != 1 {
		t.Fatalf("got %d flags, want 1", len(flags))
	}
	if flags[0].Long != "--verbose" {
		t.Errorf("Long = %q, want %q", flags[0].Long, "--verbose")
	}
}

func TestParse_ManPageBackspace(t *testing.T) {
	// Man-page bold: each char doubled with backspace (c\bc for each c).
	manHelp := "Options:\n  -\b--\b-v\bve\ber\brb\bbo\bos\bse\be             enable verbose output\n"
	flags := Parse(manHelp)

	if len(flags) != 1 {
		t.Fatalf("got %d flags, want 1", len(flags))
	}
	if flags[0].Long != "--verbose" {
		t.Errorf("Long = %q, want %q", flags[0].Long, "--verbose")
	}
}

func TestParse_EmptyInput(t *testing.T) {
	flags := Parse("")
	if len(flags) != 0 {
		t.Errorf("got %d flags for empty input, want 0", len(flags))
	}
}

func TestParse_NoOptionsSection(t *testing.T) {
	flags := Parse("This is just some text\nwith no options section.\n")
	if len(flags) != 0 {
		t.Errorf("got %d flags, want 0", len(flags))
	}
}
