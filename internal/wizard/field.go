// Package wizard implements the interactive Bubbletea-based wizard engine and field types.
package wizard

import (
	tea "charm.land/bubbletea/v2"
)

// Field is a single wizard step's input component.
type Field interface {
	Init() tea.Cmd
	Update(tea.KeyPressMsg) (submitted bool, cmd tea.Cmd)
	View() string
	Value() any
	SetValue(any)
}
