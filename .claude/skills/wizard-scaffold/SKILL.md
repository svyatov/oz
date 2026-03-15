---
name: wizard-scaffold
description: Scaffold a new oz wizard YAML config with proper structure, option types, and validation. Use when the user asks to create, scaffold, or generate a wizard config, wants to wrap a CLI tool with an interactive wizard, or mentions oz wizards, wizard YAML, or interactive CLI prompts.
---

# Wizard Scaffold

Generate an oz wizard YAML configuration from a CLI tool's help output.
Wraps any CLI tool with interactive Bubbletea prompts so users can build commands step by step.

## Workflow

1. Ask the user for:
   - **Wizard name** (kebab-case, e.g. `rails-new`)
   - **Base command** (e.g. `rails new`, `docker run`)

2. **Run `oz generate <tool> [subcommand...]`** to produce the raw scaffold YAML.
   This auto-detects help format (GNU, Cobra, kubectl, Clap, argparse, Thor, man page),
   extracts flags, types, defaults, and enum values.

   If the tool is not installed, ask the user for help text and pipe it:
   `echo "<help text>" | oz generate --stdin --name <wizard-name>`

3. **Enhance the scaffold** — this is where the skill adds value beyond raw parsing:
   - Rewrite labels into question form ("Dry run" → "Dry run?")
   - Write self-sufficient descriptions for every option and choice (see guidelines below)
   - Add `show_when`/`hide_when` for options that become irrelevant given earlier answers
   - Convert `--skip-*` flags: `confirm` with `flag: --skip-*`, label "Skip X?"
   - Convert `--no-*` flags: `confirm` with `flag_false: --no-*`, label uses the positive form
   - Add `allow_none`/`flag_none` on selects where the CLI supports skipping a category entirely
   - Add `validate` rules to input fields where patterns are obvious from help text
   - Add `version_control` if the tool has multiple installable versions
   - Reorder: positional args first, then commonly changed options, then rare ones

4. **Review with the user.** Ask if they want to:
   - Remove or reorder options
   - Add version control (multi-version tool support)
   - Add more conditional visibility rules
   - Use dynamic choices (`choices_from`) for any select

5. **Write the YAML file** to `~/.config/oz/wizards/<name>.yml`
   (or `$OZ_CONFIG_DIR/wizards/<name>.yml` if set).

6. **Validate** with `go run ./cmd/oz/ validate <name>` (or `oz validate <name>` if binary exists).

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

## Writing Self-Sufficient Descriptions

The user picking from wizard options may not know the tool well.
Every description — especially for choices — should let them make an informed decision without reading docs.

### The Three Questions

Every choice description should answer enough of these that a user can decide "should I pick this?":

1. **What is it?** — what the thing actually is or does (one clause)
2. **When to pick it?** — who benefits from this choice and under what circumstances
3. **How does it differ?** — the key trade-off vs other choices in the same list

Not every description needs all three explicitly, but together the choices in a list should make the differences clear.

### Examples

**Select choices — weak vs strong:**

```yaml
# Weak: user still doesn't know which to pick
choices:
  - value: sqlite3
    description: "File-based, no server needed."
  - value: postgresql
    description: "Production-ready relational database."

# Strong: user can pick confidently
choices:
  - value: sqlite3
    label: SQLite
    description: "Single-file database, zero setup. Best for solo dev and prototypes. No concurrent multi-server access."
  - value: postgresql
    label: PostgreSQL
    description: "Full-featured relational DB with JSON support and strong concurrency. Pick for production or team development."
  - value: mysql
    label: MySQL
    description: "Widely deployed relational DB with broad hosting support. Pick if your team or hosting already uses MySQL."
```

**Confirm (skip) — weak vs strong:**

```yaml
# Weak: user doesn't know what they'd lose
- name: skip_active_storage
  description: "Don't include the file upload framework."

# Strong: user understands the trade-off
- name: skip_active_storage
  description: "Active Storage handles file uploads to S3, GCS, or local disk. Skip if your app won't handle file uploads."
```

**Input — weak vs strong:**

```yaml
# Weak: user doesn't know the format or purpose
- name: template
  description: "Path to a template."

# Strong: user knows what to provide and why
- name: template
  description: "Path or URL to a template file (e.g. ~/template.rb or https://...). Applied after generation to customize the project."
```

### Default Rules

- **Describe every choice.** Use the object form (`value`/`label`/`description`) for all choices.
  String shorthand (`- sqlite3`) is only appropriate for universally obvious values (country codes, `json`/`yaml`/`csv`).
- **Confirm descriptions should explain what the feature does**, not just "Don't include X."
  The user needs to understand the cost of enabling/skipping.
