package main

import (
	"errors"
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

			if w.Version == nil {
				fmt.Println("  Version detection: not configured")
				return nil
			}

			ver, err := compat.DetectVersion(w.Version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Version detection: %s %v\n",
					ui.MutedStyle.Render("failed —"), err)
				return nil
			}

			fmt.Printf("  Detected version: %s\n", ver)

			if w.Version.CustomVersionCmd != "" {
				fmt.Printf("  Version template: %s\n", w.Version.CustomVersionCmd)
				fmt.Printf("  Effective command: %s\n", w.EffectiveCommand(ver))
			}
			if w.Version.CustomVersionVerify != "" {
				fmt.Printf("  Verify template: %s\n", w.Version.CustomVersionVerify)
			}
			if w.Version.AvailVersionsCmd != "" {
				fmt.Printf("  Available versions: command — %s\n", w.Version.AvailVersionsCmd)
			} else if w.Version.AvailVersions != "" {
				fmt.Printf("  Available versions: static — %s\n", w.Version.AvailVersions)
			}

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

			detectedVersion, _ := compat.DetectVersion(w.Version)
			options := compat.FilterOptions(w.Options, w.Compat, detectedVersion)

			fmt.Printf("\n  %s\n", ui.Header(w.Name, detectedVersion, versionLabel(w)))
			effectiveCmd := w.EffectiveCommand(detectedVersion)
			if effectiveCmd != w.Command {
				fmt.Printf("  %s\n", ui.MutedStyle.Render("command: "+effectiveCmd))
			}
			fmt.Println()
			for i, o := range options {
				printOptionExplanation(i+1, len(options), o)
			}
			return nil
		},
	}
}

func printOptionExplanation(step, total int, o config.Option) {
	fmt.Printf("  %s  %s\n", ui.StepCounter(step, total),
		ui.TitleStyle.Render(o.Label))
	if o.Description != "" {
		fmt.Printf("         %s\n", ui.MutedStyle.Render(o.Description))
	}

	printOptionFlags(o)
	printOptionDetails(o)

	for _, c := range o.Choices {
		desc := ""
		if c.Description != "" {
			desc = "  " + ui.MutedStyle.Render(c.Description)
		}
		fmt.Printf("           - %s%s\n", c.Label, desc)
	}

	printConditions("Show when", o.ShowWhen)
	printConditions("Hide when", o.HideWhen)
	fmt.Println()
}

func printOptionFlags(o config.Option) {
	fmt.Printf("         Type: %s", o.Type)
	if o.Positional {
		fmt.Print("  (positional)")
	}
	if o.Flag != "" {
		fmt.Printf("  Flag: %s", o.Flag)
	}
	if o.FlagTrue != "" {
		fmt.Printf("  FlagTrue: %s", o.FlagTrue)
	}
	if o.FlagFalse != "" {
		fmt.Printf("  FlagFalse: %s", o.FlagFalse)
	}
	if o.Separator != "" {
		fmt.Printf("  Separator: %q", o.Separator)
	}
	fmt.Println()
}

func printOptionDetails(o config.Option) {
	if o.Default != nil {
		fmt.Printf("         Default: %v\n", o.Default)
	}
	if o.Required {
		fmt.Println("         Required: yes")
	}
	if o.Validate != nil {
		printValidateInfo(o.Validate)
	}
	if o.ChoicesFrom != "" {
		fmt.Printf("         Choices from: %s\n",
			ui.MutedStyle.Render(o.ChoicesFrom))
	}
}

func printValidateInfo(v *config.InputRule) {
	if v.Pattern != "" {
		fmt.Printf("         Pattern: %s\n", v.Pattern)
	}
	if v.MinLength > 0 {
		fmt.Printf("         Min length: %d\n", v.MinLength)
	}
	if v.MaxLength > 0 {
		fmt.Printf("         Max length: %d\n", v.MaxLength)
	}
}

func printConditions(label string, conds map[string]any) {
	if len(conds) == 0 {
		return
	}
	conditions := make([]string, 0, len(conds))
	for k, v := range conds {
		conditions = append(conditions, fmt.Sprintf("%s=%v", k, v))
	}
	fmt.Printf("         %s: %s\n", label, strings.Join(conditions, ", "))
}

