---
name: wizard-scaffold
description: Scaffold a new oz wizard YAML config with proper structure, option types, and validation
---

# Wizard Scaffold

Generate a new oz wizard YAML configuration file from a CLI tool's help output.

## Workflow

1. Ask the user for:
   - **Wizard name** (kebab-case, e.g. `rails-new`)
   - **Base command** (e.g. `rails new`, `docker run`)
   - **Help command** (e.g. `rails new -h`, `docker run --help`)

2. **Run the help command** with Bash. Parse the output to identify every flag,
   positional arg, description, default, and enumerated value.
   Use the type selection guide below to map each to an option type.

3. **Generate a complete wizard config** with ALL options from the help text.
   Every option must have a `description` extracted from the help text.
   If there are more than ~20 options, include the most common ones and group
   advanced options behind `show_when` on an `advanced_mode` confirm.

4. **Review with the user.** Ask if they want to:
   - Remove or reorder options
   - Add version control (multi-version tool support)
   - Add conditional visibility (`show_when`/`hide_when`)
   - Use dynamic choices (`choices_from`) for any select

5. **Generate the YAML file** at `~/.config/oz/wizards/<name>.yml` (or `$OZ_CONFIG_DIR/wizards/<name>.yml` if set).

6. **Validate** with `go run ./cmd/oz/ validate <name>` (or `./oz validate <name>` if binary exists).

7. **Test** with `go run ./cmd/oz/ run <name> --dry-run`.

## Type Selection Guide

| Help text pattern             | Type                              | Notes                     |
| ----------------------------- | --------------------------------- | ------------------------- |
| `--flag` (boolean, no value)  | `confirm`                         | `--verbose`, `--force`    |
| `--skip-*` (boolean, skip)    | `confirm`                         | label: "Skip X?"          |
| `--no-*` (negation flag)      | `confirm` with `flag_false`       | label uses positive form  |
| `--flag=VALUE` listed values  | `select`                          | extract choices from help |
| `--flag=VALUE` free-form      | `input`                           | `--name=NAME`             |
| `--flag=V1,V2,...` comma list | `multi_select` + `separator: ","` |                           |
| `ARG` (positional argument)   | `input` + `positional: true`      | put first                 |

## Label & Description Guidelines

**Labels (prompt text shown to the user):**

- Use question form: "Which database?" not "Database"
- For `--skip-*` flags: "Skip X?" (not "Include X?" or "Enable X?") — matches the flag semantics
- For `--no-*` flags: use the positive form. `--no-rc` -> label "Load RC file?" with `flag_false: --no-rc`
- Be specific about the effect: "Use TypeScript instead of JavaScript?" not "TypeScript?"
- Include enough context for someone unfamiliar with the tool
- Keep under ~60 characters

**Descriptions (detail text below the label):**

- Fill a description for EVERY option — pull wording from the help text
- Explain *what it does* and *when you'd change it* — don't repeat the label
- For the wizard-level `description`: explain what the wizard helps you do
- One sentence, ~80 characters max

**Choice descriptions:**

- For select/multi_select choices, add descriptions to each choice when the values aren't self-explanatory
- Describe the trade-off or use case for each choice

## YAML Schema

```yaml
name: wizard-name                # kebab-case, required
description: "What this wizard does"  # strongly recommended
command: "base-command"           # required
# flag_style: equals             # "equals" (--flag=val) or "space" (--flag val)

# Version detection (optional)
# version_control:
#   label: "Tool Version"
#   command: tool --version
#   pattern: 'v?(\d+\.\d+\.\d+)'
#   custom_version_command: "npx tool@{{version}}"      # must contain {{version}}
#   custom_version_verify_command: "npm view tool@{{version}} version"  # requires custom_version_command
#   available_versions_command: "tool list-versions"     # shell command listing versions
#   available_versions: "1.0.0 2.0.0 3.0.0"             # or static list

# Version compatibility filters (requires version_control)
# compat:
#   - versions: ">= 8.0"
#     options: [new_feature]
#   - versions: "< 8.0"
#     options: [legacy_option]

options:
  # Input: positional argument (put these first)
  - name: app_name
    type: input
    label: "Project name"
    description: "Directory name for the new project."
    positional: true
    required: true
    validate:
      pattern: '^[a-zA-Z][a-zA-Z0-9_-]*$'
      min_length: 1
      max_length: 100
      message: "Must start with a letter, then letters/digits/hyphens/underscores."

  # Select: single choice from a list
  - name: database
    type: select
    label: "Which database?"
    description: "Database adapter for the application."
    flag: --database
    # flag_style: space           # per-option override
    default: sqlite3
    # required: true
    # allow_none: true            # adds "(none)" choice
    # flag_none: --no-database    # flag when none selected
    choices:
      - value: sqlite3
        label: SQLite
        description: "File-based, no server needed."
      - value: postgresql
        label: PostgreSQL
        description: "Production-ready relational database."
      - mysql                     # shorthand: value=label

  # Select with dynamic choices from shell command
  # - name: profile
  #   type: select
  #   label: "Which AWS profile?"
  #   description: "AWS credentials profile to use."
  #   flag: --profile
  #   choices_from: "aws configure list-profiles"
  #   # Output: one line per choice. Tab-separated: value[\tlabel[\tdescription]]
  #   # Interpolation: choices_from: "cmd --region={{region}}"
  #   #   {{option_name}} inserts a prior answer (shell-escaped)

  # Confirm: yes/no toggle
  - name: skip_git
    type: confirm
    label: "Skip git init?"
    description: "Don't initialize a git repository."
    flag: --skip-git              # shorthand: emit when true, nothing when false
    # flag_true: --skip-git       # explicit true flag
    # flag_false: --no-skip-git   # explicit false flag
    default: false

  # Conditional visibility
  - name: db_host
    type: input
    label: "Database host"
    description: "Hostname for the database server."
    flag: --db-host
    show_when:
      database: [postgresql, mysql]  # OR semantics: shown when postgres OR mysql
    # hide_when:
    #   skip_db: true

  # Multi-select: multiple choices
  - name: features
    type: multi_select
    label: "Which features to include?"
    description: "Optional features to enable in the project."
    flag: --features
    # separator: ","              # --features=a,b  (omit for --features=a --features=b)
    choices:
      - value: api
        label: API mode
        description: "Skip views and assets, API-only app."
      - value: hotwire
        label: Hotwire
        description: "Real-time page updates via Turbo and Stimulus."
```

