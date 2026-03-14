// Package wizard implements the interactive Bubbletea-based wizard engine and field types.
package wizard

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// Key name constants for Bubbletea key press matching.
const (
	keyEnter    = "enter"
	keyTab      = "tab"
	keyEsc      = "esc"
	keyCtrlC    = "ctrl+c"
	keyUp       = "up"
	keyDown     = "down"
	keyLeft     = "left"
	keyRight    = "right"
	keyShiftTab = "shift+tab"
	keySpace    = "space"
	cursorBlank = "   " // inactive cursor padding
)

// versionPinCurrent is the sentinel value for "pin to detected version".
const versionPinCurrent = "current"

// Field is a single wizard step's input component.
type Field interface {
	Init() tea.Cmd
	Update(tea.KeyPressMsg) (submitted bool, cmd tea.Cmd)
	View() string
	Value() config.FieldValue
	SetValue(config.FieldValue)
}

// buildField creates the appropriate Field for an option type.
func buildField(opt *config.Option) Field {
	switch opt.Type {
	case config.OptionSelect:
		return NewSelectField(*opt)
	case config.OptionConfirm:
		return NewConfirmField(*opt)
	case config.OptionInput:
		return NewInputField(*opt)
	case config.OptionMultiSelect:
		return NewMultiSelectField(*opt)
	default:
		return NewInputField(*opt)
	}
}

// fieldHeader renders the common title + description block for all field types.
func fieldHeader(label, description string) string {
	var b strings.Builder
	b.WriteString("  " + ui.StepCounter(0, 0) + "  ")
	b.WriteString(ui.FieldTitle(label) + "\n")
	if description != "" {
		b.WriteString("         " + ui.FieldDesc(description) + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// choiceCursor renders the cursor indicator — active arrow or blank padding.
func choiceCursor(active bool) string {
	if active {
		return " " + ui.Cursor() + " "
	}
	return cursorBlank
}

// cursorUp moves the cursor up with wrap-around.
func cursorUp(cursor, n int) int { return (cursor - 1 + n) % n }

// cursorDown moves the cursor down with wrap-around.
func cursorDown(cursor, n int) int { return (cursor + 1) % n }

// numberKeyIndex converts a number key press (1-9) to a zero-based index.
// Returns -1 if the key is not a number or the index exceeds n.
func numberKeyIndex(code rune, n int) int {
	if code >= '1' && code <= '9' {
		if idx := int(code-'0') - 1; idx < n {
			return idx
		}
	}
	return -1
}
