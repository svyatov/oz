package ui

import (
	"fmt"
	"os"
	"strconv"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

var lightDark = lipgloss.LightDark(detectDarkBackground())

func detectDarkBackground() bool {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) || !term.IsTerminal(os.Stderr.Fd()) {
		return true
	}
	return lipgloss.HasDarkBackground(os.Stdin, os.Stderr)
}

// Rich color palette — distinct hues, not just gray shades.
var (
	Green   = lightDark(lipgloss.Color("#1E8A3C"), lipgloss.Color("#5AF78E")) // active selection, answers, cursor
	Accent  = lightDark(lipgloss.Color("#5A56E0"), lipgloss.Color("#9D8CFF")) // titles, step counter
	Yellow  = lightDark(lipgloss.Color("#996B22"), lipgloss.Color("#E5C07B")) // field descriptions
	Cyan    = lightDark(lipgloss.Color("#0E7490"), lipgloss.Color("#56D6DB")) // command args
	Muted   = lightDark(lipgloss.Color("#6B7280"), lipgloss.Color("#8B95A5")) // completed labels, choice descriptions
	Dimmed  = lightDark(lipgloss.Color("#9CA3AF"), lipgloss.Color("#4B5563")) // nav hints, inactive numbers
	Normal  = lightDark(lipgloss.Color("#1F2937"), lipgloss.Color("#E5E9F0")) // choice labels, default text
	Warning = lightDark(lipgloss.Color("#B8860B"), lipgloss.Color("#DAA520"))
)

var (
	TitleStyle  = lipgloss.NewStyle().Bold(true)
	MutedStyle  = lipgloss.NewStyle().Foreground(Muted)
	AccentStyle = lipgloss.NewStyle().Foreground(Accent)
)

// CompletedStepLine renders a completed step: `  01  ✓ Label  Answer`.
func CompletedStepLine(stepNum int, label, answer string) string {
	num := lipgloss.NewStyle().Foreground(Accent).Render(fmt.Sprintf("%02d", stepNum))
	check := lipgloss.NewStyle().Foreground(Green).Render("\u2713")
	lbl := MutedStyle.Render(label)
	ans := lipgloss.NewStyle().Foreground(Green).Render(answer)
	return fmt.Sprintf("  %s   %s %s  %s", num, check, lbl, ans)
}

// CompletedStepAnswer renders just an answer value in green.
func CompletedStepAnswer(answer string) string {
	return lipgloss.NewStyle().Foreground(Green).Render(answer)
}

// FieldTitle renders the current field's title in accent bold.
func FieldTitle(title string) string {
	return lipgloss.NewStyle().Foreground(Accent).Bold(true).Render(title)
}

// FieldDesc renders the field description in warm yellow.
func FieldDesc(desc string) string {
	return lipgloss.NewStyle().Foreground(Yellow).Render(desc)
}

// ChoiceDesc renders a choice's description in muted.
func ChoiceDesc(desc string) string {
	return MutedStyle.Render(desc)
}

// NumberGutter renders a choice number — green when active, dimmed when not.
func NumberGutter(n int, active bool) string {
	s := strconv.Itoa(n)
	if active {
		return lipgloss.NewStyle().Foreground(Green).Bold(true).Render(s)
	}
	return lipgloss.NewStyle().Foreground(Dimmed).Render(s)
}

// Cursor renders the cursor indicator in green.
func Cursor() string {
	return lipgloss.NewStyle().Foreground(Green).Bold(true).Render("\u203a")
}

// ChoiceLabel renders a choice label — green bold when active, normal when not.
func ChoiceLabel(label string, active bool) string {
	if active {
		return lipgloss.NewStyle().Foreground(Green).Bold(true).Render(label)
	}
	return lipgloss.NewStyle().Foreground(Normal).Render(label)
}

// NavHint renders the navigation hint line (dimmest text).
func NavHint() string {
	return lipgloss.NewStyle().Foreground(Dimmed).Render("  shift+tab back \u00b7 tab/enter next \u00b7 esc quit")
}

// StepCounter returns a formatted step counter like "01/05" in accent color.
func StepCounter(current, total int) string {
	cur := lipgloss.NewStyle().Foreground(Accent).Bold(true).Render(fmt.Sprintf("%02d", current))
	sep := lipgloss.NewStyle().Foreground(Dimmed).Render("/")
	tot := lipgloss.NewStyle().Foreground(Dimmed).Render(fmt.Sprintf("%02d", total))
	return cur + sep + tot
}

// Header renders the wizard header with name and optional version.
func Header(name, version string) string {
	s := TitleStyle.Render(name)
	if version != "" {
		s += MutedStyle.Render(fmt.Sprintf(" \u2014 %s detected", version))
	}
	return s
}

// PinnedInfo renders the pinned options count.
func PinnedInfo(count int) string {
	if count == 0 {
		return ""
	}
	return MutedStyle.Render(fmt.Sprintf("(%d pinned options hidden)", count))
}

// DefaultTag renders a dimmed "(default)" suffix for select choices.
func DefaultTag() string {
	return lipgloss.NewStyle().Foreground(Dimmed).Render("(default)")
}

// PinIcon renders the pin indicator "●" in green.
func PinIcon() string {
	return lipgloss.NewStyle().Foreground(Green).Render("●")
}

// PinEditIndicator renders the "pin" label for edit mode headers.
func PinEditIndicator() string {
	return lipgloss.NewStyle().Foreground(Accent).Bold(true).Render("pin")
}

// PinsListNavHint renders the nav hint for pin list mode.
func PinsListNavHint() string {
	return lipgloss.NewStyle().Foreground(Dimmed).Render("  enter edit \u00b7 space toggle pin \u00b7 esc done")
}

// PinsEditNavHint renders the nav hint for pin edit mode.
func PinsEditNavHint() string {
	return lipgloss.NewStyle().Foreground(Dimmed).Render("  enter confirm \u00b7 esc cancel")
}
