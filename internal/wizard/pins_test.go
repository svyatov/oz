package wizard

import (
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
	m := newPinsModel(testOptions(), nil, nil)
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
	m := newPinsModel(testOptions(), nil, nil)
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
	m := newPinsModel(testOptions(), nil, map[string]any{"db": "pg"})
	m.Init()

	m.Update(specialKey(tea.KeySpace))
	if _, ok := m.pins["db"]; !ok {
		t.Fatal("expected db to be pinned after space")
	}
	if m.pins["db"] != "pg" {
		t.Errorf("expected pinned value pg, got %v", m.pins["db"])
	}

	m.Update(specialKey(tea.KeySpace))
	if _, ok := m.pins["db"]; ok {
		t.Fatal("expected db to be unpinned after second space")
	}
}

func TestCancelEdit(t *testing.T) {
	m := newPinsModel(testOptions(), nil, nil)
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
	m := newPinsModel(testOptions(), nil, nil)
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
	m := newPinsModel(testOptions(), nil, nil)
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
	m := newPinsModel(testOptions(), pins, nil)
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = model.(*PinsModel)

	model, _ = m.Update(key('2'))
	m = model.(*PinsModel)
	if m.pins["db"] != "mysql" {
		t.Errorf("expected db updated to mysql, got %v", m.pins["db"])
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
