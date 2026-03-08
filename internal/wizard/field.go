// Package wizard implements the interactive Bubbletea-based wizard engine and field types.
package wizard

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/ui"
)

// Field is a single wizard step's input component.
type Field interface {
	Init() tea.Cmd
	Update(tea.KeyPressMsg) (submitted bool, cmd tea.Cmd)
	View() string
	Value() any
	SetValue(any)
}

// fieldHeader renders the common title + description block for all field types.
func fieldHeader(label, description string) string {
	var b strings.Builder
	b.WriteString("  " + ui.StepCounter(0, 0) + "  ")
	b.WriteString(ui.FieldTitle(label) + "\n")
	if description != "" {
		b.WriteString("         " + ui.FieldDesc(description) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}
