package command

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Run executes the command, connecting stdin/stdout/stderr to the terminal.
func Run(parts []string) error {
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}

// DryRun prints the command without executing it.
func DryRun(parts []string) {
	fmt.Println(strings.Join(parts, " "))
}
