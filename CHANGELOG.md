# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- Rename `delete` subcommand to `remove` (with `rm` alias)
- Rename `preset` subcommand to `presets` (plural, consistent with collection semantics)
- Rename `--preset` flag to `--with-preset` (`-p`) to avoid shadowing the subcommand name
- Add `-n` shorthand for `--dry-run`
- Convert `--pins` flag to `pins` subcommand with `show` and `clear` sub-commands

### Added

- Short aliases for subcommands (`r`, `c`/`new`, `e`, `rm`, `l`/`ls`)
- Interactive Bubbletea wizard UI with rich color palette
- CLI entrypoint with `list` and `validate` subcommands
- YAML-driven wizard config with select, confirm, input, and multi_select field types
- Version detection and semver-based option compatibility filtering
- Command builder with positional args and flag styles (equals/space)
- Last-used state persistence and named presets
- Batch config validation with detailed error reporting
- golangci-lint with 37 linters enabled
