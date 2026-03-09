package wizard

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/svyatov/oz/internal/compat"
	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// VersionResult holds the outcome of version loading.
type VersionResult struct {
	Detected    string
	Selected    string
	Versions    []string
	Aborted     bool
	Interactive bool // true when the user saw a version selector
}

type versionPhase int

const (
	phaseLoading versionPhase = iota
	phaseSelect
	phaseInput
	phaseVerifying
)

// Messages for async operations.
type versionDetectedMsg struct {
	version string
	err     error
}

type versionsListedMsg struct {
	versions []string
	err      error
}

type versionVerifiedMsg struct {
	version string
	err     error
}

type spinnerDelayMsg struct{}

// VersionLoaderModel is a Bubbletea model for version selection.
type VersionLoaderModel struct {
	wizardName string
	vc         *config.VersionControl
	pin        string
	cached     *VersionResult

	phase         versionPhase
	spinner       spinner.Model
	showSpinner   bool
	detected      string
	detectErr     error
	detectDone    bool
	versions      []string
	versionsErr   error
	versionsDone  bool
	hasVersionCmd bool

	// Select phase
	preselect string // version to pre-select (from previous go-back)
	cursor    int
	items     []versionItem
	customAt  int // index of "Custom..." sentinel

	// Input phase
	ti        textinput.Model
	verifyErr string

	done   bool
	result VersionResult
}

type versionItem struct {
	version    string
	isCustom   bool
	isDetected bool
}

const customSentinel = "Custom..."

func newVersionLoaderModel(
	wizardName string, vc *config.VersionControl, pin string,
) *VersionLoaderModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(ui.Accent)

	return &VersionLoaderModel{
		wizardName:    wizardName,
		vc:            vc,
		pin:           pin,
		phase:         phaseLoading,
		spinner:       s,
		hasVersionCmd: vc.AvailVersionsCmd != "" || vc.AvailVersions != "",
	}
}

func (m *VersionLoaderModel) Init() tea.Cmd {
	if m.cached != nil {
		m.detected = m.cached.Detected
		m.versions = m.cached.Versions
		m.detectDone = true
		m.versionsDone = true
		if len(m.versions) > 0 {
			m.buildItems()
			m.phase = phaseSelect
			return nil
		}
		m.phase = phaseInput
		m.ti = textinput.New()
		return m.ti.Focus()
	}
	return tea.Batch(
		m.detectVersion,
		m.fetchVersions,
		tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { //nolint:mnd
			return spinnerDelayMsg{}
		}),
	)
}

func (m *VersionLoaderModel) detectVersion() tea.Msg {
	ver, err := compat.DetectVersion(m.vc)
	return versionDetectedMsg{version: ver, err: err}
}

func (m *VersionLoaderModel) fetchVersions() tea.Msg {
	if m.vc.AvailVersionsCmd != "" {
		versions, err := compat.FetchAvailableVersions(m.vc.AvailVersionsCmd)
		return versionsListedMsg{versions: versions, err: err}
	}
	if m.vc.AvailVersions != "" {
		versions := compat.ParseAvailableVersions(m.vc.AvailVersions)
		return versionsListedMsg{versions: versions}
	}
	return versionsListedMsg{}
}

func (m *VersionLoaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerDelayMsg:
		m.showSpinner = true
		return m, func() tea.Msg { return m.spinner.Tick() }

	case spinner.TickMsg:
		if m.phase == phaseLoading || m.phase == phaseVerifying {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case versionDetectedMsg:
		m.detectDone = true
		m.detected = msg.version
		m.detectErr = msg.err
		return m.checkLoadingDone()

	case versionsListedMsg:
		m.versionsDone = true
		m.versions = msg.versions
		m.versionsErr = msg.err
		return m.checkLoadingDone()

	case versionVerifiedMsg:
		return m.handleVerified(msg)

	case tea.KeyPressMsg:
		switch m.phase {
		case phaseLoading, phaseVerifying:
			return m.handleAbortKey(msg)
		case phaseSelect:
			return m.updateSelect(msg)
		case phaseInput:
			return m.updateInput(msg)
		}
	}

	return m, nil
}