## Common Patterns

**Positional argument (project name):**

```yaml
- name: app_name
  type: input
  label: "Project name"
  description: "Directory name for the new project."
  positional: true
  required: true
  validate:
    pattern: '^[a-zA-Z][a-zA-Z0-9_-]*$'
    message: "Must start with a letter, then letters/digits/hyphens/underscores."
```

**Skip flag:**

```yaml
- name: skip_tests
  type: confirm
  label: "Skip test framework?"
  description: "Omit test setup and dependencies."
  flag: --skip-test
  default: false
```

**Negation flag (`--no-*`):**

```yaml
- name: load_rc
  type: confirm
  label: "Load RC file?"
  description: "Source the shell RC file on startup."
  flag_false: --no-rc
  default: true
```

**Select with choice descriptions:**

```yaml
- name: css
  type: select
  label: "Which CSS framework?"
  description: "CSS approach for styling the application."
  flag: --css
  default: tailwind
  choices:
    - value: tailwind
      label: Tailwind CSS
      description: "Utility-first framework, fast prototyping."
    - value: bootstrap
      label: Bootstrap
      description: "Component library with opinionated design."
    - value: sass
      label: Sass
      description: "Plain CSS preprocessor, full control."
```

**Conditional visibility:**

```yaml
- name: advanced_mode
  type: confirm
  label: "Configure advanced options?"
  description: "Show additional settings for fine-tuning."
  default: false

- name: log_level
  type: select
  label: "Log level?"
  description: "Verbosity of application logging."
  flag: --log-level
  show_when:
    advanced_mode: true
  choices: [debug, info, warn, error]
```

**Dynamic choices with interpolation:**

```yaml
- name: region
  type: select
  label: "AWS region?"
  flag: --region
  choices_from: "aws ec2 describe-regions --query 'Regions[].RegionName' --output text | tr '\t' '\n'"

- name: vpc
  type: select
  label: "Which VPC?"
  flag: --vpc-id
  choices_from: "aws ec2 describe-vpcs --region={{region}} --query 'Vpcs[].[VpcId,Tags[?Key==`Name`].Value|[0]]' --output text"
```

**Select with none option:**

```yaml
- name: database
  type: select
  label: "Which database?"
  flag: --database
  flag_none: --skip-active-record
  allow_none: true
  choices: [sqlite3, postgresql, mysql]
```

## Rules

**Naming:** wizard `kebab-case`, option names `snake_case`, choice values lowercase (match what the CLI expects).

**Required fields:**

- Top-level: `name`, `command` (required); `description` (always fill)
- Every option: `name`, `type`, `label`, `flag` (unless `positional: true`)
- Select/multi_select: `choices` or `choices_from` (mutually exclusive)

**Ordering:** positional options first, then commonly changed options, then rare ones. Use `show_when` for advanced options.

**Visibility:**

- `show_when`/`hide_when` can only reference options that appear *earlier* in the list
- An option cannot reference itself
- List values use OR: `show_when: { db: [postgres, mysql] }`
- Don't use both `show_when` and `hide_when` on the same key with overlapping values — the option becomes invisible

**Confirms:**

- `flag:` emits the flag when true, nothing when false — use for `--skip-*` style flags
- `flag_true:`/`flag_false:` when both states need flags (e.g. `--color`/`--no-color`)
- Never use both `flag:` and `flag_true:` on the same option

**Choices:**

- Use `choices_from` for dynamic data (containers, branches, versions) — not hardcoded lists
- `choices_from` output: one line per choice, tab-separated `value[\tlabel[\tdescription]]`
- `{{option_name}}` in `choices_from` interpolates a prior answer (auto shell-escaped)
- Use string shorthand (`- sqlite3`) only when value equals label and no description is needed

**Type constraints:**

- `validate` (pattern/min_length/max_length) is only valid on `input`
- `separator` is only valid on `multi_select`
- `allow_none` and `flag_none` are only valid on `select`
- `flag_true`/`flag_false` are only valid on `confirm`
- `choices`/`choices_from` are not valid on `input` or `confirm`
- `required` and `allow_none` are mutually exclusive
- `positional` is mutually exclusive with `flag`/`flag_true`/`flag_false`
- `{{option_name}}` in `choices_from` cannot reference itself or later options

**Defaults:** select defaults to first choice, confirm defaults to `false`,
input defaults to empty. Default values must exist in the choices list.
Multi_select defaults must be a YAML list.
