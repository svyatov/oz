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
	if w.FlagStyle != "" && w.FlagStyle != FlagStyleEquals && w.FlagStyle != FlagStyleSpace {
		add("flag_style must be 'equals' or 'space', got %q", w.FlagStyle)
	}

	validateVersionControl(w.Version, add)

	if len(w.Compat) > 0 && w.Version == nil {
		add("compat requires version_control to be set")
	}

	optionNames := validateOptions(w.Options, add)
	validateReferences(w.Options, w.Compat, optionNames, add)
	validateVisibilityGraph(w.Options, add)

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
	} else if _, err := regexp.Compile(vc.Pattern); err != nil {
		add("version_control.pattern is invalid regex: %v", err)
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

var validOptionTypes = map[OptionType]bool{
	OptionSelect: true, OptionConfirm: true, OptionInput: true, OptionMultiSelect: true,
}

func optionPrefix(i int, name string) string {
	if name != "" {
		return fmt.Sprintf("options[%d] (%s)", i, name)
	}
	return fmt.Sprintf("options[%d]", i)
}

func validateOptions(options []Option, add func(string, ...any)) map[string]bool {
	optionNames := make(map[string]bool)

	for i, o := range options {
		prefix := optionPrefix(i, o.Name)

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
	if o.FlagStyle != "" && o.FlagStyle != FlagStyleEquals && o.FlagStyle != FlagStyleSpace {
		add("%s: flag_style must be 'equals' or 'space', got %q", prefix, o.FlagStyle)
	}

	validateOptionChoices(o, prefix, add)
	validateOptionTypeConstraints(o, prefix, add)
	validateOptionSemantics(o, prefix, add)
}

func validateOptionChoices(o Option, prefix string, add func(string, ...any)) {
	hasChoices := len(o.Choices) > 0
	hasChoicesFrom := o.ChoicesFrom != ""
	if hasChoices && hasChoicesFrom {
		add("%s: choices and choices_from are mutually exclusive", prefix)
	}
	if (o.Type == OptionSelect || o.Type == OptionMultiSelect) && !hasChoices && !hasChoicesFrom {
		add("%s: choices or choices_from required for type %q", prefix, o.Type)
	}
	for j, c := range o.Choices {
		if c.Value == "" {
			add("%s: choices[%d]: value is required", prefix, j)
		}
	}
}

func validateOptionTypeConstraints(o Option, prefix string, add func(string, ...any)) {
	if o.Separator != "" && o.Type != OptionMultiSelect {
		add("%s: separator is only valid for multi_select type", prefix)
	}
	if o.Validate != nil {
		if o.Type != OptionInput {
			add("%s: validate is only valid for input type", prefix)
		}
		validateInputRule(o.Validate, prefix, add)
	}
	if o.Positional && (o.Flag != "" || o.FlagTrue != "" || o.FlagFalse != "") {
		add("%s: positional is mutually exclusive with flag, flag_true, flag_false", prefix)
	}
	validateFieldTypeRestrictions(o, prefix, add)
}

func validateFieldTypeRestrictions(o Option, prefix string, add func(string, ...any)) {
	if o.AllowNone && o.Type != OptionSelect {
		add("%s: allow_none is only valid for select type", prefix)
	}
	if o.FlagTrue != "" && o.Type != OptionConfirm {
		add("%s: flag_true is only valid for confirm type", prefix)
	}
	if o.FlagFalse != "" && o.Type != OptionConfirm {
		add("%s: flag_false is only valid for confirm type", prefix)
	}
	if o.FlagNone != "" && o.Type != OptionSelect {
		add("%s: flag_none is only valid for select type", prefix)
	}
	if len(o.Choices) > 0 && o.Type == OptionInput {
		add("%s: input type does not use choices", prefix)
	}
	if len(o.Choices) > 0 && o.Type == OptionConfirm {
		add("%s: confirm type does not use choices", prefix)
	}
}

func validateInputRule(r *InputRule, prefix string, add func(string, ...any)) {
	if r.Pattern != "" {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			add("%s: validate.pattern is invalid: %v", prefix, err)
		}
	}
	if r.MinLength < 0 {
		add("%s: validate.min_length must not be negative", prefix)
	}
	if r.MaxLength < 0 {
		add("%s: validate.max_length must be positive", prefix)
	}
	if r.MaxLength > 0 && r.MinLength > r.MaxLength {
		add("%s: validate.min_length (%d) exceeds max_length (%d)", prefix, r.MinLength, r.MaxLength)
	}
}

func validateOptionSemantics(o Option, prefix string, add func(string, ...any)) {
	if o.Required && o.AllowNone {
		add("%s: required and allow_none are mutually exclusive", prefix)
	}
	if o.Type == OptionConfirm && o.Flag != "" && o.FlagTrue != "" {
		add("%s: confirm type with both flag and flag_true is ambiguous; use flag or flag_true, not both", prefix)
	}
	validateDefaultInChoices(o, prefix, add)
	validateDuplicateChoices(o, prefix, add)
}

func validateDefaultInChoices(o Option, prefix string, add func(string, ...any)) {
	if len(o.Choices) == 0 || o.Default == nil {
		return
	}
	if o.AllowNone && o.Default.Scalar() == "" {
		return
	}
	choiceSet := make(map[string]bool, len(o.Choices))
	for _, c := range o.Choices {
		choiceSet[c.Value] = true
	}
	if o.Type == OptionMultiSelect {
		if !o.Default.IsStrings() {
			add("%s: default for multi_select must be a list", prefix)
			return
		}
		for _, s := range o.Default.Strings() {
			if !choiceSet[s] {
				add("%s: default value %q is not among the defined choices", prefix, s)
			}
		}
		return
	}
	s := o.Default.Scalar()
	if !choiceSet[s] {
		add("%s: default value %q is not among the defined choices", prefix, s)
	}
}

func validateDuplicateChoices(o Option, prefix string, add func(string, ...any)) {
	seen := make(map[string]bool, len(o.Choices))
	for _, c := range o.Choices {
		if c.Value != "" && seen[c.Value] {
			add("%s: duplicate choice value %q", prefix, c.Value)
		}
		seen[c.Value] = true
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
	for _, match := range ChoicesFromInterpolationRe.FindAllStringSubmatch(o.ChoicesFrom, -1) {
		ref := match[1]
		if !optionNames[ref] {
			add("options[%d] (%s): choices_from interpolation references unknown option %q", idx, o.Name, ref)
		}
	}
}

// ChoicesFromInterpolationRe matches {{name}} but not {{.name}} (Go template syntax).
var ChoicesFromInterpolationRe = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)

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
