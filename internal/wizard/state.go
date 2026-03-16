package wizard

import (
	"slices"
	"strings"

	"github.com/svyatov/oz/internal/config"
)

// EvalShowWhen checks whether an option's show_when conditions are met.
func EvalShowWhen(showWhen config.Values, answers config.Values) bool {
	if len(showWhen) == 0 {
		return true
	}
	for name, expected := range showWhen {
		actual, ok := answers[name]
		if !ok {
			return false
		}
		if !valuesMatch(actual, expected) {
			return false
		}
	}
	return true
}

// EvalHideWhen checks whether an option's hide_when conditions are met.
func EvalHideWhen(hideWhen config.Values, answers config.Values) bool {
	if len(hideWhen) == 0 {
		return false
	}
	for name, expected := range hideWhen {
		actual, ok := answers[name]
		if !ok {
			return false
		}
		if !valuesMatch(actual, expected) {
			return false
		}
	}
	return true
}

// IsVisible returns true if the option should be shown given current answers.
func IsVisible(opt config.Option, answers config.Values) bool {
	return EvalShowWhen(opt.ShowWhen, answers) && !EvalHideWhen(opt.HideWhen, answers)
}

func valuesMatch(actual, expected config.FieldValue) bool {
	// Expected is a list: OR semantics — match if actual equals any element
	if expected.IsStrings() {
		// Actual is also a list (multi_select): match if any expected is IN actual
		if actual.IsStrings() {
			for _, e := range expected.Strings() {
				if slices.Contains(actual.Strings(), e) {
					return true
				}
			}
			return false
		}
		// Actual is scalar: match if actual equals any expected
		return slices.Contains(expected.Strings(), actual.Scalar())
	}

	// Expected is scalar, actual is a list (multi_select membership)
	if actual.IsStrings() {
		return slices.Contains(actual.Strings(), expected.Scalar())
	}

	// Both scalar: string equality
	return actual.Scalar() == expected.Scalar()
}

// MissingRequired returns labels of visible required options that have no value.
func MissingRequired(options []config.Option, values config.Values) []string {
	var missing []string
	for _, opt := range options {
		if !opt.Required || !IsVisible(opt, values) {
			continue
		}
		if _, has := values[opt.Name]; !has {
			missing = append(missing, opt.Label)
		}
	}
	return missing
}

// FilterPinned removes options that are pinned, returning the filtered list
// and the count of pinned options.
func FilterPinned(options []config.Option, pins config.Values) (filtered []config.Option, pinCount int) {
	for _, o := range options {
		if _, pinned := pins[o.Name]; pinned {
			pinCount++
			continue
		}
		filtered = append(filtered, o)
	}
	return
}

// VisibleSteps returns the indices of options that pass show_when and hide_when evaluation.
func VisibleSteps(options []config.Option, answers config.Values) []int {
	var indices []int
	for i, o := range options {
		if IsVisible(o, answers) {
			indices = append(indices, i)
		}
	}
	return indices
}

// FormatAnswer renders a field value as a human-readable string for completed-step display.
func FormatAnswer(opt *config.Option, val config.FieldValue) string {
	switch opt.Type {
	case config.OptionConfirm:
		if val.Bool() {
			return "Yes"
		}
		return "No"
	case config.OptionSelect:
		s := val.Scalar()
		for _, c := range opt.Choices {
			if c.Value == s {
				return c.Label
			}
		}
		if s == config.NoneValue {
			return "None"
		}
		return s
	case config.OptionMultiSelect:
		vals := val.Strings()
		labels := make([]string, 0, len(vals))
		choiceMap := make(map[string]string, len(opt.Choices))
		for _, c := range opt.Choices {
			choiceMap[c.Value] = c.Label
		}
		for _, v := range vals {
			if label, ok := choiceMap[v]; ok {
				labels = append(labels, label)
			} else {
				labels = append(labels, v)
			}
		}
		return strings.Join(labels, ", ")
	case config.OptionInput:
		// fall through to default
	}
	return val.Scalar()
}

// resolveDefault finds the best default for an option from the given sources (in priority order).
func resolveDefault(opt *config.Option, sources ...config.Values) *config.FieldValue {
	for _, src := range sources {
		if v, ok := src[opt.Name]; ok {
			return &v
		}
	}
	if opt.Default != nil {
		return opt.Default
	}
	switch opt.Type {
	case config.OptionSelect:
		if len(opt.Choices) > 0 {
			v := config.StringVal(opt.Choices[0].Value)
			return &v
		}
	case config.OptionConfirm:
		v := config.BoolVal(false)
		return &v
	case config.OptionInput:
		v := config.StringVal("")
		return &v
	case config.OptionMultiSelect:
		// no default for multi_select
	}
	return nil
}
