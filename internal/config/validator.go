package config

import (
	"fmt"
	"regexp"
	"strings"

	semver "github.com/Masterminds/semver/v3"
)

// errorCollector accumulates validation errors.
type errorCollector []error

func (c *errorCollector) addf(msg string, args ...any) {
	*c = append(*c, fmt.Errorf(msg, args...))
}

// Validate checks a wizard config for errors.
func Validate(w *Wizard) []error {
	var errs errorCollector

	if w.Name == "" {
		errs.addf("name is required")
	}
	if w.Command == "" {
		errs.addf("command is required")
	}
	if w.FlagStyle != "" && w.FlagStyle != FlagStyleEquals && w.FlagStyle != FlagStyleSpace {
		errs.addf("flag_style must be 'equals' or 'space', got %q", w.FlagStyle)
	}

	validateVersionControl(w.Version, &errs)

	optionNames := validateOptions(w.Options, &errs)
	validateReferences(w.Options, optionNames, &errs)
	validateVersionsConstraints(w, &errs)
	validateVersionGating(w.Options, &errs)
	validateVisibilityGraph(w.Options, &errs)

	return []error(errs)
}

func validateVersionControl(vc *VersionControl, errs *errorCollector) {
	if vc == nil {
		return
	}
	if vc.Command == "" {
		errs.addf("version_control.command is required")
	}
	if vc.Pattern == "" {
		errs.addf("version_control.pattern is required")
	} else if _, err := regexp.Compile(vc.Pattern); err != nil {
		errs.addf("version_control.pattern is invalid regex: %v", err)
	}
	if vc.CustomVersionCmd != "" && !strings.Contains(vc.CustomVersionCmd, "{{version}}") {
		errs.addf("version_control.custom_version_command must contain {{version}}")
	}
	if vc.CustomVersionVerify != "" {
		if vc.CustomVersionCmd == "" {
			errs.addf("version_control.custom_version_verify_command requires custom_version_command")
		}
		if !strings.Contains(vc.CustomVersionVerify, "{{version}}") {
			errs.addf("version_control.custom_version_verify_command must contain {{version}}")
		}
	}
}

func optionPrefix(i int, name string) string {
	if name != "" {
		return fmt.Sprintf("options[%d] (%s)", i, name)
	}
	return fmt.Sprintf("options[%d]", i)
}

func validateOptions(options []Option, errs *errorCollector) map[string]bool {
	optionNames := make(map[string]bool)

	for i, o := range options {
		prefix := optionPrefix(i, o.Name)

		if o.Name == "" {
			errs.addf("%s: name is required", prefix)
		} else if optionNames[o.Name] {
			errs.addf("%s: duplicate option name", prefix)
		}
		optionNames[o.Name] = true

		validateOptionFields(o, prefix, errs)
	}

	return optionNames
}

func validateOptionFields(o Option, prefix string, errs *errorCollector) {
	if !o.Type.IsValid() {
		errs.addf("%s: type must be one of select, confirm, input, multi_select; got %q", prefix, o.Type)
	}
	if o.Label == "" {
		errs.addf("%s: label is required", prefix)
	}
	if o.FlagStyle != "" && o.FlagStyle != FlagStyleEquals && o.FlagStyle != FlagStyleSpace {
		errs.addf("%s: flag_style must be 'equals' or 'space', got %q", prefix, o.FlagStyle)
	}

	validateOptionChoices(o, prefix, errs)
	validateOptionTypeConstraints(o, prefix, errs)
	validateOptionSemantics(o, prefix, errs)
}

func validateOptionChoices(o Option, prefix string, errs *errorCollector) {
	hasChoices := len(o.Choices) > 0
	hasChoicesFrom := o.ChoicesFrom != ""
	if hasChoices && hasChoicesFrom {
		errs.addf("%s: choices and choices_from are mutually exclusive", prefix)
	}
	if (o.Type == OptionSelect || o.Type == OptionMultiSelect) && !hasChoices && !hasChoicesFrom {
		errs.addf("%s: choices or choices_from required for type %q", prefix, o.Type)
	}
	for j, c := range o.Choices {
		if c.Value == "" {
			errs.addf("%s: choices[%d]: value is required", prefix, j)
		}
	}
}

func validateOptionTypeConstraints(o Option, prefix string, errs *errorCollector) {
	if o.Separator != "" && o.Type != OptionMultiSelect {
		errs.addf("%s: separator is only valid for multi_select type", prefix)
	}
	if o.Validate != nil {
		if o.Type != OptionInput {
			errs.addf("%s: validate is only valid for input type", prefix)
		}
		validateInputRule(o.Validate, prefix, errs)
	}
	if o.Positional && (o.Flag != "" || o.FlagTrue != "" || o.FlagFalse != "") {
		errs.addf("%s: positional is mutually exclusive with flag, flag_true, flag_false", prefix)
	}
	validateFieldTypeRestrictions(o, prefix, errs)
}

