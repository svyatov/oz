package config

import (
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
			var errs errorCollector
			validateVisibilityGraph(opts, &errs)
			var msgs []string
			for _, e := range errs {
				msgs = append(msgs, e.Error())
			}
			combined := strings.Join(msgs, "\n")
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

type versionGatingCase struct {
	name    string
	modify  func([]Option) []Option
	wantErr string
}

func versionGatingMinimal() []Option {
	return []Option{
		{
			Name: "database", Type: OptionSelect, Label: "DB",
			Versions: "< 8.0",
			Choices: FlexChoices{
				{Value: "sqlite", Label: "SQLite"},
				{Value: "postgres", Label: "PostgreSQL"},
			},
		},
		{Name: "name", Type: OptionInput, Label: "Name"},
	}
}

func TestValidateVersionGating(t *testing.T) {
	for _, tt := range versionGatingCases() {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.modify(versionGatingMinimal())
			var errs errorCollector
			validateVersionGating(opts, &errs)
			var msgs []string
			for _, e := range errs {
				msgs = append(msgs, e.Error())
			}
			combined := strings.Join(msgs, "\n")
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

func TestConstraintsOverlap(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		// Basic overlapping ranges.
		{"both_gte", ">= 1.0.0", ">= 2.0.0", true},
		{"nested_range", ">= 1.0.0, < 3.0.0", ">= 2.0.0, < 4.0.0", true},
		{"subset", ">= 2.0.0, < 3.0.0", ">= 1.0.0, < 4.0.0", true},
		{"identical", ">= 1.0.0, < 2.0.0", ">= 1.0.0, < 2.0.0", true},

		// Non-overlapping ranges.
		{"disjoint", ">= 1.0.0, < 2.0.0", ">= 3.0.0, < 4.0.0", false},
		{"touching_exclusive", "< 2.0.0", ">= 2.0.0", false},
		{"touching_strict", "< 2.0.0", "> 2.0.0", false},

		// Strict inequality boundaries.
		{"gt_lt_overlap", "> 1.0.0, < 3.0.0", "> 2.0.0, < 4.0.0", true},
		{"gt_lt_touching", "> 1.0.0, < 2.0.0", "> 2.0.0, < 3.0.0", false},

		// Single-direction constraints.
		{"lt_lt_overlap", "< 8.0", "< 7.0", true},
		{"gte_gte_overlap", ">= 5.0", ">= 8.0", true},
		{"lt_gte_no_overlap", "< 5.0", ">= 5.0", false},

		// Large version numbers.
		{"large_major_overlap", ">= 2026.0.0", "< 2027.0.0", true},
		{"large_major_no_overlap", ">= 2027.0.0", "< 2026.0.0", false},
		{"large_minor_overlap", ">= 0.180.0", "< 0.181.0", true},

		// Tilde and caret.
		{"tilde_overlap", "~1.2.0", ">= 1.2.5", true},
		{"tilde_no_overlap", "~1.2.0", ">= 1.3.0", false},
		{"caret_overlap", "^1.0.0", ">= 1.5.0", true},
		{"caret_no_overlap", "^1.0.0", ">= 2.0.0", false},

		// Wildcards.
		{"wildcard_overlap", "1.x", ">= 1.5.0", true},
		{"wildcard_no_overlap", "1.x", ">= 2.0.0", false},

		// OR constraints.
		{"or_overlap_first", ">= 1.0.0, < 2.0.0 || >= 5.0.0", ">= 1.5.0", true},
		{"or_overlap_second", ">= 1.0.0, < 2.0.0 || >= 5.0.0", ">= 6.0.0", true},

		// Invalid constraints.
		{"invalid_a", ">>> bad", ">= 1.0.0", false},
		{"invalid_b", ">= 1.0.0", ">>> bad", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constraintsOverlap(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("constraintsOverlap(%q, %q) = %v, want %v",
					tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func versionGatingCases() []versionGatingCase {
	return []versionGatingCase{
		{"valid_no_versions", func(opts []Option) []Option {
			opts[0].Versions = ""
			return opts
		}, ""},
		{"valid_choice_within_option_range", func(opts []Option) []Option {
			opts[0].Choices = append(opts[0].Choices, Choice{
				Value: "mysql", Label: "MySQL", Versions: "< 7.0",
			})
			return opts
		}, ""},
		{"choice_versions_outside_option_versions", func(opts []Option) []Option {
			opts[0].Choices = append(opts[0].Choices, Choice{
				Value: "mariadb", Label: "MariaDB", Versions: ">= 9.0",
			})
			return opts
		}, "can never match"},
		{"default_in_version_gated_choice", func(opts []Option) []Option {
			opts[0].Versions = ""
			opts[0].Choices = append(opts[0].Choices, Choice{
				Value: "mariadb", Label: "MariaDB", Versions: ">= 8.0",
			})
			defVal := StringVal("mariadb")
			opts[0].Default = &defVal
			return opts
		}, "version-gated choice"},
		{"default_in_ungated_choice_valid", func(opts []Option) []Option {
			defVal := StringVal("sqlite")
			opts[0].Default = &defVal
			return opts
		}, ""},
		{"show_when_refs_version_gated_option", func(opts []Option) []Option {
			opts[1].ShowWhen = Values{"database": StringVal("sqlite")}
			return opts
		}, "show_when references version-gated option"},
		{"hide_when_refs_version_gated_option", func(opts []Option) []Option {
			opts[1].HideWhen = Values{"database": StringVal("sqlite")}
			return opts
		}, "hide_when references version-gated option"},
		{"show_when_refs_ungated_option_valid", func(opts []Option) []Option {
			opts[0].Versions = ""
			opts[1].ShowWhen = Values{"database": StringVal("sqlite")}
			return opts
		}, ""},
		{"both_version_gated_skip_visibility_check", func(opts []Option) []Option {
			opts[1].Versions = "< 8.0"
			opts[1].ShowWhen = Values{"database": StringVal("sqlite")}
			return opts
		}, ""},
	}
}

func graphRefCases() []graphCase {
	return []graphCase{
		{"self_ref_show_when", func(opts []Option) []Option {
			opts[1].ShowWhen = Values{"second": StringVal("x")}
			return opts
		}, "references itself"},
		{"self_ref_hide_when", func(opts []Option) []Option {
			opts[0].HideWhen = Values{"first": StringVal("x")}
			return opts
		}, "references itself"},
		{"forward_ref_show_when", func(opts []Option) []Option {
			opts[0].ShowWhen = Values{"second": StringVal("x")}
			return opts
		}, "appears later"},
		{"forward_ref_hide_when", func(opts []Option) []Option {
			opts[0].HideWhen = Values{"second": StringVal("x")}
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
			opts[1].ShowWhen = Values{"first": StringVal("x")}
			return opts
		}, ""},
		{"chain_backward_valid", func(opts []Option) []Option {
			opts = append(opts, Option{
				Name: "third", Type: OptionInput, Label: "Third",
				ShowWhen: Values{"second": StringVal("val")},
			})
			return opts
		}, ""},
	}
}

func graphConflictCases() []graphCase {
	return []graphCase{
		{"conflict_scalar", func(opts []Option) []Option {
			opts[1].ShowWhen = Values{"first": StringVal("x")}
			opts[1].HideWhen = Values{"first": StringVal("x")}
			return opts
		}, "conflict on key"},
		{"conflict_bool", func(opts []Option) []Option {
			opts[1].ShowWhen = Values{"first": BoolVal(true)}
			opts[1].HideWhen = Values{"first": BoolVal(true)}
			return opts
		}, "conflict on key"},
		{"conflict_show_subset_of_hide_values", func(opts []Option) []Option {
			opts[1].ShowWhen = Values{"first": StringVal("x")}
			opts[1].HideWhen = Values{"first": StringsVal("x", "y")}
			return opts
		}, "conflict on key"},
		{"conflict_hide_subset_of_show_keys", func(opts []Option) []Option {
			opts = append(opts, Option{Name: "third", Type: OptionInput, Label: "Third"})
			opts[2].ShowWhen = Values{"first": StringVal("x"), "second": StringVal("y")}
			opts[2].HideWhen = Values{"first": StringVal("x")}
			return opts
		}, "conflict on key"},
		{"no_conflict_different_values", func(opts []Option) []Option {
			opts[1].ShowWhen = Values{"first": StringVal("x")}
			opts[1].HideWhen = Values{"first": StringVal("y")}
			return opts
		}, ""},
		{"no_conflict_different_keys", func(opts []Option) []Option {
			opts = append(opts, Option{Name: "third", Type: OptionInput, Label: "Third"})
			opts[2].ShowWhen = Values{"first": StringVal("x")}
			opts[2].HideWhen = Values{"second": StringVal("y")}
			return opts
		}, ""},
		{"no_conflict_hide_has_extra_keys", func(opts []Option) []Option {
			opts = append(opts, Option{Name: "third", Type: OptionInput, Label: "Third"})
			opts[2].ShowWhen = Values{"first": StringVal("x")}
			opts[2].HideWhen = Values{"first": StringVal("x"), "second": StringVal("y")}
			return opts
		}, ""},
		{"no_conflict_partial_list_overlap", func(opts []Option) []Option {
			opts[1].ShowWhen = Values{"first": StringsVal("a", "b")}
			opts[1].HideWhen = Values{"first": StringsVal("b", "c")}
			return opts
		}, ""},
	}
}
