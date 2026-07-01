package wizard

import (
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestNumberFieldValidate(t *testing.T) {
	tests := []struct {
		name     string
		min      *float64
		max      *float64
		required bool
		value    string
		wantErr  string
	}{
		{"accepts_int", nil, nil, false, "42", ""},
		{"accepts_float", nil, nil, false, "3.14", ""},
		{"accepts_negative", nil, nil, false, "-5", ""},
		{"rejects_non_numeric", nil, nil, false, "abc", "Must be a number"},
		{"rejects_nan_within_bounds", new(1.0), new(65535.0), false, "NaN", "finite"},
		{"rejects_inf_within_bounds", new(1.0), new(65535.0), false, "Inf", "finite"},
		{"rejects_neg_inf", nil, nil, false, "-Inf", "finite"},
		{"blank_optional_ok", nil, nil, false, "", ""},
		{"blank_required_fails", nil, nil, true, "", "required"},
		{"in_range_ok", new(1.0), new(65535.0), false, "443", ""},
		{"over_max_fails", new(1.0), new(65535.0), false, "70000", "at most 65535"},
		{"under_min_fails", new(1.0), new(65535.0), false, "0", "at least 1"},
		{"min_only_ok", new(10.0), nil, false, "10", ""},
		{"max_only_fails", nil, new(10.0), false, "11", "at most 10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewNumberField(config.Option{
				Type: config.OptionNumber, Label: "N", Min: tt.min, Max: tt.max, Required: tt.required,
			})
			f.ti.SetValue(tt.value)
			got := f.validate()
			if tt.wantErr == "" {
				if got != "" {
					t.Errorf("expected no error, got %q", got)
				}
			} else if !containsSubstring(got, tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, got)
			}
		})
	}
}

// TestNumberFieldEntryHook proves the embedded InputField.Update routes entry
// validation to numeric bounds (via validateFn). Without the hook it would call
// InputField.validate, silently skipping min/max.
func TestNumberFieldEntryHook(t *testing.T) {
	f := NewNumberField(config.Option{Type: config.OptionNumber, Label: "N", Max: new(10.0)})
	f.ti.SetValue("11")
	if got := f.validateEntry(); !containsSubstring(got, "at most 10") {
		t.Errorf("entry hook skipped numeric bounds: got %q", got)
	}
}
