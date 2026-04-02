package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/registry"
	"github.com/svyatov/oz/internal/ui"
)

func addCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "add <wizard-name|file-path>",
		Aliases: []string{"a"},
		Short: "Add a wizard from the registry or a local file",
		Long: `Download a wizard config from the remote registry and install it,
or copy a local YAML file into the wizards directory.

Local files are auto-detected when the argument contains "/" or ends
with ".yml"/".yaml". Everything else is treated as a remote wizard name.

Override the registry URL with the OZ_REGISTRY_URL environment variable.`,
		Example: "  oz add rails-new\n  oz add ./my-wizard.yml\n  oz add rails-new --force",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if isLocalPath(args[0]) {
				return addLocal(args[0], force)
			}
			return addRemote(args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing wizard")

	return cmd
}

func isLocalPath(arg string) bool {
	return strings.Contains(arg, "/") ||
		strings.HasSuffix(arg, ".yml") ||
		strings.HasSuffix(arg, ".yaml")
}

func addRemote(name string, force bool) error {
	client := registry.New(registry.DefaultBaseURL())

	data, err := client.FetchWizard(name)
	if err != nil {
		return fmt.Errorf("downloading wizard: %w", err)
	}

	return installWizard(data, force, "Added")
}

func addLocal(path string, force bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	return installWizard(data, force, "Added")
}

func installWizard(data []byte, force bool, verb string) error {
	w, err := config.ParseWizard(data)
	if err != nil {
		return fmt.Errorf("parsing wizard: %w", err)
	}

	if errs := config.Validate(w); len(errs) > 0 {
		return fmt.Errorf("validation errors:\n%s", config.FormatErrors(errs))
	}

	safeName := filepath.Base(w.Name)
	dest := config.WizardPath(configDir, safeName)

	if !force {
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("wizard %q already exists (use --force to overwrite)", safeName)
		}
	}

	if err := os.MkdirAll(config.WizardsDir(configDir), 0o755); err != nil {
		return fmt.Errorf("creating wizards directory: %w", err)
	}

	if err := os.WriteFile(dest, data, 0o644); err != nil { //nolint:gosec // G703: path sanitized via filepath.Base above
		return fmt.Errorf("writing wizard config: %w", err)
	}

	ui.SuccessMsgf("%s wizard %q", verb, safeName)

	return nil
}