- **Keep descriptions to 1-2 sentences.** Aim for under 120 characters but allow up to 160 when needed for clarity.

## Label Guidelines

- Use question form: "Which database?" not "Database"
- For `--skip-*` flags: "Skip X?" — matches the flag semantics
- For `--no-*` flags: use the positive form. `--no-rc` → label "Load RC file?" with `flag_false: --no-rc`
- Be specific about the effect: "Use TypeScript instead of JavaScript?" not "TypeScript?"
- Keep under ~60 characters

## YAML Schema

```yaml
name: wizard-name                # kebab-case, required
description: "What this wizard does"  # strongly recommended
command: "base-command"           # required
# flag_style: equals             # "equals" (--flag=val) or "space" (--flag val)

# Version detection (optional — required if any option/choice uses `versions`)
# version_control:
#   label: "Tool Version"
#   command: tool --version
#   pattern: 'v?(\d+\.\d+\.\d+)'         # regex with capture group
#   custom_version_command: "npx tool@{{version}}"      # must contain {{version}}
#   custom_version_verify_command: "npm view tool@{{version}} version"  # requires custom_version_command
#   available_versions_command: "tool list-versions"     # shell command listing versions
#   available_versions: "1.0.0 2.0.0 3.0.0"             # or static space-separated list

options:
  # Input: positional argument (put these first)
  - name: app_name
    type: input
    label: "Project name"
    description: "Directory name for the new project. Created in the current working directory."
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
    # flag_none: --skip-database  # flag emitted when none selected
    # versions: ">= 2.0"         # only show if detected version matches (semver constraint)
    choices:
      - value: sqlite3
        label: SQLite
        description: "Single-file database, zero setup. Best for solo dev and prototypes."
      - value: postgresql
        label: PostgreSQL
        description: "Full-featured relational DB with advanced queries and strong concurrency. Pick for production."
        # versions: ">= 5.0"    # choice-level version gating
      - value: mysql
        label: MySQL
        description: "Widely deployed relational DB with broad hosting support. Pick if your infra already uses MySQL."

  # Select with dynamic choices from shell command
  # - name: profile
  #   type: select
  #   label: "Which AWS profile?"
  #   description: "AWS credentials profile to use for this deployment."
  #   flag: --profile
  #   choices_from: "aws configure list-profiles"
  #   # Output: one line per choice. Tab-separated: value[\tlabel[\tdescription]]
  #   # Interpolation: choices_from: "cmd --region={{region}}"
  #   #   {{option_name}} inserts a prior answer (auto shell-escaped)

  # Confirm: yes/no toggle
  - name: skip_git
    type: confirm
    label: "Skip git init?"
    description: "Git init creates a repository with .gitignore and .gitattributes. Skip if the project is already in a git repo."
    flag: --skip-git              # shorthand: emit when true, nothing when false
    # flag_true: --skip-git       # explicit true flag (can't use both flag and flag_true)
    # flag_false: --no-skip-git   # explicit false flag
    default: false

  # Conditional visibility
  - name: db_host
    type: input
    label: "Database host?"
    description: "Hostname for the database server. Only needed for client-server databases."
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
        description: "API-only app: skip views, assets, and browser-related middleware."
      - value: hotwire
        label: Hotwire
        description: "Real-time page updates via Turbo and Stimulus. Adds SPA-like interactivity without writing JS."
```

## Common Patterns

**Positional argument (project name):**

```yaml
- name: app_name
  type: input
  label: "Project name"
  description: "Directory name for the new project. Created in the current working directory."
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
  description: "Test framework provides unit and integration test scaffolding. Skip if you use an external test suite."
  flag: --skip-test
  default: false
```

**Negation flag (`--no-*`):**

```yaml
- name: load_rc
  type: confirm
  label: "Load RC file?"
  description: "RC file applies default options from ~/.toolrc on every run. Disable to ignore it."
  flag_false: --no-rc
  default: true
```

**Select with none option:**

```yaml
- name: database
  type: select
  label: "Which database?"
  description: "Database adapter for the application."
  flag: --database
  flag_none: --skip-active-record
  allow_none: true
  choices:
    - value: sqlite3
      label: SQLite
      description: "Single-file database, zero setup. Best for solo dev and prototypes."
    - value: postgresql
      label: PostgreSQL
      description: "Full-featured relational DB with strong concurrency. Pick for production."
    - value: mysql
      label: MySQL
      description: "Widely deployed relational DB. Pick if your infra already uses MySQL."
```

**Conditional visibility (`hide_when`):**

