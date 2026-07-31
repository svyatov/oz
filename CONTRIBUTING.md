# Contributing to oz

## Development setup

Requirements: Go 1.26+, [Task](https://taskfile.dev/) runner.

```bash
git clone https://github.com/svyatov/oz.git
cd oz
task test
task lint
```

## Making changes

1. Fork the repo and create a feature branch.
2. Make your changes. A change that adds or alters functionality arrives with a test in the
   same pull request; a change that only touches prose or config does not.
3. Run `task lint` and `task test`. Both must pass.
4. Add an entry under `## [Unreleased]` in [CHANGELOG.md](CHANGELOG.md) for anything a user notices.
5. Open a pull request against `main`.

Commits follow [Conventional Commits](https://www.conventionalcommits.org/):
`type(scope): description` (e.g., `feat(wizard): add kubectl wizard`).

What an acceptable contribution has to satisfy is written down in two places: the
[Code Standards](#code-standards) section below, and [CLAUDE.md](CLAUDE.md), which carries the
lint thresholds, error-handling rules, and test conventions the linter enforces.

## Contributing a wizard

Wizard configs are YAML files in the [`wizards/`](wizards/) directory.

1. Create a YAML file: `oz create <name>` or `oz generate <tool>`.
2. Validate it: `oz validate <name>`.
3. Test it: `oz run <name> -n`.
4. **Add at least one fixture** (required: CI rejects a wizard without one).
5. Submit a PR adding your file to the `wizards/` directory.

See existing wizards in [`wizards/`](wizards/) for reference.

### Fixtures (required)

Every wizard under `wizards/` must ship at least one passing fixture, so CI can
catch a wizard drifting from the command it should build. A wizard with no
fixture fails the gate.

A fixture is a pair of files in `wizards/testdata/<wizard>/`:

- `<case>.yml`: the pinned tool `version:` and an `answers:` map of option name
  to value (string, bool, or list).
- `<case>.golden`: the expected built command (generated for you).

```yaml
# wizards/testdata/my-tool/basic.yml
version: "1.2.0"
answers:
  app_name: blog
  features: [a, b]
```

Generate or refresh the golden, then check it:

```bash
oz test my-tool --update --config-dir .   # write wizards/testdata/my-tool/*.golden
oz test my-tool --config-dir .            # verify it passes
```

Execution is hermetic: the pinned `version:` drives option filtering (the real
tool is never run or detected) and dynamic-choice (`choices_from`) answers are
taken as the literal values you supply (no shell runs). Fixtures therefore pass
on any machine, including one without the wrapped tool installed.

## Code standards

- All code is linted with [golangci-lint](https://golangci-lint.run/) (50+ linters).
- Tests are table-driven with `t.Run` subtests, stdlib `testing` only.
- Wrap errors: `fmt.Errorf("context: %w", err)`.
- Functions stay under 60 lines, lines under 120 chars.

## Reporting issues

Use the GitHub issue templates for bug reports, feature requests, or new wizard
suggestions. For a suspected security vulnerability, use the private channel in
[SECURITY.md](SECURITY.md) instead of the issue tracker.

## Versioning

The public API oz versions is its command-line surface: command and subcommand names, flags and
their aliases, the wizard YAML schema, the config directory layout and the `OZ_CONFIG_DIR` and
`OZ_REGISTRY_URL` environment variables, the registry index format, and exit codes. The Go packages
are not part of it: they all live under `internal/`, no other module can import them, and their
signatures change without a version bump.

oz is in `0.y.z` initial development. Until `1.0.0`, an incompatible change to the surface above
takes a MINOR bump and everything else takes a PATCH bump, so a MINOR release is the one to read
the changelog entry for before upgrading.

## Governance

oz has one maintainer, [@svyatov](https://github.com/svyatov), who reviews and merges every change
and cuts every release. There is no succession arranged: if that maintainer stops, the project
stops, and the MIT license lets anyone fork and continue it. Anyone weighing whether to depend on
oz should read that as the honest answer rather than as a placeholder.
