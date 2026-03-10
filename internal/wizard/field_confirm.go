package wizard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// ConfirmField is a Yes/No toggle with y/n shortcuts.
type ConfirmField struct {
	label        string
	description  string
	cursor       int // 0=Yes, 1=No
	value        bool
	defaultValue *bool
}

// NewConfirmField creates a ConfirmField from a config option.
func NewConfirmField(opt config.Option) *ConfirmField {
	return &ConfirmField{label: opt.Label, description: opt.Description}
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

	b.WriteString(fieldHeader(f.label, f.description))

	items := []string{"Yes", "No"}
	for i, item := range items {
		active := i == f.cursor
		num := ui.NumberGutter(i+1, 1, active)

		var cursor string
		if active {
			cursor = " " + ui.Cursor() + " "
		} else {
			cursor = "   "
		}

		styledLabel := ui.ChoiceLabel(item, active)
		tag := ""
		if f.defaultValue != nil && (i == 0) == *f.defaultValue {
			tag = " " + ui.DefaultTag()
		}
		fmt.Fprintf(&b, "   %s%s  %s%s\n", cursor, num, styledLabel, tag)
	}

	return b.String()
}

func (f *ConfirmField) Value() config.FieldValue { return config.BoolVal(f.value) }

// SetDefault records which value is the default so View can show a "(default)" tag.
func (f *ConfirmField) SetDefault(v config.FieldValue) {
	b := v.Bool()
	f.defaultValue = &b
}

func (f *ConfirmField) SetValue(v config.FieldValue) {
	f.value = v.Bool()
	if f.value {
		f.cursor = 0
	} else {
		f.cursor = 1
	}
}
