package wizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
)

func mustPresets(t *testing.T, model tea.Model) *PresetsModel {
	t.Helper()
	m, ok := model.(*PresetsModel)
	if !ok {
		t.Fatalf("expected *PresetsModel, got %T", model)
	}
	return m
}

func presetOptions() []config.Option {
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
			Name:  "verbose",
			Type:  config.OptionConfirm,
			Label: "Verbose",
		},
	}
}

func TestPresetsEmptyList(t *testing.T) {
	m := newPresetsModel(presetOptions(), nil, nil, nil)
	m.Init()

	view := m.viewList()
	if view == "" {
		t.Fatal("expected non-empty view")
	}
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
}

func TestPresetsCreateEmpty(t *testing.T) {
	m := newPresetsModel(presetOptions(), nil, nil, nil)
	m.Init()

	// Press 'n' to start creating.
	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)
	if m.mode != presetsNameMode {
		t.Fatalf("expected name mode, got %d", m.mode)
	}

	// Type "my-preset".
	for _, c := range "my-preset" {
		model, _ = m.Update(key(c))
		m = mustPresets(t, model)
	}

	// Submit name — only "Empty" source since no presets/last-used exist.
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	// Should go straight to values mode (single source = Empty).
	if m.mode != presetsValuesMode {
		t.Fatalf("expected values mode, got %d", m.mode)
	}
	if m.activeName != "my-preset" {
		t.Errorf("expected activeName=my-preset, got %q", m.activeName)
	}
	if _, ok := m.presets["my-preset"]; !ok {
		t.Fatal("expected preset to exist")
	}
}

func TestPresetsCreateFromLastUsed(t *testing.T) {
	lastUsed := config.Values{"db": config.StringVal("pg")}
	m := newPresetsModel(presetOptions(), nil, lastUsed, nil)
	m.Init()

	// Press 'n' and type name.
	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)
	for _, c := range "from-lu" {
		model, _ = m.Update(key(c))
		m = mustPresets(t, model)
	}

	// Submit name — should show source selection (has last-used).
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsSourceMode {
		t.Fatalf("expected source mode, got %d", m.mode)
	}

	// Select "Last-used values" (index 1).
	model, _ = m.Update(key('2'))
	m = mustPresets(t, model)
	if m.mode != presetsValuesMode {
		t.Fatalf("expected values mode after source selection, got %d", m.mode)
	}
	vals := m.presets["from-lu"]
	if vals["db"].String() != "pg" {
		t.Errorf("expected db=pg from last-used, got %v", vals["db"])
	}
}

