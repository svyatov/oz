package wizard

import (
	"fmt"
	"maps"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

type pinsMode int

const (
	pinsListMode pinsMode = iota
	pinsEditMode
)

// PinsResult is returned by RunPins.
type PinsResult struct {
	Pins map[string]any
}

// PinsModel is a Bubbletea model for interactive pin management.
type PinsModel struct {
	options  []config.Option
	pins     map[string]any
	lastUsed map[string]any

	mode      pinsMode
	cursor    int
	editIdx   int
	editField Field

	done bool
}

func newPinsModel(options []config.Option, pins, lastUsed map[string]any) *PinsModel {
	if pins == nil {
		pins = make(map[string]any)
	}
	if lastUsed == nil {
		lastUsed = make(map[string]any)
	}
	return &PinsModel{
		options:  options,
		pins:     pins,
		lastUsed: lastUsed,
	}
}

func (m *PinsModel) Init() tea.Cmd { return nil }

func (m *PinsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		if m.mode == pinsEditMode {
			return m.updateEdit(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m *PinsModel) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	n := len(m.options)

	switch msg.String() {
	case "up", "k":
		m.cursor = (m.cursor - 1 + n) % n
	case "down", "j":
		m.cursor = (m.cursor + 1) % n
	case "enter":
		return m.enterEdit(m.cursor)
	case "space":
		m.togglePin(m.cursor)
	case "esc", "ctrl+c":
		m.done = true
		return m, tea.Quit
	}

	if msg.Code >= '1' && msg.Code <= '9' {
		idx := int(msg.Code-'0') - 1
		if idx < n {
			return m.enterEdit(idx)
		}
	}

	return m, nil
}

func (m *PinsModel) updateEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" || msg.String() == "ctrl+c" {
		m.mode = pinsListMode
		m.editField = nil
		return m, nil
	}

	submitted, cmd := m.editField.Update(msg)
	if submitted {
		opt := &m.options[m.editIdx]
		m.pins[opt.Name] = m.editField.Value()
		m.mode = pinsListMode
		m.editField = nil
	}
	return m, cmd
}

func (m *PinsModel) enterEdit(idx int) (tea.Model, tea.Cmd) {
	m.cursor = idx
	m.editIdx = idx
	opt := &m.options[idx]
	m.editField = buildPinsField(opt)

	val := resolveDefault(opt, m.pins, m.lastUsed)
	if val != nil {
		m.editField.SetValue(val)
	}

	m.mode = pinsEditMode
	return m, m.editField.Init()
}

func (m *PinsModel) togglePin(idx int) {
	name := m.options[idx].Name
	if _, pinned := m.pins[name]; pinned {
		delete(m.pins, name)
	} else {
		opt := &m.options[idx]
		m.pins[name] = resolveDefault(opt, m.pins, m.lastUsed)
	}
}

func (m *PinsModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	if m.mode == pinsEditMode {
		return tea.NewView(m.viewEdit())
	}
	return tea.NewView(m.viewList())
}

func (m *PinsModel) viewList() string {
	var b strings.Builder

	b.WriteString("\n  " + ui.TitleStyle.Render("Manage pins") + "\n\n")

	maxLabel := 0
	for _, o := range m.options {
		if len(o.Label) > maxLabel {
			maxLabel = len(o.Label)
		}
	}

	for i, o := range m.options {
		active := i == m.cursor
		num := ui.NumberGutter(i+1, active)

		var cursor string
		if active {
			cursor = " " + ui.Cursor() + " "
		} else {
			cursor = "   "
		}

		_, pinned := m.pins[o.Name]
		pin := "  "
		if pinned {
			pin = ui.PinIcon() + " "
		}

		label := ui.ChoiceLabel(o.Label, active)
		pad := strings.Repeat(" ", maxLabel-len(o.Label))

		var value string
		if pinned {
			value = ui.CompletedStepAnswer(FormatAnswer(&o, m.pins[o.Name]))
		} else {
			value = ui.MutedStyle.Render("\u2500")
		}

		fmt.Fprintf(&b, "   %s%s  %s%s%s  %s\n", cursor, num, pin, label, pad, value)
	}

	b.WriteString("\n" + ui.PinsListNavHint() + "\n")
	return b.String()
}

func (m *PinsModel) viewEdit() string {
	var b strings.Builder

	fieldView := m.editField.View()
	placeholder := ui.StepCounter(0, 0)
	fieldView = strings.Replace(fieldView, placeholder, ui.PinEditIndicator(), 1)

	b.WriteString("\n")
	b.WriteString(fieldView)
	b.WriteString("\n" + ui.PinsEditNavHint() + "\n")

	return b.String()
}

func buildPinsField(opt *config.Option) Field {
	switch opt.Type {
	case "select":
		return NewSelectField(*opt)
	case "confirm":
		return NewConfirmField(opt.Label, opt.Description)
	case "input":
		return NewInputField(opt.Label, opt.Description)
	case "multi_select":
		return NewMultiSelectField(*opt)
	default:
		return NewInputField(opt.Label, opt.Description)
	}
}

func resolveDefault(opt *config.Option, pins, lastUsed map[string]any) any {
	if v, ok := pins[opt.Name]; ok {
		return v
	}
	if v, ok := lastUsed[opt.Name]; ok {
		return v
	}
	if opt.Default != nil {
		return opt.Default
	}
	switch opt.Type {
	case "select":
		if len(opt.Choices) > 0 {
			return opt.Choices[0].Value
		}
	case "confirm":
		return false
	case "input":
		return ""
	}
	return nil
}

// RunPins shows the interactive pin management UI and returns updated pins.
func RunPins(options []config.Option, currentPins, lastUsed map[string]any) (*PinsResult, error) {
	pins := make(map[string]any, len(currentPins))
	maps.Copy(pins, currentPins)

	model := newPinsModel(options, pins, lastUsed)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("pins UI error: %w", err)
	}

	return &PinsResult{Pins: finalModel.(*PinsModel).pins}, nil
}
