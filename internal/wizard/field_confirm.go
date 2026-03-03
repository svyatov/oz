package wizard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/ui"
)

// ConfirmField is a Yes/No toggle with y/n shortcuts.
type ConfirmField struct {
	label       string
	description string
	cursor      int // 0=Yes, 1=No
	value       bool
}

func NewConfirmField(label, description string) *ConfirmField {
	return &ConfirmField{label: label, description: description}
}

func (f *ConfirmField) Init() tea.Cmd { return nil }

func (f *ConfirmField) Update(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		f.cursor = 0
	case "down", "j":
		f.cursor = 1
	case "enter", "tab":
		f.value = f.cursor == 0
		return true, nil
	}

	switch msg.Code {
	case 'y':
		f.value = true
		return true, nil
	case 'n':
		f.value = false
		return true, nil
	case '1':
		f.cursor = 0
		f.value = true
		return true, nil
	case '2':
		f.cursor = 1
		f.value = false
		return true, nil
	}

	return false, nil
}

func (f *ConfirmField) View() string {
	var b strings.Builder

	b.WriteString("  " + ui.StepCounter(0, 0) + "  ")
	b.WriteString(ui.FieldTitle(f.label) + "\n")
	if f.description != "" {
		b.WriteString("         " + ui.FieldDesc(f.description) + "\n")
	}
	b.WriteString("\n")

	items := []string{"Yes", "No"}
	for i, item := range items {
		active := i == f.cursor
		num := ui.NumberGutter(i+1, active)

		var cursor string
		if active {
			cursor = " " + ui.Cursor() + " "
		} else {
			cursor = "   "
		}

		styledLabel := ui.ChoiceLabel(item, active)
		fmt.Fprintf(&b, "   %s%s  %s\n", cursor, num, styledLabel)
	}

	return b.String()
}

func (f *ConfirmField) Value() any { return f.value }

func (f *ConfirmField) SetValue(v any) {
	switch val := v.(type) {
	case bool:
		f.value = val
	default:
		f.value = fmt.Sprintf("%v", v) == "true"
	}
	if f.value {
		f.cursor = 0
	} else {
		f.cursor = 1
	}
}
