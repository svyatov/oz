package wizard

import (
	"fmt"
	"slices"
	"strings"

	"github.com/svyatov/oz/internal/config"
)

// Answers tracks the current values for each option.
type Answers map[string]any

// EvalShowWhen checks whether an option's show_when conditions are met.
func EvalShowWhen(showWhen map[string]any, answers Answers) bool {
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
func EvalHideWhen(hideWhen map[string]any, answers Answers) bool {
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
func IsVisible(opt config.Option, answers Answers) bool {
	return EvalShowWhen(opt.ShowWhen, answers) && !EvalHideWhen(opt.HideWhen, answers)
}

func valuesMatch(actual, expected any) bool {
	// Expected is a list: OR semantics — match if actual equals any element
	if expectedList, ok := toStringSlice(expected); ok {
		// Actual is also a list (multi_select): match if any expected is IN actual
		if actualList, ok := toStringSlice(actual); ok {
			for _, e := range expectedList {
				if slices.Contains(actualList, e) {
					return true
				}
			}
			return false
		}
		// Actual is scalar: match if actual equals any expected
		actualStr := fmt.Sprintf("%v", actual)
		return slices.Contains(expectedList, actualStr)
	}

	// Expected is scalar, actual is a list (multi_select membership)
	if actualList, ok := toStringSlice(actual); ok {
		return slices.Contains(actualList, fmt.Sprintf("%v", expected))
	}

	// Both scalar: string equality
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
}

// toStringSlice converts []any or []string to []string, returns false if not a slice.
func toStringSlice(v any) ([]string, bool) {
	switch vv := v.(type) {
	case []string:
		return vv, true
	case []any:
		out := make([]string, len(vv))
		for i, item := range vv {
			out[i] = fmt.Sprintf("%v", item)
		}
		return out, true
	}
	return nil, false
}

// FilterPinned removes options that are pinned, returning the filtered list
// and the count of pinned options.
func FilterPinned(options []config.Option, pins map[string]any) (filtered []config.Option, pinCount int) {
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
func VisibleSteps(options []config.Option, answers Answers) []int {
	var indices []int
	for i, o := range options {
		if IsVisible(o, answers) {
			indices = append(indices, i)
		}
	}
	return indices
}

// FormatAnswer renders a field value as a human-readable string for completed-step display.
func FormatAnswer(opt *config.Option, val any) string {
	switch opt.Type {
	case "confirm":
		if b, ok := val.(bool); ok {
			if b {
				return "Yes"
			}
			return "No"
		}
	case "select":
		s := fmt.Sprintf("%v", val)
		for _, c := range opt.Choices {
			if c.Value == s {
				return c.Label
			}
		}
		if s == noneValue {
			return "None"
		}
		return s
	case "multi_select":
		if vals, ok := val.([]string); ok {
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
		}
	}
	return fmt.Sprintf("%v", val)
}
