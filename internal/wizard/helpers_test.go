package wizard

import (
	"regexp"

	tea "charm.land/bubbletea/v2"
)

var ansiRE = regexp.MustCompile(`\x1b(?:\[[0-9;]*[a-zA-Z]|\([A-Za-z])`)

// stripANSI removes ANSI escape sequences for text assertions.
func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func key(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: string(code)}
}

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}