func TestPresetsCreateDuplicate(t *testing.T) {
	existing := map[string]config.Values{
		"original": {"db": config.StringVal("mysql")},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	// Create new preset.
	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)
	for _, c := range "copy" {
		model, _ = m.Update(key(c))
		m = mustPresets(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsSourceMode {
		t.Fatalf("expected source mode, got %d", m.mode)
	}

	// Select "Copy: original" (index 1, since no last-used).
	model, _ = m.Update(key('2'))
	m = mustPresets(t, model)
	vals := m.presets["copy"]
	if vals["db"].String() != "mysql" {
		t.Errorf("expected db=mysql from duplicate, got %v", vals["db"])
	}
}

func TestPresetsRename(t *testing.T) {
	existing := map[string]config.Values{
		"old-name": {"db": config.StringVal("pg")},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	// Press 'r' to rename.
	model, _ := m.Update(key('r'))
	m = mustPresets(t, model)
	if m.mode != presetsNameMode {
		t.Fatalf("expected name mode, got %d", m.mode)
	}
	if m.renamingFrom != "old-name" {
		t.Errorf("expected renamingFrom=old-name, got %q", m.renamingFrom)
	}

	// Clear and type new name.
	// Select all text then type over.
	m.nameInput.SetValue("")
	for _, c := range "new-name" {
		model, _ = m.Update(key(c))
		m = mustPresets(t, model)
	}

	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode after rename, got %d", m.mode)
	}
	if _, ok := m.presets["old-name"]; ok {
		t.Error("expected old-name to be removed")
	}
	if _, ok := m.presets["new-name"]; !ok {
		t.Fatal("expected new-name to exist")
	}
	if m.presets["new-name"]["db"].String() != "pg" {
		t.Error("expected values preserved after rename")
	}
}

func TestPresetsDelete(t *testing.T) {
	existing := map[string]config.Values{
		"to-delete": {"db": config.StringVal("pg")},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	// Press 'd' to delete.
	model, _ := m.Update(key('d'))
	m = mustPresets(t, model)
	if m.mode != presetsDeleteMode {
		t.Fatalf("expected delete mode, got %d", m.mode)
	}

	// Confirm with 'y'.
	model, _ = m.Update(key('y'))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode after delete, got %d", m.mode)
	}
	if len(m.presets) != 0 {
		t.Errorf("expected 0 presets after delete, got %d", len(m.presets))
	}
}

func TestPresetsDeleteCancel(t *testing.T) {
	existing := map[string]config.Values{
		"keep-me": {"db": config.StringVal("pg")},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	model, _ := m.Update(key('d'))
	m = mustPresets(t, model)

	// Press 'n' to cancel.
	model, _ = m.Update(key('n'))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode after cancel, got %d", m.mode)
	}
	if len(m.presets) != 1 {
		t.Errorf("expected 1 preset preserved, got %d", len(m.presets))
	}
}

func TestPresetsDuplicateNameRejected(t *testing.T) {
	existing := map[string]config.Values{
		"taken": {"db": config.StringVal("pg")},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	// Try to create with existing name.
	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)
	for _, c := range "taken" {
		model, _ = m.Update(key(c))
		m = mustPresets(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsNameMode {
		t.Fatalf("expected to stay in name mode, got %d", m.mode)
	}
	if m.nameErr == "" {
		t.Error("expected name error for duplicate")
	}
}

func TestPresetsEmptyNameRejected(t *testing.T) {
	m := newPresetsModel(presetOptions(), nil, nil, nil)
	m.Init()

	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)

	// Submit empty name.
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsNameMode {
		t.Fatalf("expected to stay in name mode, got %d", m.mode)
	}
	if m.nameErr == "" {
		t.Error("expected name error for empty name")
	}
}

func TestPresetsEditValues(t *testing.T) {
	existing := map[string]config.Values{
		"test": {
			"db":      config.StringVal("pg"),
			"verbose": config.BoolVal(false),
		},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	// Enter preset values.
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsValuesMode {
		t.Fatalf("expected values mode, got %d", m.mode)
	}

	// Cycle db value right (pg → mysql).
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPresets(t, model)
	if m.editor.values["db"].String() != "mysql" {
		t.Errorf("expected db=mysql after cycle, got %v", m.editor.values["db"])
	}

	// Esc back to list — all values present, should save.
	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
	if m.presets["test"]["db"].String() != "mysql" {
		t.Errorf("expected db=mysql saved, got %v", m.presets["test"]["db"])
	}
}

func TestPresetsValueEditMode(t *testing.T) {
	existing := map[string]config.Values{
		"test": {},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	// Enter preset values.
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)

	// Enter edit mode for db option.
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if !m.editor.Editing() {
		t.Fatal("expected editor to be in editing mode")
	}

	// Select second choice (mysql) via number key.
	model, _ = m.Update(key('2'))
	m = mustPresets(t, model)
	if m.editor.Editing() {
		t.Fatal("expected editor to exit editing after submit")
	}
	if m.editor.values["db"].String() != "mysql" {
		t.Errorf("expected db=mysql, got %v", m.editor.values["db"])
	}
}

func TestPresetsCursorWrapping(t *testing.T) {
	existing := map[string]config.Values{
		"a": {},
		"b": {},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	if m.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.cursor)
	}

	// Up from 0 wraps to last.
	m.Update(specialKey(tea.KeyUp))
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}

	// Down from last wraps to 0.
	m.Update(specialKey(tea.KeyDown))
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
}

func TestPresetsInvalidNameChars(t *testing.T) {
	m := newPresetsModel(presetOptions(), nil, nil, nil)
	m.Init()

	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)

	// Type name with path separator.
	for _, c := range "../bad" {
		model, _ = m.Update(key(c))
		m = mustPresets(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.nameErr == "" {
		t.Error("expected error for path separator in name")
	}
}

func TestPresetsNameCancelReturnsToList(t *testing.T) {
	m := newPresetsModel(presetOptions(), nil, nil, nil)
	m.Init()

	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)
	if m.mode != presetsNameMode {
		t.Fatalf("expected name mode")
	}

	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode after cancel, got %d", m.mode)
	}
}

func TestPresetsSourceCancelReturnsToList(t *testing.T) {
	lastUsed := config.Values{"db": config.StringVal("pg")}
	m := newPresetsModel(presetOptions(), nil, lastUsed, nil)
	m.Init()

	// Create with source selection.
	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)
	for _, c := range "test" {
		model, _ = m.Update(key(c))
		m = mustPresets(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsSourceMode {
		t.Fatalf("expected source mode, got %d", m.mode)
	}

	// Cancel source selection.
	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode after cancel, got %d", m.mode)
	}
	if _, exists := m.presets["test"]; exists {
		t.Error("expected preset not to be created on cancel")
	}
}

func requiredPresetOptions() []config.Option {
	return []config.Option{
		{
			Name:     "name",
			Type:     config.OptionInput,
			Label:    "App name",
			Required: true,
		},
		{
			Name:  "db",
			Type:  config.OptionSelect,
			Label: "Database",
			Choices: []config.Choice{
				{Value: "pg", Label: "PostgreSQL"},
				{Value: "mysql", Label: "MySQL"},
			},
		},
	}
}

func TestPresetsExitWarnsOnMissingRequired(t *testing.T) {
	existing := map[string]config.Values{
		"incomplete": {"db": config.StringVal("pg")}, // missing required "name"
	}
	m := newPresetsModel(requiredPresetOptions(), existing, nil, nil)
	m.Init()

	// Enter preset values.
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)

	// First Esc — should warn about missing required, stay in values mode.
	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustPresets(t, model)
	if m.mode != presetsValuesMode {
		t.Fatalf("expected to stay in values mode, got %d", m.mode)
	}
	if m.valuesWarning == "" {
		t.Fatal("expected warning about missing required values")
	}

	// Second Esc — should force exit.
	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode after second esc, got %d", m.mode)
	}
}

