package wizard

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
)

func mustPins(t *testing.T, model tea.Model) *PinsModel {
	t.Helper()
	m, ok := model.(*PinsModel)
	if !ok {
		t.Fatalf("expected *PinsModel, got %T", model)
	}
	return m
}

func testOptions() []config.Option {
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
			Name:    "name",
			Type:    config.OptionInput,
			Label:   "App name",
			Default: new(config.StringVal("myapp")),
		},
	}
}

func TestPinViaEditSelect(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}

	model, _ = m.Update(key('2'))
	m = mustPins(t, model)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode after submit, got %d", m.mode)
	}
	if v, ok := m.editor.values["db"]; !ok || v.String() != "mysql" {
		t.Errorf("expected db pinned to mysql, got %v", m.editor.values["db"])
	}
}

func TestPinViaEditConfirm(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	m.Update(specialKey(tea.KeyDown))
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	model, _ = m.Update(key('1'))
	m = mustPins(t, model)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
	if v, ok := m.editor.values["api"]; !ok || v.Bool() != true {
		t.Errorf("expected api pinned to true, got %v", m.editor.values["api"])
	}
}

func TestTogglePinSpace(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), LastUsed: config.Values{"db": config.StringVal("pg")}})
	m.Init()

	model, _ := m.Update(specialKey(tea.KeySpace))
	m = mustPins(t, model)
	if _, ok := m.editor.values["db"]; !ok {
		t.Fatal("expected db to be pinned after space")
	}
	if m.editor.values["db"].String() != "pg" {
		t.Errorf("expected pinned value pg, got %v", m.editor.values["db"])
	}

	model, _ = m.Update(specialKey(tea.KeySpace))
	m = mustPins(t, model)
	if _, ok := m.editor.values["db"]; ok {
		t.Fatal("expected db to be unpinned after second space")
	}
}

func TestCancelEdit(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode")
	}

	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustPins(t, model)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode after esc")
	}
	if _, ok := m.editor.values["db"]; ok {
		t.Error("expected no pin after cancel")
	}
}

func TestCursorWrapping(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}

	m.Update(specialKey(tea.KeyUp))
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2 after up-wrap, got %d", m.cursor)
	}

	m.Update(specialKey(tea.KeyDown))
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 after down-wrap, got %d", m.cursor)
	}
}

func TestNumberKeyEntersEdit(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	model, _ := m.Update(key('2'))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode")
	}
	if m.editor.editIdx != 1 {
		t.Errorf("expected editIdx=1, got %d", m.editor.editIdx)
	}
}

func TestEditUpdatesExistingPin(t *testing.T) {
	pins := config.Values{"db": config.StringVal("pg")}
	m := newPinsModel(PinsParams{Options: testOptions(), Pins: pins})
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	model, _ = m.Update(key('2'))
	m = mustPins(t, model)
	if m.editor.values["db"].String() != "mysql" {
		t.Errorf("expected db updated to mysql, got %v", m.editor.values["db"])
	}
}

func TestVersionPinWithoutVerify(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true})
	m.Init()

	// Enter edit for version (index 0)
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}

	// Type "7.2"
	for _, c := range "7.2" {
		model, _ = m.Update(key(c))
		m = mustPins(t, model)
	}

	// Submit
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
	if m.versionPin != "7.2" {
		t.Errorf("expected version pin 7.2, got %q", m.versionPin)
	}
}

func TestVersionPinWithVerifyEntersVerifying(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true, CustomVersionVerify: "echo ok"})
	m.Init()

	// Enter edit for version (index 0)
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	// Type "7.2"
	for _, c := range "7.2" {
		model, _ = m.Update(key(c))
		m = mustPins(t, model)
	}

	// Submit → should enter verifying mode
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsVerifyingMode {
		t.Fatalf("expected verifying mode, got %d", m.mode)
	}
}

func TestHandleVersionVerifiedError(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true, CustomVersionVerify: "echo ok"})
	m.Init()

	// Enter edit for version
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	// Type version and submit to enter verifying mode
	for _, c := range "bad" {
		model, _ = m.Update(key(c))
		m = mustPins(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	// Simulate verification failure
	model, _ = m.Update(versionVerifiedMsg{version: "bad", err: errors.New("not found")})
	m = mustPins(t, model)

	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode after verify error, got %d", m.mode)
	}
	if m.verifyErr == "" {
		t.Error("expected verifyErr to be set")
	}
	if m.versionPin != "" {
		t.Errorf("expected version pin unchanged (empty), got %q", m.versionPin)
	}
}

