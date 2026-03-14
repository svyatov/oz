package wizard

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// MultiSelectField is a checkbox list with number toggles and space/x toggle.
type MultiSelectField struct {
	selected    map[int]bool
	label       string
	description string
	choices     []config.Choice
	cursor      int
}

// NewMultiSelectField creates a MultiSelectField from a config option.
func NewMultiSelectField(opt config.Option) *MultiSelectField {
	return &MultiSelectField{
		label:       opt.Label,
		description: opt.Description,
		choices:     opt.Choices,
		selected:    make(map[int]bool),
	}
}

func (f *MultiSelectField) Init() tea.Cmd { return nil }

func (f *MultiSelectField) Update(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	n := len(f.choices)

	switch msg.String() {
	case keyUp, "k":
		f.cursor = cursorUp(f.cursor, n)
	case keyDown, "j":
		f.cursor = cursorDown(f.cursor, n)
	case keySpace, "x":
		f.selected[f.cursor] = !f.selected[f.cursor]
	case "a":
		allSelected := true
		for i := range f.choices {
			if !f.selected[i] {
				allSelected = false
				break
			}
		}
		for i := range f.choices {
			f.selected[i] = !allSelected
		}
	case keyEnter, keyTab:
		return true, nil
	}

	// Number keys 1–9: toggle selection.
	if idx := numberKeyIndex(msg.Code, n); idx >= 0 {
		f.cursor = idx
		f.selected[idx] = !f.selected[idx]
	}

	return false, nil
}

func (f *MultiSelectField) View() string {
	var b strings.Builder

	b.WriteString(fieldHeader(f.label, f.description))

	maxLabel := 0
	for _, c := range f.choices {
		if len(c.Label) > maxLabel {
			maxLabel = len(c.Label)
		}
	}

	gutterWidth := len(strconv.Itoa(len(f.choices)))
	for i, c := range f.choices {
		active := i == f.cursor
		num := ui.NumberGutter(i+1, gutterWidth, active)

		check := "[ ]"
		if f.selected[i] {
			check = "[x]"
		}

		styledLabel := ui.ChoiceLabel(c.Label, active)
		pad := strings.Repeat(" ", maxLabel-len(c.Label))

		line := fmt.Sprintf("   %s%s  %s %s%s", choiceCursor(active), num, check, styledLabel, pad)
		if c.Description != "" {
			line += "   " + ui.ChoiceDesc(c.Description)
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (f *MultiSelectField) Value() config.FieldValue {
	var vals []string
	for i, c := range f.choices {
		if f.selected[i] {
			vals = append(vals, c.Value)
		}
	}
	return config.StringsVal(vals...)
}

func (f *MultiSelectField) SetValue(v config.FieldValue) {
	f.selected = make(map[int]bool)
	vals := v.Strings()
	set := make(map[string]bool, len(vals))
	for _, s := range vals {
		set[s] = true
	}
	for i, c := range f.choices {
		if set[c.Value] {
			f.selected[i] = true
		}
	}
}
