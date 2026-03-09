package wizard

import (
	"fmt"
	"maps"
	"strconv"
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
	Pins       map[string]any
	VersionPin string
}

// PinsModel is a Bubbletea model for interactive pin management.
type PinsModel struct {
	options  []config.Option
	pins     map[string]any
	lastUsed map[string]any
	hints    map[string]string

	hasCustomVersion bool
	versionPin       string

	mode      pinsMode
	cursor    int
	editIdx   int
	editField Field

	done bool
}

func newPinsModel(
	options []config.Option, pins, lastUsed map[string]any,
	hints map[string]string,
	hasCustomVersion bool, versionPin string,
) *PinsModel {
	if pins == nil {
		pins = make(map[string]any)
	}
	if lastUsed == nil {
		lastUsed = make(map[string]any)
	}
	return &PinsModel{
		options:          options,
		pins:             pins,
		lastUsed:         lastUsed,
		hints:            hints,
		hasCustomVersion: hasCustomVersion,
		versionPin:       versionPin,
	}
}

// itemCount returns the total number of items in the list.
func (m *PinsModel) itemCount() int {
	n := len(m.options)
	if m.hasCustomVersion {
		n++
	}
	return n
}

// versionOffset returns 1 if the version pin entry exists (at index 0), else 0.
func (m *PinsModel) versionOffset() int {
	if m.hasCustomVersion {
		return 1
	}
	return 0
}

// isVersionIdx returns true if idx points to the synthetic version pin entry.
func (m *PinsModel) isVersionIdx(idx int) bool {
	return m.hasCustomVersion && idx == 0
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
	n := m.itemCount()

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
		if m.isVersionIdx(m.editIdx) {
			if v := fmt.Sprintf("%v", m.editField.Value()); v == "" {
				m.versionPin = "current"
			} else {
				m.versionPin = v
			}
		} else {
			optIdx := m.editIdx - m.versionOffset()
			opt := &m.options[optIdx]
			m.pins[opt.Name] = m.editField.Value()
		}
		m.mode = pinsListMode
		m.editField = nil
	}
	return m, cmd
}

func (m *PinsModel) enterEdit(idx int) (tea.Model, tea.Cmd) {
	m.cursor = idx
	m.editIdx = idx

	if m.isVersionIdx(idx) {
		m.editField = NewInputField("Version", "Leave blank for current version", nil, false)
		if m.versionPin != "" && m.versionPin != "current" {
			m.editField.SetValue(m.versionPin)
		}
		m.mode = pinsEditMode
		return m, m.editField.Init()
	}

	optIdx := idx - m.versionOffset()
	opt := &m.options[optIdx]
	m.editField = buildPinsField(opt)

	val := resolveDefault(opt, m.pins, m.lastUsed)
	if val != nil {
		m.editField.SetValue(val)
	}

	m.mode = pinsEditMode
	return m, m.editField.Init()
}

func (m *PinsModel) togglePin(idx int) {
	if m.isVersionIdx(idx) {
		if m.versionPin != "" {
			m.versionPin = ""
		} else {
			m.versionPin = "current"
		}
		return
	}

	optIdx := idx - m.versionOffset()
	name := m.options[optIdx].Name
	if _, pinned := m.pins[name]; pinned {
		delete(m.pins, name)
	} else {
		opt := &m.options[optIdx]
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

	versionLabel := "Version"
	maxLabel := 0
	if m.hasCustomVersion && len(versionLabel) > maxLabel {
		maxLabel = len(versionLabel)
	}
	for _, o := range m.options {
		if len(o.Label) > maxLabel {
			maxLabel = len(o.Label)
		}
	}

	n := m.itemCount()
	gutterWidth := len(strconv.Itoa(n))
	for i := range n {
		active := i == m.cursor
		if m.isVersionIdx(i) {
			b.WriteString(m.viewVersionRow(i, active, maxLabel, gutterWidth, versionLabel))
		} else {
			b.WriteString(m.viewOptionRow(i, active, maxLabel, gutterWidth))
		}
	}

	b.WriteString("\n" + ui.PinsListNavHint() + "\n")
	return b.String()
}

func (m *PinsModel) viewVersionRow(i int, active bool, maxLabel, gutterWidth int, label string) string {
	num := ui.NumberGutter(i+1, gutterWidth, active)
	cursor := "   "
	if active {
		cursor = " " + ui.Cursor() + " "
	}
	pinned := m.versionPin != ""
	pin := "  "
	if pinned {
		pin = ui.PinIcon() + " "
	}
	styledLabel := ui.ChoiceLabel(label, active)
	pad := strings.Repeat(" ", maxLabel-len(label))
	value := ui.MutedStyle.Render("\u2500")
	if pinned {
		value = ui.CompletedStepAnswer(m.versionPin)
	}
	return fmt.Sprintf("   %s%s  %s%s%s  %s\n", cursor, num, pin, styledLabel, pad, value)
}

func (m *PinsModel) viewOptionRow(i int, active bool, maxLabel, gutterWidth int) string {
	num := ui.NumberGutter(i+1, gutterWidth, active)
	cursor := "   "
	if active {
		cursor = " " + ui.Cursor() + " "
	}
	o := m.options[i-m.versionOffset()]
	_, pinned := m.pins[o.Name]
	pin := "  "
	if pinned {
		pin = ui.PinIcon() + " "
	}
	label := ui.ChoiceLabel(o.Label, active)
	pad := strings.Repeat(" ", maxLabel-len(o.Label))
	value := ui.MutedStyle.Render("\u2500")
	if pinned {
		value = ui.CompletedStepAnswer(FormatAnswer(&o, m.pins[o.Name]))
	}
	hint := ""
	if h := m.hints[o.Name]; h != "" {
		hint = " " + ui.NavHintText(h)
	}
	return fmt.Sprintf("   %s%s  %s%s%s  %s%s\n", cursor, num, pin, label, pad, value, hint)
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
		return NewInputField(opt.Label, opt.Description, nil, false)
	case "multi_select":
		return NewMultiSelectField(*opt)
	default:
		return NewInputField(opt.Label, opt.Description, nil, false)
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
func RunPins(
	options []config.Option, currentPins, lastUsed map[string]any,
	hints map[string]string,
	hasCustomVersion bool, currentVersionPin string,
) (*PinsResult, error) {
	pins := make(map[string]any, len(currentPins))
	maps.Copy(pins, currentPins)

	model := newPinsModel(options, pins, lastUsed, hints, hasCustomVersion, currentVersionPin)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("pins UI error: %w", err)
	}

	final := finalModel.(*PinsModel)
	return &PinsResult{Pins: final.pins, VersionPin: final.versionPin}, nil
}
