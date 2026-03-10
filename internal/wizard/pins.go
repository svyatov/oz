package wizard

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/svyatov/oz/internal/compat"
	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

type pinsMode int

const (
	pinsListMode pinsMode = iota
	pinsEditMode
	pinsVerifyingMode
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

	hasCustomVersion    bool
	versionPin          string
	customVersionVerify string

	mode      pinsMode
	cursor    int
	editIdx   int
	editField Field
	spinner   spinner.Model
	verifyErr string

	done bool
}

func newPinsModel(
	options []config.Option, pins, lastUsed map[string]any,
	hints map[string]string,
	hasCustomVersion bool, versionPin string,
	customVersionVerify string,
) *PinsModel {
	if pins == nil {
		pins = make(map[string]any)
	}
	if lastUsed == nil {
		lastUsed = make(map[string]any)
	}

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ui.Accent)

	return &PinsModel{
		options:             options,
		pins:                pins,
		lastUsed:            lastUsed,
		hints:               hints,
		hasCustomVersion:    hasCustomVersion,
		versionPin:          versionPin,
		customVersionVerify: customVersionVerify,
		spinner:             s,
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
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch m.mode {
		case pinsListMode:
			return m.updateList(msg)
		case pinsEditMode:
			return m.updateEdit(msg)
		case pinsVerifyingMode:
			if msg.String() == "esc" || msg.String() == "ctrl+c" {
				m.mode = pinsEditMode
				return m, m.editField.Init()
			}
			return m, nil
		}
	case versionVerifiedMsg:
		return m.handleVersionVerified(msg)
	case spinner.TickMsg:
		if m.mode == pinsVerifyingMode {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
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
		return m.togglePin(m.cursor)
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
		m.verifyErr = ""
		return m, nil
	}

	submitted, cmd := m.editField.Update(msg)
	if submitted {
		if m.isVersionIdx(m.editIdx) {
			return m.submitVersionPin()
		}
		optIdx := m.editIdx - m.versionOffset()
		opt := &m.options[optIdx]
		m.pins[opt.Name] = m.editField.Value()
		m.mode = pinsListMode
		m.editField = nil
		return m, cmd
	}
	m.verifyErr = ""
	return m, cmd
}

func (m *PinsModel) enterEdit(idx int) (tea.Model, tea.Cmd) {
	m.cursor = idx
	m.editIdx = idx

	if m.isVersionIdx(idx) {
		m.editField = NewInputField(config.Option{Label: "Version", Description: "Leave blank for current version"})
		if m.versionPin != "" && m.versionPin != "current" {
			m.editField.SetValue(m.versionPin)
		}
		m.mode = pinsEditMode
		return m, m.editField.Init()
	}

	optIdx := idx - m.versionOffset()
	opt := &m.options[optIdx]
	m.editField = buildPinsField(opt)

	switch f := m.editField.(type) {
	case *SelectField:
		if opt.Default != nil {
			f.SetDefault(opt.Default)
		}
	case *ConfirmField:
		defVal := opt.Default
		if defVal == nil {
			defVal = false
		}
		f.SetDefault(defVal)
	}

	val := resolveDefault(opt, m.pins, m.lastUsed)
	if val != nil {
		m.editField.SetValue(val)
	}

	m.mode = pinsEditMode
	return m, m.editField.Init()
}

func (m *PinsModel) togglePin(idx int) (tea.Model, tea.Cmd) {
	if m.isVersionIdx(idx) {
		if m.versionPin != "" {
			m.versionPin = ""
		} else {
			m.versionPin = "current"
		}
		return m, nil
	}

	optIdx := idx - m.versionOffset()
	name := m.options[optIdx].Name
	if _, pinned := m.pins[name]; pinned {
		delete(m.pins, name)
		return m, nil
	}

	opt := &m.options[optIdx]
	val := resolveDefault(opt, m.pins, m.lastUsed)
	if opt.Type == config.OptionInput && !m.isValidInputValue(opt, val) {
		return m.enterEdit(idx)
	}
	m.pins[name] = val
	return m, nil
}

