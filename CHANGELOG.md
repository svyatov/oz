# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-XX-XX

Initial public release.

### Features

- Interactive TUI wizard engine powered by Bubbletea
- YAML-based wizard configuration with select, confirm, input, and multi-select field types
- Auto-generate wizard configs from any CLI tool's `--help` output (59 formats supported)
- Wizard registry with `add` and `update` commands
- Version detection with semver-based option filtering
- Presets and pinned options for quick reuse
- Conditional visibility (`show_when` / `hide_when`)
- Dynamic choices via shell commands
- Dry-run mode and command copy-to-clipboard
- Shell completions (bash, zsh, fish, powershell)

[Unreleased]: https://github.com/svyatov/oz/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/svyatov/oz/releases/tag/v0.1.0
