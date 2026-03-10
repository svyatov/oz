package config

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func fvptr(v FieldValue) *FieldValue { return &v }

func TestValidate(t *testing.T) {
	minimal := func() *Wizard {
		return &Wizard{
			Name:    "test",
			Command: "cmd",
			Options: []Option{
				{Name: "opt1", Type: OptionInput, Label: "Opt 1"},
			},
		}
	}

	for _, tt := range validationCases() {
		t.Run(tt.name, func(t *testing.T) {
			w := minimal()
			tt.modify(w)
			errs := Validate(w)
			combined := FormatErrors(errs)
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

type validationCase struct {
	name    string
	modify  func(*Wizard)
	wantErr string
}

func validationCases() []validationCase {
	cases := baseCases()
	cases = append(cases, versionControlCases()...)
	cases = append(cases, newFeatureCases()...)
	cases = append(cases, semanticCases()...)
	return cases
}

func baseCases() []validationCase {
	return []validationCase{
		{"valid_minimal", func(w *Wizard) {}, ""},
		{"missing_name", func(w *Wizard) { w.Name = "" }, "name is required"},
		{"missing_command", func(w *Wizard) { w.Command = "" }, "command is required"},
		{"invalid_flag_style", func(w *Wizard) { w.FlagStyle = "bad" }, "flag_style must be"},
		{"compat_without_detect", func(w *Wizard) {
			w.Compat = []CompatEntry{{Versions: ">= 1.0", Options: []string{"opt1"}}}
		}, "compat requires version_control"},
		{"duplicate_option_name", func(w *Wizard) {
			w.Options = append(w.Options, Option{Name: "opt1", Type: OptionInput, Label: "Dup"})
		}, "duplicate option name"},
		{"invalid_option_type", func(w *Wizard) {
			w.Options[0].Type = "bad"
		}, "type must be one of"},
		{"missing_label", func(w *Wizard) {
			w.Options[0].Label = ""
		}, "label is required"},
		{"select_without_choices", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = nil
		}, "choices or choices_from required"},
		{"multi_select_without_choices", func(w *Wizard) {
			w.Options[0].Type = OptionMultiSelect
			w.Options[0].Choices = nil
		}, "choices or choices_from required"},
		{"choice_empty_value", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "", Label: "x"}}
		}, "value is required"},
		{"show_when_unknown_option", func(w *Wizard) {
			w.Options[0].ShowWhen = Values{"nonexistent": BoolVal(true)}
		}, "references unknown option"},
		{"compat_unknown_option", func(w *Wizard) {
			w.Version = &VersionControl{Command: "cmd", Pattern: `(\d+)`}
			w.Compat = []CompatEntry{{Versions: ">= 1.0", Options: []string{"nope"}}}
		}, "references unknown option"},
	}
}

func versionControlCases() []validationCase {
	return []validationCase{
		{"version_control_missing_command", func(w *Wizard) {
			w.Version = &VersionControl{Pattern: `(\d+)`}
		}, "version_control.command is required"},
		{"version_control_missing_pattern", func(w *Wizard) {
			w.Version = &VersionControl{Command: "cmd --version"}
		}, "version_control.pattern is required"},
		{"custom_version_cmd_missing_placeholder", func(w *Wizard) {
			w.Version = &VersionControl{Command: "cmd", Pattern: `(\d+)`, CustomVersionCmd: "cmd new"}
		}, "must contain {{version}}"},
		{"custom_version_verify_missing_placeholder", func(w *Wizard) {
			w.Version = &VersionControl{
				Command: "cmd", Pattern: `(\d+)`,
				CustomVersionCmd: "cmd _{{version}}_ new", CustomVersionVerify: "cmd --version",
			}
		}, "custom_version_verify_command must contain {{version}}"},
		{"custom_version_verify_without_cmd", func(w *Wizard) {
			w.Version = &VersionControl{
				Command: "cmd", Pattern: `(\d+)`,
				CustomVersionVerify: "cmd _{{version}}_ --version",
			}
		}, "requires custom_version_command"},
		{"valid_full_version_control", func(w *Wizard) {
			w.Version = &VersionControl{
				Command: "cmd --version", Pattern: `(\d+\.\d+\.\d+)`,
				CustomVersionCmd:    "cmd _{{version}}_ new",
				CustomVersionVerify: "cmd _{{version}}_ --version",
				AvailVersions:       "7.2.1, 7.1.0",
			}
		}, ""},
	}
}

