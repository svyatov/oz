// Package command builds and executes shell commands from wizard answers.
package command

import (
	"fmt"
	"strings"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// PartKind tags each segment of the built command.
type PartKind int

const (
	PartCommand PartKind = iota
	PartArg
	PartFlag
	PartExtra // passthrough args from --.
)

// Part is a tagged segment of the built command.
type Part struct {
	Text   string
	Kind   PartKind
	Secret bool // password flag: mask the value in human-facing renderers only.
}

// Build constructs the full CLI command from the wizard config and answers.
func Build(w *config.Wizard, answers config.Values) []Part {
	var parts []Part
	for s := range strings.FieldsSeq(w.Command) {
		parts = append(parts, Part{Text: s, Kind: PartCommand})
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
			s := val.Scalar()
			if s != "" && s != config.NoneValue {
				positionalParts = append(positionalParts, Part{Text: s, Kind: PartArg})
			}
			continue
		}

		secret := opt.Type == config.OptionPassword
		flags := buildOptionFlags(opt, val, defaultStyle)
		for _, f := range flags {
			parts = append(parts, Part{Text: f, Kind: PartFlag, Secret: secret})
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
	return strings.Join(PlainParts(parts), " ")
}

// PrintCommand prints the colored command with consistent spacing (blank line above and below).
func PrintCommand(parts []Part) {
	fmt.Printf("\n  %s\n\n", formatCommandColored(parts))
}

// formatCommandColored returns the command with color-coded segments.
func formatCommandColored(parts []Part) string {
	highlightStyle := ui.CyanBoldStyle
	flagStyle := ui.AccentStyle

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
			flag, val, hasVal := splitFlag(p.Text, p.Secret)
			if !hasVal {
				b.WriteString(flagStyle.Render(p.Text))
			} else {
				if p.Secret {
					val = config.SecretMask
				}
				b.WriteString(flagStyle.Render(flag) + ui.CompletedStepAnswer(val))
			}
		case PartExtra:
			b.WriteString(highlightStyle.Render(p.Text))
		}
	}
	return b.String()
}

// AppendExtra appends passthrough args (from --) as PartExtra parts.
func AppendExtra(parts []Part, extra []string) []Part {
	for _, e := range extra {
		parts = append(parts, Part{Text: e, Kind: PartExtra})
	}
	return parts
}

// PlainParts returns just the text strings (for execution).
func PlainParts(parts []Part) []string {
	strs := make([]string, len(parts))
	for i, p := range parts {
		strs[i] = p.Text
	}
	return strs
}

func buildOptionFlags(opt config.Option, val config.FieldValue, defaultStyle config.FlagStyle) []string {
	style := opt.EffectiveFlagStyle(defaultStyle)

	switch opt.Type {
	case config.OptionConfirm:
		return buildConfirmFlags(opt, val)
	case config.OptionSelect:
		return buildSelectFlags(opt, val, style)
	case config.OptionInput, config.OptionNumber:
		return buildInputFlags(opt, val, style)
	case config.OptionMultiSelect:
		return buildMultiSelectFlags(opt, val, style)
	case config.OptionPassword:
		// An env-delivered secret emits no argv token; env wins over flag.
		if opt.SecretEnv != "" {
			return nil
		}
		return buildInputFlags(opt, val, style)
	}
	return nil
}

func buildConfirmFlags(opt config.Option, val config.FieldValue) []string {
	b := val.Bool()

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

func buildSelectFlags(opt config.Option, val config.FieldValue, style config.FlagStyle) []string {
	s := val.Scalar()
	if s == "" || s == config.NoneValue {
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

func buildInputFlags(opt config.Option, val config.FieldValue, style config.FlagStyle) []string {
	s := val.Scalar()
	if s == "" || opt.Flag == "" {
		return nil
	}
	return []string{formatFlag(opt.Flag, s, style)}
}

func buildMultiSelectFlags(opt config.Option, val config.FieldValue, style config.FlagStyle) []string {
	vals := val.Strings()
	if len(vals) == 0 || opt.Flag == "" {
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

// splitFlag separates a flag part into its flag and value for colored rendering.
// For secrets it splits on the earliest delimiter (space before "="), so a
// space-style secret whose value contains "=" (e.g. a base64 token) masks the
// whole value rather than leaking the pre-"=" prefix. Non-secret flags split on
// "=" only, preserving how space-style flags render.
func splitFlag(text string, secret bool) (flag, val string, hasVal bool) {
	eq := strings.Index(text, "=")
	if secret {
		if sp := strings.Index(text, " "); sp >= 0 && (eq < 0 || sp < eq) {
			return text[:sp+1], text[sp+1:], true
		}
	}
	if eq >= 0 {
		return text[:eq+1], text[eq+1:], true
	}
	return text, "", false
}

func formatFlag(flag, value string, style config.FlagStyle) string {
	if style == config.FlagStyleSpace {
		return flag + " " + value
	}
	return flag + "=" + value
}
