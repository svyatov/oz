# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Fixed

- `oz list` / `oz list --remote`: add column spacing so the wizard name, description,
  and `(installed)` tag no longer run together
- Registry now served from the main repo (`svyatov/oz` `wizards/` + `index.yml`)
  instead of a separate `oz-wizards` repo, so `oz add`, `oz list --remote`, and
  `oz update` work out of the box; override with `OZ_REGISTRY_URL`
- rails-new wizard: expose `--skip-asset-pipeline` on Rails 8.x via a version-gated
  toggle (the asset-pipeline selector is `< 8.0` only, so 8.x had no way to skip it)

## [0.1.0] - 2026-04-02

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
