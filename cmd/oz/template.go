package main

import "fmt"

const wizardTemplateFmt = `# Wizard: %[1]s
name: %[1]s
description: ""
command: ""

# Flag style: "equals" (--flag=value) or "space" (--flag value)
# flag_style: equals

# Version detection and multi-version support
# version_control:
#   label: "Tool Version"
#   command: mytool --version
#   pattern: 'v?(\d+\.\d+\.\d+)'
#   custom_version_command: "npx mytool@{{version}}"
#   custom_version_verify_command: "npm view mytool versions --json"
#   available_versions_command: "ls /opt/mytool/versions"
#   available_versions: "1.0.0 2.0.0 3.0.0"

# Version compatibility filters
# compat:
#   - versions: ">=2.0.0"
#     options: [new_feature]
#   - versions: "<2.0.0"
#     options: [legacy_option]

options:
  # Select: single choice from a list
  - name: example_select
    type: select
    label: Pick one
    # description: Shown below the label
    flag: --example
    # flag_style: space          # per-option override
    # default: value1
    # required: true
    # allow_none: true           # add "(none)" choice
    # flag_none: --no-example    # flag when none is selected
    # show_when:                 # conditional visibility
    #   other_option: some_value
    # hide_when:                 # inverse of show_when
    #   other_option: some_value
    choices:
      - value: value1
        label: Value 1
        description: First option
      - value2                   # string shorthand: value=label

  # Select with dynamic choices from shell command
  # - name: container
  #   type: select
  #   label: Select container
  #   flag: --name
  #   choices_from: "docker ps --format '{{.Names}}\t{{.Status}}'"
  #   # Each line = one choice. Tab-separated: value[\tlabel[\tdescription]]

  # Confirm: yes/no toggle
  # - name: example_confirm
  #   type: confirm
  #   label: Enable feature?
  #   flag: --enable              # shorthand: emit when true
  #   # flag_true: --enable       # explicit true flag
  #   # flag_false: --disable     # explicit false flag
  #   # default: false

  # Input: free-text entry with validation
  # - name: example_input
  #   type: input
  #   label: Enter a value
  #   flag: --value
  #   # default: ""
  #   # required: true            # prevents empty submission
  #   # positional: true          # emit as bare arg, not flag
  #   # validate:
  #   #   pattern: '^\d+$'
  #   #   min_length: 1
  #   #   max_length: 100
  #   #   message: "Must be a number"

  # Multi-select: multiple choices
  # - name: example_multi
  #   type: multi_select
  #   label: Pick several
  #   flag: --items
  #   # separator: ","            # --items=a,b instead of --items=a --items=b
  #   choices:
  #     - value: a
  #       label: Option A
  #     - value: b
  #       label: Option B
`

func wizardTemplate(name string) string {
	return fmt.Sprintf(wizardTemplateFmt, name)
}