func TestHandleVersionVerifiedSuccess(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true, CustomVersionVerify: "echo ok"})
	m.Init()

	// Enter edit for version
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	// Type version and submit to enter verifying mode
	for _, c := range "7.2" {
		model, _ = m.Update(key(c))
		m = mustPins(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	// Simulate verification success
	model, _ = m.Update(versionVerifiedMsg{version: "7.2", err: nil})
	m = mustPins(t, model)

	if m.mode != pinsListMode {
		t.Fatalf("expected list mode after verify success, got %d", m.mode)
	}
	if m.versionPin != "7.2" {
		t.Errorf("expected version pin 7.2, got %q", m.versionPin)
	}
	if m.verifyErr != "" {
		t.Errorf("expected verifyErr cleared, got %q", m.verifyErr)
	}
}

func TestEmptyVersionPinMapsToCurrent(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true, CustomVersionVerify: "echo ok"})
	m.Init()

	// Enter edit for version
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	// Submit empty → maps to "current", no verification
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
	if m.versionPin != "current" {
		t.Errorf("expected version pin 'current', got %q", m.versionPin)
	}
}

func validatedInputOptions() []config.Option {
	return []config.Option{
		{
			Name:     "port",
			Type:     config.OptionInput,
			Label:    "Port",
			Required: true,
			Validate: &config.InputRule{Pattern: `^\d+$`, Message: "must be a number"},
		},
	}
}

func TestSpaceOnInputWithoutValidDefault(t *testing.T) {
	m := newPinsModel(PinsParams{Options: validatedInputOptions()})
	m.Init()

	// Space on required input with no stored value → enters edit mode
	model, _ := m.Update(specialKey(tea.KeySpace))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}
	if _, ok := m.editor.values["port"]; ok {
		t.Error("expected no pin saved")
	}
}

func TestSpaceOnInputWithValidLastUsed(t *testing.T) {
	m := newPinsModel(PinsParams{
		Options:  validatedInputOptions(),
		LastUsed: config.Values{"port": config.StringVal("3000")},
	})
	m.Init()

	// Space on input with valid last-used → quick-pins
	model, _ := m.Update(specialKey(tea.KeySpace))
	m = mustPins(t, model)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
	if v, ok := m.editor.values["port"]; !ok || v.String() != "3000" {
		t.Errorf("expected port pinned to 3000, got %v", m.editor.values["port"])
	}
}

func TestSpaceOnInputWithInvalidLastUsed(t *testing.T) {
	m := newPinsModel(PinsParams{
		Options:  validatedInputOptions(),
		LastUsed: config.Values{"port": config.StringVal("abc")},
	})
	m.Init()

	// Space on input with invalid last-used → enters edit mode
	model, _ := m.Update(specialKey(tea.KeySpace))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}
	if _, ok := m.editor.values["port"]; ok {
		t.Error("expected no pin saved for invalid last-used value")
	}
}

func TestPinInputRejectsInvalidValue(t *testing.T) {
	m := newPinsModel(PinsParams{Options: validatedInputOptions()})
	m.Init()

	// Enter edit mode
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}

	// Type invalid value (non-numeric)
	for _, c := range "abc" {
		model, _ = m.Update(key(c))
		m = mustPins(t, model)
	}

	// Submit — should stay in edit mode
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode after invalid input, got %d", m.mode)
	}
	if _, ok := m.editor.values["port"]; ok {
		t.Error("expected no pin saved for invalid input")
	}
}

func TestPinInputRejectsBlankRequired(t *testing.T) {
	m := newPinsModel(PinsParams{Options: validatedInputOptions()})
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	// Submit empty — should stay in edit mode (required field)
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode after blank required input, got %d", m.mode)
	}
	if _, ok := m.editor.values["port"]; ok {
		t.Error("expected no pin saved for blank required input")
	}
}

func TestPinInputAcceptsValidValue(t *testing.T) {
	m := newPinsModel(PinsParams{Options: validatedInputOptions()})
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)

	for _, c := range "8080" {
		model, _ = m.Update(key(c))
		m = mustPins(t, model)
	}

	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode after valid input, got %d", m.mode)
	}
	if v, ok := m.editor.values["port"]; !ok || v.String() != "8080" {
		t.Errorf("expected port pinned to 8080, got %v", m.editor.values["port"])
	}
}

func TestCyclePinSelectForward(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	// First right → pins db to first choice (pg).
	model, _ := m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if v, ok := m.editor.values["db"]; !ok || v.String() != "pg" {
		t.Fatalf("expected db=pg, got %v", m.editor.values["db"])
	}

	// Second right → pg → mysql.
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if v := m.editor.values["db"]; v.String() != "mysql" {
		t.Fatalf("expected db=mysql, got %v", v)
	}

	// Third right → mysql → unpinned.
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if _, ok := m.editor.values["db"]; ok {
		t.Fatal("expected db unpinned")
	}

	// Fourth right → wraps to pg.
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if v := m.editor.values["db"]; v.String() != "pg" {
		t.Fatalf("expected db=pg after wrap, got %v", v)
	}
}