func newFeatureCases() []validationCase {
	cases := choicesCases()
	cases = append(cases, constraintCases()...)
	return cases
}

func choicesCases() []validationCase {
	return []validationCase{
		{"choices_from_valid", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].ChoicesFrom = "ls *.txt"
		}, ""},
		{"choices_and_choices_from_conflict", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "a"}}
			w.Options[0].ChoicesFrom = "ls"
		}, "choices and choices_from are mutually exclusive"},
		{"separator_on_non_multi_select", func(w *Wizard) {
			w.Options[0].Separator = ","
		}, "separator is only valid for multi_select"},
		{"separator_on_multi_select_valid", func(w *Wizard) {
			w.Options[0].Type = OptionMultiSelect
			w.Options[0].Separator = ","
			w.Options[0].ChoicesFrom = "echo a"
		}, ""},
		{"validate_on_non_input", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "a"}}
			w.Options[0].Validate = &InputRule{Pattern: ".*"}
		}, "validate is only valid for input"},
		{"validate_bad_pattern", func(w *Wizard) {
			w.Options[0].Validate = &InputRule{Pattern: "[invalid"}
		}, "validate.pattern is invalid"},
		{"validate_valid", func(w *Wizard) {
			w.Options[0].Validate = &InputRule{Pattern: `^\d+$`, Message: "must be number"}
		}, ""},
	}
}

func constraintCases() []validationCase {
	return []validationCase{
		{"positional_conflicts_with_flag", func(w *Wizard) {
			w.Options[0].Positional = true
			w.Options[0].Flag = "--name"
		}, "positional is mutually exclusive with flag"},
		{"positional_conflicts_with_flag_true", func(w *Wizard) {
			w.Options[0].Type = OptionConfirm
			w.Options[0].Positional = true
			w.Options[0].FlagTrue = "--yes"
		}, "positional is mutually exclusive with flag"},
		{"positional_valid", func(w *Wizard) {
			w.Options[0].Positional = true
		}, ""},
		{"hide_when_unknown_option", func(w *Wizard) {
			w.Options[0].HideWhen = Values{"nonexistent": BoolVal(true)}
		}, "hide_when references unknown option"},
		{"hide_when_valid", func(w *Wizard) {
			w.Options = append(w.Options, Option{
				Name: "opt2", Type: OptionInput, Label: "Opt 2",
				HideWhen: Values{"opt1": StringVal("x")},
			})
		}, ""},
		{"choices_from_unknown_interpolation", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].ChoicesFrom = "cmd --profile={{unknown}}"
		}, "choices_from interpolation references unknown option"},
		{"choices_from_dot_interpolation_ignored", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].ChoicesFrom = "docker images --format '{{.Names}}'"
		}, ""},
	}
}

func semanticCases() []validationCase {
	cases := semanticDefaultCases()
	cases = append(cases, semanticConstraintCases()...)
	cases = append(cases, semanticTypeCases()...)
	return cases
}

func semanticDefaultCases() []validationCase {
	return []validationCase{
		{"default_not_in_choices", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}
			w.Options[0].Default = fvptr(StringVal("c"))
		}, "not among the defined choices"},
		{"default_in_choices_valid", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}
			w.Options[0].Default = fvptr(StringVal("a"))
		}, ""},
		{"default_multi_select_not_in_choices", func(w *Wizard) {
			w.Options[0].Type = OptionMultiSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}
			w.Options[0].Default = fvptr(StringsVal("a", "z"))
		}, "not among the defined choices"},
		{"default_multi_select_all_valid", func(w *Wizard) {
			w.Options[0].Type = OptionMultiSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}
			w.Options[0].Default = fvptr(StringsVal("a", "b"))
		}, ""},
		{"default_multi_select_scalar_rejected", func(w *Wizard) {
			w.Options[0].Type = OptionMultiSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}}
			w.Options[0].Default = fvptr(StringVal("a"))
		}, "must be a list"},
		{"default_with_choices_from_skipped", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].ChoicesFrom = "echo a b c"
			w.Options[0].Default = fvptr(StringVal("anything"))
		}, ""},
		{"default_empty_with_allow_none_valid", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}}
			w.Options[0].AllowNone = true
			w.Options[0].Default = fvptr(StringVal(""))
		}, ""},
		{"duplicate_choice_values", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "x", Label: "X"}, {Value: "x", Label: "X2"}}
		}, "duplicate choice value"},
	}
}

