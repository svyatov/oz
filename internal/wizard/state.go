package wizard

import (
	"fmt"
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

func valuesMatch(actual, expected any) bool {
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
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

// VisibleSteps returns the indices of options that pass show_when evaluation.
func VisibleSteps(options []config.Option, answers Answers) []int {
	var indices []int
	for i, o := range options {
		if EvalShowWhen(o.ShowWhen, answers) {
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
