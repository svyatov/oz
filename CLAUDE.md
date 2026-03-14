# oz

Config-driven CLI wizard framework in Go. Reads YAML wizard definitions, runs interactive Bubbletea prompts,
builds and executes shell commands. Wizard configs live in `~/.config/oz/wizards/`
(override with `OZ_CONFIG_DIR` env var or `--config-dir` flag).

## Working Style

When implementing a plan, proceed directly to code changes. Do not spend excessive time on exploration
or planning agents if the plan is already provided. If you need to explore, timebox it to 2-3 minutes
max before starting implementation.

## Commands

```bash
task lint          # golangci-lint run ./...
task test          # go test ./...
task cover         # test with coverage report
go test ./internal/config/ -run TestValidate  # single test
go build ./cmd/oz/                            # build binary
```

## Linting

50+ linters via golangci-lint (including govet analyzers). Key thresholds:

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

```txt
cmd/oz/
  main.go              CLI entrypoint (cobra)
  subcmds.go           Wizard subcommand definitions
  wizard.go            Wizard runner wiring
  prompt.go            Post-run confirmation prompts
  template.go          Wizard YAML template for create
  generate.go          Generate wizard YAML from --help output
  → generate/
      parse.go         Parse --help text into []Flag (GNU, Cobra, kubectl, Clap, argparse, Thor, man page)
      emit.go          Convert []Flag to scaffold YAML string
      testdata/        60 real-world --help fixtures for regression testing
  → config/
      schema.go        Type definitions (Wizard, Option, OptionType, FlagStyle)
      value.go         FieldValue sum type (string | bool | []string)
      loader.go        Load + parse YAML wizard definitions
      validator.go     Validate config, return []error
      validator_graph.go  Dependency graph validation (show_when/hide_when cycles)
  → compat/version.go   Detect tool version, filter options by semver range
  → wizard/
      engine.go        Bubbletea Model driving step-by-step prompts
      field.go         Field interface definition
      field_select.go  Single-choice select
      field_confirm.go Yes/no toggle
      field_input.go   Free-text input with validation
      field_multi.go   Multi-select
      version_loader.go  Async version detection + interactive picker
      pins.go          Interactive TUI for managing pinned options
      choices.go       Dynamic choice loading from shell commands
      state.go         Session state (completed steps, navigation history)
  → command/builder.go  Build CLI command parts from answers
  → command/runner.go   Execute or copy the built command
  → store/store.go      Persist last-used state, pins + presets as YAML
  → ui/theme.go         Lipgloss color palette and styles

oz list (l, ls)              list available wizards
oz validate <wizard>         validate wizard config
oz edit (e) <wizard>         open wizard YAML in $EDITOR
oz create (c, new) <wizard>  scaffold new wizard from template
oz generate (g, gen) <tool>  generate wizard YAML from --help output
oz remove (rm) <wizard>      delete wizard config (--force to skip confirm)
oz run (r) <wizard>
├── -n, --dry-run
├── -p, --preset <name>
├── doctor
├── show (s)                show all options with descriptions
├── pins                    interactive TUI manager
│   ├── list (l, ls)        display current pins
│   └── clear               remove all pins
└── presets                 show help
    ├── list (l, ls)
    ├── show <name> (s)     preset values and command (-v for verbose)
    ├── save <name>
    └── remove <name> (rm)
```

**Key abstractions:**

- `Field` interface (`Init`, `Update`, `View`, `Value`, `SetValue`) — each option type implements this
- `FieldValue` — type-safe sum type (string | bool | []string) replacing `any` across the codebase
- `Engine` is the Bubbletea `Model` — manages step navigation, visibility (`show_when`/`hide_when`), back/forward
- `Validate()` returns `[]error` (batch validation pattern); includes dependency graph cycle detection
- Config structs use `yaml` struct tags, parsed via `gopkg.in/yaml.v3`
- Version constraints use `github.com/Masterminds/semver/v3` — supports `>=`, `<=`, `>`, `<`, `=`, `!=`,
  tilde (`~1.2`), caret (`^2.0`), wildcards (`1.2.x`), hyphen ranges (`1.2 - 1.4`), and OR (`||`)
- `generate.Parse()` — multi-format help parser with ANSI stripping, section detection, GNU/kubectl/Thor
  preprocessing, best-of-both (section-aware vs full-scan) strategy, and enum/default extraction
- `generate.Emit()` — converts `[]Flag` to valid scaffold YAML with type inference, name dedup, and
  clean descriptions. 60 fixture regression tests across 59 CLI tools

## Go Conventions

- Wrap errors with `fmt.Errorf("context: %w", err)` — never bare returns
- All internal packages live under `internal/` — nothing is exported outside the module
- Batch validation: collect all errors into `[]error`, don't fail on the first
- Non-fatal warnings go to stderr via `fmt.Fprintf(os.Stderr, ...)`
- Use `maps.Copy`, `strings.FieldsSeq`, `slices` — prefer stdlib over hand-rolled utilities
- Exhaustive switch via `exhaustive` linter — all enum-like cases must be handled

### Error handling (wrapcheck, errorlint, errname, nilerr)

- Use `errors.Is()`/`errors.As()` for comparisons — never `==` or direct type assertion on errors.
- Error type names: `ErrFoo` for sentinel values, `FooError` for types.
- Never return nil error when err != nil.
- Don't return `(nil, nil)` — always return a value or an error.

### Type safety (forcetypeassert, unconvert)

- Always use two-value form: `v, ok := x.(Type)` — never bare `x.(Type)`.
- Don't add redundant type conversions.

### Performance (perfsprint, bodyclose, usestdlibvars)

- Use `strconv.Itoa(n)` / `strconv.FormatBool(b)` instead of `fmt.Sprintf` for simple conversions.
- Always close HTTP response bodies (`defer resp.Body.Close()`).
- Use stdlib constants (`http.StatusOK`, `http.MethodGet`) instead of literals.

### Context (fatcontext)

- Never store `context.Context` in a struct — pass it as a function parameter.

### Style (godot, nakedret, nestif)

- Comments must end with a period.
- No naked returns in functions longer than 5 lines.
- Prefer early returns over deeply nested `if` blocks.

### Modern Go (intrange, modernize)

- `for i := range n` instead of `for i := 0; i < n; i++`.

### Strings (goconst)

- Extract string literals used 3+ times into constants.

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

## Pre-commit Checklist

Always run lints (`golangci-lint run`) and tests (`go test ./...`) before committing.
Fix any lint issues before presenting work as complete.

## CLI / UI Guidelines

When implementing UI changes (TUI, CLI output), consider ALL field types and edge cases
(e.g., confirm fields, select fields, text inputs). Don't assume a fix for one field type covers all others.

## Code Quality

- All new code must have tests and pass lint
- Match existing patterns — check neighboring files before adding new ones
- Keep functions under 60 lines; extract helpers if needed
- No `//nolint` without a comment explaining why
