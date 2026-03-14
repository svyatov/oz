package wizard

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
)

func mustEngine(t *testing.T, model tea.Model) *Engine {
	t.Helper()
	m, ok := model.(*Engine)
	if !ok {
		t.Fatalf("expected *Engine, got %T", model)
	}
	return m
}

func engineOptions() []config.Option {
	return []config.Option{
		{
			Name:  "db",
			Type:  config.OptionSelect,
			Label: "Database",
			Choices: []config.Choice{
				{Value: "pg", Label: "PostgreSQL"},
				{Value: "mysql", Label: "MySQL"},
			},
		},
		{
			Name:  "api",
			Type:  config.OptionConfirm,
			Label: "API mode?",
		},
		{
			Name:  "name",
			Type:  config.OptionInput,
			Label: "App name",
		},
	}
}

func newTestEngine(opts []config.Option) *Engine {
	return NewEngine("test-wizard", "1.0.0", "", false, opts, 0, nil)
}

func initEngine(t *testing.T, e *Engine) *Engine {
	t.Helper()
	e.Init()
	return e
}

// submitSelect submits the current selection (first choice by default).
func submitSelect(t *testing.T, e *Engine) *Engine {
	t.Helper()
	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return mustEngine(t, model)
}

// submitConfirm submits the current confirm value (false by default, cursor=0 → true).
func submitConfirm(t *testing.T, e *Engine) *Engine {
	t.Helper()
	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return mustEngine(t, model)
}

// typeAndSubmitInput types text and submits.
func typeAndSubmitInput(t *testing.T, e *Engine, text string) *Engine {
	t.Helper()
	for _, c := range text {
		model, _ := e.Update(key(c))
		e = mustEngine(t, model)
	}
	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return mustEngine(t, model)
}

func TestEngineForwardFlow(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	if e.done {
		t.Fatal("should not be done after init")
	}
	if len(e.completedSteps) != 0 {
		t.Fatalf("expected 0 completed steps, got %d", len(e.completedSteps))
	}

	// Step 1: select db.
	e = submitSelect(t, e)
	if len(e.completedSteps) != 1 {
		t.Fatalf("expected 1 completed step, got %d", len(e.completedSteps))
	}
	if e.answers["db"].String() != "pg" {
		t.Errorf("expected db=pg, got %v", e.answers["db"])
	}

	// Step 2: confirm api.
	e = submitConfirm(t, e)
	if len(e.completedSteps) != 2 {
		t.Fatalf("expected 2 completed steps, got %d", len(e.completedSteps))
	}

	// Step 3: input name.
	e = typeAndSubmitInput(t, e, "myapp")
	if !e.done {
		t.Fatal("expected done after last step")
	}
	if e.answers["name"].String() != "myapp" {
		t.Errorf("expected name=myapp, got %v", e.answers["name"])
	}

	result := e.GetResult()
	if result.Aborted {
		t.Error("should not be aborted")
	}
	if result.GoBack {
		t.Error("should not be GoBack")
	}
	if len(result.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(result.Values))
	}
}

func TestEngineBackNavigation(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	// Submit step 1 (select).
	e = submitSelect(t, e)
	// Submit step 2 (confirm).
	e = submitConfirm(t, e)

	if len(e.completedSteps) != 2 {
		t.Fatalf("expected 2 completed, got %d", len(e.completedSteps))
	}

	// Go back.
	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	e = mustEngine(t, model)

	if e.done {
		t.Fatal("should not be done after going back")
	}
	if len(e.completedSteps) != 1 {
		t.Errorf("expected 1 completed step after back, got %d", len(e.completedSteps))
	}
	if e.stepIndex != 1 {
		t.Errorf("expected stepIndex=1, got %d", e.stepIndex)
	}

	// Re-submit step 2.
	e = submitConfirm(t, e)
	if len(e.completedSteps) != 2 {
		t.Fatalf("expected 2 completed steps after re-submit, got %d", len(e.completedSteps))
	}
}

