package wizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
)

func newTestConfirmField() *ConfirmField {
	return NewConfirmField(config.Option{
		Label:       "Enable API?",
		Description: "Run in API mode",
		Type:        config.OptionConfirm,
	})
}

func TestConfirmFieldSubmitYes(t *testing.T) {
	f := newTestConfirmField()
	// Cursor starts at 0 (Yes).
	submitted, _ := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !submitted {
		t.Fatal("expected submitted")
	}
	if !f.Value().Bool() {
		t.Error("expected true")
	}
}

func TestConfirmFieldSubmitNo(t *testing.T) {
	f := newTestConfirmField()
	f.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	submitted, _ := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !submitted {
		t.Fatal("expected submitted")
	}
	if f.Value().Bool() {
		t.Error("expected false")
	}
}

func TestConfirmFieldNavigation(t *testing.T) {
	f := newTestConfirmField()
	if f.cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", f.cursor)
	}

	f.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if f.cursor != 1 {
		t.Errorf("expected cursor=1 after down, got %d", f.cursor)
	}

	f.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if f.cursor != 0 {
		t.Errorf("expected cursor=0 after up, got %d", f.cursor)
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

func TestConfirmFieldShortcutKeys(t *testing.T) {
	tests := []struct {
		name string
		code rune
		want bool
	}{
		{"y_submits_true", 'y', true},
		{"n_submits_false", 'n', false},
		{"1_submits_true", '1', true},
		{"2_submits_false", '2', false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestConfirmField()
			submitted, _ := f.Update(key(tt.code))
			if !submitted {
				t.Fatal("expected submitted")
			}
			if f.Value().Bool() != tt.want {
				t.Errorf("got %v, want %v", f.Value().Bool(), tt.want)
			}
		})
	}
}

func TestConfirmFieldSetValue(t *testing.T) {
	f := newTestConfirmField()
	f.SetValue(config.BoolVal(false))
	if f.cursor != 1 {
		t.Errorf("expected cursor=1 for false, got %d", f.cursor)
	}
	if f.Value().Bool() {
		t.Error("expected false")
	}

	f.SetValue(config.BoolVal(true))
	if f.cursor != 0 {
		t.Errorf("expected cursor=0 for true, got %d", f.cursor)
	}
	if !f.Value().Bool() {
		t.Error("expected true")
	}
}

func TestConfirmFieldDefaultTag(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue bool
		wantYesTag   bool
		wantNoTag    bool
	}{
		{"default_true_tags_yes", true, true, false},
		{"default_false_tags_no", false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTestConfirmField()
			f.SetDefault(config.BoolVal(tt.defaultValue))
			view := stripANSI(f.View())
			lines := strings.Split(view, "\n")

			var yesLine, noLine string
			for _, line := range lines {
				if strings.Contains(line, "Yes") {
					yesLine = line
				}
				if strings.Contains(line, "No") {
					noLine = line
				}
			}

			if hasTag := strings.Contains(yesLine, "(default)"); hasTag != tt.wantYesTag {
				t.Errorf("Yes line has (default) = %v, want %v", hasTag, tt.wantYesTag)
			}
			if hasTag := strings.Contains(noLine, "(default)"); hasTag != tt.wantNoTag {
				t.Errorf("No line has (default) = %v, want %v", hasTag, tt.wantNoTag)
			}
		})
	}
}

func TestConfirmFieldNoDefaultTag(t *testing.T) {
	f := newTestConfirmField()
	view := stripANSI(f.View())
	if strings.Contains(view, "(default)") {
		t.Error("expected no (default) tag when SetDefault not called")
	}
}

func TestConfirmFieldViewContainsYesNo(t *testing.T) {
	f := newTestConfirmField()
	view := stripANSI(f.View())
	if !strings.Contains(view, "Yes") {
		t.Error("View missing Yes")
	}
	if !strings.Contains(view, "No") {
		t.Error("View missing No")
	}
	if !strings.Contains(view, "Enable API?") {
		t.Error("View missing label")
	}
}

func TestConfirmFieldUnhandledKey(t *testing.T) {
	f := newTestConfirmField()
	submitted, _ := f.Update(key('z'))
	if submitted {
		t.Error("unexpected submission on unhandled key")
	}
}
