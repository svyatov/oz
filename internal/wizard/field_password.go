package wizard

import (
	"charm.land/bubbles/v2/textinput"

	"github.com/svyatov/oz/internal/config"
)

// PasswordField is an InputField whose entry is masked on screen.
// It reuses InputField's validation (required/length/pattern); only the echo
// mode differs. The real value is still returned by Value() — masking is a
// display concern handled by the command renderer and FormatAnswer.
type PasswordField struct {
	*InputField
}

// NewPasswordField creates a masked-entry field from a config option.
func NewPasswordField(opt config.Option) *PasswordField {
	f := NewInputField(opt)
	f.ti.EchoMode = textinput.EchoPassword
	return &PasswordField{InputField: f}
}