func validateFieldTypeRestrictions(o Option, prefix string, errs *errorCollector) {
	if o.AllowNone && o.Type != OptionSelect {
		errs.addf("%s: allow_none is only valid for select type", prefix)
	}
	if o.FlagTrue != "" && o.Type != OptionConfirm {
		errs.addf("%s: flag_true is only valid for confirm type", prefix)
	}
	if o.FlagFalse != "" && o.Type != OptionConfirm {
		errs.addf("%s: flag_false is only valid for confirm type", prefix)
	}
	if o.FlagNone != "" && o.Type != OptionSelect {
		errs.addf("%s: flag_none is only valid for select type", prefix)
	}
	if len(o.Choices) > 0 && o.Type == OptionInput {
		errs.addf("%s: input type does not use choices", prefix)
	}
	if len(o.Choices) > 0 && o.Type == OptionConfirm {
		errs.addf("%s: confirm type does not use choices", prefix)
	}
}

func validateInputRule(r *InputRule, prefix string, errs *errorCollector) {
	if r.Pattern != "" {
		if _, err := regexp.Compile(r.Pattern); err != nil {
			errs.addf("%s: validate.pattern is invalid: %v", prefix, err)
		}
	}
	if r.MinLength < 0 {
		errs.addf("%s: validate.min_length must not be negative", prefix)
	}
	if r.MaxLength < 0 {
		errs.addf("%s: validate.max_length must be positive", prefix)
	}
	if r.MaxLength > 0 && r.MinLength > r.MaxLength {
		errs.addf("%s: validate.min_length (%d) exceeds max_length (%d)", prefix, r.MinLength, r.MaxLength)
	}
}

func validateOptionSemantics(o Option, prefix string, errs *errorCollector) {
	if o.Required && o.AllowNone {
		errs.addf("%s: required and allow_none are mutually exclusive", prefix)
	}
	if o.Type == OptionConfirm && o.Flag != "" && o.FlagTrue != "" {
		errs.addf("%s: confirm type with both flag and flag_true is ambiguous; use flag or flag_true, not both", prefix)
	}
	validateDefaultInChoices(o, prefix, errs)
	validateDuplicateChoices(o, prefix, errs)
}

func validateDefaultInChoices(o Option, prefix string, errs *errorCollector) {
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
			errs.addf("%s: default for multi_select must be a list", prefix)
			return
		}
		for _, s := range o.Default.Strings() {
			if !choiceSet[s] {
				errs.addf("%s: default value %q is not among the defined choices", prefix, s)
			}
		}
		return
	}
	s := o.Default.Scalar()
	if !choiceSet[s] {
		errs.addf("%s: default value %q is not among the defined choices", prefix, s)
	}
}

func validateDuplicateChoices(o Option, prefix string, errs *errorCollector) {
	seen := make(map[string]bool, len(o.Choices))
	for _, c := range o.Choices {
		if c.Value != "" && seen[c.Value] {
			errs.addf("%s: duplicate choice value %q", prefix, c.Value)
		}
		seen[c.Value] = true
	}
}

func validateReferences(options []Option, optionNames map[string]bool, errs *errorCollector) {
	for i, o := range options {
		for ref := range o.ShowWhen {
			if !optionNames[ref] {
				errs.addf("options[%d] (%s): show_when references unknown option %q", i, o.Name, ref)
			}
		}
		for ref := range o.HideWhen {
			if !optionNames[ref] {
				errs.addf("options[%d] (%s): hide_when references unknown option %q", i, o.Name, ref)
			}
		}
		validateChoicesFromInterpolations(o, i, optionNames, errs)
	}
}

// validateVersionsConstraints checks that versions fields are valid and consistent.
func validateVersionsConstraints(w *Wizard, errs *errorCollector) {
	hasVersions := false
	for _, o := range w.Options {
		if o.Versions != "" {
			hasVersions = true
			if _, err := semver.NewConstraint(o.Versions); err != nil {
				prefix := optionPrefix(indexOf(w.Options, o.Name), o.Name)
				errs.addf("%s: invalid versions constraint %q: %v", prefix, o.Versions, err)
			}
		}
		for j, c := range o.Choices {
			if c.Versions != "" {
				hasVersions = true
				prefix := optionPrefix(indexOf(w.Options, o.Name), o.Name)
				if _, err := semver.NewConstraint(c.Versions); err != nil {
					errs.addf("%s: choices[%d]: invalid versions constraint %q: %v", prefix, j, c.Versions, err)
				}
			}
		}
	}
	if hasVersions && w.Version == nil {
		errs.addf("versions constraints require version_control to be set")
	}
}

func indexOf(options []Option, name string) int {
	for i, o := range options {
		if o.Name == name {
			return i
		}
	}
	return -1
}


func validateChoicesFromInterpolations(o Option, idx int, optionNames map[string]bool, errs *errorCollector) {
	if o.ChoicesFrom == "" {
		return
	}
	// Find {{name}} interpolations (without leading dot, which is Go template syntax like {{.Names}})
	for _, match := range ChoicesFromInterpolationRe.FindAllStringSubmatch(o.ChoicesFrom, -1) {
		ref := match[1]
		if !optionNames[ref] {
			errs.addf("options[%d] (%s): choices_from interpolation references unknown option %q", idx, o.Name, ref)
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
