package wizard

import (
	"strings"
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestPasswordFieldMasksInput(t *testing.T) {
	f := NewPasswordField(config.Option{Type: config.OptionPassword, Label: "Token"})
	f.Init()
	for _, r := range "abc123" {
		f.Update(key(r))
	}

	view := stripANSI(f.View())
	if strings.Contains(view, "abc123") {
		t.Errorf("password value leaked in view: %q", view)
	}
	if !strings.Contains(view, "******") {
		t.Errorf("expected masked characters in view, got %q", view)
	}
	if got := f.Value().Scalar(); got != "abc123" {
		t.Errorf("Value() should return real secret, got %q", got)
	}
}

func TestPasswordFieldReusesInputValidation(t *testing.T) {
	f := NewPasswordField(config.Option{Type: config.OptionPassword, Label: "Token", Required: true})
	if got := f.validate(); !containsSubstring(got, "required") {
		t.Errorf("expected required error, got %q", got)
	}
	f.ti.SetValue("secret")
	if got := f.validate(); got != "" {
		t.Errorf("expected no error, got %q", got)
	}
}