func TestPresetsExitAllowedWithoutRequired(t *testing.T) {
	// No required options — should exit freely even with missing values.
	existing := map[string]config.Values{
		"partial": {}, // no values, but none required
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)

	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}
	if m.valuesWarning != "" {
		t.Errorf("expected no warning, got %q", m.valuesWarning)
	}
}

func TestPresetsWarningClearsOnValueChange(t *testing.T) {
	existing := map[string]config.Values{
		"incomplete": {},
	}
	m := newPresetsModel(requiredPresetOptions(), existing, nil, nil)
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)

	// First Esc — triggers warning.
	model, _ = m.Update(specialKey(tea.KeyEscape))
	m = mustPresets(t, model)
	if m.valuesWarning == "" {
		t.Fatal("expected warning")
	}

	// Cycle a value — warning should clear.
	model, _ = m.Update(specialKey(tea.KeyDown))
	m = mustPresets(t, model)
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPresets(t, model)
	if m.valuesWarning != "" {
		t.Errorf("expected warning cleared after value change, got %q", m.valuesWarning)
	}
	if m.exitWarned {
		t.Error("expected exitWarned reset after value change")
	}
}

func TestPresetsNoActionOnEmptyList(t *testing.T) {
	m := newPresetsModel(presetOptions(), nil, nil, nil)
	m.Init()

	// Enter on empty list does nothing.
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode, got %d", m.mode)
	}

	// Rename on empty list does nothing.
	model, _ = m.Update(key('r'))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode after r on empty, got %d", m.mode)
	}

	// Delete on empty list does nothing.
	model, _ = m.Update(key('d'))
	m = mustPresets(t, model)
	if m.mode != presetsListMode {
		t.Fatalf("expected list mode after d on empty, got %d", m.mode)
	}
}

