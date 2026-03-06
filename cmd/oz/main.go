package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

var version = "dev"

var configDir string

func main() {
	root := newRootCmd(os.Args[1:])

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd(args []string) *cobra.Command {
	root := &cobra.Command{
		Use:           "oz",
		Short:         "Config-driven CLI wizard framework",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}
	root.PersistentFlags().StringVar(&configDir, "config-dir", config.DefaultConfigDir(), "config directory")

	run := runCmd()
	root.AddCommand(run)
	root.AddCommand(listCmd())
	root.AddCommand(validateCmd())
	root.AddCommand(editCmd())
	root.AddCommand(deleteCmd())
	root.AddCommand(createCmd())

	if name := detectWizardName(args); name != "" {
		run.AddCommand(wizardCmd(name))
	}

	return root
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "run",
		Short:             "Run a wizard",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeWizardNames,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
}

// detectWizardName finds the wizard name argument after "run" in os.Args.
// Returns empty during shell completion to avoid registering partial input as a subcommand.
func detectWizardName(args []string) string {
	foundRun := false
	for _, a := range args {
		if a == "__complete" || a == "__completeNoDesc" {
			return ""
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		if !foundRun {
			if a == "run" {
				foundRun = true
			}
			continue
		}
		return a
	}
	return ""
}

func completeWizardNames(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	wizards, err := config.ListWizards(configDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(wizards))
	for _, w := range wizards {
		if w.Description != "" {
			names = append(names, fmt.Sprintf("%s\t%s", w.Name, w.Description))
		} else {
			names = append(names, w.Name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func editCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <wizard>",
		Short: "Open wizard config in editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := config.WizardPath(configDir, args[0])
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("wizard config not found: %s", path)
			}
			editor, err := findEditor()
			if err != nil {
				return fmt.Errorf("finding editor: %w", err)
			}
			return syscall.Exec(editor, []string{editor, path}, os.Environ())
		},
		ValidArgsFunction: completeWizardNames,
	}
}

func findEditor() (string, error) {
	name := "vi"
	if v := os.Getenv("VISUAL"); v != "" {
		name = v
	} else if v := os.Getenv("EDITOR"); v != "" {
		name = v
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("looking up %q: %w", name, err)
	}
	return path, nil
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available wizards",
		RunE: func(_ *cobra.Command, _ []string) error {
			wizards, err := config.ListWizards(configDir)
			if err != nil {
				return fmt.Errorf("listing wizards: %w", err)
			}
			if len(wizards) == 0 {
				fmt.Println("No wizards found in", config.WizardsDir(configDir))
				return nil
			}
			maxLen := 0
			for _, w := range wizards {
				if len(w.Name) > maxLen {
					maxLen = len(w.Name)
				}
			}

			fmt.Println()
			for _, w := range wizards {
				name := ui.AccentStyle.Render(fmt.Sprintf("  %-*s", maxLen, w.Name))
				desc := ""
				if w.Description != "" {
					desc = "  " + ui.MutedStyle.Render(w.Description)
				}
				fmt.Println(name + desc)
			}
			fmt.Println()
			return nil
		},
	}
}

func deleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <wizard>",
		Short: "Delete a wizard config",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := config.WizardPath(configDir, args[0])
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("wizard config not found: %s", path)
			}
			if !confirmPrompt(fmt.Sprintf("Delete %s?", path)) {
				return nil
			}
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("deleting wizard: %w", err)
			}
			fmt.Printf("  Deleted %s\n", args[0])
			return nil
		},
		ValidArgsFunction: completeWizardNames,
	}
}

func createCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <wizard>",
		Short: "Create a new wizard config from template",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := config.WizardPath(configDir, args[0])
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("wizard %q already exists: %s", args[0], path)
			}
			if err := os.MkdirAll(config.WizardsDir(configDir), 0o755); err != nil {
				return fmt.Errorf("creating wizards directory: %w", err)
			}
			if err := os.WriteFile(path, []byte(wizardTemplate(args[0])), 0o644); err != nil {
				return fmt.Errorf("writing wizard config: %w", err)
			}
			fmt.Printf("  Created %s\n", path)
			editor, err := findEditor()
			if err != nil {
				return fmt.Errorf("finding editor: %w", err)
			}
			return syscall.Exec(editor, []string{editor, path}, os.Environ())
		},
	}
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <path>",
		Short: "Validate a wizard config file",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			w, err := config.LoadWizard(args[0])
			if err != nil {
				return fmt.Errorf("loading wizard: %w", err)
			}
			errs := config.Validate(w)
			if len(errs) > 0 {
				return fmt.Errorf("validation errors:\n%s", config.FormatErrors(errs))
			}
			fmt.Println("Valid!")
			return nil
		},
	}
}
