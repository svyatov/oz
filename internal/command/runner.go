package command

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// ExecCommand is the function that runs shell commands. Tests can override it.
var ExecCommand = func(parts []string) error {
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Run executes the command, connecting stdin/stdout/stderr to the terminal.
func Run(parts []string) error {
	if len(parts) == 0 {
		return errors.New("empty command")
	}
	if err := ExecCommand(parts); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}