func TestPresetsViewListMode(t *testing.T) {
	existing := map[string]config.Values{
		"dev":  {"db": config.StringVal("pg")},
		"prod": {"db": config.StringVal("mysql")},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view content in list mode")
	}
	if !strings.Contains(content, "Manage presets") {
		t.Error("expected title 'Manage presets' in list view")
	}
	if !strings.Contains(content, "dev") {
		t.Error("expected preset name 'dev' in list view")
	}
	if !strings.Contains(content, "prod") {
		t.Error("expected preset name 'prod' in list view")
	}
}

func TestPresetsViewValuesMode(t *testing.T) {
	existing := map[string]config.Values{
		"test": {"db": config.StringVal("pg")},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsValuesMode {
		t.Fatalf("expected values mode, got %d", m.mode)
	}

	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view content in values mode")
	}
	if !strings.Contains(content, "test") {
		t.Error("expected preset name 'test' in values view title")
	}
	if !strings.Contains(content, "Database") {
		t.Error("expected option label 'Database' in values view")
	}
}

func TestPresetsViewNameMode(t *testing.T) {
	m := newPresetsModel(presetOptions(), nil, nil, nil)
	m.Init()

	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)
	if m.mode != presetsNameMode {
		t.Fatalf("expected name mode, got %d", m.mode)
	}

	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view content in name mode")
	}
	if !strings.Contains(content, "New preset") {
		t.Error("expected 'New preset' title in name view")
	}
}

func TestPresetsViewSourceMode(t *testing.T) {
	lastUsed := config.Values{"db": config.StringVal("pg")}
	m := newPresetsModel(presetOptions(), nil, lastUsed, nil)
	m.Init()

	// Create new to get to source mode.
	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)
	for _, c := range "test" {
		model, _ = m.Update(key(c))
		m = mustPresets(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsSourceMode {
		t.Fatalf("expected source mode, got %d", m.mode)
	}

	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view content in source mode")
	}
	if !strings.Contains(content, "Start from") {
		t.Error("expected 'Start from' text in source view")
	}
	if !strings.Contains(content, "Empty") {
		t.Error("expected 'Empty' source option")
	}
	if !strings.Contains(content, "Last-used") {
		t.Error("expected 'Last-used' source option")
	}
}

func TestPresetsViewDeleteMode(t *testing.T) {
	existing := map[string]config.Values{
		"doomed": {},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	model, _ := m.Update(key('d'))
	m = mustPresets(t, model)
	if m.mode != presetsDeleteMode {
		t.Fatalf("expected delete mode, got %d", m.mode)
	}

	v := m.View()
	content := stripANSI(v.Content)
	if content == "" {
		t.Fatal("expected non-empty view content in delete mode")
	}
	if !strings.Contains(content, "Delete") {
		t.Error("expected 'Delete' text in delete view")
	}
	if !strings.Contains(content, "doomed") {
		t.Error("expected preset name 'doomed' in delete view")
	}
}

func TestPresetsViewWhenDone(t *testing.T) {
	m := newPresetsModel(presetOptions(), nil, nil, nil)
	m.Init()
	m.done = true

	v := m.View()
	if v.Content != "" {
		t.Errorf("expected empty content when done, got %q", v.Content)
	}
}

func TestPresetsViewPresetRow(t *testing.T) {
	existing := map[string]config.Values{
		"staging": {"db": config.StringVal("pg"), "verbose": config.BoolVal(true)},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	content := stripANSI(m.viewList())
	if !strings.Contains(content, "staging") {
		t.Error("expected preset name 'staging' in viewList")
	}
	if !strings.Contains(content, "2 values") {
		t.Error("expected '2 values' count in preset row")
	}
}

func TestPresetsUpdateValuesNav(t *testing.T) {
	existing := map[string]config.Values{
		"test": {},
	}
	m := newPresetsModel(presetOptions(), existing, nil, nil)
	m.Init()

	// Enter preset values.
	model, _ := m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.editor.cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.editor.cursor)
	}

	// Navigate down.
	model, _ = m.Update(specialKey(tea.KeyDown))
	m = mustPresets(t, model)
	if m.editor.cursor != 1 {
		t.Errorf("expected cursor at 1 after down, got %d", m.editor.cursor)
	}

	// Navigate up.
	model, _ = m.Update(specialKey(tea.KeyUp))
	m = mustPresets(t, model)
	if m.editor.cursor != 0 {
		t.Errorf("expected cursor at 0 after up, got %d", m.editor.cursor)
	}

	// Cycle right on select field.
	model, _ = m.Update(specialKey(tea.KeyRight))
	m = mustPresets(t, model)
	if v := m.editor.values["db"]; v.String() != "pg" {
		t.Errorf("expected db=pg after cycle right, got %v", v)
	}

	// Cycle left on select field.
	model, _ = m.Update(specialKey(tea.KeyLeft))
	m = mustPresets(t, model)
	if _, ok := m.editor.values["db"]; ok {
		t.Error("expected db unpinned after cycle left")
	}
}

