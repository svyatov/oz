package wizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
)

func newTestMultiSelectField() *MultiSelectField {
	return NewMultiSelectField(config.Option{
		Label:       "Features",
		Description: "Select features",
		Type:        config.OptionMultiSelect,
		Choices: []config.Choice{
			{Value: "auth", Label: "Authentication"},
			{Value: "logs", Label: "Logging"},
			{Value: "cache", Label: "Caching"},
		},
	})
}

func TestMultiSelectInit(t *testing.T) {
	f := newTestMultiSelectField()
	cmd := f.Init()
	if cmd != nil {
		t.Error("expected nil cmd from Init")
	}
	if f.cursor != 0 {
		t.Errorf("expected cursor=0 after Init, got %d", f.cursor)
	}
	if len(f.selected) != 0 {
		t.Errorf("expected empty selected map, got %d entries", len(f.selected))
	}
}

func TestMultiSelectToggleSpace(t *testing.T) {
	f := newTestMultiSelectField()

	// Toggle first item on.
	f.Update(specialKey(tea.KeySpace))
	if !f.selected[0] {
		t.Fatal("expected item 0 selected")
	}

	// Toggle first item off.
	f.Update(specialKey(tea.KeySpace))
	if f.selected[0] {
		t.Fatal("expected item 0 deselected")
	}
}

func TestMultiSelectToggleX(t *testing.T) {
	f := newTestMultiSelectField()
	f.Update(key('x'))
	if !f.selected[0] {
		t.Fatal("expected item 0 selected via x")
	}
}

func TestMultiSelectToggleAll(t *testing.T) {
	f := newTestMultiSelectField()

	// Select all.
	f.Update(key('a'))
	for i := range f.choices {
		if !f.selected[i] {
			t.Errorf("expected item %d selected after select-all", i)
		}
	}

	// Deselect all.
	f.Update(key('a'))
	for i := range f.choices {
		if f.selected[i] {
			t.Errorf("expected item %d deselected after deselect-all", i)
		}
	}
}

func TestMultiSelectNumberKeys(t *testing.T) {
	f := newTestMultiSelectField()

	f.Update(key('1'))
	f.Update(key('3'))

	if !f.selected[0] {
		t.Error("expected item 0 selected via '1'")
	}
	if f.selected[1] {
		t.Error("expected item 1 not selected")
	}
	if !f.selected[2] {
		t.Error("expected item 2 selected via '3'")
	}

	vals := f.Value().Strings()
	if len(vals) != 2 || vals[0] != "auth" || vals[1] != "cache" {
		t.Errorf("expected [auth cache], got %v", vals)
	}
}

func TestMultiSelectNavigationWrapping(t *testing.T) {
	f := newTestMultiSelectField()
	if f.cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", f.cursor)
	}

	// Up from 0 wraps to last.
	f.Update(specialKey(tea.KeyUp))
	if f.cursor != 2 {
		t.Errorf("expected cursor=2 after up-wrap, got %d", f.cursor)
	}

	// Down from last wraps to 0.
	f.Update(specialKey(tea.KeyDown))
	if f.cursor != 0 {
		t.Errorf("expected cursor=0 after down-wrap, got %d", f.cursor)
	}

	// Vim keys.
	f.Update(key('j'))
	if f.cursor != 1 {
		t.Errorf("expected cursor=1 after j, got %d", f.cursor)
	}
	f.Update(key('k'))
	if f.cursor != 0 {
		t.Errorf("expected cursor=0 after k, got %d", f.cursor)
	}
}

func TestMultiSelectSubmit(t *testing.T) {
	f := newTestMultiSelectField()
	f.Update(key('1'))
	f.Update(key('3'))

	submitted, _ := f.Update(specialKey(tea.KeyEnter))
	if !submitted {
		t.Fatal("expected submitted")
	}
	vals := f.Value().Strings()
	if len(vals) != 2 || vals[0] != "auth" || vals[1] != "cache" {
		t.Errorf("expected [auth cache], got %v", vals)
	}
}

func TestMultiSelectSubmitTab(t *testing.T) {
	f := newTestMultiSelectField()
	submitted, _ := f.Update(specialKey(tea.KeyTab))
	if !submitted {
		t.Fatal("expected submitted via tab")
	}
}

func TestMultiSelectEmptySubmit(t *testing.T) {
	f := newTestMultiSelectField()
	submitted, _ := f.Update(specialKey(tea.KeyEnter))
	if !submitted {
		t.Fatal("expected submitted")
	}
	vals := f.Value().Strings()
	if len(vals) != 0 {
		t.Errorf("expected empty, got %v", vals)
	}
}

func TestMultiSelectSetValue(t *testing.T) {
	f := newTestMultiSelectField()
	f.SetValue(config.StringsVal("auth", "cache"))

	if !f.selected[0] {
		t.Error("expected item 0 selected")
	}
	if f.selected[1] {
		t.Error("expected item 1 not selected")
	}
	if !f.selected[2] {
		t.Error("expected item 2 selected")
	}
}

func TestMultiSelectNumberKeyOutOfRange(t *testing.T) {
	f := newTestMultiSelectField()
	// Key '9' is out of range (only 3 choices).
	f.Update(key('9'))
	for i := range f.choices {
		if f.selected[i] {
			t.Errorf("expected no selection from out-of-range key, item %d selected", i)
		}
	}
}

func TestMultiSelectViewContainsLabels(t *testing.T) {
	f := newTestMultiSelectField()
	view := stripANSI(f.View())
	for _, label := range []string{"Authentication", "Logging", "Caching", "Features"} {
		if !strings.Contains(view, label) {
			t.Errorf("View missing %q", label)
		}
	}
}

func TestMultiSelectViewCheckboxes(t *testing.T) {
	f := newTestMultiSelectField()
	f.Update(key('1'))

	view := stripANSI(f.View())
	if !strings.Contains(view, "[x]") {
		t.Error("expected [x] for selected item")
	}
	if !strings.Contains(view, "[ ]") {
		t.Error("expected [ ] for unselected items")
	}
}

func TestMultiSelectNumberKeySetsAndTogglesCursor(t *testing.T) {
	f := newTestMultiSelectField()
	f.Update(key('2'))
	if f.cursor != 1 {
		t.Errorf("expected cursor=1 after pressing '2', got %d", f.cursor)
	}
	if !f.selected[1] {
		t.Error("expected item 1 toggled on")
	}

	// Press '2' again to toggle off.
	f.Update(key('2'))
	if f.selected[1] {
		t.Error("expected item 1 toggled off")
	}
}
