package command

import (
	"fmt"
	"sort"
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

// Build constructs the full CLI command from the wizard config, positional args, and answers.
func Build(w *config.Wizard, positionalArgs map[string]string, answers map[string]any) []Part {
	var parts []Part
	for _, s := range strings.Fields(w.Command) {
		parts = append(parts, Part{s, PartCommand})
	}

	// Add positional args sorted by position
	type posArg struct {
		position int
		value    string
	}
	var posArgs []posArg
	for _, a := range w.Args {
		if v, ok := positionalArgs[a.Name]; ok && v != "" {
			posArgs = append(posArgs, posArg{a.Position, v})
		}
	}
	sort.Slice(posArgs, func(i, j int) bool { return posArgs[i].position < posArgs[j].position })
	for _, pa := range posArgs {
		parts = append(parts, Part{pa.value, PartArg})
	}

	// Add flags from options
	defaultStyle := w.EffectiveFlagStyle()
	for _, opt := range w.Options {
		val, ok := answers[opt.Name]
		if !ok {
			continue
		}
		flags := buildOptionFlags(opt, val, defaultStyle)
		for _, f := range flags {
			parts = append(parts, Part{f, PartFlag})
		}
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
	if b && opt.FlagTrue != "" {
		return []string{opt.FlagTrue}
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
