package main

import (
	"fmt"
	"os"
	"strings"

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
	root.AddCommand(listCmd())
	root.AddCommand(validateCmd())

	// Before cobra parses, check if the first positional arg is a wizard name.
	// If so, build a dynamic subcommand tree for it.
	args := os.Args[1:]
	if wizardName, ok := detectWizardArg(args); ok {
		root.AddCommand(wizardCmd(wizardName))
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// detectWizardArg returns the wizard name if the first positional arg isn't a builtin.
func detectWizardArg(args []string) (string, bool) {
	builtins := map[string]bool{
		"list": true, "validate": true, "help": true, "completion": true,
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if builtins[a] {
			return "", false
		}
		return a, true
	}
	return "", false
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