func TestEngineAbort(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	// Submit one step, then abort.
	e = submitSelect(t, e)
	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	e = mustEngine(t, model)

	if !e.done {
		t.Fatal("expected done after abort")
	}
	result := e.GetResult()
	if !result.Aborted {
		t.Error("expected aborted")
	}
}

func TestEngineAbortCtrlC(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	model, _ := e.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	e = mustEngine(t, model)
	if !e.GetResult().Aborted {
		t.Error("expected aborted via ctrl+c")
	}
}

func TestEngineBackBeforeFirst(t *testing.T) {
	e := newTestEngine(engineOptions())
	e.canGoBack = true
	e = initEngine(t, e)

	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	e = mustEngine(t, model)

	if !e.done {
		t.Fatal("expected done")
	}
	result := e.GetResult()
	if !result.GoBack {
		t.Error("expected GoBack")
	}
	if result.Aborted {
		t.Error("should not be aborted")
	}
}

func TestEngineBackBeforeFirstDisabled(t *testing.T) {
	e := newTestEngine(engineOptions())
	// canGoBack defaults to false.
	e = initEngine(t, e)

	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	e = mustEngine(t, model)

	if e.done {
		t.Fatal("should not be done when canGoBack is false")
	}
}

func TestEngineHiddenFieldSkipping(t *testing.T) {
	opts := []config.Option{
		{
			Name:  "mode",
			Type:  config.OptionSelect,
			Label: "Mode",
			Choices: []config.Choice{
				{Value: "simple", Label: "Simple"},
				{Value: "advanced", Label: "Advanced"},
			},
		},
		{
			Name:     "detail",
			Type:     config.OptionInput,
			Label:    "Detail level",
			ShowWhen: config.Values{"mode": config.StringVal("advanced")},
		},
		{
			Name:  "confirm",
			Type:  config.OptionConfirm,
			Label: "Proceed?",
		},
	}

	e := newTestEngine(opts)
	e = initEngine(t, e)

	// Submit "simple" → detail should be hidden, skip to confirm.
	e = submitSelect(t, e)
	if e.done {
		t.Fatal("should not be done yet")
	}

	// Current field should be confirm (not detail).
	if _, ok := e.currentField.(*ConfirmField); !ok {
		t.Fatalf("expected ConfirmField, got %T", e.currentField)
	}

	// Finish wizard.
	e = submitConfirm(t, e)
	if !e.done {
		t.Fatal("expected done")
	}

	result := e.GetResult()
	if _, has := result.Values["detail"]; has {
		t.Error("hidden field should not appear in results")
	}
	if len(result.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(result.Values))
	}
}

func TestEngineHiddenFieldShownWhenConditionMet(t *testing.T) {
	opts := []config.Option{
		{
			Name:  "mode",
			Type:  config.OptionSelect,
			Label: "Mode",
			Choices: []config.Choice{
				{Value: "advanced", Label: "Advanced"},
				{Value: "simple", Label: "Simple"},
			},
		},
		{
			Name:     "detail",
			Type:     config.OptionInput,
			Label:    "Detail level",
			ShowWhen: config.Values{"mode": config.StringVal("advanced")},
		},
		{
			Name:  "done",
			Type:  config.OptionConfirm,
			Label: "Done?",
		},
	}

	e := newTestEngine(opts)
	e = initEngine(t, e)

	// Submit "advanced" → detail should be visible.
	e = submitSelect(t, e)
	if _, ok := e.currentField.(*InputField); !ok {
		t.Fatalf("expected InputField for detail, got %T", e.currentField)
	}

	e = typeAndSubmitInput(t, e, "high")
	e = submitConfirm(t, e)
	if !e.done {
		t.Fatal("expected done")
	}
	if e.answers["detail"].String() != "high" {
		t.Errorf("expected detail=high, got %v", e.answers["detail"])
	}
}

