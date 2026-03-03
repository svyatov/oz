package wizard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// MultiSelectField is a checkbox list with number toggles and space/x toggle.
type MultiSelectField struct {
	label       string
	description string
	choices     []config.Choice
	cursor      int
	selected    map[int]bool
}

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
	case "up", "k":
		f.cursor = (f.cursor - 1 + n) % n
	case "down", "j":
		f.cursor = (f.cursor + 1) % n
	case "space", "x":
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
	case "enter", "tab":
		return true, nil
	}

	// Number keys 1–9: toggle selection
	if msg.Code >= '1' && msg.Code <= '9' {
		idx := int(msg.Code-'0') - 1
		if idx < n {
			f.cursor = idx
			f.selected[idx] = !f.selected[idx]
		}
	}

	return false, nil
}

func (f *MultiSelectField) View() string {
	var b strings.Builder

	b.WriteString("  " + ui.StepCounter(0, 0) + "  ")
	b.WriteString(ui.FieldTitle(f.label) + "\n")
	if f.description != "" {
		b.WriteString("         " + ui.FieldDesc(f.description) + "\n")
	}
	b.WriteString("\n")

	maxLabel := 0
	for _, c := range f.choices {
		if len(c.Label) > maxLabel {
			maxLabel = len(c.Label)
		}
	}

	for i, c := range f.choices {
		active := i == f.cursor
		num := ui.NumberGutter(i+1, active)

		var cursor string
		if active {
			cursor = " " + ui.Cursor() + " "
		} else {
			cursor = "   "
		}

		check := "[ ]"
		if f.selected[i] {
			check = "[x]"
		}

		styledLabel := ui.ChoiceLabel(c.Label, active)
		pad := strings.Repeat(" ", maxLabel-len(c.Label))

		line := fmt.Sprintf("   %s%s  %s %s%s", cursor, num, check, styledLabel, pad)
		if c.Description != "" {
			line += "   " + ui.ChoiceDesc(c.Description)
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}

func (f *MultiSelectField) Value() any {
	var vals []string
	for i, c := range f.choices {
		if f.selected[i] {
			vals = append(vals, c.Value)
		}
	}
	return vals
}

func (f *MultiSelectField) SetValue(v any) {
	f.selected = make(map[int]bool)
	var vals []string
	switch vv := v.(type) {
	case []string:
		vals = vv
	case []any:
		for _, item := range vv {
			vals = append(vals, fmt.Sprintf("%v", item))
		}
	}
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
