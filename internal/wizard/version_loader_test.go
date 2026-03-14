package wizard

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
)

func mustVL(t *testing.T, model tea.Model) *VersionLoaderModel {
	t.Helper()
	m, ok := model.(*VersionLoaderModel)
	if !ok {
		t.Fatalf("expected *VersionLoaderModel, got %T", model)
	}
	return m
}

func testVC() *config.VersionControl {
	return &config.VersionControl{
		Command:          "ruby --version",
		Pattern:          `(\d+\.\d+\.\d+)`,
		CustomVersionCmd: "rbenv versions",
		Label:            "Ruby",
	}
}

func TestVersionLoaderDetectAndList(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	if m.phase != phaseLoading {
		t.Fatalf("expected phaseLoading, got %d", m.phase)
	}

	// Inject detection.
	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	if m.detected != "3.2.1" {
		t.Errorf("expected detected=3.2.1, got %q", m.detected)
	}

	// Inject versions list.
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1", "3.1.0", "3.0.0"}})
	m = mustVL(t, model)

	if m.phase != phaseSelect {
		t.Fatalf("expected phaseSelect, got %d", m.phase)
	}
	if len(m.items) != 4 { // 3 versions + Custom...
		t.Errorf("expected 4 items, got %d", len(m.items))
	}
	// Detected version should be preselected.
	if m.cursor != 0 {
		t.Errorf("expected cursor=0 (detected version), got %d", m.cursor)
	}
}

func TestVersionLoaderPinCurrent(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, versionPinCurrent)
	m.Init()

	// Inject detection and versions.
	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1"}})
	m = mustVL(t, model)

	if !m.done {
		t.Fatal("expected done for pin=current")
	}
	if m.result.Selected != "3.2.1" {
		t.Errorf("expected selected=3.2.1, got %q", m.result.Selected)
	}
}

func TestVersionLoaderPinSpecificNoVerify(t *testing.T) {
	vc := testVC()
	// No custom_version_verify.
	m := newVersionLoaderModel("rails", vc, "3.0.0")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1", "3.0.0"}})
	m = mustVL(t, model)

	if !m.done {
		t.Fatal("expected done for specific pin without verify")
	}
	if m.result.Selected != "3.0.0" {
		t.Errorf("expected selected=3.0.0, got %q", m.result.Selected)
	}
}

func TestVersionLoaderPinSpecificWithVerify(t *testing.T) {
	vc := testVC()
	vc.CustomVersionVerify = "rbenv versions --bare | grep -q {{version}}"
	m := newVersionLoaderModel("rails", vc, "3.0.0")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1"}})
	m = mustVL(t, model)

	if m.phase != phaseVerifying {
		t.Fatalf("expected phaseVerifying, got %d", m.phase)
	}
	if !m.verifyingPin {
		t.Fatal("expected verifyingPin=true")
	}

	// Simulate success.
	model, _ = m.Update(versionVerifiedMsg{version: "3.0.0"})
	m = mustVL(t, model)
	if !m.done {
		t.Fatal("expected done after verify success")
	}
	if m.result.Selected != "3.0.0" {
		t.Errorf("expected selected=3.0.0, got %q", m.result.Selected)
	}
}

func TestVersionLoaderPinVerifyFailure(t *testing.T) {
	vc := testVC()
	vc.CustomVersionVerify = "false"
	m := newVersionLoaderModel("rails", vc, "bad")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1"}})
	m = mustVL(t, model)

	// Simulate verification failure.
	model, _ = m.Update(versionVerifiedMsg{version: "bad", err: errors.New("not found")})
	m = mustVL(t, model)

	// Should fall through to UI since pin is invalid.
	if m.pin != "" {
		t.Errorf("expected pin cleared, got %q", m.pin)
	}
	if m.phase != phaseSelect {
		t.Errorf("expected phaseSelect after pin verify failure, got %d", m.phase)
	}
}

func TestVersionLoaderSelectItem(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1", "3.0.0"}})
	m = mustVL(t, model)

	// Select second item (3.0.0).
	model, _ = m.Update(specialKey(tea.KeyDown))
	m = mustVL(t, model)
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustVL(t, model)

	if !m.done {
		t.Fatal("expected done after selecting version")
	}
	if m.result.Selected != "3.0.0" {
		t.Errorf("expected selected=3.0.0, got %q", m.result.Selected)
	}
}

