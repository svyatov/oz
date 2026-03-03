package wizard

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/ui"
)

// InputField wraps bubbles/textinput for free-text entry.
type InputField struct {
	label       string
	description string
	ti          textinput.Model
}

func NewInputField(label, description string) *InputField {
	ti := textinput.New()
	return &InputField{
		label:       label,
		description: description,
		ti:          ti,
	}
}

func (f *InputField) Init() tea.Cmd {
	return f.ti.Focus()
}

func (f *InputField) Update(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "enter", "tab":
		return true, nil
	}

	var cmd tea.Cmd
	f.ti, cmd = f.ti.Update(msg)
	return false, cmd
}

func (f *InputField) View() string {
	var b strings.Builder

	b.WriteString("  " + ui.StepCounter(0, 0) + "  ")
	b.WriteString(ui.FieldTitle(f.label) + "\n")
	if f.description != "" {
		b.WriteString("         " + ui.FieldDesc(f.description) + "\n")
	}
	b.WriteString("\n")
	b.WriteString("    " + f.ti.View() + "\n")

	return b.String()
}

func (f *InputField) Value() any { return f.ti.Value() }

func (f *InputField) SetValue(v any) {
	if v != nil {
		f.ti.SetValue(fmt.Sprintf("%v", v))
	}
}
