package command

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// RunWithEnv executes the command with extra env vars (NAME=value) appended to
// the inherited environment. Used to deliver secrets out-of-band, off argv.
func RunWithEnv(parts, env []string) error {
	if len(parts) == 0 {
		return errors.New("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}