func TestVersionLoaderCustomInput(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1"}})
	m = mustVL(t, model)

	// Navigate to "Custom..." (last item).
	n := len(m.items)
	for range n - 1 {
		model, _ = m.Update(specialKey(tea.KeyDown))
		m = mustVL(t, model)
	}
	// Select Custom...
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustVL(t, model)

	if m.phase != phaseInput {
		t.Fatalf("expected phaseInput, got %d", m.phase)
	}

	// Type a version.
	for _, c := range "2.7.0" {
		model, _ = m.Update(key(c))
		m = mustVL(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustVL(t, model)

	if !m.done {
		t.Fatal("expected done after custom input")
	}
	if m.result.Selected != "2.7.0" {
		t.Errorf("expected selected=2.7.0, got %q", m.result.Selected)
	}
}

func TestVersionLoaderCustomInputEmptyReturnDetected(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	// No available versions → goes straight to input.
	model, _ = m.Update(versionsListedMsg{})
	m = mustVL(t, model)

	if m.phase != phaseInput {
		t.Fatalf("expected phaseInput, got %d", m.phase)
	}

	// Submit empty → use detected.
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustVL(t, model)
	if !m.done {
		t.Fatal("expected done")
	}
	if m.result.Selected != "3.2.1" {
		t.Errorf("expected selected=3.2.1, got %q", m.result.Selected)
	}
}

func TestVersionLoaderCustomInputWithVerify(t *testing.T) {
	vc := testVC()
	vc.CustomVersionVerify = "test-verify"
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{})
	m = mustVL(t, model)

	for _, c := range "2.7.0" {
		model, _ = m.Update(key(c))
		m = mustVL(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustVL(t, model)

	if m.phase != phaseVerifying {
		t.Fatalf("expected phaseVerifying, got %d", m.phase)
	}

	// Success.
	model, _ = m.Update(versionVerifiedMsg{version: "2.7.0"})
	m = mustVL(t, model)
	if !m.done {
		t.Fatal("expected done")
	}
	if m.result.Selected != "2.7.0" {
		t.Errorf("expected selected=2.7.0, got %q", m.result.Selected)
	}
}

func TestVersionLoaderVerifyFailureReturnsToInput(t *testing.T) {
	vc := testVC()
	vc.CustomVersionVerify = "test-verify"
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{})
	m = mustVL(t, model)

	for _, c := range "bad" {
		model, _ = m.Update(key(c))
		m = mustVL(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustVL(t, model)

	// Verification fails.
	model, _ = m.Update(versionVerifiedMsg{version: "bad", err: errors.New("not found")})
	m = mustVL(t, model)

	if m.phase != phaseInput {
		t.Fatalf("expected phaseInput after verify error, got %d", m.phase)
	}
	if m.verifyErr == "" {
		t.Error("expected verifyErr set")
	}
}

func TestVersionLoaderDetectionError(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{err: errors.New("command not found")})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{})
	m = mustVL(t, model)

	// Should fall through to input since no versions and detection failed.
	if m.phase != phaseInput {
		t.Fatalf("expected phaseInput after detect error, got %d", m.phase)
	}
}

func TestVersionLoaderAbortLoading(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEscape))
	m = mustVL(t, model)

	if !m.done {
		t.Fatal("expected done")
	}
	if !m.result.Aborted {
		t.Error("expected aborted")
	}
}

func TestVersionLoaderAbortSelect(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1"}})
	m = mustVL(t, model)

	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustVL(t, model)
	if !m.result.Aborted {
		t.Error("expected aborted from select phase")
	}
}

func TestVersionLoaderEscFromInputWithItems(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1"}})
	m = mustVL(t, model)

	// Go to Custom...
	model, _ = m.Update(specialKey(tea.KeyDown))
	m = mustVL(t, model)
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustVL(t, model)
	if m.phase != phaseInput {
		t.Fatalf("expected phaseInput")
	}

	// Esc should go back to select (not abort), since items exist.
	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustVL(t, model)
	if m.phase != phaseSelect {
		t.Errorf("expected phaseSelect after esc from input, got %d", m.phase)
	}
	if m.result.Aborted {
		t.Error("should not abort when items exist")
	}
}