func (m *PinsModel) isValidInputValue(opt *config.Option, val any) bool {
	s, _ := val.(string)
	if opt.Required && s == "" {
		return false
	}
	if opt.Validate == nil || s == "" {
		return true
	}
	f := NewInputField(config.Option{Validate: opt.Validate, Required: opt.Required})
	f.SetValue(val)
	return f.validate() == ""
}

func (m *PinsModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	switch m.mode {
	case pinsListMode:
		return tea.NewView(m.viewList())
	case pinsEditMode:
		return tea.NewView(m.viewEdit())
	case pinsVerifyingMode:
		return tea.NewView(m.viewVerifying())
	}
	return tea.NewView("")
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
	replacement := ui.PinEditIndicator()
	fieldView = strings.Replace(fieldView, placeholder, replacement, 1)

	oldIndent := "\n" + strings.Repeat(" ", 2+lipgloss.Width(placeholder)+2)
	newIndent := "\n" + strings.Repeat(" ", 2+lipgloss.Width(replacement)+2)
	fieldView = strings.Replace(fieldView, oldIndent, newIndent, 1)

	b.WriteString("\n")
	b.WriteString(fieldView)
	if m.verifyErr != "" {
		b.WriteString("\n  " + ui.WarningText(m.verifyErr))
	}

	hint := ui.PinsEditNavHint()
	switch m.editField.(type) {
	case *SelectField, *ConfirmField:
		hint = ui.PinsSelectEditNavHint()
	case *MultiSelectField:
		hint = ui.PinsMultiSelectEditNavHint()
	}
	b.WriteString("\n" + hint + "\n")

	return b.String()
}

func (m *PinsModel) viewVerifying() string {
	v := fmt.Sprintf("%v", m.editField.Value())
	return fmt.Sprintf("\n  %s Verifying version %s...\n", m.spinner.View(), v)
}

func (m *PinsModel) submitVersionPin() (tea.Model, tea.Cmd) {
	v := fmt.Sprintf("%v", m.editField.Value())
	if v == "" {
		m.versionPin = "current"
		m.mode = pinsListMode
		m.editField = nil
		m.verifyErr = ""
		return m, nil
	}

	if m.customVersionVerify != "" {
		m.mode = pinsVerifyingMode
		m.verifyErr = ""
		verifyCmd := m.customVersionVerify
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			err := compat.VerifyVersion(verifyCmd, v)
			return versionVerifiedMsg{version: v, err: err}
		})
	}

	m.versionPin = v
	m.mode = pinsListMode
	m.editField = nil
	return m, nil
}

func (m *PinsModel) handleVersionVerified(msg versionVerifiedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.mode = pinsEditMode
		m.verifyErr = msg.err.Error()
		return m, m.editField.Init()
	}
	m.versionPin = msg.version
	m.mode = pinsListMode
	m.editField = nil
	m.verifyErr = ""
	return m, nil
}

func buildPinsField(opt *config.Option) Field {
	switch opt.Type {
	case config.OptionSelect:
		return NewSelectField(*opt)
	case config.OptionConfirm:
		return NewConfirmField(*opt)
	case config.OptionInput:
		return NewInputField(*opt)
	case config.OptionMultiSelect:
		return NewMultiSelectField(*opt)
	default:
		return NewInputField(*opt)
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
	case config.OptionSelect:
		if len(opt.Choices) > 0 {
			return opt.Choices[0].Value
		}
	case config.OptionConfirm:
		return false
	case config.OptionInput:
		return ""
	case config.OptionMultiSelect:
		// no default for multi_select
	}
	return nil
}

// RunPins shows the interactive pin management UI and returns updated pins.
func RunPins(
	options []config.Option, currentPins, lastUsed map[string]any,
	hints map[string]string,
	hasCustomVersion bool, currentVersionPin string,
	customVersionVerify string,
) (*PinsResult, error) {
	pins := make(map[string]any, len(currentPins))
	maps.Copy(pins, currentPins)

	model := newPinsModel(options, pins, lastUsed, hints, hasCustomVersion, currentVersionPin, customVersionVerify)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("pins UI error: %w", err)
	}

	final := finalModel.(*PinsModel)
	return &PinsResult{Pins: final.pins, VersionPin: final.versionPin}, nil
}
