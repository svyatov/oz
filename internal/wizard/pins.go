package wizard

import (
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

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

// PinsParams groups the parameters for RunPins.
type PinsParams struct {
	Pins                config.Values
	LastUsed            config.Values
	Hints               map[string]string
	VersionPin          string
	CustomVersionVerify string
	Options             []config.Option
	HasCustomVersion    bool
}

// PinsResult is returned by RunPins.
type PinsResult struct {
	Pins       config.Values
	VersionPin string
}

// PinsModel is a Bubbletea model for interactive pin management.
type PinsModel struct {
	editField           Field // version pin editing only
	editor              *ValuesEditor
	customVersionVerify string
	versionPin          string
	verifyErr           string
	spinner             spinner.Model
	mode                pinsMode
	cursor              int
	hasCustomVersion    bool
	done                bool
}

func newPinsModel(p PinsParams) *PinsModel {
	editor := NewValuesEditor(p.Options, p.Pins, p.LastUsed, p.Hints)

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = ui.AccentStyle

	return &PinsModel{
		editor:              editor,
		hasCustomVersion:    p.HasCustomVersion,
		versionPin:          p.VersionPin,
		customVersionVerify: p.CustomVersionVerify,
		spinner:             s,
	}
}

// itemCount returns the total number of items in the list.
func (m *PinsModel) itemCount() int {
	n := len(m.editor.options)
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
			if msg.String() == keyEsc || msg.String() == keyCtrlC {
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
	case keyUp, "k":
		m.cursor = (m.cursor - 1 + n) % n
	case keyDown, "j":
		m.cursor = (m.cursor + 1) % n
	case keyLeft, "h":
		return m.handleCycle(-1)
	case keyRight, "l":
		return m.handleCycle(1)
	case keyEnter:
		return m.handleEnter()
	case keySpace:
		return m.handleToggle()
	case keyEsc, keyCtrlC:
		m.done = true
		return m, tea.Quit
	}

	if msg.Code >= '1' && msg.Code <= '9' {
		idx := int(msg.Code-'0') - 1
		if idx < n {
			m.cursor = idx
			return m.handleEnter()
		}
	}

	return m, nil
}

func (m *PinsModel) handleCycle(direction int) (tea.Model, tea.Cmd) {
	if m.isVersionIdx(m.cursor) {
		m.toggleVersionPin()
		return m, nil
	}
	m.editor.CycleValue(m.cursor-m.versionOffset(), direction)
	return m, nil
}

func (m *PinsModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.isVersionIdx(m.cursor) {
		return m.enterVersionEdit()
	}
	cmd := m.editor.EnterEdit(m.cursor - m.versionOffset())
	m.mode = pinsEditMode
	return m, cmd
}

func (m *PinsModel) handleToggle() (tea.Model, tea.Cmd) {
	if m.isVersionIdx(m.cursor) {
		m.toggleVersionPin()
		return m, nil
	}
	cmd := m.editor.ToggleValue(m.cursor - m.versionOffset())
	if m.editor.Editing() {
		m.mode = pinsEditMode
	}
	return m, cmd
}

func (m *PinsModel) toggleVersionPin() {
	if m.versionPin != "" {
		m.versionPin = ""
	} else {
		m.versionPin = versionPinCurrent
	}
}

func (m *PinsModel) enterVersionEdit() (tea.Model, tea.Cmd) {
	m.editField = NewInputField(config.Option{
		Label: "Version", Description: "Leave blank for current version",
	})
	if m.versionPin != "" && m.versionPin != versionPinCurrent {
		m.editField.SetValue(config.StringVal(m.versionPin))
	}
	m.mode = pinsEditMode
	return m, m.editField.Init()
}

func (m *PinsModel) updateEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.editor.Editing() {
		exited, cmd := m.editor.UpdateEdit(msg)
		if exited {
			m.mode = pinsListMode
		}
		return m, cmd
	}

	// Version pin editing.
	if msg.String() == keyEsc || msg.String() == keyCtrlC {
		m.mode = pinsListMode
		m.editField = nil
		m.verifyErr = ""
		return m, nil
	}

	submitted, cmd := m.editField.Update(msg)
	if submitted {
		return m.submitVersionPin()
	}
	m.verifyErr = ""
	return m, cmd
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
	maxLabel := m.editor.MaxLabelWidth()
	if m.hasCustomVersion && len(versionLabel) > maxLabel {
		maxLabel = len(versionLabel)
	}

	n := m.itemCount()
	gutterWidth := len(strconv.Itoa(n))
	for i := range n {
		active := i == m.cursor
		if m.isVersionIdx(i) {
			b.WriteString(m.viewVersionRow(i, active, maxLabel, gutterWidth, versionLabel))
		} else {
			optIdx := i - m.versionOffset()
			b.WriteString(m.editor.ViewOptionRow(optIdx, active, maxLabel, gutterWidth, i+1))
		}
	}

	b.WriteString("\n" + ui.NavHints(
			ui.HintUp, ui.HintDown, ui.HintCycle, ui.HintEdit,
			ui.Hint{Key: "space", Desc: "toggle pin"}, ui.HintEscDone,
		) + "\n")
	return b.String()
}

func (m *PinsModel) viewVersionRow(
	i int, active bool, maxLabel, gutterWidth int, label string,
) string {
	num := ui.NumberGutter(i+1, gutterWidth, active)
	cursor := cursorBlank
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

func (m *PinsModel) viewEdit() string {
	if m.editor.Editing() {
		return m.editor.ViewEdit(ui.PinEditIndicator())
	}
	// Version pin editing.
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderFieldWithIndicator(m.editField.View(), ui.PinEditIndicator()))
	if m.verifyErr != "" {
		b.WriteString("\n  " + ui.WarningText(m.verifyErr))
	}
	b.WriteString("\n" + ui.NavHints(ui.HintEnter, ui.HintEsc) + "\n")
	return b.String()
}

func (m *PinsModel) viewVerifying() string {
	v := m.editField.Value().Scalar()
	return fmt.Sprintf("\n  %s Verifying version %s...\n", m.spinner.View(), v)
}

func (m *PinsModel) submitVersionPin() (tea.Model, tea.Cmd) {
	v := m.editField.Value().Scalar()
	if v == "" {
		m.versionPin = versionPinCurrent
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

// RunPins shows the interactive pin management UI and returns updated pins.
func RunPins(p PinsParams) (*PinsResult, error) {
	pins := make(config.Values, len(p.Pins))
	maps.Copy(pins, p.Pins)

	p.Pins = pins
	model := newPinsModel(p)
	prog := tea.NewProgram(model)
	finalModel, err := prog.Run()
	if err != nil {
		return nil, fmt.Errorf("pins UI error: %w", err)
	}

	final, ok := finalModel.(*PinsModel)
	if !ok {
		return nil, errors.New("unexpected model type")
	}
	return &PinsResult{Pins: final.editor.Values(), VersionPin: final.versionPin}, nil
}