func TestVersionLoaderCachedResult(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.cached = &VersionResult{
		Detected: "3.2.1",
		Selected: "3.1.0",
		Versions: []string{"3.2.1", "3.1.0", "3.0.0"},
	}
	m.preselect = "3.1.0"
	m.Init()

	if m.phase != phaseSelect {
		t.Fatalf("expected phaseSelect from cache, got %d", m.phase)
	}
	// Preselect should position cursor on 3.1.0.
	if m.items[m.cursor].version != "3.1.0" {
		t.Errorf("expected cursor on 3.1.0, got %q", m.items[m.cursor].version)
	}
}

func TestVersionLoaderNoCustomVersionCmd(t *testing.T) {
	vc := &config.VersionControl{
		Command: "ruby --version",
		Pattern: `(\d+\.\d+\.\d+)`,
		// No CustomVersionCmd → auto-return detected.
	}
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{})
	m = mustVL(t, model)

	if !m.done {
		t.Fatal("expected done (no custom version support)")
	}
	if m.result.Selected != "3.2.1" {
		t.Errorf("expected selected=3.2.1, got %q", m.result.Selected)
	}
}

func TestVersionLoaderNumberKeySelect(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1", "3.0.0"}})
	m = mustVL(t, model)

	// Press '2' to select second item (3.0.0).
	model, _ = m.Update(key('2'))
	m = mustVL(t, model)

	if !m.done {
		t.Fatal("expected done after number key select")
	}
	if m.result.Selected != "3.0.0" {
		t.Errorf("expected selected=3.0.0, got %q", m.result.Selected)
	}
}

func TestVersionLoaderViewLoading(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.showSpinner = true

	v := m.View()
	content := stripANSI(v.Content)
	if !strings.Contains(content, "Detecting version") {
		t.Error("expected 'Detecting version' in loading view")
	}
}

func TestVersionLoaderViewLoadingBeforeDelay(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")

	v := m.View()
	if v.Content != "" {
		t.Errorf("expected empty content before spinner delay, got %q", v.Content)
	}
}

func TestVersionLoaderViewSelect(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{versions: []string{"3.2.1", "3.0.0"}})
	m = mustVL(t, model)

	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view in select phase")
	}
	if !strings.Contains(content, "3.2.1") {
		t.Error("expected version '3.2.1' in select view")
	}
	if !strings.Contains(content, "Custom...") {
		t.Error("expected 'Custom...' in select view")
	}
}

func TestVersionLoaderViewInput(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{})
	m = mustVL(t, model)

	if m.phase != phaseInput {
		t.Fatalf("expected phaseInput, got %d", m.phase)
	}
	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view in input phase")
	}
	if !strings.Contains(content, "version") {
		t.Error("expected 'version' text in input view")
	}
}

func TestVersionLoaderViewVerifying(t *testing.T) {
	vc := testVC()
	vc.CustomVersionVerify = "test-verify"
	m := newVersionLoaderModel("rails", vc, "")
	m.Init()

	model, _ := m.Update(versionDetectedMsg{version: "3.2.1"})
	m = mustVL(t, model)
	model, _ = m.Update(versionsListedMsg{})
	m = mustVL(t, model)

	for _, c := range "2.7.0" {
		model, _ = m.Update(key(c))
		m = mustVL(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustVL(t, model)
	if m.phase != phaseVerifying {
		t.Fatalf("expected phaseVerifying, got %d", m.phase)
	}

	m.showSpinner = true
	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view in verifying phase")
	}
	if !strings.Contains(content, "verifying") {
		t.Error("expected 'verifying' text in verifying view")
	}
}

func TestVersionLoaderViewDone(t *testing.T) {
	vc := testVC()
	m := newVersionLoaderModel("rails", vc, "")
	m.done = true

	v := m.View()
	if v.Content != "" {
		t.Errorf("expected empty content when done, got %q", v.Content)
	}
}