func TestCyclePinSelectBackward(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	// Left from unpinned → wraps to last choice (mysql).
	model, _ := m.Update(specialKey(tea.KeyLeft))
	m = mustPins(t, model)
	if v := m.editor.values["db"]; v.String() != "mysql" {
		t.Fatalf("expected db=mysql, got %v", v)
	}

	// Left → mysql → pg.
	model, _ = m.Update(specialKey(tea.KeyLeft))
	m = mustPins(t, model)
	if v := m.editor.values["db"]; v.String() != "pg" {
		t.Fatalf("expected db=pg, got %v", v)
	}

	// Left → pg → unpinned.
	model, _ = m.Update(specialKey(tea.KeyLeft))
	m = mustPins(t, model)
	if _, ok := m.editor.values["db"]; ok {
		t.Fatal("expected db unpinned")
	}
}

func TestCyclePinConfirm(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	// Move cursor to confirm field (index 1).
	m.Update(specialKey(tea.KeyDown))

	// Right → Yes (true).
	model, _ := m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if v, ok := m.editor.values["api"]; !ok || v.Bool() != true {
		t.Fatalf("expected api=true, got %v", m.editor.values["api"])
	}

	// Right → No (false).
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if v := m.editor.values["api"]; v.Bool() != false {
		t.Fatalf("expected api=false, got %v", v)
	}

	// Right → unpinned.
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if _, ok := m.editor.values["api"]; ok {
		t.Fatal("expected api unpinned")
	}

	// Right → wraps to Yes.
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if v := m.editor.values["api"]; v.Bool() != true {
		t.Fatalf("expected api=true after wrap, got %v", v)
	}
}

func TestCyclePinSelectAllowNone(t *testing.T) {
	opts := []config.Option{
		{
			Name:      "db",
			Type:      config.OptionSelect,
			Label:     "Database",
			AllowNone: true,
			Choices:   []config.Choice{{Value: "pg", Label: "PostgreSQL"}},
		},
	}
	m := newPinsModel(PinsParams{Options: opts})
	m.Init()

	// Right → pg.
	model, _ := m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if v := m.editor.values["db"]; v.String() != "pg" {
		t.Fatalf("expected db=pg, got %v", v)
	}

	// Right → None.
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if v := m.editor.values["db"]; v.String() != config.NoneValue {
		t.Fatalf("expected db=%s, got %v", config.NoneValue, v)
	}

	// Right → unpinned.
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if _, ok := m.editor.values["db"]; ok {
		t.Fatal("expected db unpinned")
	}
}

func TestCyclePinInputNoop(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	// Move cursor to input field (index 2).
	m.Update(specialKey(tea.KeyDown))
	m.Update(specialKey(tea.KeyDown))

	model, _ := m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if _, ok := m.editor.values["name"]; ok {
		t.Fatal("expected no pin change for input field")
	}
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
}

func TestCyclePinMultiSelectNoop(t *testing.T) {
	opts := []config.Option{
		{
			Name:  "tags",
			Type:  config.OptionMultiSelect,
			Label: "Tags",
			Choices: []config.Choice{
				{Value: "a", Label: "A"},
				{Value: "b", Label: "B"},
			},
		},
	}
	m := newPinsModel(PinsParams{Options: opts})
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if _, ok := m.editor.values["tags"]; ok {
		t.Fatal("expected no pin change for multi-select field")
	}
}

func TestCyclePinVersionToggle(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true})
	m.Init()

	// Right on version (index 0) → pins to "current".
	model, _ := m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if m.versionPin != versionPinCurrent {
		t.Fatalf("expected version pin 'current', got %q", m.versionPin)
	}

	// Right again → unpins.
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPins(t, model)
	if m.versionPin != "" {
		t.Fatalf("expected version unpinned, got %q", m.versionPin)
	}
}

func TestCyclePinWithHLKeys(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	// 'l' key should cycle forward like right arrow.
	model, _ := m.Update(key('l'))
	m = mustPins(t, model)
	if v, ok := m.editor.values["db"]; !ok || v.String() != "pg" {
		t.Fatalf("expected db=pg via 'l', got %v", m.editor.values["db"])
	}

	// 'h' key should cycle backward like left arrow.
	model, _ = m.Update(key('h'))
	m = mustPins(t, model)
	if _, ok := m.editor.values["db"]; ok {
		t.Fatal("expected db unpinned via 'h'")
	}
}

func TestPinsViewListMode(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view content in list mode")
	}
	if !strings.Contains(content, "Database") {
		t.Error("expected option label 'Database' in list view")
	}
	if !strings.Contains(content, "API mode?") {
		t.Error("expected option label 'API mode?' in list view")
	}
	if !strings.Contains(content, "App name") {
		t.Error("expected option label 'App name' in list view")
	}
}