func (m *VersionLoaderModel) finish() tea.Cmd {
	m.done = true
	return tea.Quit
}

func (m *VersionLoaderModel) handleAbortKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" || msg.String() == "ctrl+c" {
		m.result.Aborted = true
		return m, m.finish()
	}
	return m, nil
}

func (m *VersionLoaderModel) handleVerified(msg versionVerifiedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.phase = phaseInput
		m.verifyErr = msg.err.Error()
		return m, m.ti.Focus()
	}
	m.result = VersionResult{
		Detected: m.detected,
		Selected: msg.version,
		Versions: m.versions,
	}
	return m, m.finish()
}

func (m *VersionLoaderModel) checkLoadingDone() (tea.Model, tea.Cmd) {
	if !m.detectDone || !m.versionsDone {
		return m, nil
	}

	// Handle pin
	if m.pin != "" {
		if m.pin == "current" {
			m.result = VersionResult{
				Detected: m.detected,
				Selected: m.detected,
				Versions: m.versions,
			}
		} else {
			m.result = VersionResult{
				Detected: m.detected,
				Selected: m.pin,
				Versions: m.versions,
			}
		}
		return m, m.finish()
	}

	// No custom version support → return detected
	if m.vc.CustomVersionCmd == "" {
		m.result = VersionResult{
			Detected: m.detected,
			Selected: m.detected,
			Versions: m.versions,
		}
		return m, m.finish()
	}

	// Build version list or go to input
	if len(m.versions) > 0 {
		m.buildItems()
		m.phase = phaseSelect
		return m, nil
	}

	m.phase = phaseInput
	m.ti = textinput.New()
	return m, m.ti.Focus()
}

func (m *VersionLoaderModel) buildItems() {
	for _, v := range m.versions {
		m.items = append(m.items, versionItem{
			version:    v,
			isDetected: v == m.detected,
		})
	}
	m.customAt = len(m.items)
	m.items = append(m.items, versionItem{isCustom: true})

	// Pre-select: prefer previous choice, fall back to detected version
	target := m.preselect
	if target == "" {
		target = m.detected
	}
	for i, item := range m.items {
		if item.version == target {
			m.cursor = i
			break
		}
	}
}

func (m *VersionLoaderModel) updateSelect(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	n := len(m.items)

	switch msg.String() {
	case "up", "k":
		m.cursor = (m.cursor - 1 + n) % n
	case "down", "j":
		m.cursor = (m.cursor + 1) % n
	case "enter", "tab":
		return m.selectItem(m.cursor)
	case "esc", "ctrl+c":
		m.result.Aborted = true
		return m, m.finish()
	}

	if msg.Code >= '1' && msg.Code <= '9' {
		idx := int(msg.Code-'0') - 1
		if idx < n {
			m.cursor = idx
			return m.selectItem(idx)
		}
	}

	return m, nil
}

func (m *VersionLoaderModel) selectItem(idx int) (tea.Model, tea.Cmd) {
	item := m.items[idx]
	if item.isCustom {
		m.phase = phaseInput
		m.ti = textinput.New()
		return m, m.ti.Focus()
	}
	m.result = VersionResult{
		Detected: m.detected,
		Selected: item.version,
		Versions: m.versions,
	}
	return m, m.finish()
}

func (m *VersionLoaderModel) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "shift+tab":
		if len(m.items) > 0 {
			m.phase = phaseSelect
			m.verifyErr = ""
			return m, nil
		}
		return m, nil
	case "esc", "ctrl+c":
		if len(m.items) > 0 {
			m.phase = phaseSelect
			m.verifyErr = ""
			return m, nil
		}
		m.result.Aborted = true
		return m, m.finish()
	case "enter", "tab":
		return m.submitInput()
	}

	// Clear error when user types
	m.verifyErr = ""
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m *VersionLoaderModel) submitInput() (tea.Model, tea.Cmd) {
	version := strings.TrimSpace(m.ti.Value())
	if version == "" {
		m.result = VersionResult{
			Detected: m.detected,
			Selected: m.detected,
			Versions: m.versions,
		}
		return m, m.finish()
	}

	if m.vc.CustomVersionVerify != "" {
		m.phase = phaseVerifying
		m.verifyErr = ""
		verifyCmd := m.vc.CustomVersionVerify
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			err := compat.VerifyVersion(verifyCmd, version)
			return versionVerifiedMsg{version: version, err: err}
		})
	}

	m.result = VersionResult{
		Detected: m.detected,
		Selected: version,
		Versions: m.versions,
	}
	return m, m.finish()
}

