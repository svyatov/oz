package wizard

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// CompletedStep records a finished wizard step for display.
type CompletedStep struct {
	StepNum int
	Label   string
	Answer  string
}

// Result is what the wizard returns after completion.
type Result struct {
	Values  config.Values
	Aborted bool
	GoBack  bool
}

// choicesLoadedMsg is sent when a choices_from command completes.
type choicesLoadedMsg struct {
	choices []config.Choice
	err     error
}

// Engine is a bubbletea model that runs the wizard step-by-step.
type Engine struct {
	wizardName    string
	version       string
	versionLabel  string
	overridden    bool
	options    []config.Option
	pinnedCount  int
	answers  config.Values
	defaults config.Values // from last-used state

	// Navigation
	stepIndex int   // index into visibleSteps
	history   []int // stack of visited step indices for back navigation
	canGoBack bool  // allow back-navigation before the first step

	// Current field
	currentField Field

	// Loading state for choices_from
	loading  bool
	loadErr  error
	spinner  spinner.Model

	// Completed steps for display
	completedSteps []CompletedStep

	// State
	done     bool
	aborted  bool
	wentBack bool
	width    int
}

// NewEngine creates a new wizard engine.
func NewEngine(
	wizardName, version, versionLabel string, overridden bool,
	options []config.Option, pinnedCount int, defaults config.Values,
) *Engine {
	if defaults == nil {
		defaults = make(config.Values)
	}
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return &Engine{
		wizardName:   wizardName,
		version:      version,
		versionLabel: versionLabel,
		overridden:   overridden,
		options:    options,
		pinnedCount:  pinnedCount,
		answers:    make(config.Values),
		defaults:   defaults,
		spinner:    s,
		width:      80,
	}
}

func (m *Engine) headerLine() string {
	h := ui.Header(m.wizardName, m.version, m.versionLabel)
	if m.overridden {
		h += " " + ui.VersionOverrideTag()
	}
	return h
}

// SetPinnedValues sets the answers for pinned options (needed for show_when).
func (m *Engine) SetPinnedValues(pins config.Values) {
	maps.Copy(m.answers, pins)
}

func (m *Engine) Init() tea.Cmd {
	m.advanceToNextVisible()
	return m.initCurrentField()
}

func (m *Engine) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case choicesLoadedMsg:
		return m.handleChoicesLoaded(msg)

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

func (m *Engine) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.aborted = true
		m.done = true
		return m, tea.Quit
	case "shift+tab":
		if m.loading {
			return m, nil
		}
		if m.goBack() {
			return m, m.initCurrentField()
		}
		if m.canGoBack {
			m.wentBack = true
			m.done = true
			return m, tea.Quit
		}
		return m, nil
	}

	// During loading: enter retries, other keys ignored
	if m.loading {
		if m.loadErr != nil && msg.String() == "enter" {
			m.loadErr = nil
			return m, m.initCurrentField()
		}
		return m, nil
	}

	if m.currentField == nil {
		return m, nil
	}

	submitted, cmd := m.currentField.Update(msg)
	if !submitted {
		return m, cmd
	}

	m.saveCurrentAnswer()
	m.recordCompletedStep()
	m.stepIndex++
	m.advanceToNextVisible()

	visible := VisibleSteps(m.options, m.answers)
	if m.stepIndex >= len(visible) {
		m.done = true
		return m, tea.Quit
	}

	m.history = append(m.history, m.stepIndex)
	return m, m.initCurrentField()
}

func (m *Engine) View() tea.View {
	if m.done {
		return tea.NewView(m.finalView())
	}

	var b strings.Builder

	// Header
	b.WriteString("\n  " + m.headerLine() + "\n")

	// Pinned info
	if m.pinnedCount > 0 {
		b.WriteString("  " + ui.PinnedInfo(m.pinnedCount) + "\n")
	}

	// Completed steps
	if len(m.completedSteps) > 0 {
		b.WriteString("\n")
		for _, cs := range m.completedSteps {
			b.WriteString(ui.CompletedStepLine(cs.StepNum, cs.Label, cs.Answer) + "\n")
		}
	}

	b.WriteString("\n")

	// Loading state
	if m.loading {
		opt := m.currentOption()
		if opt != nil {
			visible := VisibleSteps(m.options, m.answers)
			total := len(visible)
			displayPos := m.stepIndex + 1
			b.WriteString("  " + ui.StepCounter(displayPos, total) + "  ")
			if m.loadErr != nil {
				b.WriteString(ui.WarningText("Error loading choices: "+m.loadErr.Error()) + "\n")
				b.WriteString("         " + ui.NavHintText("enter=retry  shift+tab=back") + "\n")
			} else {
				b.WriteString(m.spinner.View() + " " + ui.FieldTitle("Loading "+opt.Label+"...") + "\n")
			}
		}
	} else {
		// Current field with step counter
		visible := VisibleSteps(m.options, m.answers)
		total := len(visible)
		displayPos := m.stepIndex + 1

		if m.currentField != nil {
			fieldView := m.currentField.View()
			// Replace placeholder step counter with real one
			placeholder := ui.StepCounter(0, 0)
			actual := ui.StepCounter(displayPos, total)
			fieldView = strings.Replace(fieldView, placeholder, actual, 1)
			b.WriteString(fieldView)
		}
	}

	// Nav hint
	b.WriteString("\n" + ui.NavHint() + "\n")

	return tea.NewView(b.String())
}