func TestEngineAsyncChoicesLoading(t *testing.T) {
	opts := []config.Option{
		{
			Name:        "tool",
			Type:        config.OptionSelect,
			Label:       "Tool",
			ChoicesFrom: "echo 'a\nb\nc'",
		},
	}

	e := newTestEngine(opts)
	e.Init()

	if !e.loading {
		t.Fatal("expected loading state")
	}
	if e.currentField != nil {
		t.Fatal("expected nil currentField during loading")
	}

	// Inject choices loaded message.
	choices := []config.Choice{
		{Value: "a", Label: "a"},
		{Value: "b", Label: "b"},
		{Value: "c", Label: "c"},
	}
	model, _ := e.Update(choicesLoadedMsg{choices: choices})
	e = mustEngine(t, model)

	if e.loading {
		t.Fatal("expected loading=false after choices loaded")
	}
	if e.currentField == nil {
		t.Fatal("expected currentField initialized")
	}
}

func TestEngineChoicesLoadingError(t *testing.T) {
	opts := []config.Option{
		{
			Name:        "tool",
			Type:        config.OptionSelect,
			Label:       "Tool",
			ChoicesFrom: "false",
		},
	}

	e := newTestEngine(opts)
	e.Init()

	// Inject error.
	model, _ := e.Update(choicesLoadedMsg{err: errTest})
	e = mustEngine(t, model)

	if e.loadErr == nil {
		t.Fatal("expected loadErr set")
	}

	// Retry via Enter.
	model, _ = e.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	e = mustEngine(t, model)
	if e.loadErr != nil {
		t.Error("expected loadErr cleared after retry")
	}
	if !e.loading {
		t.Error("expected loading after retry")
	}
}

func TestEngineChoicesLoadingEmpty(t *testing.T) {
	opts := []config.Option{
		{
			Name:        "tool",
			Type:        config.OptionSelect,
			Label:       "Tool",
			ChoicesFrom: "echo ''",
		},
	}

	e := newTestEngine(opts)
	e.Init()

	// Inject empty choices.
	model, _ := e.Update(choicesLoadedMsg{choices: nil})
	e = mustEngine(t, model)

	if e.loadErr == nil {
		t.Fatal("expected loadErr for empty choices")
	}
}

func TestEnginePinnedValues(t *testing.T) {
	e := newTestEngine(engineOptions())
	e.SetPinnedValues(config.Values{"db": config.StringVal("mysql")})
	e = initEngine(t, e)

	if e.answers["db"].String() != "mysql" {
		t.Errorf("expected pinned db=mysql in answers, got %v", e.answers["db"])
	}
}

func TestEngineWindowResize(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	model, _ := e.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	e = mustEngine(t, model)

	if e.width != 120 {
		t.Errorf("expected width=120, got %d", e.width)
	}
	if e.done {
		t.Error("resize should not finish wizard")
	}
}

func TestEngineCompletedStepsFormatting(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	e = submitSelect(t, e)
	if len(e.completedSteps) != 1 {
		t.Fatalf("expected 1 completed step")
	}
	cs := e.completedSteps[0]
	if cs.Label != "Database" {
		t.Errorf("expected label Database, got %s", cs.Label)
	}
	if cs.Answer != "PostgreSQL" {
		t.Errorf("expected answer PostgreSQL, got %s", cs.Answer)
	}
	if cs.StepNum != 1 {
		t.Errorf("expected StepNum=1, got %d", cs.StepNum)
	}
}

func TestEngineFinalViewOnCompletion(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	e = submitSelect(t, e)
	e = submitConfirm(t, e)
	e = typeAndSubmitInput(t, e, "app")

	view := e.finalView()
	if view == "" {
		t.Error("expected non-empty final view on completion")
	}
}

func TestEngineFinalViewOnAbort(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	e = mustEngine(t, model)

	view := e.finalView()
	if view != "" {
		t.Errorf("expected empty final view on abort, got %q", view)
	}
}

