# Contributing to oz

## Development Setup

Requirements: Go 1.26+, [Task](https://taskfile.dev/) runner.

```bash
git clone https://github.com/svyatov/oz.git
cd oz
task test
task lint
```

## Making Changes

1. Fork the repo and create a feature branch.
2. Make your changes.
3. Run `task lint` and `task test` -- both must pass.
4. Open a pull request against `main`.

Commits follow [Conventional Commits](https://www.conventionalcommits.org/):
`type(scope): description` (e.g., `feat(wizard): add kubectl wizard`).

## Contributing a Wizard

Wizard configs are YAML files in the [`wizards/`](wizards/) directory.

1. Create a YAML file: `oz create <name>` or `oz generate <tool>`.
2. Validate it: `oz validate <name>`.
3. Test it: `oz run <name> -n`.
4. Submit a PR adding your file to the `wizards/` directory.

See existing wizards in [`wizards/`](wizards/) for reference.

## Code Standards

- All code is linted with [golangci-lint](https://golangci-lint.run/) (50+ linters).
- Tests are table-driven with `t.Run` subtests, stdlib `testing` only.
- Wrap errors: `fmt.Errorf("context: %w", err)`.
- Functions stay under 60 lines, lines under 120 chars.

## Reporting Issues

Use the GitHub issue templates for bug reports, feature requests, or new wizard
suggestions.