func pinsCmd(wizardName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pins",
		Short: "Manage pinned options",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runPins(wizardName)
		},
	}

	cmd.AddCommand(pinsShowCmd(wizardName))
	cmd.AddCommand(pinsClearCmd(wizardName))

	return cmd
}

func pinsShowCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display current pins",
		RunE: func(_ *cobra.Command, _ []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			st := store.New(configDir)
			pins, err := st.LoadPins(w.Name)
			if err != nil {
				return fmt.Errorf("loading pins: %w", err)
			}
			pinnedVer, _ := st.LoadPinnedVersion(w.Name)
			if pinnedVer == "" && len(pins) == 0 {
				fmt.Println("  No pins set.")
				return nil
			}
			if pinnedVer != "" {
				fmt.Printf("  version: %s\n", pinnedVer)
			}
			for k, v := range pins {
				fmt.Printf("  %s: %v\n", k, v)
			}
			return nil
		},
	}
}

func pinsClearCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all pins",
		RunE: func(_ *cobra.Command, _ []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			st := store.New(configDir)
			if err := st.SavePins(w.Name, nil); err != nil {
				return fmt.Errorf("saving pins: %w", err)
			}
			if err := st.SavePinnedVersion(w.Name, ""); err != nil {
				return fmt.Errorf("saving pinned version: %w", err)
			}
			fmt.Println("  Pins cleared.")
			return nil
		},
	}
}

func presetsCmd(wizardName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "presets",
		Short: "Manage presets",
	}

	cmd.AddCommand(presetsListCmd(wizardName))
	cmd.AddCommand(presetsShowCmd(wizardName))
	cmd.AddCommand(presetsExplainCmd(wizardName))
	cmd.AddCommand(presetsSaveCmd(wizardName))
	cmd.AddCommand(presetsDeleteCmd(wizardName))

	return cmd
}

func presetsListCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List presets",
		RunE: func(_ *cobra.Command, _ []string) error {
			st := store.New(configDir)
			names, err := st.ListPresets(wizardName)
			if err != nil {
				return fmt.Errorf("listing presets: %w", err)
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

func presetsShowCmd(wizardName string) *cobra.Command {
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
				return fmt.Errorf("loading preset %q: %w", args[0], err)
			}

			detectedVersion, _ := compat.DetectVersion(w.Version)
			w.Command = w.EffectiveCommand(detectedVersion)

			fmt.Printf("\n  Preset: %s\n\n", args[0])
			for k, v := range values {
				fmt.Printf("  %s: %v\n", k, v)
			}

			parts := command.Build(w, values)
			command.PrintCommand(parts)
			return nil
		},
	}
}

func presetsExplainCmd(wizardName string) *cobra.Command {
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
				return fmt.Errorf("loading preset %q: %w", args[0], err)
			}

			detectedVersion, _ := compat.DetectVersion(w.Version)
			w.Command = w.EffectiveCommand(detectedVersion)

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

			parts := command.Build(w, values)
			command.PrintCommand(parts)
			return nil
		},
	}
}

func presetsSaveCmd(wizardName string) *cobra.Command {
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

			detectedVersion, _ := compat.DetectVersion(w.Version)
			majorVersion := majorVer(detectedVersion)

			state, err := st.LoadState(w.Name, majorVersion)
			if err != nil {
				return fmt.Errorf("no state found: %w", err)
			}
			if len(state.LastUsed) == 0 {
				return errors.New("no last-used values to save — run the wizard first")
			}

			if err := st.SavePreset(wizardName, args[0], state.LastUsed); err != nil {
				return fmt.Errorf("saving preset %q: %w", args[0], err)
			}
			fmt.Printf("  Preset %q saved.\n", args[0])
			return nil
		},
	}
}

func presetsDeleteCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a preset",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st := store.New(configDir)
			if err := st.DeletePreset(wizardName, args[0]); err != nil {
				return fmt.Errorf("deleting preset %q: %w", args[0], err)
			}
			fmt.Printf("  Preset %q deleted.\n", args[0])
			return nil
		},
	}
}