func (m *VersionLoaderModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	switch m.phase {
	case phaseLoading:
		return tea.NewView(m.viewLoading())
	case phaseSelect:
		return tea.NewView(m.viewSelect())
	case phaseInput:
		return tea.NewView(m.viewInput())
	case phaseVerifying:
		return tea.NewView(m.viewVerifying())
	}
	return tea.NewView("")
}

func (m *VersionLoaderModel) viewLoading() string {
	if !m.showSpinner {
		return ""
	}
	return fmt.Sprintf("\n  %s Detecting version...\n", m.spinner.View())
}

func (m *VersionLoaderModel) viewSelect() string {
	var b strings.Builder

	b.WriteString("\n  " + ui.Header(m.wizardName, m.detected, m.vc.Label) + "\n\n")

	gutterWidth := len(strconv.Itoa(len(m.items)))
	for i, item := range m.items {
		active := i == m.cursor
		num := ui.NumberGutter(i+1, gutterWidth, active)

		cursor := "   "
		if active {
			cursor = " " + ui.Cursor() + " "
		}

		var label string
		if item.isCustom {
			label = ui.ChoiceLabel(customSentinel, active)
		} else {
			label = ui.ChoiceLabel(item.version, active)
		}

		suffix := ""
		if item.isDetected {
			suffix = "  " + ui.MutedStyle.Render("(current)")
		}

		fmt.Fprintf(&b, "   %s%s  %s%s\n", cursor, num, label, suffix)
	}

	b.WriteString("\n" + ui.NavHint() + "\n")
	return b.String()
}

func (m *VersionLoaderModel) viewInput() string {
	var b strings.Builder

	b.WriteString("\n  " + ui.Header(m.wizardName, m.detected, m.vc.Label) + "\n\n")

	prompt := "Use different version?"
	if m.detected != "" {
		prompt = fmt.Sprintf(
			"Use different version? (leave blank for %s)", m.detected,
		)
	}
	b.WriteString("  " + ui.FieldTitle(prompt) + "\n\n")

	if m.verifyErr != "" {
		b.WriteString("  " + ui.WarningText(m.verifyErr) + "\n\n")
	}

	b.WriteString("    " + m.ti.View() + "\n")
	return b.String()
}

func (m *VersionLoaderModel) viewVerifying() string {
	version := strings.TrimSpace(m.ti.Value())
	tag := ""
	if m.showSpinner {
		tag = " " + ui.VersionVerifyingTag(m.spinner.View())
	}
	return fmt.Sprintf("\n  %s%s\n", ui.Header(m.wizardName, version, m.vc.Label), tag)
}

// RunVersionLoader runs version detection, fetches available versions,
// and presents a version selector if applicable.
func RunVersionLoader(
	wizardName string, vc *config.VersionControl, pin string, cached *VersionResult,
) (*VersionResult, error) {
	if vc == nil {
		return &VersionResult{}, nil
	}

	// No custom version support and no pin → just detect
	if vc.CustomVersionCmd == "" && pin == "" {
		ver, err := compat.DetectVersion(vc)
		if err != nil {
			return nil, fmt.Errorf("detecting version: %w", err)
		}
		return &VersionResult{
			Detected: ver,
			Selected: ver,
		}, nil
	}

	model := newVersionLoaderModel(wizardName, vc, pin)
	if cached != nil {
		model.cached = cached
		model.preselect = cached.Selected
	}
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("version loader error: %w", err)
	}

	result := finalModel.(*VersionLoaderModel).result
	result.Interactive = true
	return &result, nil
}