// GetResult returns the wizard result after completion.
func (m *Engine) GetResult() Result {
	return Result{
		Values:  m.answers,
		Aborted:  m.aborted,
		GoBack:   m.wentBack,
	}
}

// finalView renders the summary that persists in scrollback after the wizard ends.
func (m *Engine) finalView() string {
	if m.aborted || m.wentBack {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n  " + m.headerLine() + "\n\n")
	for _, cs := range m.completedSteps {
		b.WriteString(ui.CompletedStepLine(cs.StepNum, cs.Label, cs.Answer) + "\n")
	}
	return b.String()
}

func (m *Engine) handleChoicesLoaded(msg choicesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.loadErr = msg.err
		return m, nil
	}

	opt := m.currentOption()
	if opt == nil {
		m.done = true
		return m, tea.Quit
	}

	if len(msg.choices) == 0 {
		m.loadErr = errors.New("no choices available")
		return m, nil
	}

	// Update the option's choices in-place
	opt.Choices = config.FlexChoices(msg.choices)
	m.loading = false
	m.loadErr = nil

	m.currentField = buildField(opt)
	m.setFieldDefault(opt)
	return m, m.currentField.Init()
}

func (m *Engine) advanceToNextVisible() {
	visible := VisibleSteps(m.options, m.answers)
	for m.stepIndex < len(visible) {
		idx := visible[m.stepIndex]
		opt := m.options[idx]
		if IsVisible(opt, m.answers) {
			break
		}
		m.stepIndex++
	}
}

func (m *Engine) goBack() bool {
	if len(m.history) <= 1 {
		return false
	}
	// Pop current
	m.history = m.history[:len(m.history)-1]
	m.stepIndex = m.history[len(m.history)-1]

	// Remove last completed step from display
	if len(m.completedSteps) > 0 {
		m.completedSteps = m.completedSteps[:len(m.completedSteps)-1]
	}
	// Keep answer in m.answers so setFieldDefault restores the previous selection.
	// saveCurrentAnswer will overwrite it when the user submits again.

	return true
}

func (m *Engine) currentOption() *config.Option {
	visible := VisibleSteps(m.options, m.answers)
	if m.stepIndex >= len(visible) {
		return nil
	}
	return &m.options[visible[m.stepIndex]]
}

func (m *Engine) initCurrentField() tea.Cmd {
	opt := m.currentOption()
	if opt == nil {
		m.done = true
		return tea.Quit
	}

	// Record in history if not already the last entry
	if len(m.history) == 0 || m.history[len(m.history)-1] != m.stepIndex {
		m.history = append(m.history, m.stepIndex)
	}

	// If choices_from is set, enter loading state
	if opt.ChoicesFrom != "" && len(opt.Choices) == 0 {
		m.loading = true
		m.loadErr = nil
		m.currentField = nil
		resolveCmd := func() tea.Msg {
			choices, err := ResolveChoices(opt.ChoicesFrom, m.answers)
			return choicesLoadedMsg{choices: choices, err: err}
		}
		return tea.Batch(m.spinner.Tick, resolveCmd)
	}

	m.loading = false
	m.currentField = buildField(opt)
	m.setFieldDefault(opt)

	return m.currentField.Init()
}

func (m *Engine) setFieldDefault(opt *config.Option) {
	val := resolveDefault(opt, m.answers, m.defaults)

	switch f := m.currentField.(type) {
	case *SelectField:
		if opt.Default != nil {
			f.SetDefault(*opt.Default)
		}
	case *ConfirmField:
		defVal := config.BoolVal(false)
		if opt.Default != nil {
			defVal = *opt.Default
		}
		f.SetDefault(defVal)
	}

	if val != nil {
		m.currentField.SetValue(*val)
	}
}

func (m *Engine) saveCurrentAnswer() {
	opt := m.currentOption()
	if opt == nil || m.currentField == nil {
		return
	}
	m.answers[opt.Name] = m.currentField.Value()
}

func (m *Engine) recordCompletedStep() {
	opt := m.currentOption()
	if opt == nil || m.currentField == nil {
		return
	}
	m.completedSteps = append(m.completedSteps, CompletedStep{
		StepNum: m.stepIndex + 1,
		Label:   opt.Label,
		Answer:  FormatAnswer(opt, m.currentField.Value()),
	})
}

// RunParams groups the arguments for Run.
type RunParams struct {
	WizardName    string
	Version       string
	VersionLabel  string
	Overridden    bool
	Options       []config.Option
	PinnedCount   int
	Defaults      config.Values
	PinnedValues  config.Values
	CanGoBack     bool
}

// Run executes the wizard and returns the result.
func Run(p RunParams) (*Result, error) {
	engine := NewEngine(p.WizardName, p.Version, p.VersionLabel, p.Overridden, p.Options, p.PinnedCount, p.Defaults)
	engine.canGoBack = p.CanGoBack
	engine.SetPinnedValues(p.PinnedValues)

	prog := tea.NewProgram(engine)
	finalModel, err := prog.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard error: %w", err)
	}

	result := finalModel.(*Engine).GetResult()
	return &result, nil
}
