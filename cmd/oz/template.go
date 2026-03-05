package main

import "fmt"

const wizardTemplateFmt = `# Wizard: %[1]s
name: %[1]s
description: ""
command: ""

# Flag style: "equals" (--flag=value) or "space" (--flag value)
# flag_style: equals

# Positional arguments
# args:
#   - name: target
#     label: Target path
#     required: true
#     position: 1

# Version detection and multi-version support
# version_control:
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
    choices:
      - value: value1
        label: Value 1
        description: First option
      - value: value2
        label: Value 2
        description: Second option

  # Confirm: yes/no toggle
  # - name: example_confirm
  #   type: confirm
  #   label: Enable feature?
  #   flag_true: --enable
  #   flag_false: --disable
  #   default: false

  # Input: free-text entry
  # - name: example_input
  #   type: input
  #   label: Enter a value
  #   flag: --value
  #   default: ""
  #   required: true

  # Multi-select: multiple choices
  # - name: example_multi
  #   type: multi_select
  #   label: Pick several
  #   flag: --items
  #   choices:
  #     - value: a
  #       label: Option A
  #     - value: b
  #       label: Option B
`

func wizardTemplate(name string) string {
	return fmt.Sprintf(wizardTemplateFmt, name)
}
