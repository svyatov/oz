package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/generate"
	"github.com/svyatov/oz/internal/ui"
)

func generateCmd() *cobra.Command {
	var (
		name   string
		output string
		force  bool
	)

	cmd := &cobra.Command{
		Use:     "generate <tool> [subcommand...]",
		Aliases: []string{"g", "gen"},
		Short:   "Generate wizard config from --help output",
		Long: `Run <tool> [subcommand...] --help, parse the output, and scaffold
a wizard YAML config. Saves to the config directory by default.

Use -o to write to a specific file instead. Use -n to override the wizard name.`,
		Example: `  oz generate docker run
  oz generate kubectl apply
  oz generate ffmpeg
  oz generate docker run --name docker-run
  oz generate docker run -o custom-path.yml
  oz generate docker run --force`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("specify a tool name")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return runGenerate(args, name, output, force)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "override wizard name")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write to file instead of config dir")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing file")

	return cmd
}

func runGenerate(args []string, name, output string, force bool) error {
	helpText := acquireHelpText(args)

	flags := generate.Parse(helpText)
	if len(flags) == 0 {
		return errors.New("no flags found in help output")
	}

	wizardName := name
	if wizardName == "" {
		wizardName = strings.Join(args, "-")
	}

	yamlStr := generate.Emit(generate.EmitConfig{
		Name:    wizardName,
		Command: strings.Join(args, " "),
	}, flags)

	dest := output
	if dest == "" {
		dest = config.WizardPath(configDir, wizardName)
	}

	if !force {
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", dest)
		}
	}

	if output == "" {
		if err := os.MkdirAll(config.WizardsDir(configDir), 0o755); err != nil {
			return fmt.Errorf("creating wizards directory: %w", err)
		}
	}

	if err := os.WriteFile(dest, []byte(yamlStr), 0o644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	ui.SuccessMsgf("Created %s", dest)

	return nil
}

func acquireHelpText(args []string) string {
	text := runHelp(args[0], args[1:], "--help")
	if strings.TrimSpace(text) != "" {
		return text
	}

	// Fallback to -h if --help produced nothing.
	return runHelp(args[0], args[1:], "-h")
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