func TestEngineIgnoresKeysWhileLoading(t *testing.T) {
	opts := []config.Option{
		{
			Name:        "tool",
			Type:        config.OptionSelect,
			Label:       "Tool",
			ChoicesFrom: "echo test",
		},
	}

	e := newTestEngine(opts)
	e.Init()

	if !e.loading {
		t.Fatal("expected loading")
	}

	// Non-special keys should be ignored during loading.
	model, _ := e.Update(key('a'))
	e = mustEngine(t, model)
	if e.done {
		t.Error("typing during loading should not end wizard")
	}

	// Shift+Tab during loading should be ignored.
	model, _ = e.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	e = mustEngine(t, model)
	if e.done {
		t.Error("shift+tab during loading should not end wizard")
	}
}

func TestEngineSingleOption(t *testing.T) {
	opts := []config.Option{
		{
			Name:  "confirm",
			Type:  config.OptionConfirm,
			Label: "Continue?",
		},
	}

	e := newTestEngine(opts)
	e = initEngine(t, e)
	e = submitConfirm(t, e)

	if !e.done {
		t.Fatal("expected done after single option")
	}
	result := e.GetResult()
	if len(result.Values) != 1 {
		t.Errorf("expected 1 value, got %d", len(result.Values))
	}
}

var errTest = errors.New("test error")

func TestEngineUpdateSpinnerTickNotLoading(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	// Engine is not loading, so spinner tick should be a no-op.
	model, cmd := e.Update(spinner.TickMsg{})
	e = mustEngine(t, model)
	if cmd != nil {
		t.Error("expected nil cmd when not loading")
	}
	if e.done {
		t.Error("should not be done")
	}
}

func TestEngineUpdateUnknownMessage(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	// Send a message type the engine doesn't handle.
	type customMsg struct{}
	model, cmd := e.Update(customMsg{})
	e = mustEngine(t, model)
	if cmd != nil {
		t.Error("expected nil cmd for unknown message type")
	}
	if e.done {
		t.Error("should not be done")
	}
}

func TestEngineViewWhenDone(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	e = submitSelect(t, e)
	e = submitConfirm(t, e)
	e = typeAndSubmitInput(t, e, "app")

	if !e.done {
		t.Fatal("expected done")
	}
	view := e.View()
	if view.Content == "" {
		t.Error("expected non-empty view when done")
	}
}

func TestEngineViewWhenLoading(t *testing.T) {
	opts := []config.Option{
		{
			Name:        "tool",
			Type:        config.OptionSelect,
			Label:       "Tool",
			ChoicesFrom: "echo test",
		},
	}
	e := newTestEngine(opts)
	e.Init()

	if !e.loading {
		t.Fatal("expected loading")
	}
	view := e.View()
	plain := stripANSI(view.Content)
	if !strings.Contains(plain, "Loading Tool...") {
		t.Errorf("expected loading text in view, got:\n%s", plain)
	}
}

func TestEngineViewWithLoadErr(t *testing.T) {
	opts := []config.Option{
		{
			Name:        "tool",
			Type:        config.OptionSelect,
			Label:       "Tool",
			ChoicesFrom: "echo test",
		},
	}
	e := newTestEngine(opts)
	e.Init()

	// Inject error.
	e.loadErr = errors.New("connection failed")

	view := e.View()
	plain := stripANSI(view.Content)
	if !strings.Contains(plain, "connection failed") {
		t.Errorf("expected error text in view, got:\n%s", plain)
	}
	if !strings.Contains(plain, "enter=retry") {
		t.Errorf("expected retry hint in view, got:\n%s", plain)
	}
}

func TestEngineViewWithPinnedCount(t *testing.T) {
	e := NewEngine("test-wizard", "1.0.0", "", false, engineOptions(), 3, nil)
	e = initEngine(t, e)

	view := e.View()
	plain := stripANSI(view.Content)
	if !strings.Contains(plain, "3") {
		t.Errorf("expected pinned count in view, got:\n%s", plain)
	}
}

func TestEngineViewWithCompletedSteps(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	e = submitSelect(t, e)
	if !e.loading && e.currentField != nil {
		view := e.View()
		plain := stripANSI(view.Content)
		if !strings.Contains(plain, "Database") {
			t.Errorf("expected completed step label in view, got:\n%s", plain)
		}
		if !strings.Contains(plain, "PostgreSQL") {
			t.Errorf("expected completed step answer in view, got:\n%s", plain)
		}
	}
}