func semanticConstraintCases() []validationCase {
	return []validationCase{
		{"required_and_allow_none", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}}
			w.Options[0].Required = true
			w.Options[0].AllowNone = true
		}, "mutually exclusive"},
		{"confirm_with_choices", func(w *Wizard) {
			w.Options[0].Type = OptionConfirm
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}}
		}, "does not use choices"},
		{"confirm_flag_and_flag_true", func(w *Wizard) {
			w.Options[0].Type = OptionConfirm
			w.Options[0].Flag = "--verbose"
			w.Options[0].FlagTrue = "--yes"
		}, "ambiguous"},
		{"confirm_flag_only_valid", func(w *Wizard) {
			w.Options[0].Type = OptionConfirm
			w.Options[0].Flag = "--verbose"
		}, ""},
		{"input_rule_min_gt_max", func(w *Wizard) {
			w.Options[0].Validate = &InputRule{MinLength: 10, MaxLength: 5}
		}, "exceeds max_length"},
		{"input_rule_negative_max", func(w *Wizard) {
			w.Options[0].Validate = &InputRule{MaxLength: -1}
		}, "must be positive"},
		{"input_rule_negative_min", func(w *Wizard) {
			w.Options[0].Validate = &InputRule{MinLength: -1}
		}, "must not be negative"},
		{"input_rule_min_eq_max_valid", func(w *Wizard) {
			w.Options[0].Validate = &InputRule{MinLength: 5, MaxLength: 5}
		}, ""},
		{"version_control_invalid_pattern", func(w *Wizard) {
			w.Version = &VersionControl{Command: "cmd", Pattern: "[invalid"}
		}, "invalid regex"},
	}
}

func semanticTypeCases() []validationCase {
	return []validationCase{
		{"allow_none_on_input", func(w *Wizard) {
			w.Options[0].AllowNone = true
		}, "only valid for select"},
		{"allow_none_on_confirm", func(w *Wizard) {
			w.Options[0].Type = OptionConfirm
			w.Options[0].AllowNone = true
		}, "only valid for select"},
		{"allow_none_on_multi_select", func(w *Wizard) {
			w.Options[0].Type = OptionMultiSelect
			w.Options[0].ChoicesFrom = "echo a"
			w.Options[0].AllowNone = true
		}, "only valid for select"},
		{"flag_true_on_select", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}}
			w.Options[0].FlagTrue = "--yes"
		}, "only valid for confirm"},
		{"flag_false_on_input", func(w *Wizard) {
			w.Options[0].FlagFalse = "--no"
		}, "only valid for confirm"},
		{"flag_false_on_select", func(w *Wizard) {
			w.Options[0].Type = OptionSelect
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}}
			w.Options[0].FlagFalse = "--no"
		}, "only valid for confirm"},
		{"flag_true_on_multi_select", func(w *Wizard) {
			w.Options[0].Type = OptionMultiSelect
			w.Options[0].ChoicesFrom = "echo a"
			w.Options[0].FlagTrue = "--yes"
		}, "only valid for confirm"},
		{"flag_none_on_input", func(w *Wizard) {
			w.Options[0].FlagNone = "--skip"
		}, "only valid for select"},
		{"flag_none_on_confirm", func(w *Wizard) {
			w.Options[0].Type = OptionConfirm
			w.Options[0].FlagNone = "--skip"
		}, "only valid for select"},
		{"flag_none_on_multi_select", func(w *Wizard) {
			w.Options[0].Type = OptionMultiSelect
			w.Options[0].ChoicesFrom = "echo a"
			w.Options[0].FlagNone = "--skip"
		}, "only valid for select"},
		{"choices_on_input", func(w *Wizard) {
			w.Options[0].Choices = FlexChoices{{Value: "a", Label: "A"}}
		}, "does not use choices"},
	}
}

