package wizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
)

func editorOptions() []config.Option {
	return []config.Option{
		{
			Name:  "db",
			Type:  config.OptionSelect,
			Label: "Database",
			Choices: []config.Choice{
				{Value: "pg", Label: "PostgreSQL"},
				{Value: "mysql", Label: "MySQL"},
				{Value: "sqlite", Label: "SQLite"},
			},
		},
		{
			Name:  "api",
			Type:  config.OptionConfirm,
			Label: "API mode?",
		},
		{
			Name:     "port",
			Type:     config.OptionInput,
			Label:    "Port",
			Required: true,
		},
	}
}

func TestCycleValueSelectForward(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)

	// First cycle → first choice (pg).
	e.CycleValue(0, 1)
	if v := e.values["db"]; v.String() != "pg" {
		t.Fatalf("expected db=pg, got %v", v)
	}

	// Second → mysql.
	e.CycleValue(0, 1)
	if v := e.values["db"]; v.String() != "mysql" {
		t.Fatalf("expected db=mysql, got %v", v)
	}

	// Third → sqlite.
	e.CycleValue(0, 1)
	if v := e.values["db"]; v.String() != "sqlite" {
		t.Fatalf("expected db=sqlite, got %v", v)
	}

	// Fourth → unpinned.
	e.CycleValue(0, 1)
	if _, ok := e.values["db"]; ok {
		t.Fatal("expected db unpinned")
	}

	// Fifth → wraps to pg.
	e.CycleValue(0, 1)
	if v := e.values["db"]; v.String() != "pg" {
		t.Fatalf("expected db=pg after wrap, got %v", v)
	}
}

func TestCycleValueSelectBackward(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)

	// Left from unpinned → wraps to last (sqlite).
	e.CycleValue(0, -1)
	if v := e.values["db"]; v.String() != "sqlite" {
		t.Fatalf("expected db=sqlite, got %v", v)
	}

	// Left → mysql.
	e.CycleValue(0, -1)
	if v := e.values["db"]; v.String() != "mysql" {
		t.Fatalf("expected db=mysql, got %v", v)
	}
}

func TestCycleValueConfirm(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)

	// Cycle confirm (index 1): unpinned → true → false → unpinned.
	e.CycleValue(1, 1)
	if v := e.values["api"]; !v.Bool() {
		t.Fatalf("expected api=true, got %v", v)
	}

	e.CycleValue(1, 1)
	if v := e.values["api"]; v.Bool() {
		t.Fatalf("expected api=false, got %v", v)
	}

	e.CycleValue(1, 1)
	if _, ok := e.values["api"]; ok {
		t.Fatal("expected api unpinned")
	}
}

func TestCycleValueInputNoop(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)
	e.CycleValue(2, 1)
	if _, ok := e.values["port"]; ok {
		t.Fatal("expected no change for input field")
	}
}

func TestToggleValueSelect(t *testing.T) {
	opts := editorOptions()
	e := NewValuesEditor(opts, nil, config.Values{"db": config.StringVal("mysql")}, nil)

	// Toggle on → uses last-used value.
	e.ToggleValue(0)
	if v := e.values["db"]; v.String() != "mysql" {
		t.Fatalf("expected db=mysql, got %v", v)
	}

	// Toggle off.
	e.ToggleValue(0)
	if _, ok := e.values["db"]; ok {
		t.Fatal("expected db removed")
	}
}

func TestToggleValueConfirm(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)
	e.ToggleValue(1)
	if _, ok := e.values["api"]; !ok {
		t.Fatal("expected api toggled on")
	}
	e.ToggleValue(1)
	if _, ok := e.values["api"]; ok {
		t.Fatal("expected api toggled off")
	}
}

func TestEnterEditAndSubmit(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)
	e.EnterEdit(0)
	if !e.Editing() {
		t.Fatal("expected editing=true")
	}
	if e.editIdx != 0 {
		t.Errorf("expected editIdx=0, got %d", e.editIdx)
	}

	// Submit by selecting first choice (enter).
	exited, _ := e.UpdateEdit(specialKey(tea.KeyEnter))
	if !exited {
		t.Fatal("expected exited after submit")
	}
	if e.Editing() {
		t.Fatal("expected editing=false after submit")
	}
	if _, ok := e.values["db"]; !ok {
		t.Fatal("expected db value set after edit submit")
	}
}

