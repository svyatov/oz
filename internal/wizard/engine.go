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
	Answers Answers
	Aborted bool
}

// choicesLoadedMsg is sent when a choices_from command completes.
type choicesLoadedMsg struct {
	choices []config.Choice
	err     error
}

// Engine is a bubbletea model that runs the wizard step-by-step.
type Engine struct {
	wizardName string
	version    string
	overridden bool
	options    []config.Option
	pinnedCnt  int
	answers    Answers
	defaults   map[string]any // from last-used state

	// Navigation
	stepIndex int   // index into visibleSteps
	history   []int // stack of visited step indices for back navigation

	// Current field
	currentField Field

	// Loading state for choices_from
	loading  bool
	loadErr  error
	spinner  spinner.Model

	// Completed steps for display
	completedSteps []CompletedStep

	// State
	done    bool
	aborted bool
	width   int
}

// NewEngine creates a new wizard engine.
func NewEngine(
	wizardName, version string, overridden bool,
	options []config.Option, pinnedCount int, defaults map[string]any,
) *Engine {
	if defaults == nil {
		defaults = make(map[string]any)
	}
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return &Engine{
		wizardName: wizardName,
		version:    version,
		overridden: overridden,
		options:    options,
		pinnedCnt:  pinnedCount,
		answers:    make(Answers),
		defaults:   defaults,
		spinner:    s,
		width:      80,
	}
}

func (e *Engine) headerLine() string {
	h := ui.Header(e.wizardName, e.version)
	if e.overridden {
		h += " " + ui.VersionOverrideTag()
	}
	return h
}

// SetPinnedAnswers sets the answers for pinned options (needed for show_when).
func (e *Engine) SetPinnedAnswers(pins map[string]any) {
	maps.Copy(e.answers, pins)
}

func (e *Engine) Init() tea.Cmd {
	e.advanceToNextVisible()
	return e.initCurrentField()
}

func (e *Engine) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.width = msg.Width
		return e, nil

	case spinner.TickMsg:
		if e.loading {
			var cmd tea.Cmd
			e.spinner, cmd = e.spinner.Update(msg)
			return e, cmd
		}
		return e, nil

	case choicesLoadedMsg:
		return e.handleChoicesLoaded(msg)

	case tea.KeyPressMsg:
		return e.handleKeyPress(msg)
	}

	return e, nil
}

func (e *Engine) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		e.aborted = true
		e.done = true
		return e, tea.Quit
	case "shift+tab":
		if !e.loading && e.goBack() {
			return e, e.initCurrentField()
		}
		return e, nil
	}

	// During loading: enter retries, other keys ignored
	if e.loading {
		if e.loadErr != nil && msg.String() == "enter" {
			e.loadErr = nil
			return e, e.initCurrentField()
		}
		return e, nil
	}

	if e.currentField == nil {
		return e, nil
	}

	submitted, cmd := e.currentField.Update(msg)
	if !submitted {
		return e, cmd
	}

	e.saveCurrentAnswer()
	e.recordCompletedStep()
	e.stepIndex++
	e.advanceToNextVisible()

	visible := VisibleSteps(e.options, e.answers)
	if e.stepIndex >= len(visible) {
		e.done = true
		return e, tea.Quit
	}

	e.history = append(e.history, e.stepIndex)
	return e, e.initCurrentField()
}

