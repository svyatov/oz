package wizard

import (
	"fmt"
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
	Detected string
	Selected string
	Versions []string
	Aborted  bool
}

type versionPhase int

const (
	phaseLoading versionPhase = iota
	phaseSelect
	phaseInput
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

type spinnerDelayMsg struct{}

// VersionLoaderModel is a Bubbletea model for version selection.
type VersionLoaderModel struct {
	wizardName string
	vc         *config.VersionControl
	pin        string

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
	cursor   int
	items    []versionItem
	customAt int // index of "Custom..." sentinel

	// Input phase
	ti        textinput.Model
	verifyErr string

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
		if m.phase != phaseLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

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

	case tea.KeyPressMsg:
		switch m.phase {
		case phaseLoading:
			// Keys ignored during loading
		case phaseSelect:
			return m.updateSelect(msg)
		case phaseInput:
			return m.updateInput(msg)
		}
	}

	return m, nil
}

func (m *VersionLoaderModel) checkLoadingDone() (tea.Model, tea.Cmd) {
	if !m.detectDone || !m.versionsDone {
		return m, nil
	}

	// Handle pin
	if m.pin != "" {
		if m.pin == "default" {
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
		return m, tea.Quit
	}

	// No custom version support → return detected
	if m.vc.CustomVersionCmd == "" {
		m.result = VersionResult{
			Detected: m.detected,
			Selected: m.detected,
			Versions: m.versions,
		}
		return m, tea.Quit
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

	// Pre-select detected version
	for i, item := range m.items {
		if item.isDetected {
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
		return m, tea.Quit
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
	return m, tea.Quit
}

func (m *VersionLoaderModel) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		if len(m.items) > 0 {
			m.phase = phaseSelect
			m.verifyErr = ""
			return m, nil
		}
		m.result.Aborted = true
		return m, tea.Quit
	case "enter", "tab":
		version := strings.TrimSpace(m.ti.Value())
		if version == "" {
			// Use detected version
			m.result = VersionResult{
				Detected: m.detected,
				Selected: m.detected,
				Versions: m.versions,
			}
			return m, tea.Quit
		}
		if m.vc.CustomVersionVerify != "" {
			if err := compat.VerifyVersion(m.vc.CustomVersionVerify, version); err != nil {
				m.verifyErr = err.Error()
				return m, nil
			}
		}
		m.result = VersionResult{
			Detected: m.detected,
			Selected: version,
			Versions: m.versions,
		}
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m *VersionLoaderModel) View() tea.View {
	switch m.phase {
	case phaseLoading:
		return tea.NewView(m.viewLoading())
	case phaseSelect:
		return tea.NewView(m.viewSelect())
	case phaseInput:
		return tea.NewView(m.viewInput())
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

	b.WriteString("\n  " + ui.Header(m.wizardName, m.detected) + "\n\n")

	for i, item := range m.items {
		active := i == m.cursor
		num := ui.NumberGutter(i+1, active)

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
			suffix = "  " + ui.MutedStyle.Render("(detected)")
		}

		fmt.Fprintf(&b, "   %s%s  %s%s\n", cursor, num, label, suffix)
	}

	b.WriteString("\n" + ui.NavHint() + "\n")
	return b.String()
}

func (m *VersionLoaderModel) viewInput() string {
	var b strings.Builder

	b.WriteString("\n  " + ui.Header(m.wizardName, m.detected) + "\n\n")

	prompt := "Use different version?"
	if m.detected != "" {
		prompt = fmt.Sprintf("Use different version? (leave blank for %s)", m.detected)
	}
	b.WriteString("  " + ui.FieldTitle(prompt) + "\n\n")

	if m.verifyErr != "" {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(ui.Warning).Render(m.verifyErr) + "\n\n")
	}

	b.WriteString("    " + m.ti.View() + "\n")
	return b.String()
}

// RunVersionLoader runs version detection, fetches available versions,
// and presents a version selector if applicable.
func RunVersionLoader(
	wizardName string, vc *config.VersionControl, pin string,
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
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("version loader error: %w", err)
	}

	result := finalModel.(*VersionLoaderModel).result
	return &result, nil
}