func TestEnterEditAndCancel(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)
	e.EnterEdit(0)
	if !e.Editing() {
		t.Fatal("expected editing=true")
	}

	exited, _ := e.UpdateEdit(specialKey(tea.KeyEscape))
	if !exited {
		t.Fatal("expected exited after cancel")
	}
	if e.Editing() {
		t.Fatal("expected editing=false after cancel")
	}
	if _, ok := e.values["db"]; ok {
		t.Fatal("expected no value after cancel")
	}
}

func TestMaxLabelWidth(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)
	w := e.MaxLabelWidth()
	// "API mode?" is 9, "Database" is 8, "Port" is 4.
	if w != 9 {
		t.Errorf("expected max label width 9, got %d", w)
	}
}

func TestMaxLabelWidthWithRequired(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)
	e.showRequired = true
	w := e.MaxLabelWidth()
	// "Database" is 8, "API mode?" is 9, "Port *" is 6 → 9 is max.
	if w != 9 {
		t.Errorf("expected max label width 9, got %d", w)
	}
}

func TestViewOptionRowPinIcon(t *testing.T) {
	e := NewValuesEditor(editorOptions(), config.Values{"db": config.StringVal("pg")}, nil, nil)
	row := stripANSI(e.ViewOptionRow(0, true, 10, 1, 1))
	if !strings.Contains(row, "PostgreSQL") {
		t.Error("expected formatted value in row")
	}
}

func TestViewOptionRowUnpinned(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)
	row := stripANSI(e.ViewOptionRow(0, false, 10, 1, 1))
	// Should show dash for unpinned.
	if !strings.Contains(row, "\u2500") {
		t.Error("expected dash for unpinned value")
	}
}

func TestValuesEditorViewEdit(t *testing.T) {
	e := NewValuesEditor(editorOptions(), nil, nil, nil)
	e.EnterEdit(0) // select field
	if !e.Editing() {
		t.Fatal("expected editing=true")
	}

	output := stripANSI(e.ViewEdit("PIN"))
	if output == "" {
		t.Fatal("expected non-empty ViewEdit output")
	}
	if !strings.Contains(output, "PIN") {
		t.Error("expected indicator 'PIN' in ViewEdit output")
	}
}

func TestValuesEditorEditNavHint(t *testing.T) {
	tests := []struct {
		name    string
		optIdx  int
		options []config.Option
		want    string
	}{
		{
			"select_field",
			0,
			editorOptions(), // index 0 = select
			"enter",
		},
		{
			"confirm_field",
			1,
			editorOptions(), // index 1 = confirm
			"enter",
		},
		{
			"input_field",
			2,
			editorOptions(), // index 2 = input
			"enter",
		},
		{
			"multi_select_field",
			0,
			[]config.Option{{
				Name:  "tags",
				Type:  config.OptionMultiSelect,
				Label: "Tags",
				Choices: []config.Choice{
					{Value: "a", Label: "A"},
				},
			}},
			"toggle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewValuesEditor(tt.options, nil, nil, nil)
			e.EnterEdit(tt.optIdx)
			hint := stripANSI(e.editNavHint())
			if !strings.Contains(hint, tt.want) {
				t.Errorf("editNavHint() = %q, want substring %q", hint, tt.want)
			}
		})
	}
}

func TestRenderFieldWithIndicator(t *testing.T) {
	f := NewInputField(config.Option{Label: "Test", Description: "A test field"})
	view := f.View()
	result := renderFieldWithIndicator(view, "CUSTOM")
	stripped := stripANSI(result)
	if !strings.Contains(stripped, "CUSTOM") {
		t.Error("expected 'CUSTOM' indicator in rendered output")
	}
}

func TestCycleValueSelectAllowNone(t *testing.T) {
	opts := []config.Option{
		{
			Name:      "db",
			Type:      config.OptionSelect,
			Label:     "Database",
			AllowNone: true,
			Choices:   []config.Choice{{Value: "pg", Label: "PostgreSQL"}},
		},
	}
	e := NewValuesEditor(opts, nil, nil, nil)

	// Right → pg.
	e.CycleValue(0, 1)
	if v := e.values["db"]; v.String() != "pg" {
		t.Fatalf("expected db=pg, got %v", v)
	}

	// Right → None.
	e.CycleValue(0, 1)
	if v := e.values["db"]; v.String() != config.NoneValue {
		t.Fatalf("expected db=%s, got %v", config.NoneValue, v)
	}

	// Right → unpinned.
	e.CycleValue(0, 1)
	if _, ok := e.values["db"]; ok {
		t.Fatal("expected db unpinned")
	}
}
