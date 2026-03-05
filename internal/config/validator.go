package config

import (
	"fmt"
	"strings"
)

// Validate checks a wizard config for errors.
func Validate(w *Wizard) []error {
	var errs []error
	add := func(msg string, args ...any) {
		errs = append(errs, fmt.Errorf(msg, args...))
	}

	if w.Name == "" {
		add("name is required")
	}
	if w.Command == "" {
		add("command is required")
	}
	if w.FlagStyle != "" && w.FlagStyle != "equals" && w.FlagStyle != "space" {
		add("flag_style must be 'equals' or 'space', got %q", w.FlagStyle)
	}

	validateArgs(w.Args, add)
	validateVersionControl(w.Version, add)

	if len(w.Compat) > 0 && w.Version == nil {
		add("compat requires version_control to be set")
	}

	optionNames := validateOptions(w.Options, add)
	validateReferences(w.Options, w.Compat, optionNames, add)

	return errs
}

func validateArgs(args []Arg, add func(string, ...any)) {
	for i, a := range args {
		if a.Name == "" {
			add("args[%d]: name is required", i)
		}
		if a.Position < 1 {
			add("args[%d] (%s): position must be >= 1", i, a.Name)
		}
	}
}

func validateVersionControl(vc *VersionControl, add func(string, ...any)) {
	if vc == nil {
		return
	}
	if vc.Command == "" {
		add("version_control.command is required")
	}
	if vc.Pattern == "" {
		add("version_control.pattern is required")
	}
	if vc.CustomVersionCmd != "" && !strings.Contains(vc.CustomVersionCmd, "{{version}}") {
		add("version_control.custom_version_command must contain {{version}}")
	}
	if vc.CustomVersionVerify != "" {
		if vc.CustomVersionCmd == "" {
			add("version_control.custom_version_verify_command requires custom_version_command")
		}
		if !strings.Contains(vc.CustomVersionVerify, "{{version}}") {
			add("version_control.custom_version_verify_command must contain {{version}}")
		}
	}
}

var validOptionTypes = map[string]bool{
	"select": true, "confirm": true, "input": true, "multi_select": true,
}

func validateOptions(options []Option, add func(string, ...any)) map[string]bool {
	optionNames := make(map[string]bool)

	for i, o := range options {
		prefix := fmt.Sprintf("options[%d]", i)
		if o.Name != "" {
			prefix = fmt.Sprintf("options[%d] (%s)", i, o.Name)
		}

		if o.Name == "" {
			add("%s: name is required", prefix)
		} else if optionNames[o.Name] {
			add("%s: duplicate option name", prefix)
		}
		optionNames[o.Name] = true

		if !validOptionTypes[o.Type] {
			add("%s: type must be one of select, confirm, input, multi_select; got %q", prefix, o.Type)
		}
		if o.Label == "" {
			add("%s: label is required", prefix)
		}
		if o.FlagStyle != "" && o.FlagStyle != "equals" && o.FlagStyle != "space" {
			add("%s: flag_style must be 'equals' or 'space', got %q", prefix, o.FlagStyle)
		}
		if (o.Type == "select" || o.Type == "multi_select") && len(o.Choices) == 0 {
			add("%s: choices are required for type %q", prefix, o.Type)
		}
		for j, c := range o.Choices {
			if c.Value == "" {
				add("%s: choices[%d]: value is required", prefix, j)
			}
		}
	}

	return optionNames
}

func validateReferences(options []Option, compat []CompatEntry, optionNames map[string]bool, add func(string, ...any)) {
	for i, o := range options {
		for ref := range o.ShowWhen {
			if !optionNames[ref] {
				add("options[%d] (%s): show_when references unknown option %q", i, o.Name, ref)
			}
		}
	}
	for i, c := range compat {
		for _, name := range c.Options {
			if !optionNames[name] {
				add("compat[%d]: references unknown option %q", i, name)
			}
		}
	}
}

// FormatErrors formats validation errors as a single string.
func FormatErrors(errs []error) string {
	if len(errs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, e := range errs {
		fmt.Fprintf(&b, "  - %s\n", e)
	}
	return b.String()
}
