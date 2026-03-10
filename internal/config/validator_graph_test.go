package config

import (
	"fmt"
	"strings"
	"testing"
)

type graphCase struct {
	name    string
	modify  func([]Option) []Option
	wantErr string
}

func graphMinimal() []Option {
	return []Option{
		{Name: "first", Type: OptionSelect, Label: "First", ChoicesFrom: "echo a"},
		{Name: "second", Type: OptionInput, Label: "Second"},
	}
}

func TestValidateVisibilityGraph(t *testing.T) {
	for _, tt := range graphCases() {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.modify(graphMinimal())
			var errs []string
			add := func(msg string, args ...any) {
				errs = append(errs, fmt.Sprintf(msg, args...))
			}
			validateVisibilityGraph(opts, add)
			combined := strings.Join(errs, "\n")
			if tt.wantErr == "" {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got:\n%s", combined)
				}
			} else {
				if !strings.Contains(combined, tt.wantErr) {
					t.Errorf("expected error containing %q, got:\n%s", tt.wantErr, combined)
				}
			}
		})
	}
}

func graphCases() []graphCase {
	cases := graphRefCases()
	cases = append(cases, graphConflictCases()...)
	return cases
}

func graphRefCases() []graphCase {
	return []graphCase{
		{"self_ref_show_when", func(opts []Option) []Option {
			opts[1].ShowWhen = map[string]any{"second": "x"}
			return opts
		}, "references itself"},
		{"self_ref_hide_when", func(opts []Option) []Option {
			opts[0].HideWhen = map[string]any{"first": "x"}
			return opts
		}, "references itself"},
		{"forward_ref_show_when", func(opts []Option) []Option {
			opts[0].ShowWhen = map[string]any{"second": "x"}
			return opts
		}, "appears later"},
		{"forward_ref_hide_when", func(opts []Option) []Option {
			opts[0].HideWhen = map[string]any{"second": "x"}
			return opts
		}, "appears later"},
		{"forward_ref_choices_from", func(opts []Option) []Option {
			opts[0].ChoicesFrom = "cmd --profile={{second}}"
			return opts
		}, "appears later"},
		{"choices_from_self_interpolation", func(opts []Option) []Option {
			opts[0].ChoicesFrom = "cmd --db={{first}}"
			return opts
		}, "references itself"},
		{"backward_ref_valid", func(opts []Option) []Option {
			opts[1].ShowWhen = map[string]any{"first": "x"}
			return opts
		}, ""},
		{"chain_backward_valid", func(opts []Option) []Option {
			opts = append(opts, Option{
				Name: "third", Type: OptionInput, Label: "Third",
				ShowWhen: map[string]any{"second": "val"},
			})
			return opts
		}, ""},
	}
}

func graphConflictCases() []graphCase {
	return []graphCase{
		{"conflict_scalar", func(opts []Option) []Option {
			opts[1].ShowWhen = map[string]any{"first": "x"}
			opts[1].HideWhen = map[string]any{"first": "x"}
			return opts
		}, "conflict on key"},
		{"conflict_bool", func(opts []Option) []Option {
			opts[1].ShowWhen = map[string]any{"first": true}
			opts[1].HideWhen = map[string]any{"first": true}
			return opts
		}, "conflict on key"},
		{"conflict_show_subset_of_hide_values", func(opts []Option) []Option {
			opts[1].ShowWhen = map[string]any{"first": "x"}
			opts[1].HideWhen = map[string]any{"first": []any{"x", "y"}}
			return opts
		}, "conflict on key"},
		{"conflict_hide_subset_of_show_keys", func(opts []Option) []Option {
			opts = append(opts, Option{Name: "third", Type: OptionInput, Label: "Third"})
			opts[2].ShowWhen = map[string]any{"first": "x", "second": "y"}
			opts[2].HideWhen = map[string]any{"first": "x"}
			return opts
		}, "conflict on key"},
		{"no_conflict_different_values", func(opts []Option) []Option {
			opts[1].ShowWhen = map[string]any{"first": "x"}
			opts[1].HideWhen = map[string]any{"first": "y"}
			return opts
		}, ""},
		{"no_conflict_different_keys", func(opts []Option) []Option {
			opts = append(opts, Option{Name: "third", Type: OptionInput, Label: "Third"})
			opts[2].ShowWhen = map[string]any{"first": "x"}
			opts[2].HideWhen = map[string]any{"second": "y"}
			return opts
		}, ""},
		{"no_conflict_hide_has_extra_keys", func(opts []Option) []Option {
			opts = append(opts, Option{Name: "third", Type: OptionInput, Label: "Third"})
			opts[2].ShowWhen = map[string]any{"first": "x"}
			opts[2].HideWhen = map[string]any{"first": "x", "second": "y"}
			return opts
		}, ""},
		{"no_conflict_partial_list_overlap", func(opts []Option) []Option {
			opts[1].ShowWhen = map[string]any{"first": []any{"a", "b"}}
			opts[1].HideWhen = map[string]any{"first": []any{"b", "c"}}
			return opts
		}, ""},
	}
}
