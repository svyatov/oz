package config

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	minimal := func() *Wizard {
		return &Wizard{
			Name:    "test",
			Command: "cmd",
			Options: []Option{
				{Name: "opt1", Type: "input", Label: "Opt 1"},
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
	return append(cases, versionControlCases()...)
}

func baseCases() []validationCase {
	return []validationCase{
		{"valid_minimal", func(w *Wizard) {}, ""},
		{"missing_name", func(w *Wizard) { w.Name = "" }, "name is required"},
		{"missing_command", func(w *Wizard) { w.Command = "" }, "command is required"},
		{"invalid_flag_style", func(w *Wizard) { w.FlagStyle = "bad" }, "flag_style must be"},
		{"arg_missing_name", func(w *Wizard) {
			w.Args = []Arg{{Position: 1}}
		}, "name is required"},
		{"arg_position_zero", func(w *Wizard) {
			w.Args = []Arg{{Name: "a", Position: 0}}
		}, "position must be >= 1"},
		{"compat_without_detect", func(w *Wizard) {
			w.Compat = []CompatEntry{{Versions: ">= 1.0", Options: []string{"opt1"}}}
		}, "compat requires version_control"},
		{"duplicate_option_name", func(w *Wizard) {
			w.Options = append(w.Options, Option{Name: "opt1", Type: "input", Label: "Dup"})
		}, "duplicate option name"},
		{"invalid_option_type", func(w *Wizard) {
			w.Options[0].Type = "bad"
		}, "type must be one of"},
		{"missing_label", func(w *Wizard) {
			w.Options[0].Label = ""
		}, "label is required"},
		{"select_without_choices", func(w *Wizard) {
			w.Options[0].Type = "select"
			w.Options[0].Choices = nil
		}, "choices are required"},
		{"multi_select_without_choices", func(w *Wizard) {
			w.Options[0].Type = "multi_select"
			w.Options[0].Choices = nil
		}, "choices are required"},
		{"choice_empty_value", func(w *Wizard) {
			w.Options[0].Type = "select"
			w.Options[0].Choices = []Choice{{Value: "", Label: "x"}}
		}, "value is required"},
		{"show_when_unknown_option", func(w *Wizard) {
			w.Options[0].ShowWhen = map[string]any{"nonexistent": true}
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
		if got := w.EffectiveFlagStyle(); got != "equals" {
			t.Errorf("got %q, want equals", got)
		}
	})
	t.Run("wizard_space", func(t *testing.T) {
		w := &Wizard{FlagStyle: "space"}
		if got := w.EffectiveFlagStyle(); got != "space" {
			t.Errorf("got %q, want space", got)
		}
	})
	t.Run("option_inherits", func(t *testing.T) {
		o := &Option{}
		if got := o.EffectiveFlagStyle("space"); got != "space" {
			t.Errorf("got %q, want space", got)
		}
	})
	t.Run("option_overrides", func(t *testing.T) {
		o := &Option{FlagStyle: "equals"}
		if got := o.EffectiveFlagStyle("space"); got != "equals" {
			t.Errorf("got %q, want equals", got)
		}
	})
}