```yaml
- name: api
  type: confirm
  label: "API-only application?"
  description: "API-only mode strips views, assets, and browser middleware. Pick for backend services or SPAs with a separate frontend."
  flag: --api
  default: false

- name: css
  type: select
  label: "Which CSS framework?"
  description: "CSS approach for styling the application."
  flag: --css
  hide_when:
    api: true           # irrelevant for API-only apps
  choices:
    - value: tailwind
      label: Tailwind CSS
      description: "Utility classes composed in HTML. Fast to prototype, no pre-built components. Requires a build step."
    - value: bootstrap
      label: Bootstrap
      description: "Pre-built components (navbars, modals, cards) with consistent design. Pick for admin panels or quick UIs."
    - value: sass
      label: Sass
      description: "CSS preprocessor with variables and nesting, no pre-built components. Pick for full design control."
```

**Conditional visibility (`show_when`):**

```yaml
- name: database
  type: select
  label: "Which database?"
  flag: --database
  choices:
    - value: sqlite3
      label: SQLite
      description: "File-based, zero setup. Best for local dev."
    - value: postgresql
      label: PostgreSQL
      description: "Production-ready with strong concurrency."
    - value: mysql
      label: MySQL
      description: "Widely deployed, broad hosting support."

- name: db_host
  type: input
  label: "Database host?"
  description: "Hostname for the database server. SQLite uses a local file and doesn't need this."
  flag: --db-host
  show_when:
    database: [postgresql, mysql]  # OR semantics: shown for either
```

**Dynamic choices with interpolation:**

```yaml
- name: region
  type: select
  label: "AWS region?"
  description: "Region where resources will be deployed."
  flag: --region
  choices_from: "aws ec2 describe-regions --query 'Regions[].RegionName' --output text | tr '\t' '\n'"

- name: vpc
  type: select
  label: "Which VPC?"
  description: "Virtual network to deploy into. Filtered by the selected region."
  flag: --vpc-id
  choices_from: "aws ec2 describe-vpcs --region={{region}} --query 'Vpcs[].[VpcId,Tags[?Key==`Name`].Value|[0]]' --output text"
```

## Rules

**Naming:** wizard name `kebab-case`, option names `snake_case`, choice values lowercase (match CLI expectations).

**Required fields:**

- Top-level: `name`, `command` (required); `description` (always fill)
- Every option: `name`, `type`, `label`
- Select/multi_select: `choices` or `choices_from` (mutually exclusive)
- Every option needs a `flag` (unless `positional: true`)

**Ordering:** positional options first, then commonly changed options, then rare ones.
Use `hide_when` for options irrelevant given earlier answers (e.g. CSS when API-only).
Don't gate behind `advanced_mode` — users pin frequently-used options instead.

**Visibility:**

- `show_when`/`hide_when` can only reference options that appear *earlier* in the list
- An option cannot reference itself
- List values use OR: `show_when: { db: [postgres, mysql] }`
- Don't combine `show_when` and `hide_when` on the same key with overlapping values

**Confirms:**

- `flag:` emits the flag when true, nothing when false — use for `--skip-*` style flags
- `flag_true:`/`flag_false:` when both states need flags (e.g. `--color`/`--no-color`)
- Never use both `flag:` and `flag_true:` on the same option

**Choices:**

- Default to object form with descriptions for every choice.
  String shorthand (`- value`) is only for universally obvious values where no description helps.
- Use `choices_from` for dynamic data (containers, branches, versions) — not hardcoded lists
- `choices_from` output: one line per choice, tab-separated `value[\tlabel[\tdescription]]`
- `{{option_name}}` in `choices_from` interpolates a prior answer (auto shell-escaped, cannot reference self or later options)

**Type constraints:**

- `validate` (pattern/min_length/max_length) — only on `input`
- `separator` — only on `multi_select`
- `allow_none` and `flag_none` — only on `select`
- `flag_true`/`flag_false` — only on `confirm`
- `choices`/`choices_from` — not valid on `input` or `confirm`
- `required` and `allow_none` — mutually exclusive
- `positional` — mutually exclusive with `flag`/`flag_true`/`flag_false`

**Versions:** `versions` on an option or choice is a semver constraint.
Supports `>=`, `<=`, `>`, `<`, `=`, `!=`, tilde (`~1.2`),
caret (`^2.0`), wildcards (`1.2.x`), hyphen ranges (`1.2 - 1.4`),
and `||` OR. Requires `version_control` at the wizard level.

**Defaults:** select defaults to first choice, confirm defaults to `false`,
input defaults to empty. Default values must exist in the choices list.
Multi_select defaults must be a YAML list.