func TestPinsViewEditMode(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()

	// Enter edit mode for first option.
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}

	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view content in edit mode")
	}
}

func TestPinsViewVerifyingMode(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true, CustomVersionVerify: "echo ok"})
	m.Init()

	// Enter version edit, type a value, submit to enter verifying.
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	for _, c := range "7.2" {
		model, _ = m.Update(key(c))
		m = mustPins(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsVerifyingMode {
		t.Fatalf("expected verifying mode, got %d", m.mode)
	}

	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view content in verifying mode")
	}
	if !strings.Contains(content, "Verifying") {
		t.Error("expected 'Verifying' text in verifying view")
	}
}

func TestPinsViewWhenDone(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions()})
	m.Init()
	m.done = true

	v := m.View()
	if v.Content != "" {
		t.Errorf("expected empty content when done, got %q", v.Content)
	}
}

func TestPinsViewVersionRow(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true, VersionPin: "3.2.1"})
	m.Init()

	content := stripANSI(m.viewList())
	if !strings.Contains(content, "Version") {
		t.Error("expected 'Version' label in list view with custom version")
	}
	if !strings.Contains(content, "3.2.1") {
		t.Error("expected pinned version '3.2.1' in list view")
	}
}

func TestPinsUpdateVerifyingKeyPress(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true, CustomVersionVerify: "echo ok"})
	m.Init()

	// Enter version edit, type, submit.
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	for _, c := range "7.2" {
		model, _ = m.Update(key(c))
		m = mustPins(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPins(t, model)
	if m.mode != pinsVerifyingMode {
		t.Fatalf("expected verifying mode, got %d", m.mode)
	}

	// Pressing Esc during verifying returns to edit mode.
	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustPins(t, model)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode after esc in verifying, got %d", m.mode)
	}

	// Pressing a random key during verifying is ignored.
	m.mode = pinsVerifyingMode
	model, _ = m.Update(key('x'))
	m = mustPins(t, model)
	if m.mode != pinsVerifyingMode {
		t.Fatalf("expected still in verifying mode, got %d", m.mode)
	}
}

func TestPinsHandleToggleVersionRow(t *testing.T) {
	m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: true})
	m.Init()

	// Cursor should be at 0 (version row when hasCustomVersion=true).
	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}
	if !m.isVersionIdx(m.cursor) {
		t.Fatal("expected cursor to be on version index")
	}

	// Space toggles version pin.
	model, _ := m.Update(specialKey(tea.KeySpace))
	m = mustPins(t, model)
	if m.versionPin != versionPinCurrent {
		t.Errorf("expected version pin 'current', got %q", m.versionPin)
	}

	// Space again unpins.
	model, _ = m.Update(specialKey(tea.KeySpace))
	m = mustPins(t, model)
	if m.versionPin != "" {
		t.Errorf("expected version pin empty, got %q", m.versionPin)
	}
}

func TestPinsVersionOffset(t *testing.T) {
	tests := []struct {
		name             string
		hasCustomVersion bool
		wantOffset       int
	}{
		{"with_custom_version", true, 1},
		{"without_custom_version", false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newPinsModel(PinsParams{Options: testOptions(), HasCustomVersion: tt.hasCustomVersion})
			got := m.versionOffset()
			if got != tt.wantOffset {
				t.Errorf("versionOffset() = %d, want %d", got, tt.wantOffset)
			}
		})
	}
}

func TestResolveDefault(t *testing.T) {
	tests := []struct {
		name     string
		opt      config.Option
		pins     config.Values
		lastUsed config.Values
		want     *config.FieldValue
	}{
		{
			"from_pins",
			config.Option{Name: "db", Type: config.OptionSelect},
			config.Values{"db": config.StringVal("pg")},
			nil,
			new(config.StringVal("pg")),
		},
		{
			"from_last_used",
			config.Option{Name: "db", Type: config.OptionSelect},
			config.Values{},
			config.Values{"db": config.StringVal("mysql")},
			new(config.StringVal("mysql")),
		},
		{
			"from_default",
			config.Option{Name: "name", Type: config.OptionInput, Default: new(config.StringVal("myapp"))},
			config.Values{},
			config.Values{},
			new(config.StringVal("myapp")),
		},
		{
			"confirm_fallback",
			config.Option{Name: "api", Type: config.OptionConfirm},
			config.Values{},
			config.Values{},
			new(config.BoolVal(false)),
		},
		{
			"select_first_choice",
			config.Option{
				Name: "db", Type: config.OptionSelect,
				Choices: []config.Choice{{Value: "pg"}},
			},
			config.Values{},
			config.Values{},
			new(config.StringVal("pg")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDefault(&tt.opt, tt.pins, tt.lastUsed)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolveDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
