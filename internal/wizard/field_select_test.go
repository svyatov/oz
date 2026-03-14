package wizard

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
)

func newTestSelectField(choices []config.Choice) *SelectField {
	return NewSelectField(config.Option{
		Label:   "Language",
		Choices: choices,
		Type:    config.OptionSelect,
	})
}

func testChoices() []config.Choice {
	return []config.Choice{
		{Label: "Python 3", Value: "python3", Description: "Latest Python"},
		{Label: "Node.js", Value: "nodejs", Description: "JavaScript runtime"},
		{Label: "Go", Value: "go", Description: "Fast compiled"},
	}
}

var ansiRE = regexp.MustCompile(`\x1b(?:\[[0-9;]*[a-zA-Z]|\([A-Za-z])`)

// stripANSI removes ANSI escape sequences for text assertions.
func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func TestSelectFieldDefaultTag(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue config.FieldValue
		wantDefault  bool
	}{
		{"shows_default_for_matching_choice", config.StringVal("python3"), true},
		{"no_default_when_unset", config.FieldValue{}, false},
		{"no_default_for_nonexistent_value", config.StringVal("ruby"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestSelectField(testChoices())
			if !reflect.DeepEqual(tt.defaultValue, config.FieldValue{}) {
				f.SetDefault(tt.defaultValue)
			}

			view := stripANSI(f.View())
			hasDefault := strings.Contains(view, "(default)")
			if hasDefault != tt.wantDefault {
				t.Errorf("View() contains (default) = %v, want %v\nview:\n%s", hasDefault, tt.wantDefault, view)
			}
		})
	}
}

func TestSelectFieldSubmitTab(t *testing.T) {
	f := newTestSelectField(testChoices())
	submitted, _ := f.Update(specialKey(tea.KeyTab))
	if !submitted {
		t.Fatal("expected submitted via tab")
	}
	if f.value != "python3" {
		t.Errorf("expected value python3, got %s", f.value)
	}
}

func TestSelectFieldNumberKeyZero(t *testing.T) {
	// Key '0' is not in 1-9 range, numberKeyIndex should return -1.
	idx := numberKeyIndex('0', 3)
	if idx != -1 {
		t.Errorf("expected -1 for key '0', got %d", idx)
	}
}

func TestSelectFieldNumberKeyBeyondRange(t *testing.T) {
	f := newTestSelectField(testChoices()) // 3 choices
	submitted, _ := f.Update(key('5'))
	if submitted {
		t.Error("should not submit with out-of-range number key")
	}
	// Cursor should remain at 0.
	if f.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", f.cursor)
	}
}

func TestSelectFieldAllowNoneMaxDisplayWidth(t *testing.T) {
	choices := []config.Choice{
		{Value: "ab", Label: "AB"},
	}
	f := NewSelectField(config.Option{
		Label:     "Test",
		Choices:   choices,
		Type:      config.OptionSelect,
		AllowNone: true,
	})
	// "None" is 4 chars, "AB" is 2 chars — maxDisplayWidth should be at least 4.
	w := f.maxDisplayWidth()
	if w < 4 {
		t.Errorf("expected maxDisplayWidth >= 4 with allowNone, got %d", w)
	}
}

func TestSelectFieldAllowNoneDefaultMaxDisplayWidth(t *testing.T) {
	choices := []config.Choice{
		{Value: "ab", Label: "AB"},
	}
	f := NewSelectField(config.Option{
		Label:     "Test",
		Choices:   choices,
		Type:      config.OptionSelect,
		AllowNone: true,
	})
	// Set default to NoneValue — should add " (default)" to "None" width.
	f.SetDefault(config.StringVal(config.NoneValue))
	w := f.maxDisplayWidth()
	expectedMin := len("None") + len(defaultSuffix)
	if w < expectedMin {
		t.Errorf("expected maxDisplayWidth >= %d with allowNone default, got %d", expectedMin, w)
	}
}

func TestBuildFieldUnknownType(t *testing.T) {
	opt := &config.Option{
		Name:  "test",
		Type:  config.OptionType("unknown"),
		Label: "Test",
	}
	f := buildField(opt)
	if _, ok := f.(*InputField); !ok {
		t.Errorf("expected InputField for unknown type, got %T", f)
	}
}

func TestBuildFieldMultiSelect(t *testing.T) {
	opt := &config.Option{
		Name:  "features",
		Type:  config.OptionMultiSelect,
		Label: "Features",
		Choices: []config.Choice{
			{Value: "a", Label: "A"},
		},
	}
	f := buildField(opt)
	if _, ok := f.(*MultiSelectField); !ok {
		t.Errorf("expected MultiSelectField, got %T", f)
	}
}

func TestSelectFieldDefaultTagAlignment(t *testing.T) {
	f := newTestSelectField(testChoices())
	f.SetDefault(config.StringVal("python3"))

	view := stripANSI(f.View())
	lines := strings.Split(view, "\n")

	// Collect description start columns (rune-based, not byte-based)
	// across choice lines. The cursor › is multi-byte UTF-8, so we
	// must count runes for correct visual column comparison.
	descs := []string{"Latest Python", "JavaScript runtime", "Fast compiled"}
	var descColumns []int
	for _, line := range lines {
		for _, d := range descs {
			if before, _, ok := strings.Cut(line, d); ok {
				descColumns = append(descColumns, len([]rune(before)))
			}
		}
	}

	if len(descColumns) < 2 {
		t.Fatalf("expected at least 2 description positions, got %d\nview:\n%s", len(descColumns), view)
	}

	first := descColumns[0]
	for i, col := range descColumns[1:] {
		if col != first {
			t.Errorf("description column %d at rune position %d, want %d\nview:\n%s", i+1, col, first, view)
		}
	}
}
