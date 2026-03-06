package wizard

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/svyatov/oz/internal/config"
)

const choicesFromTimeout = 10 * time.Second

// ResolveChoices runs a choices_from command and parses stdout into choices.
// Each non-empty line = one choice. Tab-separated columns: value[\tlabel[\tdescription]].
func ResolveChoices(choicesFrom string, answers Answers) ([]config.Choice, error) {
	cmd := interpolateCommand(choicesFrom, answers)

	ctx, cancel := context.WithTimeout(context.Background(), choicesFromTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).Output()
	if err != nil {
		return nil, fmt.Errorf("running choices_from command: %w", err)
	}

	return parseChoicesOutput(string(out)), nil
}

// interpolateCommand replaces {{name}} (no dot) with shell-escaped answer values.
func interpolateCommand(cmd string, answers Answers) string {
	return config.ChoicesFromInterpolationRe().ReplaceAllStringFunc(cmd, func(match string) string {
		name := match[2 : len(match)-2] // strip {{ and }}
		val, ok := answers[name]
		if !ok {
			return match
		}
		return shellEscape(fmt.Sprintf("%v", val))
	})
}

// shellEscape wraps a value in single quotes with proper escaping.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// parseChoicesOutput parses lines of output into choices.
func parseChoicesOutput(output string) []config.Choice {
	var choices []config.Choice
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		c := config.Choice{Value: parts[0], Label: parts[0]}
		if len(parts) >= 2 && parts[1] != "" {
			c.Label = parts[1]
		}
		if len(parts) >= 3 {
			c.Description = parts[2]
		}
		choices = append(choices, c)
	}
	return choices
}
