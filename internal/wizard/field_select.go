package wizard

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)
const defaultSuffix = " (default)"

// SelectField is a numbered-list select with ↑↓jk navigation and 1–9 instant select.
type SelectField struct {
	label        string
	description  string
	choices      []config.Choice
	allowNone    bool
	cursor       int
	value        string
	defaultValue string
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

	b.WriteString(fieldHeader(f.label, f.description))

	maxDisplay := f.maxDisplayWidth()

	n := f.itemCount()
	gutterWidth := len(strconv.Itoa(n))
	for i := range n {
		active := i == f.cursor
		num := ui.NumberGutter(i+1, gutterWidth, active)

		cursor := "   "
		if active {
			cursor = " " + ui.Cursor() + " "
		}

		label, desc := f.itemAt(i)
		styledLabel := ui.ChoiceLabel(label, active)

		displayLen := len(label)
		tag := ""
		if f.defaultValue != "" && f.valueAt(i) == f.defaultValue {
			tag = " " + ui.DefaultTag()
			displayLen += len(defaultSuffix)
		}
		pad := strings.Repeat(" ", maxDisplay-displayLen)

		line := fmt.Sprintf("   %s%s  %s%s%s", cursor, num, styledLabel, tag, pad)
		if desc != "" {
			line += "   " + ui.ChoiceDesc(desc)
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

// maxDisplayWidth returns the widest label column width, accounting for the
// " (default)" suffix on the matching choice.
func (f *SelectField) maxDisplayWidth() int {
	w := 0
	for _, c := range f.choices {
		n := len(c.Label)
		if f.defaultValue != "" && c.Value == f.defaultValue {
			n += len(defaultSuffix)
		}
		if n > w {
			w = n
		}
	}
	if f.allowNone {
		n := len("None")
		if f.defaultValue == config.NoneValue {
			n += len(defaultSuffix)
		}
		if n > w {
			w = n
		}
	}
	return w
}

func (f *SelectField) Value() any { return f.value }

// SetDefault records which value is the default so View can show a "(default)" tag.
func (f *SelectField) SetDefault(v any) {
	f.defaultValue = fmt.Sprintf("%v", v)
}

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
	return config.NoneValue
}

func (f *SelectField) itemAt(i int) (label, desc string) {
	if i < len(f.choices) {
		return f.choices[i].Label, f.choices[i].Description
	}
	return "None", ""
}
