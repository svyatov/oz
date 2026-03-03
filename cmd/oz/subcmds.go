package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/command"
	"github.com/svyatov/oz/internal/compat"
	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/store"
	"github.com/svyatov/oz/internal/ui"
)

func doctorCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check tool installation and detected version",
		RunE: func(_ *cobra.Command, _ []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			fmt.Printf("  Wizard: %s\n", w.Name)
			fmt.Printf("  Command: %s\n", w.Command)

			if w.Detect == nil {
				fmt.Println("  Version detection: not configured")
				return nil
			}

			ver, err := compat.DetectVersion(w.Detect)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Version detection: %s %v\n",
					ui.MutedStyle.Render("failed —"), err)
				return nil
			}

			fmt.Printf("  Detected version: %s\n", ver)

			// Show which compat entry matches
			if len(w.Compat) > 0 {
				matchedRange := compat.MatchedRange(w.Compat, ver)
				if matchedRange != "" {
					filtered := compat.FilterOptions(w.Options, w.Compat, ver)
					fmt.Printf("  Compat match: %s (%d options)\n", matchedRange, len(filtered))
				} else {
					fmt.Println("  Compat match: none (all options shown)")
				}
			}

			return nil
		},
	}
}

func explainCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "explain",
		Short: "Show all options with descriptions",
		RunE: func(_ *cobra.Command, _ []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			detectedVersion, _ := compat.DetectVersion(w.Detect)
			options := compat.FilterOptions(w.Options, w.Compat, detectedVersion)

			fmt.Printf("\n  %s\n\n", ui.Header(w.Name, detectedVersion))

			for i, o := range options {
				fmt.Printf("  %s  %s\n", ui.StepCounter(i+1, len(options)),
					ui.TitleStyle.Render(o.Label))
				if o.Description != "" {
					fmt.Printf("         %s\n", ui.MutedStyle.Render(o.Description))
				}
				fmt.Printf("         Type: %s", o.Type)
				if o.Flag != "" {
					fmt.Printf("  Flag: %s", o.Flag)
				}
				if o.FlagTrue != "" {
					fmt.Printf("  FlagTrue: %s", o.FlagTrue)
				}
				if o.FlagFalse != "" {
					fmt.Printf("  FlagFalse: %s", o.FlagFalse)
				}
				fmt.Println()

				if o.Default != nil {
					fmt.Printf("         Default: %v\n", o.Default)
				}

				for _, c := range o.Choices {
					desc := ""
					if c.Description != "" {
						desc = "  " + ui.MutedStyle.Render(c.Description)
					}
					fmt.Printf("           - %s%s\n", c.Label, desc)
				}

				if len(o.ShowWhen) > 0 {
					conditions := make([]string, 0, len(o.ShowWhen))
					for k, v := range o.ShowWhen {
						conditions = append(conditions, fmt.Sprintf("%s=%v", k, v))
					}
					fmt.Printf("         Show when: %s\n", strings.Join(conditions, ", "))
				}

				fmt.Println()
			}
			return nil
		},
	}
}

func presetCmd(wizardName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preset",
		Short: "Manage presets",
	}

	cmd.AddCommand(presetListCmd(wizardName))
	cmd.AddCommand(presetShowCmd(wizardName))
	cmd.AddCommand(presetExplainCmd(wizardName))
	cmd.AddCommand(presetSaveCmd(wizardName))
	cmd.AddCommand(presetDeleteCmd(wizardName))

	return cmd
}

func presetListCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List presets",
		RunE: func(_ *cobra.Command, _ []string) error {
			st := store.New(configDir)
			names, err := st.ListPresets(wizardName)
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Println("  No presets found.")
				return nil
			}
			for _, n := range names {
				fmt.Printf("  %s\n", n)
			}
			return nil
		},
	}
}

func presetShowCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show preset values and generated command",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			st := store.New(configDir)
			values, err := st.LoadPreset(wizardName, args[0])
			if err != nil {
				return err
			}

			fmt.Printf("\n  Preset: %s\n\n", args[0])
			for k, v := range values {
				fmt.Printf("  %s: %v\n", k, v)
			}

			parts := command.Build(w, nil, values)
			command.PrintCommand(parts)
			return nil
		},
	}
}

func presetExplainCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "explain <name>",
		Short: "Annotated view with labels and descriptions",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			st := store.New(configDir)
			values, err := st.LoadPreset(wizardName, args[0])
			if err != nil {
				return err
			}

			// Build option lookup
			optMap := make(map[string]config.Option)
			for _, o := range w.Options {
				optMap[o.Name] = o
			}

			fmt.Printf("\n  Preset: %s\n\n", args[0])
			for k, v := range values {
				opt, known := optMap[k]
				if known {
					fmt.Printf("  %s: %v\n", ui.TitleStyle.Render(opt.Label), v)
					if opt.Description != "" {
						fmt.Printf("    %s\n", ui.MutedStyle.Render(opt.Description))
					}
					// Find matching choice description
					for _, c := range opt.Choices {
						if fmt.Sprintf("%v", v) == c.Value && c.Description != "" {
							fmt.Printf("    %s\n", ui.MutedStyle.Render(c.Description))
							break
						}
					}
				} else {
					fmt.Printf("  %s: %v\n", k, v)
				}
			}

			parts := command.Build(w, nil, values)
			command.PrintCommand(parts)
			return nil
		},
	}
}

func presetSaveCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "save <name>",
		Short: "Save last-used values as named preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			st := store.New(configDir)

			detectedVersion, _ := compat.DetectVersion(w.Detect)
			majorVersion := majorVer(detectedVersion)

			state, err := st.LoadState(w.Name, majorVersion)
			if err != nil {
				return fmt.Errorf("no state found: %w", err)
			}
			if len(state.LastUsed) == 0 {
				return fmt.Errorf("no last-used values to save — run the wizard first")
			}

			if err := st.SavePreset(wizardName, args[0], state.LastUsed); err != nil {
				return err
			}
			fmt.Printf("  Preset %q saved.\n", args[0])
			return nil
		},
	}
}

func presetDeleteCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st := store.New(configDir)
			if err := st.DeletePreset(wizardName, args[0]); err != nil {
				return err
			}
			fmt.Printf("  Preset %q deleted.\n", args[0])
			return nil
		},
	}
}
