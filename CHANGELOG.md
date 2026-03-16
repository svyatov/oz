# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- `oz generate` now writes to config directory by default instead of stdout
- Added `--force/-f` flag to `oz generate` for overwrite protection

### Removed

- `--stdin` and `--install` flags from `oz generate`

## [0.1.0] - 2026-03-14

### Added

- Interactive Bubbletea wizard UI with rich color palette
- YAML-driven wizard config with select, confirm, input, and multi_select field types
- CLI entrypoint with `list`, `validate`, `create`, `edit`, `remove`, and `run` commands
- Short aliases for subcommands (`r`, `c`/`new`, `e`, `rm`, `l`/`ls`, `u`)
- Version detection and semver-based option/choice compatibility filtering
- Custom version selection with optional verification
- Command builder with positional args and flag styles (equals/space)
- Last-used state persistence and named presets
- Pinned options with interactive TUI manager
- Conditional visibility (`show_when`/`hide_when`)
- Dynamic choices via `choices_from` shell commands
- Registry integration (`add`, `update`, `list --remote`)
- `doctor` subcommand for tool installation and version checks
- `show` subcommand to display all options with descriptions
- Batch config validation with detailed error reporting and cycle detection
- Dry-run mode (`-n`/`--dry-run`)
- Preset management (list, show, save, remove)
- Shell completion for all commands
- golangci-lint with 50+ linters enabled

[Unreleased]: https://github.com/svyatov/oz/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/svyatov/oz/releases/tag/v0.1.0
