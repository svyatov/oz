# oz

Config-driven CLI wizard framework. Define interactive prompts in YAML, run them with [Bubbletea](https://github.com/charmbracelet/bubbletea), and execute the resulting shell commands.

## Install

```bash
go install github.com/svyatov/oz/cmd/oz@latest
```

## Quick Start

```bash
oz create mywizard    # scaffold a new wizard YAML and open in $EDITOR
oz run mywizard       # run the interactive wizard
oz run mywizard -n    # dry-run: print command without executing
```

## Example Wizard

```yaml
name: docker-run
description: Run a Docker container
command: docker run
flag_style: space

options:
  - name: image
    type: input
    label: Image name
    flag: ""
    positional: true
    required: true

  - name: detach
    type: confirm
    label: Run detached?
    flag: -d

  - name: port
    type: input
    label: Port mapping
    flag: -p
    validate:
      pattern: '^\d+:\d+$'
      message: "Use host:container format (e.g. 8080:80)"

  - name: env
    type: multi_select
    label: Environment
    flag: -e
    choices:
      - value: NODE_ENV=production
        label: Production
      - value: NODE_ENV=development
        label: Development
```

## Commands

| Command | Description |
|---------|-------------|
| `oz run <wizard>` | Run a wizard interactively |
| `oz run <wizard> -n` | Dry-run (print command only) |
| `oz run <wizard> -p <preset>` | Run with a saved preset |
| `oz list` | List available wizards |
| `oz create <name>` | Create a new wizard from template |
| `oz edit <wizard>` | Open wizard config in `$EDITOR` |
| `oz remove <wizard>` | Remove a wizard config |
| `oz validate <path>` | Validate a wizard YAML file |

**Aliases:** `r` (run), `c`/`new` (create), `e` (edit), `rm` (remove), `l`/`ls` (list).

### Per-Wizard Subcommands

| Command | Description |
|---------|-------------|
| `oz run <wizard> doctor` | Check tool installation and version |
| `oz run <wizard> explain` | Show all options with descriptions |
| `oz run <wizard> pins` | Interactive pin manager |
| `oz run <wizard> pins show` | Display current pins |
| `oz run <wizard> pins clear` | Remove all pins |
| `oz run <wizard> presets list` | List saved presets |
| `oz run <wizard> presets show <name>` | Show preset values and command |
| `oz run <wizard> presets explain <name>` | Annotated preset view |
| `oz run <wizard> presets save <name>` | Save last-used values as preset |
| `oz run <wizard> presets delete <name>` | Delete a preset |

## Option Types

- **select** — single choice from a list (`choices` or `choices_from`)
- **multi_select** — multiple choices with optional `separator`
- **confirm** — yes/no toggle (`flag`, `flag_true`, `flag_false`)
- **input** — free-text entry with optional `validate` (pattern, min/max length)

## Configuration

Wizards live in `~/.config/oz/wizards/` (override with `OZ_CONFIG_DIR` or `--config-dir`).

### Key Fields

| Field | Description |
|-------|-------------|
| `flag` | CLI flag (e.g. `--output`) |
| `flag_style` | `equals` (default) or `space` |
| `positional` | Emit as bare argument, not flag |
| `default` | Pre-selected value |
| `required` | Prevent empty input submission |
| `allow_none` | Add "(none)" choice to select |
| `show_when` / `hide_when` | Conditional visibility based on other answers |
| `choices_from` | Shell command for dynamic choices |
| `version_control` | Auto-detect tool version and filter options |
| `compat` | Map version ranges to allowed options |
