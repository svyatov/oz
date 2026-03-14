package wizard

import (
	"strings"
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestInputFieldValidate(t *testing.T) {
	tests := []struct {
		name     string
		rule     *config.InputRule
		required bool
		value    string
		wantErr  string
	}{
		{"no_rule_no_required", nil, false, "", ""},
		{"no_rule_with_value", nil, false, "hello", ""},
		{"required_empty", nil, true, "", "This field is required"},
		{"required_filled", nil, true, "hello", ""},
		{"required_custom_message",
			&config.InputRule{Message: "fill this in"},
			true, "", "fill this in"},
		{"pattern_match",
			&config.InputRule{Pattern: `^\d+$`},
			false, "123", ""},
		{"pattern_no_match",
			&config.InputRule{Pattern: `^\d+$`},
			false, "abc", "Must match pattern"},
		{"pattern_custom_message",
			&config.InputRule{Pattern: `^\d+$`, Message: "numbers only"},
			false, "abc", "numbers only"},
		{"min_length_ok",
			&config.InputRule{MinLength: 3},
			false, "abc", ""},
		{"min_length_fail",
			&config.InputRule{MinLength: 3},
			false, "ab", "at least 3"},
		{"max_length_ok",
			&config.InputRule{MaxLength: 5},
			false, "hello", ""},
		{"max_length_fail",
			&config.InputRule{MaxLength: 5},
			false, "toolong", "at most 5"},
		{"empty_value_skips_rule",
			&config.InputRule{Pattern: `^\d+$`},
			false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewInputField(config.Option{Label: "Test", Validate: tt.rule, Required: tt.required})
			f.ti.SetValue(tt.value)
			got := f.validate()
			if tt.wantErr == "" {
				if got != "" {
					t.Errorf("expected no error, got %q", got)
				}
			} else {
				if got == "" {
					t.Errorf("expected error containing %q, got empty", tt.wantErr)
				} else if !containsSubstring(got, tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, got)
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestInputFieldView(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		f := NewInputField(config.Option{
			Label:       "Port",
			Description: "The server port",
		})
		view := stripANSI(f.View())
		if view == "" {
			t.Fatal("expected non-empty view")
		}
		if !strings.Contains(view, "Port") {
			t.Error("expected label 'Port' in view")
		}
	})

	t.Run("with_value", func(t *testing.T) {
		f := NewInputField(config.Option{Label: "Name"})
		f.SetValue(config.StringVal("hello"))
		view := stripANSI(f.View())
		if !strings.Contains(view, "hello") {
			t.Error("expected value 'hello' in view")
		}
	})

	t.Run("with_error", func(t *testing.T) {
		f := NewInputField(config.Option{
			Label:    "Port",
			Required: true,
			Validate: &config.InputRule{Pattern: `^\d+$`, Message: "numbers only"},
		})
		f.ti.SetValue("abc")
		f.validate() // populate errMsg
		f.errMsg = "numbers only"
		view := stripANSI(f.View())
		if !strings.Contains(view, "numbers only") {
			t.Error("expected error message in view")
		}
	})
}
