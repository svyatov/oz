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
	root.AddCommand(removeCmd())
	root.AddCommand(createCmd())

	if name := detectWizardName(args); name != "" {
		run.AddCommand(wizardCmd(name))
	}

	return root
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "run",
		Aliases:           []string{"r"},
		Short:             "Run a wizard",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeWizardNames,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
}

// detectWizardName finds the wizard name argument after "run" in os.Args.
// During shell completion the last arg is the word being typed, so we skip it
// to avoid registering partial input (e.g. "cre") as a subcommand.
func detectWizardName(args []string) string {
	completion := false
	foundRun := false
	nameIdx := -1
	name := ""
	for i, a := range args {
		if a == "__complete" || a == "__completeNoDesc" {
			completion = true
			continue
		}
		if a == "" || strings.HasPrefix(a, "-") {
			continue
		}
		if !foundRun {
			if a == "run" || a == "r" {
				foundRun = true
			}
			continue
		}
		name = a
		nameIdx = i
		break
	}
	if completion && nameIdx == len(args)-1 {
		return ""
	}
	return name
}

func completeWizardNames(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
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
		Use:     "edit <wizard>",
		Aliases: []string{"e"},
		Short:   "Open wizard config in editor",
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
		Use:     "list",
		Aliases: []string{"l", "ls"},
		Short:   "List available wizards",
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

func removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <wizard>",
		Aliases: []string{"rm"},
		Short:   "Remove a wizard config",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := config.WizardPath(configDir, args[0])
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("wizard config not found: %s", path)
			}
			if !confirmPrompt(fmt.Sprintf("Remove %s?", path)) {
				return nil
			}
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing wizard: %w", err)
			}
			fmt.Printf("  Removed %s\n", args[0])
			return nil
		},
		ValidArgsFunction: completeWizardNames,
	}
}

func createCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "create <wizard>",
		Aliases: []string{"c", "new"},
		Short:   "Create a new wizard config from template",
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
