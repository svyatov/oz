# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `oz test [wizard]`: a hermetic snapshot harness that asserts a wizard's fixture answers
  build the expected command. Fixtures live in `wizards/testdata/<wizard>/` as a `<case>.yml`
  (pinned `version:` + `answers:`) and a sibling `<case>.golden`; `--update` regenerates the
  goldens. Execution is hermetic — the pinned version drives option filtering (no detection)
  and dynamic-choice answers are taken as literal values (no shell runs) — so fixtures pass on
  any machine. Runs in CI over `wizards/` as a gate: a wizard that ships no fixture fails
- Proving-pack registry wizards exercising the harness's hard cases: `docker-run` (multi-select),
  `git-switch` (dynamic-choice supplied as a literal), with `rails-new` and `bundle-gem` covering
  version gating — each with passing fixtures
- `bundle-gem` registry wizard: wraps Bundler's `bundle gem` generator with version-aware
  options verified against real Bundler binaries from 2.2 to 4.x — `--changelog` (≥2.2.8),
  `--github-username` (≥2.2.16), `--linter` (≥2.2.31, supersedes the `<2.2.31` `--rubocop`),
  `--ext` values `c`/`rust` (≥2.4)/`go` (≥4.0), and `--bundle` (≥2.7); `--rubocop` is hidden on
  4.x where it was removed. Includes a version picker via `bundle _<version>_ gem`

### Fixed

- `oz list` / `oz list --remote`: add column spacing so the wizard name, description,
  and `(installed)` tag no longer run together, and a trailing blank line to match
  every other command's output
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
