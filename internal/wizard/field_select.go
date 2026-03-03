package wizard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

const noneValue = "_none"

// SelectField is a numbered-list select with ↑↓jk navigation and 1–9 instant select.
type SelectField struct {
	label       string
	description string
	choices     []config.Choice
	allowNone   bool
	cursor      int
	value       string
}

// NewSelectField creates a SelectField from a config option.
func NewSelectField(opt config.Option) *SelectField {
	return &SelectField{
		label:       opt.Label,
		description: opt.Description,
		choices:     opt.Choices,
		allowNone:   opt.AllowNone,
	}
}

func (f *SelectField) Init() tea.Cmd { return nil }

func (f *SelectField) Update(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	n := f.itemCount()

	switch msg.String() {
	case "up", "k":
		f.cursor = (f.cursor - 1 + n) % n
	case "down", "j":
		f.cursor = (f.cursor + 1) % n
	case "enter", "tab":
		f.value = f.valueAt(f.cursor)
		return true, nil
	}

	// Number keys 1–9: select and submit
	if msg.Code >= '1' && msg.Code <= '9' {
		idx := int(msg.Code-'0') - 1
		if idx < n {
			f.cursor = idx
			f.value = f.valueAt(idx)
			return true, nil
		}
	}

	return false, nil
}

func (f *SelectField) View() string {
	var b strings.Builder

	b.WriteString("  " + ui.StepCounter(0, 0) + "  ")
	b.WriteString(ui.FieldTitle(f.label) + "\n")
	if f.description != "" {
		b.WriteString("         " + ui.FieldDesc(f.description) + "\n")
	}
	b.WriteString("\n")

	// Find max label width for column alignment
	maxLabel := 0
	for _, c := range f.choices {
		if len(c.Label) > maxLabel {
			maxLabel = len(c.Label)
		}
	}
	if f.allowNone && len("None") > maxLabel {
		maxLabel = len("None")
	}

	n := f.itemCount()
	for i := range n {
		active := i == f.cursor
		num := ui.NumberGutter(i+1, active)

		var cursor string
		if active {
			cursor = " " + ui.Cursor() + " "
		} else {
			cursor = "   "
		}

		label, desc := f.itemAt(i)
		styledLabel := ui.ChoiceLabel(label, active)
		pad := strings.Repeat(" ", maxLabel-len(label))

		line := fmt.Sprintf("   %s%s  %s%s", cursor, num, styledLabel, pad)
		if desc != "" {
			line += "   " + ui.ChoiceDesc(desc)
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (f *SelectField) Value() any { return f.value }

func (f *SelectField) SetValue(v any) {
	s := fmt.Sprintf("%v", v)
	f.value = s
	for i := range f.itemCount() {
		if f.valueAt(i) == s {
			f.cursor = i
			return
		}
	}
}

func (f *SelectField) itemCount() int {
	n := len(f.choices)
	if f.allowNone {
		n++
	}
	return n
}

func (f *SelectField) valueAt(i int) string {
	if i < len(f.choices) {
		return f.choices[i].Value
	}
	return noneValue
}

func (f *SelectField) itemAt(i int) (label, desc string) {
	if i < len(f.choices) {
		return f.choices[i].Label, f.choices[i].Description
	}
	return "None", ""
}
