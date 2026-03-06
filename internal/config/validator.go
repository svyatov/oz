package config

import (
	"fmt"
	"regexp"
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

	validateVersionControl(w.Version, add)

	if len(w.Compat) > 0 && w.Version == nil {
		add("compat requires version_control to be set")
	}

	optionNames := validateOptions(w.Options, add)
	validateReferences(w.Options, w.Compat, optionNames, add)

	return errs
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

		validateOptionFields(o, prefix, add)
	}

	return optionNames
}

func validateOptionFields(o Option, prefix string, add func(string, ...any)) {
	if !validOptionTypes[o.Type] {
		add("%s: type must be one of select, confirm, input, multi_select; got %q", prefix, o.Type)
	}
	if o.Label == "" {
		add("%s: label is required", prefix)
	}
	if o.FlagStyle != "" && o.FlagStyle != "equals" && o.FlagStyle != "space" {
		add("%s: flag_style must be 'equals' or 'space', got %q", prefix, o.FlagStyle)
	}

	validateOptionChoices(o, prefix, add)
	validateOptionTypeConstraints(o, prefix, add)
}

func validateOptionChoices(o Option, prefix string, add func(string, ...any)) {
	hasChoices := len(o.Choices) > 0
	hasChoicesFrom := o.ChoicesFrom != ""
	if hasChoices && hasChoicesFrom {
		add("%s: choices and choices_from are mutually exclusive", prefix)
	}
	if (o.Type == "select" || o.Type == "multi_select") && !hasChoices && !hasChoicesFrom {
		add("%s: choices or choices_from required for type %q", prefix, o.Type)
	}
	for j, c := range o.Choices {
		if c.Value == "" {
			add("%s: choices[%d]: value is required", prefix, j)
		}
	}
}

func validateOptionTypeConstraints(o Option, prefix string, add func(string, ...any)) {
	if o.Separator != "" && o.Type != "multi_select" {
		add("%s: separator is only valid for multi_select type", prefix)
	}
	if o.Validate != nil {
		if o.Type != "input" {
			add("%s: validate is only valid for input type", prefix)
		}
		if o.Validate.Pattern != "" {
			if _, err := regexp.Compile(o.Validate.Pattern); err != nil {
				add("%s: validate.pattern is invalid: %v", prefix, err)
			}
		}
	}
	if o.Positional && (o.Flag != "" || o.FlagTrue != "" || o.FlagFalse != "") {
		add("%s: positional is mutually exclusive with flag, flag_true, flag_false", prefix)
	}
}

func validateReferences(options []Option, compat []CompatEntry, optionNames map[string]bool, add func(string, ...any)) {
	for i, o := range options {
		for ref := range o.ShowWhen {
			if !optionNames[ref] {
				add("options[%d] (%s): show_when references unknown option %q", i, o.Name, ref)
			}
		}
		for ref := range o.HideWhen {
			if !optionNames[ref] {
				add("options[%d] (%s): hide_when references unknown option %q", i, o.Name, ref)
			}
		}
		validateChoicesFromInterpolations(o, i, optionNames, add)
	}
	for i, c := range compat {
		for _, name := range c.Options {
			if !optionNames[name] {
				add("compat[%d]: references unknown option %q", i, name)
			}
		}
	}
}

func validateChoicesFromInterpolations(o Option, idx int, optionNames map[string]bool, add func(string, ...any)) {
	if o.ChoicesFrom == "" {
		return
	}
	// Find {{name}} interpolations (without leading dot, which is Go template syntax like {{.Names}})
	for _, match := range choicesFromInterpolationRe.FindAllStringSubmatch(o.ChoicesFrom, -1) {
		ref := match[1]
		if !optionNames[ref] {
			add("options[%d] (%s): choices_from interpolation references unknown option %q", idx, o.Name, ref)
		}
	}
}

// choicesFromInterpolationRe matches {{name}} but not {{.name}} (Go template syntax).
var choicesFromInterpolationRe = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)

// ChoicesFromInterpolationRe returns the compiled regex for choices_from interpolation.
func ChoicesFromInterpolationRe() *regexp.Regexp {
	return choicesFromInterpolationRe
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
