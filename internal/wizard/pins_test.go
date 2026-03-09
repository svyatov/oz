package wizard

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
)

func key(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: string(code)}
}

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func testOptions() []config.Option {
	return []config.Option{
		{
			Name:  "db",
			Type:  "select",
			Label: "Database",
			Choices: []config.Choice{
				{Value: "pg", Label: "PostgreSQL"},
				{Value: "mysql", Label: "MySQL"},
			},
		},
		{
			Name:  "api",
			Type:  "confirm",
			Label: "API mode?",
		},
		{
			Name:    "name",
			Type:    "input",
			Label:   "App name",
			Default: "myapp",
		},
	}
}

func TestPinViaEditSelect(t *testing.T) {
	m := newPinsModel(testOptions(), nil, nil, nil, false, "", "")
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}

	model, _ = m.Update(key('2'))
	m = model.(*PinsModel)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode after submit, got %d", m.mode)
	}
	if v, ok := m.pins["db"]; !ok || v != "mysql" {
		t.Errorf("expected db pinned to mysql, got %v", m.pins["db"])
	}
}

func TestPinViaEditConfirm(t *testing.T) {
	m := newPinsModel(testOptions(), nil, nil, nil, false, "", "")
	m.Init()

	m.Update(specialKey(tea.KeyDown))
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	model, _ = m.Update(key('1'))
	m = model.(*PinsModel)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
	if v, ok := m.pins["api"]; !ok || v != true {
		t.Errorf("expected api pinned to true, got %v", m.pins["api"])
	}
}

func TestTogglePinSpace(t *testing.T) {
	m := newPinsModel(testOptions(), nil, map[string]any{"db": "pg"}, nil, false, "", "")
	m.Init()

	model, _ := m.Update(specialKey(tea.KeySpace))
	m = model.(*PinsModel)
	if _, ok := m.pins["db"]; !ok {
		t.Fatal("expected db to be pinned after space")
	}
	if m.pins["db"] != "pg" {
		t.Errorf("expected pinned value pg, got %v", m.pins["db"])
	}

	model, _ = m.Update(specialKey(tea.KeySpace))
	m = model.(*PinsModel)
	if _, ok := m.pins["db"]; ok {
		t.Fatal("expected db to be unpinned after second space")
	}
}

func TestCancelEdit(t *testing.T) {
	m := newPinsModel(testOptions(), nil, nil, nil, false, "", "")
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode")
	}

	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = model.(*PinsModel)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode after esc")
	}
	if _, ok := m.pins["db"]; ok {
		t.Error("expected no pin after cancel")
	}
}

func TestCursorWrapping(t *testing.T) {
	m := newPinsModel(testOptions(), nil, nil, nil, false, "", "")
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
	m := newPinsModel(testOptions(), nil, nil, nil, false, "", "")
	m.Init()

	model, _ := m.Update(key('2'))
	m = model.(*PinsModel)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode")
	}
	if m.editIdx != 1 {
		t.Errorf("expected editIdx=1, got %d", m.editIdx)
	}
}

func TestEditUpdatesExistingPin(t *testing.T) {
	pins := map[string]any{"db": "pg"}
	m := newPinsModel(testOptions(), pins, nil, nil, false, "", "")
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	model, _ = m.Update(key('2'))
	m = model.(*PinsModel)
	if m.pins["db"] != "mysql" {
		t.Errorf("expected db updated to mysql, got %v", m.pins["db"])
	}
}

func TestVersionPinWithoutVerify(t *testing.T) {
	m := newPinsModel(testOptions(), nil, nil, nil, true, "", "")
	m.Init()

	// Enter edit for version (index 0)
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}

	// Type "7.2"
	for _, c := range "7.2" {
		model, _ = m.Update(key(c))
		m = model.(*PinsModel)
	}

	// Submit
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
	if m.versionPin != "7.2" {
		t.Errorf("expected version pin 7.2, got %q", m.versionPin)
	}
}

func TestVersionPinWithVerifyEntersVerifying(t *testing.T) {
	m := newPinsModel(testOptions(), nil, nil, nil, true, "", "echo ok")
	m.Init()

	// Enter edit for version (index 0)
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	// Type "7.2"
	for _, c := range "7.2" {
		model, _ = m.Update(key(c))
		m = model.(*PinsModel)
	}

	// Submit → should enter verifying mode
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
	if m.mode != pinsVerifyingMode {
		t.Fatalf("expected verifying mode, got %d", m.mode)
	}
}

