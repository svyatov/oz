# oz

Config-driven CLI wizard framework in Go. Reads YAML wizard definitions, runs interactive Bubbletea prompts, builds and executes shell commands.

## Commands

```bash
task lint          # golangci-lint run ./...
task test          # go test ./...
task cover         # test with coverage report
go test ./internal/config/ -run TestValidate  # single test
go build ./cmd/oz/                            # build binary
```

## Linting

37 linters via golangci-lint. Key thresholds:
- **funlen**: 60 lines / 40 statements
- **gocyclo**: 15
- **gocognit**: 30
- **lll**: 120 chars

All new code must pass `task lint` with zero issues.

## Testing

- Table-driven tests with `t.Run` subtests
- stdlib `testing` only — no testify or assertion libraries
- `t.Helper()` on all test helpers
- `t.TempDir()` for file-system tests (auto-cleanup)
- `t.Fatalf` for setup failures, `t.Errorf` for assertion failures

## Architecture

```
cmd/oz/main.go          CLI entrypoint (cobra)
  → config/loader.go    Load + parse YAML wizard definitions
  → config/validator.go  Validate config, return []error
  → compat/version.go   Detect tool version, filter options by semver range
  → wizard/engine.go    Bubbletea Model driving step-by-step prompts
  → wizard/field*.go    Field interface implementations (select, confirm, input, multi_select)
  → command/builder.go  Build CLI command parts from answers
  → command/runner.go   Execute or copy the built command
  → store/store.go      Persist last-used state + presets as YAML
  → ui/theme.go         Lipgloss color palette and styles

oz run <wizard>
├── -n, --dry-run
├── -p, --with-preset <name>
├── doctor
├── explain
├── pins                    interactive TUI manager
│   ├── show                display current pins
│   └── clear               remove all pins
└── presets
    ├── list
    ├── show <name>
    ├── explain <name>
    ├── save <name>
    └── delete <name>
```

**Key abstractions:**
- `Field` interface (`Init`, `Update`, `View`, `Value`, `SetValue`) — each option type implements this
- `Engine` is the Bubbletea `Model` — manages step navigation, visibility (`show_when`), back/forward
- `Validate()` returns `[]error` (batch validation pattern)
- Config structs use `yaml` struct tags, parsed via `gopkg.in/yaml.v3`

## Go Conventions

- Wrap errors with `fmt.Errorf("context: %w", err)` — never bare returns
- All internal packages live under `internal/` — nothing is exported outside the module
- Batch validation: collect all errors into `[]error`, don't fail on the first
- Non-fatal warnings go to stderr via `fmt.Fprintf(os.Stderr, ...)`
- Use `maps.Copy`, `strings.FieldsSeq`, `slices` — prefer stdlib over hand-rolled utilities
- Exhaustive switch via `exhaustive` linter — all enum-like cases must be handled

## Versioning and Releases

- [Semantic Versioning](https://semver.org/) for all releases
- [Conventional Commits](https://www.conventionalcommits.org/) for all commit messages: `type(scope): description`
  - Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf`, `ci`, `build`, `style`
  - `feat` → minor bump, `fix` → patch bump, `feat!`/`BREAKING CHANGE` → major bump
- [Keep a Changelog](https://keepachangelog.com/) format in `CHANGELOG.md`
  - Sections: Added, Changed, Deprecated, Removed, Fixed, Security
  - Every user-facing change gets a changelog entry under `[Unreleased]`

## Shell Completion

When adding or modifying subcommands, verify shell completion works:
- Run `go build ./cmd/oz/ && ./oz __complete "" 2>/dev/null` — all top-level commands must appear
- Cobra hides commands without `Run`/`RunE` and without subcommands — always provide a `RunE` (can just call `cmd.Help()`)
- Test nested completions too: `./oz __complete <parent> "" 2>/dev/null`

## Code Quality

- All new code must have tests and pass lint
- Match existing patterns — check neighboring files before adding new ones
- Keep functions under 60 lines; extract helpers if needed
- No `//nolint` without a comment explaining why
