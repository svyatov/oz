package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/generate"
	"github.com/svyatov/oz/internal/ui"
)

func generateCmd() *cobra.Command {
	var (
		name    string
		output  string
		install bool
		stdin   bool
	)

	cmd := &cobra.Command{
		Use:     "generate <tool> [subcommand...]",
		Aliases: []string{"g", "gen"},
		Short:   "Generate wizard config from --help output",
		Long: `Run <tool> [subcommand...] --help, parse the output, and scaffold
a wizard YAML config. Writes to stdout by default.

Use --install to save directly to the config directory and open in your editor.`,
		Example: `  oz generate docker run
  oz generate kubectl apply
  oz generate ffmpeg
  oz generate docker run --install
  oz generate docker run -o docker-run.yml
  echo "..." | oz generate --stdin --name my-tool`,
		Args: func(_ *cobra.Command, args []string) error {
			if stdin {
				if len(args) > 0 {
					return errors.New("--stdin does not accept arguments")
				}
				if name == "" {
					return errors.New("--stdin requires --name")
				}
				return nil
			}
			if len(args) == 0 {
				return errors.New("specify a tool name")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return runGenerate(args, name, output, install, stdin)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "override wizard name")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write to file instead of stdout")
	cmd.Flags().BoolVarP(&install, "install", "i", false, "install to config dir and open in editor")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "read help text from stdin")

	return cmd
}

func runGenerate(args []string, name, output string, install, stdin bool) error {
	helpText, err := acquireHelpText(args, stdin)
	if err != nil {
		return err
	}

	flags := generate.Parse(helpText)
	if len(flags) == 0 {
		return errors.New("no flags found in help output")
	}

	wizardName := name
	if wizardName == "" {
		wizardName = strings.Join(args, "-")
	}
	command := strings.Join(args, " ")

	yamlStr := generate.Emit(generate.EmitConfig{
		Name:    wizardName,
		Command: command,
	}, flags)

	if install {
		return installGenerated(wizardName, yamlStr)
	}

	if output != "" {
		if err := os.WriteFile(output, []byte(yamlStr), 0o644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		ui.SuccessMsgf("Written to %s", output)
		return nil
	}

	fmt.Print(yamlStr)
	return nil
}

func acquireHelpText(args []string, stdin bool) (string, error) {
	if stdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	text := runHelp(args[0], args[1:], "--help")
	if strings.TrimSpace(text) != "" {
		return text, nil
	}

	// Fallback to -h if --help produced nothing.
	return runHelp(args[0], args[1:], "-h"), nil
}

func runHelp(tool string, subcmds []string, helpFlag string) string {
	cmdArgs := make([]string, 0, len(subcmds)+1)
	cmdArgs = append(cmdArgs, subcmds...)
	cmdArgs = append(cmdArgs, helpFlag)

	cmd := exec.Command(tool, cmdArgs...)
	cmd.Env = append(os.Environ(), "PAGER=cat", "GIT_PAGER=cat", "MANPAGER=cat")

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	// Ignore exit code — many tools exit non-zero for --help.
	_ = cmd.Run()

	return buf.String()
}

func installGenerated(name, yamlStr string) error {
	path := config.WizardPath(configDir, name)

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("wizard %q already exists: %s", name, path)
	}

	if err := os.MkdirAll(config.WizardsDir(configDir), 0o755); err != nil {
		return fmt.Errorf("creating wizards directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(yamlStr), 0o644); err != nil {
		return fmt.Errorf("writing wizard config: %w", err)
	}

	ui.SuccessMsgf("Created %s", path)

	editor, err := findEditor()
	if err != nil {
		return fmt.Errorf("finding editor: %w", err)
	}

	if err := syscall.Exec(editor, []string{editor, path}, os.Environ()); err != nil {
		return fmt.Errorf("opening editor: %w", err)
	}

	return nil
}