func TestHandleVersionVerifiedError(t *testing.T) {
	m := newPinsModel(testOptions(), nil, nil, nil, true, "", "echo ok")
	m.Init()

	// Enter edit for version
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	// Type version and submit to enter verifying mode
	for _, c := range "bad" {
		model, _ = m.Update(key(c))
		m = model.(*PinsModel)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	// Simulate verification failure
	model, _ = m.Update(versionVerifiedMsg{version: "bad", err: errors.New("not found")})
	m = model.(*PinsModel)

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
	m := newPinsModel(testOptions(), nil, nil, nil, true, "", "echo ok")
	m.Init()

	// Enter edit for version
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	// Type version and submit to enter verifying mode
	for _, c := range "7.2" {
		model, _ = m.Update(key(c))
		m = model.(*PinsModel)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	// Simulate verification success
	model, _ = m.Update(versionVerifiedMsg{version: "7.2", err: nil})
	m = model.(*PinsModel)

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
	m := newPinsModel(testOptions(), nil, nil, nil, true, "", "echo ok")
	m.Init()

	// Enter edit for version
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	// Submit empty → maps to "current", no verification
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
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
			Type:     "input",
			Label:    "Port",
			Required: true,
			Validate: &config.InputRule{Pattern: `^\d+$`, Message: "must be a number"},
		},
	}
}

func TestSpaceOnInputWithoutValidDefault(t *testing.T) {
	m := newPinsModel(validatedInputOptions(), nil, nil, nil, false, "", "")
	m.Init()

	// Space on required input with no stored value → enters edit mode
	model, _ := m.Update(specialKey(tea.KeySpace))
	m = model.(*PinsModel)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}
	if _, ok := m.pins["port"]; ok {
		t.Error("expected no pin saved")
	}
}

func TestSpaceOnInputWithValidLastUsed(t *testing.T) {
	m := newPinsModel(validatedInputOptions(), nil, map[string]any{"port": "3000"}, nil, false, "", "")
	m.Init()

	// Space on input with valid last-used → quick-pins
	model, _ := m.Update(specialKey(tea.KeySpace))
	m = model.(*PinsModel)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
	if v, ok := m.pins["port"]; !ok || v != "3000" {
		t.Errorf("expected port pinned to 3000, got %v", m.pins["port"])
	}
}

func TestSpaceOnInputWithInvalidLastUsed(t *testing.T) {
	m := newPinsModel(validatedInputOptions(), nil, map[string]any{"port": "abc"}, nil, false, "", "")
	m.Init()

	// Space on input with invalid last-used → enters edit mode
	model, _ := m.Update(specialKey(tea.KeySpace))
	m = model.(*PinsModel)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}
	if _, ok := m.pins["port"]; ok {
		t.Error("expected no pin saved for invalid last-used value")
	}
}

func TestPinInputRejectsInvalidValue(t *testing.T) {
	m := newPinsModel(validatedInputOptions(), nil, nil, nil, false, "", "")
	m.Init()

	// Enter edit mode
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode, got %d", m.mode)
	}

	// Type invalid value (non-numeric)
	for _, c := range "abc" {
		model, _ = m.Update(key(c))
		m = model.(*PinsModel)
	}

	// Submit — should stay in edit mode
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode after invalid input, got %d", m.mode)
	}
	if _, ok := m.pins["port"]; ok {
		t.Error("expected no pin saved for invalid input")
	}
}

func TestPinInputRejectsBlankRequired(t *testing.T) {
	m := newPinsModel(validatedInputOptions(), nil, nil, nil, false, "", "")
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	// Submit empty — should stay in edit mode (required field)
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
	if m.mode != pinsEditMode {
		t.Fatalf("expected edit mode after blank required input, got %d", m.mode)
	}
	if _, ok := m.pins["port"]; ok {
		t.Error("expected no pin saved for blank required input")
	}
}

func TestPinInputAcceptsValidValue(t *testing.T) {
	m := newPinsModel(validatedInputOptions(), nil, nil, nil, false, "", "")
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	for _, c := range "8080" {
		model, _ = m.Update(key(c))
		m = model.(*PinsModel)
	}

	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)
	if m.mode != pinsListMode {
		t.Fatalf("expected list mode after valid input, got %d", m.mode)
	}
	if v, ok := m.pins["port"]; !ok || v != "8080" {
		t.Errorf("expected port pinned to 8080, got %v", m.pins["port"])
	}
}

func TestResolveDefault(t *testing.T) {
	tests := []struct {
		name     string
		opt      config.Option
		pins     map[string]any
		lastUsed map[string]any
		want     any
	}{
		{
			"from_pins",
			config.Option{Name: "db", Type: "select"},
			map[string]any{"db": "pg"},
			nil,
			"pg",
		},
		{
			"from_last_used",
			config.Option{Name: "db", Type: "select"},
			map[string]any{},
			map[string]any{"db": "mysql"},
			"mysql",
		},
		{
			"from_default",
			config.Option{Name: "name", Type: "input", Default: "myapp"},
			map[string]any{},
			map[string]any{},
			"myapp",
		},
		{
			"confirm_fallback",
			config.Option{Name: "api", Type: "confirm"},
			map[string]any{},
			map[string]any{},
			false,
		},
		{
			"select_first_choice",
			config.Option{
				Name: "db", Type: "select",
				Choices: []config.Choice{{Value: "pg"}},
			},
			map[string]any{},
			map[string]any{},
			"pg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDefault(&tt.opt, tt.pins, tt.lastUsed)
			if got != tt.want {
				t.Errorf("resolveDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