func TestEngineHeaderLineOverridden(t *testing.T) {
	e := NewEngine("test-wizard", "1.0.0", "", true, engineOptions(), 0, nil)
	header := e.headerLine()
	plain := stripANSI(header)
	if !strings.Contains(plain, "override") {
		t.Errorf("expected override tag in header, got: %s", plain)
	}
}

func TestEngineShiftTabEmptyHistoryWithCanGoBack(t *testing.T) {
	opts := []config.Option{
		{
			Name:  "db",
			Type:  config.OptionSelect,
			Label: "Database",
			Choices: []config.Choice{
				{Value: "pg", Label: "PostgreSQL"},
			},
		},
		{
			Name:  "name",
			Type:  config.OptionInput,
			Label: "Name",
		},
	}
	e := newTestEngine(opts)
	e.canGoBack = true
	e = initEngine(t, e)

	// Submit one step to have history, then go back.
	e = submitSelect(t, e)

	// Now go back — history has 2 entries, pops to first step.
	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	e = mustEngine(t, model)
	if e.done {
		t.Fatal("should not be done yet, should be at first step")
	}

	// Now go back again from first step — history has 1 entry, goBack returns false.
	// canGoBack is true, so wentBack=true and done=true.
	model, _ = e.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	e = mustEngine(t, model)
	if !e.done {
		t.Fatal("expected done")
	}
	if !e.wentBack {
		t.Error("expected wentBack=true")
	}
	result := e.GetResult()
	if !result.GoBack {
		t.Error("expected GoBack in result")
	}
}

func TestEngineKeyIgnoredDuringLoading(t *testing.T) {
	opts := []config.Option{
		{
			Name:        "tool",
			Type:        config.OptionSelect,
			Label:       "Tool",
			ChoicesFrom: "echo test",
		},
	}
	e := newTestEngine(opts)
	e.Init()

	if !e.loading {
		t.Fatal("expected loading")
	}

	// Non-enter, non-escape key should be ignored.
	model, cmd := e.Update(key('z'))
	e = mustEngine(t, model)
	if cmd != nil {
		t.Error("expected nil cmd for ignored key during loading")
	}
	if e.done {
		t.Error("should not be done")
	}
}

func TestEngineRetryOnLoadErrEnter(t *testing.T) {
	opts := []config.Option{
		{
			Name:        "tool",
			Type:        config.OptionSelect,
			Label:       "Tool",
			ChoicesFrom: "echo test",
		},
	}
	e := newTestEngine(opts)
	e.Init()

	// Inject error.
	model, _ := e.Update(choicesLoadedMsg{err: errTest})
	e = mustEngine(t, model)
	if e.loadErr == nil {
		t.Fatal("expected loadErr set")
	}

	// Press enter to retry.
	model, cmd := e.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	e = mustEngine(t, model)
	if e.loadErr != nil {
		t.Error("expected loadErr cleared after enter retry")
	}
	if !e.loading {
		t.Error("expected loading state after retry")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after retry (spinner + fetch)")
	}
}

func TestEngineKeyPressNilCurrentField(t *testing.T) {
	e := newTestEngine(engineOptions())
	e = initEngine(t, e)

	// Force nil currentField.
	e.currentField = nil
	e.loading = false

	model, cmd := e.Update(key('a'))
	e = mustEngine(t, model)
	if cmd != nil {
		t.Error("expected nil cmd when currentField is nil")
	}
	if e.done {
		t.Error("should not be done")
	}
}

func TestEngineFinalViewOnWentBack(t *testing.T) {
	e := newTestEngine(engineOptions())
	e.canGoBack = true
	e = initEngine(t, e)

	model, _ := e.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	e = mustEngine(t, model)

	if !e.wentBack {
		t.Fatal("expected wentBack")
	}
	view := e.finalView()
	if view != "" {
		t.Errorf("expected empty final view on wentBack, got %q", view)
	}
}
