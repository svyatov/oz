package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/registry"
	"github.com/svyatov/oz/internal/ui"
)

func updateCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "update <wizard>",
		Short: "Re-fetch a wizard from the registry",
		Long: `Download the latest version of a wizard config from the remote
registry and overwrite the local copy.

Use --all to update every locally-installed wizard that exists
in the remote registry.`,
		Example:           "  oz update rails-new\n  oz update --all",
		ValidArgsFunction: completeWizardNames,
		Args: func(_ *cobra.Command, args []string) error {
			if all {
				return nil
			}
			if len(args) == 0 {
				return errors.New("specify a wizard name or use --all")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if all {
				return updateAll()
			}
			return updateOne(args[0])
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "update all locally-installed wizards")

	return cmd
}

func updateOne(name string) error {
	client := registry.New(registry.DefaultBaseURL())

	data, err := client.FetchWizard(name)
	if err != nil {
		return fmt.Errorf("downloading wizard: %w", err)
	}

	return installWizard(data, true)
}

func updateAll() error {
	wizards, err := config.ListWizards(configDir)
	if err != nil {
		return fmt.Errorf("listing local wizards: %w", err)
	}

	if len(wizards) == 0 {
		ui.InfoMsgf("No local wizards to update")
		return nil
	}

	client := registry.New(registry.DefaultBaseURL())

	idx, err := client.FetchIndex()
	if err != nil {
		return fmt.Errorf("fetching registry index: %w", err)
	}

	remote := make(map[string]bool, len(idx.Wizards))
	for _, e := range idx.Wizards {
		remote[e.Name] = true
	}

	var updated int

	for _, w := range wizards {
		if !remote[w.Name] {
			continue
		}

		data, err := client.FetchWizard(w.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skipping %s: %v\n", w.Name, err)
			continue
		}

		if err := installWizard(data, true); err != nil {
			fmt.Fprintf(os.Stderr, "  skipping %s: %v\n", w.Name, err)
			continue
		}

		updated++
	}

	if updated == 0 {
		ui.InfoMsgf("No wizards found in registry to update")
	}

	return nil
}
