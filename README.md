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
| `oz list --remote` | Browse wizards in the registry |
| `oz add <name>` | Install a wizard from registry or local file |
| `oz update <wizard>` | Re-fetch a wizard from the registry |
| `oz create <name>` | Create a new wizard from template |
| `oz edit <wizard>` | Open wizard config in `$EDITOR` |
| `oz remove <wizard>` | Remove a wizard config |
| `oz validate <path>` | Validate a wizard YAML file |

**Aliases:** `r` (run), `c`/`new` (create), `e` (edit), `rm` (remove), `l`/`ls` (list), `u` (update).

### Per-Wizard Subcommands

| Command | Description |
|---------|-------------|
| `oz run <wizard> doctor` | Check tool installation and version |
| `oz run <wizard> show` | Show all options with descriptions |
| `oz run <wizard> pins` | Interactive pin manager |
| `oz run <wizard> pins list` | Display current pins |
| `oz run <wizard> pins clear` | Remove all pins |
| `oz run <wizard> presets list` | List saved presets |
| `oz run <wizard> presets show <name>` | Show preset values and command |
| `oz run <wizard> presets show <name> -v` | Annotated view with labels and descriptions |
| `oz run <wizard> presets save <name>` | Save last-used values as preset |
| `oz run <wizard> presets remove <name>` | Remove a preset |

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
| `versions` | Semver constraint to show option only for matching versions |

### Version Control

Wizards can detect the installed tool version and filter options by semver range:

```yaml
version_control:
  command: ruby --version
  pattern: '(\d+\.\d+\.\d+)'
  label: Ruby
  custom_version_command: rbenv versions --bare
  available_versions: "3.2,3.1,3.0"        # or use available_versions_command
  custom_version_verify: rbenv versions --bare | grep -q {{version}}

options:
  - name: yjit
    type: confirm
    label: Enable YJIT?
    flag: --yjit
    versions: ">= 3.1"     # only shown when version is 3.1+
```

Supports all semver constraint syntax: `>=`, `<=`, `>`, `<`, `!=`, tilde (`~1.2`), caret (`^2.0`), wildcards (`1.2.x`), hyphen ranges (`1.2 - 1.4`), and OR (`||`).

### Conditional Visibility

Show or hide options based on previous answers:

```yaml
- name: db
  type: select
  label: Database
  choices:
    - { value: pg, label: PostgreSQL }
    - { value: sqlite, label: SQLite }

- name: pool_size
  type: input
  label: Connection pool size
  flag: --pool
  show_when:
    db: pg            # only shown when db=pg
```

### Dynamic Choices

Load choices from a shell command at runtime:

```yaml
- name: branch
  type: select
  label: Branch
  choices_from: git branch --format='%(refname:short)'
```