func TestPresetsUpdateSource(t *testing.T) {
	lastUsed := config.Values{"db": config.StringVal("pg")}
	existing := map[string]config.Values{
		"base": {"db": config.StringVal("mysql")},
	}
	m := newPresetsModel(presetOptions(), existing, lastUsed, nil)
	m.Init()

	// Create new preset with source selection.
	model, _ := m.Update(key('n'))
	m = mustPresets(t, model)
	for _, c := range "newp" {
		model, _ = m.Update(key(c))
		m = mustPresets(t, model)
	}
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsSourceMode {
		t.Fatalf("expected source mode, got %d", m.mode)
	}

	// Navigate down in source selection.
	if m.sourceCursor != 0 {
		t.Fatalf("expected sourceCursor at 0, got %d", m.sourceCursor)
	}
	model, _ = m.Update(specialKey(tea.KeyDown))
	m = mustPresets(t, model)
	if m.sourceCursor != 1 {
		t.Errorf("expected sourceCursor at 1 after down, got %d", m.sourceCursor)
	}

	// Navigate up wraps.
	model, _ = m.Update(specialKey(tea.KeyUp))
	m = mustPresets(t, model)
	if m.sourceCursor != 0 {
		t.Errorf("expected sourceCursor at 0 after up, got %d", m.sourceCursor)
	}

	// Select "Empty" via Enter.
	model, _ = m.Update(specialKey(tea.KeyEnter))
	m = mustPresets(t, model)
	if m.mode != presetsValuesMode {
		t.Fatalf("expected values mode after source select, got %d", m.mode)
	}
	if len(m.presets["newp"]) != 0 {
		t.Error("expected empty values from 'Empty' source")
	}
}

func TestPresetsFinishRename(t *testing.T) {
	t.Run("empty_name_rejected", func(t *testing.T) {
		existing := map[string]config.Values{
			"old": {"db": config.StringVal("pg")},
		}
		m := newPresetsModel(presetOptions(), existing, nil, nil)
		m.Init()

		// Start rename.
		model, _ := m.Update(key('r'))
		m = mustPresets(t, model)

		// Clear name and submit empty.
		m.nameInput.SetValue("")
		model, _ = m.Update(specialKey(tea.KeyEnter))
		m = mustPresets(t, model)
		if m.mode != presetsNameMode {
			t.Fatalf("expected name mode for empty name, got %d", m.mode)
		}
		if m.nameErr == "" {
			t.Error("expected error for empty name")
		}
	})

	t.Run("same_name_returns_to_list", func(t *testing.T) {
		existing := map[string]config.Values{
			"keep": {"db": config.StringVal("pg")},
		}
		m := newPresetsModel(presetOptions(), existing, nil, nil)
		m.Init()

		// Start rename.
		model, _ := m.Update(key('r'))
		m = mustPresets(t, model)

		// Submit same name.
		model, _ = m.Update(specialKey(tea.KeyEnter))
		m = mustPresets(t, model)
		if m.mode != presetsListMode {
			t.Fatalf("expected list mode for same name, got %d", m.mode)
		}
		if _, ok := m.presets["keep"]; !ok {
			t.Error("expected preset 'keep' to still exist")
		}
	})
}
