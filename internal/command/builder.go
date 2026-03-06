package command

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// PartKind tags each segment of the built command.
type PartKind int

const (
	PartCommand PartKind = iota
	PartArg
	PartFlag
)

// Part is a tagged segment of the built command.
type Part struct {
	Text string
	Kind PartKind
}

// Build constructs the full CLI command from the wizard config and answers.
func Build(w *config.Wizard, answers map[string]any) []Part {
	var parts []Part
	for s := range strings.FieldsSeq(w.Command) {
		parts = append(parts, Part{s, PartCommand})
	}

	// Collect positional options (emitted between command and flags)
	var positionalParts []Part
	defaultStyle := w.EffectiveFlagStyle()

	for _, opt := range w.Options {
		val, ok := answers[opt.Name]
		if !ok {
			continue
		}

		if opt.Positional {
			s := fmt.Sprintf("%v", val)
			if s != "" && s != "_none" {
				positionalParts = append(positionalParts, Part{s, PartArg})
			}
			continue
		}

		flags := buildOptionFlags(opt, val, defaultStyle)
		for _, f := range flags {
			parts = append(parts, Part{f, PartFlag})
		}
	}

	// Insert positional args right after command words, before flags
	if len(positionalParts) > 0 {
		// Find where command words end and flags begin
		cmdEnd := 0
		for i, p := range parts {
			if p.Kind == PartCommand {
				cmdEnd = i + 1
			}
		}
		// Insert positional args after command
		result := make([]Part, 0, len(parts)+len(positionalParts))
		result = append(result, parts[:cmdEnd]...)
		result = append(result, positionalParts...)
		result = append(result, parts[cmdEnd:]...)
		parts = result
	}

	return parts
}

// FormatCommand returns the command as a plain display string.
func FormatCommand(parts []Part) string {
	strs := make([]string, len(parts))
	for i, p := range parts {
		strs[i] = p.Text
	}
	return strings.Join(strs, " ")
}

// PrintCommand prints the colored command with consistent spacing (blank line above and below).
func PrintCommand(parts []Part) {
	fmt.Printf("\n  %s\n\n", formatCommandColored(parts))
}

// formatCommandColored returns the command with color-coded segments.
func formatCommandColored(parts []Part) string {
	highlightStyle := lipgloss.NewStyle().Foreground(ui.Cyan).Bold(true)
	flagStyle := lipgloss.NewStyle().Foreground(ui.Accent)

	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteString(" ")
		}
		switch p.Kind {
		case PartCommand:
			b.WriteString(ui.TitleStyle.Render(p.Text))
		case PartArg:
			b.WriteString(highlightStyle.Render(p.Text))
		case PartFlag:
			if eqIdx := strings.Index(p.Text, "="); eqIdx >= 0 {
				flag := p.Text[:eqIdx+1]
				val := p.Text[eqIdx+1:]
				b.WriteString(flagStyle.Render(flag) + ui.CompletedStepAnswer(val))
			} else {
				b.WriteString(flagStyle.Render(p.Text))
			}
		}
	}
	return b.String()
}

// PlainParts returns just the text strings (for execution).
func PlainParts(parts []Part) []string {
	strs := make([]string, len(parts))
	for i, p := range parts {
		strs[i] = p.Text
	}
	return strs
}

func buildOptionFlags(opt config.Option, val any, defaultStyle string) []string {
	style := opt.EffectiveFlagStyle(defaultStyle)

	switch opt.Type {
	case "confirm":
		return buildConfirmFlags(opt, val)
	case "select":
		return buildSelectFlags(opt, val, style)
	case "input":
		return buildInputFlags(opt, val, style)
	case "multi_select":
		return buildMultiSelectFlags(opt, val, style)
	}
	return nil
}

func buildConfirmFlags(opt config.Option, val any) []string {
	b, ok := val.(bool)
	if !ok {
		return nil
	}

	flagTrue := opt.FlagTrue
	// Shorthand: if flag is set and flag_true is empty, use flag as flag_true
	if flagTrue == "" && opt.Flag != "" {
		flagTrue = opt.Flag
	}

	if b && flagTrue != "" {
		return []string{flagTrue}
	}
	if !b && opt.FlagFalse != "" {
		return []string{opt.FlagFalse}
	}
	return nil
}

func buildSelectFlags(opt config.Option, val any, style string) []string {
	s := fmt.Sprintf("%v", val)
	if s == "" || s == "_none" {
		if opt.FlagNone != "" {
			return []string{opt.FlagNone}
		}
		return nil
	}

	if opt.Flag == "" {
		return nil
	}
	return []string{formatFlag(opt.Flag, s, style)}
}

func buildInputFlags(opt config.Option, val any, style string) []string {
	s := fmt.Sprintf("%v", val)
	if s == "" || opt.Flag == "" {
		return nil
	}
	return []string{formatFlag(opt.Flag, s, style)}
}

func buildMultiSelectFlags(opt config.Option, val any, style string) []string {
	vals, ok := val.([]string)
	if !ok || len(vals) == 0 || opt.Flag == "" {
		return nil
	}

	// If separator is set, join values into a single flag
	if opt.Separator != "" {
		joined := strings.Join(vals, opt.Separator)
		return []string{formatFlag(opt.Flag, joined, style)}
	}

	var flags []string
	for _, v := range vals {
		flags = append(flags, formatFlag(opt.Flag, v, style))
	}
	return flags
}

func formatFlag(flag, value, style string) string {
	if style == "space" {
		return flag + " " + value
	}
	return flag + "=" + value
}
