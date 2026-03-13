---
name: wizard-scaffold
description: Scaffold a new oz wizard YAML config with proper structure, option types, and validation
---

# Wizard Scaffold

Generate a new oz wizard YAML configuration file.

## Workflow

1. Ask the user for:
   - **Wizard name** (kebab-case, e.g. `create-react-app`)
   - **Target command** (e.g. `docker run`, `kubectl apply`)
   - **Options** — for each option, ask: name, type (select/confirm/input/multi_select), label, flag, and choices if applicable

2. Generate the YAML file at `~/.config/oz/wizards/<name>.yml` (or `$OZ_CONFIG_DIR/wizards/<name>.yml` if set).

3. Validate with `./oz validate <name>` (or `go run ./cmd/oz/ validate <name>` if no binary).

## Wizard YAML Schema

```yaml
name: wizard-name
description: "What this wizard does"
command: "base-command"
# flag_style: equals | space (default: equals)

# Optional version detection
# version_control:
#   label: "Tool Version"
#   command: tool --version
#   pattern: 'v?(\d+\.\d+\.\d+)'

options:
  # Select: single choice from a list
  - name: option_name        # snake_case
    type: select
    label: "Prompt text"
    flag: --flag-name
    # description: "Shown below the label"
    # default: value1
    # required: true
    # allow_none: true         # adds "(none)" choice
    # flag_none: --no-flag     # flag when none selected
    # show_when:               # conditional visibility
    #   other_option: value
    # hide_when:
    #   other_option: value
    choices:
      - value: val1
        label: "Label 1"
        description: "Description"
      - simple_value          # shorthand: value=label

  # Select with dynamic choices from a shell command
  # - name: container
  #   type: select
  #   label: "Select container"
  #   flag: --name
  #   choices_from: "docker ps --format '{{.Names}}\t{{.Status}}'"
  #   # Tab-separated: value[\tlabel[\tdescription]]

  # Confirm: yes/no toggle
  - name: enable_feature
    type: confirm
    label: "Enable feature?"
    flag: --enable             # shorthand: emit when true
    # flag_true: --enable      # explicit true flag
    # flag_false: --disable    # explicit false flag
    # default: false

  # Input: free-text entry
  - name: project_name
    type: input
    label: "Project name"
    flag: --name
    # required: true
    # positional: true         # bare arg, not flag
    # validate:
    #   pattern: '^\w+$'
    #   min_length: 1
    #   max_length: 100
    #   message: "Must be alphanumeric"

  # Multi-select: multiple choices
  - name: features
    type: multi_select
    label: "Select features"
    flag: --features
    # separator: ","           # --features=a,b vs --features=a --features=b
    choices:
      - value: a
        label: "Feature A"
      - value: b
        label: "Feature B"
```

## Rules

- Option names use `snake_case`.
- Wizard name uses `kebab-case`.
- Always include `name`, `description`, and `command` at the top level.
- Every option needs `name`, `type`, `label`, and `flag` (unless `positional: true`).
- Use `choices_from` for dynamic choices instead of hardcoding when the values come from a command.
- Keep descriptions concise — they appear inline in the TUI.
