package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/config"
)

var version = "dev"

var configDir string

func main() {
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

	if name := detectWizardName(os.Args[1:]); name != "" {
		run.AddCommand(wizardCmd(name))
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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
func detectWizardName(args []string) string {
	foundRun := false
	for _, a := range args {
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
			names = append(names, fmt.Sprintf("%s\t★ %s", w.Name, w.Description))
		} else {
			names = append(names, w.Name+"\t★")
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
			for _, w := range wizards {
				desc := ""
				if w.Description != "" {
					desc = "  " + w.Description
				}
				fmt.Printf("  %s%s\n", w.Name, desc)
			}
			return nil
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