func (e *Engine) View() tea.View {
	if e.done {
		return tea.NewView(e.finalView())
	}

	var b strings.Builder

	// Header
	b.WriteString("\n  " + e.headerLine() + "\n")

	// Pinned info
	if e.pinnedCnt > 0 {
		b.WriteString("  " + ui.PinnedInfo(e.pinnedCnt) + "\n")
	}

	// Completed steps
	if len(e.completedSteps) > 0 {
		b.WriteString("\n")
		for _, cs := range e.completedSteps {
			b.WriteString(ui.CompletedStepLine(cs.StepNum, cs.Label, cs.Answer) + "\n")
		}
	}

	b.WriteString("\n")

	// Loading state
	if e.loading {
		opt := e.currentOption()
		if opt != nil {
			visible := VisibleSteps(e.options, e.answers)
			total := len(visible)
			displayPos := e.stepIndex + 1
			b.WriteString("  " + ui.StepCounter(displayPos, total) + "  ")
			if e.loadErr != nil {
				b.WriteString(ui.WarningText("Error loading choices: "+e.loadErr.Error()) + "\n")
				b.WriteString("         " + ui.NavHintText("enter=retry  shift+tab=back") + "\n")
			} else {
				b.WriteString(e.spinner.View() + " " + ui.FieldTitle("Loading "+opt.Label+"...") + "\n")
			}
		}
	} else {
		// Current field with step counter
		visible := VisibleSteps(e.options, e.answers)
		total := len(visible)
		displayPos := e.stepIndex + 1

		if e.currentField != nil {
			fieldView := e.currentField.View()
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
func (e *Engine) GetResult() Result {
	return Result{
		Answers: e.answers,
		Aborted: e.aborted,
	}
}

// finalView renders the summary that persists in scrollback after the wizard ends.
func (e *Engine) finalView() string {
	if e.aborted {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n  " + e.headerLine() + "\n\n")
	for _, cs := range e.completedSteps {
		b.WriteString(ui.CompletedStepLine(cs.StepNum, cs.Label, cs.Answer) + "\n")
	}
	return b.String()
}

func (e *Engine) handleChoicesLoaded(msg choicesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		e.loadErr = msg.err
		return e, nil
	}

	opt := e.currentOption()
	if opt == nil {
		e.done = true
		return e, tea.Quit
	}

	if len(msg.choices) == 0 {
		e.loadErr = errors.New("no choices available")
		return e, nil
	}

	// Update the option's choices in-place
	opt.Choices = config.FlexChoices(msg.choices)
	e.loading = false
	e.loadErr = nil

	e.currentField = e.buildField(opt)
	e.setFieldDefault(opt)
	return e, e.currentField.Init()
}

func (e *Engine) advanceToNextVisible() {
	visible := VisibleSteps(e.options, e.answers)
	for e.stepIndex < len(visible) {
		idx := visible[e.stepIndex]
		opt := e.options[idx]
		if IsVisible(opt, e.answers) {
			break
		}
		e.stepIndex++
	}
}

func (e *Engine) goBack() bool {
	if len(e.history) <= 1 {
		return false
	}
	// Pop current
	e.history = e.history[:len(e.history)-1]
	e.stepIndex = e.history[len(e.history)-1]

	// Remove last completed step from display
	if len(e.completedSteps) > 0 {
		e.completedSteps = e.completedSteps[:len(e.completedSteps)-1]
	}
	// Keep answer in e.answers so setFieldDefault restores the previous selection.
	// saveCurrentAnswer will overwrite it when the user submits again.

	return true
}

func (e *Engine) currentOption() *config.Option {
	visible := VisibleSteps(e.options, e.answers)
	if e.stepIndex >= len(visible) {
		return nil
	}
	return &e.options[visible[e.stepIndex]]
}

func (e *Engine) initCurrentField() tea.Cmd {
	opt := e.currentOption()
	if opt == nil {
		e.done = true
		return tea.Quit
	}

	// Record in history if not already the last entry
	if len(e.history) == 0 || e.history[len(e.history)-1] != e.stepIndex {
		e.history = append(e.history, e.stepIndex)
	}

	// If choices_from is set, enter loading state
	if opt.ChoicesFrom != "" && len(opt.Choices) == 0 {
		e.loading = true
		e.loadErr = nil
		e.currentField = nil
		resolveCmd := func() tea.Msg {
			choices, err := ResolveChoices(opt.ChoicesFrom, e.answers)
			return choicesLoadedMsg{choices: choices, err: err}
		}
		return tea.Batch(e.spinner.Tick, resolveCmd)
	}

	e.loading = false
	e.currentField = e.buildField(opt)
	e.setFieldDefault(opt)

	return e.currentField.Init()
}

func (e *Engine) buildField(opt *config.Option) Field {
	switch opt.Type {
	case "select":
		return NewSelectField(*opt)
	case "confirm":
		return NewConfirmField(opt.Label, opt.Description)
	case "input":
		return NewInputField(opt.Label, opt.Description, opt.Validate, opt.Required)
	case "multi_select":
		return NewMultiSelectField(*opt)
	default:
		return NewInputField(opt.Label, opt.Description, nil, false)
	}
}

func (e *Engine) setFieldDefault(opt *config.Option) {
	// Priority: existing answer > last-used > config default
	var val any
	if existing, ok := e.answers[opt.Name]; ok {
		val = existing
	} else if lastUsed, ok := e.defaults[opt.Name]; ok {
		val = lastUsed
	} else {
		val = opt.Default
	}

	if val == nil {
		// Set sensible defaults
		switch opt.Type {
		case "select":
			if len(opt.Choices) > 0 {
				val = opt.Choices[0].Value
			}
		case "confirm":
			val = false
		case "input":
			val = ""
		}
	}

	if sf, ok := e.currentField.(*SelectField); ok && opt.Default != nil {
		sf.SetDefault(opt.Default)
	}

	if val != nil {
		e.currentField.SetValue(val)
	}
}

func (e *Engine) saveCurrentAnswer() {
	opt := e.currentOption()
	if opt == nil || e.currentField == nil {
		return
	}
	e.answers[opt.Name] = e.currentField.Value()
}

func (e *Engine) recordCompletedStep() {
	opt := e.currentOption()
	if opt == nil || e.currentField == nil {
		return
	}
	e.completedSteps = append(e.completedSteps, CompletedStep{
		StepNum: e.stepIndex + 1,
		Label:   opt.Label,
		Answer:  FormatAnswer(opt, e.currentField.Value()),
	})
}

// Run executes the wizard and returns the result.
func Run(
	wizardName, version string, overridden bool, options []config.Option,
	pinnedCount int, defaults, pinnedAnswers map[string]any,
) (*Result, error) {
	engine := NewEngine(wizardName, version, overridden, options, pinnedCount, defaults)
	engine.SetPinnedAnswers(pinnedAnswers)

	p := tea.NewProgram(engine)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard error: %w", err)
	}

	result := finalModel.(*Engine).GetResult()
	return &result, nil
}