func TestFormatErrors(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got := FormatErrors(nil); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
	t.Run("multiple", func(t *testing.T) {
		errs := []error{
			&validationError{"a"},
			&validationError{"b"},
		}
		got := FormatErrors(errs)
		if !strings.Contains(got, "a") || !strings.Contains(got, "b") {
			t.Errorf("expected both errors, got: %s", got)
		}
	})
}

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

func TestEffectiveCommand(t *testing.T) {
	t.Run("no_version_control", func(t *testing.T) {
		w := &Wizard{Command: "rails new"}
		if got := w.EffectiveCommand("7.1.0"); got != "rails new" {
			t.Errorf("got %q, want %q", got, "rails new")
		}
	})
	t.Run("no_template", func(t *testing.T) {
		w := &Wizard{
			Command: "rails new",
			Version: &VersionControl{Command: "rails --version", Pattern: `(\d+)`},
		}
		if got := w.EffectiveCommand("7.1.0"); got != "rails new" {
			t.Errorf("got %q, want %q", got, "rails new")
		}
	})
	t.Run("with_template", func(t *testing.T) {
		w := &Wizard{
			Command: "rails new",
			Version: &VersionControl{
				Command: "rails --version", Pattern: `(\d+)`,
				CustomVersionCmd: "rails _{{version}}_ new",
			},
		}
		if got := w.EffectiveCommand("7.1.0"); got != "rails _7.1.0_ new" {
			t.Errorf("got %q, want %q", got, "rails _7.1.0_ new")
		}
	})
	t.Run("empty_version", func(t *testing.T) {
		w := &Wizard{
			Command: "rails new",
			Version: &VersionControl{
				Command: "rails --version", Pattern: `(\d+)`,
				CustomVersionCmd: "rails _{{version}}_ new",
			},
		}
		if got := w.EffectiveCommand(""); got != "rails new" {
			t.Errorf("got %q, want %q", got, "rails new")
		}
	})
}

func TestEffectiveFlagStyle(t *testing.T) {
	t.Run("wizard_default", func(t *testing.T) {
		w := &Wizard{}
		if got := w.EffectiveFlagStyle(); got != FlagStyleEquals {
			t.Errorf("got %q, want equals", got)
		}
	})
	t.Run("wizard_space", func(t *testing.T) {
		w := &Wizard{FlagStyle: FlagStyleSpace}
		if got := w.EffectiveFlagStyle(); got != FlagStyleSpace {
			t.Errorf("got %q, want space", got)
		}
	})
	t.Run("option_inherits", func(t *testing.T) {
		o := &Option{}
		if got := o.EffectiveFlagStyle(FlagStyleSpace); got != FlagStyleSpace {
			t.Errorf("got %q, want space", got)
		}
	})
	t.Run("option_overrides", func(t *testing.T) {
		o := &Option{FlagStyle: FlagStyleEquals}
		if got := o.EffectiveFlagStyle(FlagStyleSpace); got != FlagStyleEquals {
			t.Errorf("got %q, want equals", got)
		}
	})
}

func TestFlexChoicesUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    []Choice
		wantErr bool
	}{
		{
			"string_shorthand",
			"choices:\n  - sqlite\n  - postgres",
			[]Choice{{Value: "sqlite", Label: "sqlite"}, {Value: "postgres", Label: "postgres"}},
			false,
		},
		{
			"full_object",
			"choices:\n  - value: mysql\n    label: MySQL 8\n    description: Popular",
			[]Choice{{Value: "mysql", Label: "MySQL 8", Description: "Popular"}},
			false,
		},
		{
			"mixed",
			"choices:\n  - sqlite\n  - value: mysql\n    label: MySQL 8",
			[]Choice{{Value: "sqlite", Label: "sqlite"}, {Value: "mysql", Label: "MySQL 8"}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out struct {
				Choices FlexChoices `yaml:"choices"`
			}
			err := yamlUnmarshal([]byte(tt.yaml), &out)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(out.Choices) != len(tt.want) {
				t.Fatalf("got %d choices, want %d", len(out.Choices), len(tt.want))
			}
			for i, c := range out.Choices {
				if c.Value != tt.want[i].Value || c.Label != tt.want[i].Label || c.Description != tt.want[i].Description {
					t.Errorf("[%d] got %+v, want %+v", i, c, tt.want[i])
				}
			}
		})
	}
}

func yamlUnmarshal(data []byte, v any) error {
	if err := yaml.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}
