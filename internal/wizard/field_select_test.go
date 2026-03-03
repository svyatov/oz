package wizard

import (
	"regexp"
	"strings"
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func newTestSelectField(choices []config.Choice) *SelectField {
	return NewSelectField(config.Option{
		Label:   "Language",
		Choices: choices,
		Type:    "select",
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
		defaultValue string
		wantDefault  bool
	}{
		{"shows_default_for_matching_choice", "python3", true},
		{"no_default_when_unset", "", false},
		{"no_default_for_nonexistent_value", "ruby", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestSelectField(testChoices())
			if tt.defaultValue != "" {
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

func TestSelectFieldDefaultTagAlignment(t *testing.T) {
	f := newTestSelectField(testChoices())
	f.SetDefault("python3")

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
