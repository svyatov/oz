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

type completionFunc = func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)

func completePresetNames(wizardName string) completionFunc {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= 1 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		st := store.New(configDir)
		names, err := st.ListPresets(wizardName)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

func doctorCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check tool installation and detected version",
		Long: `Run diagnostics for the wizard's underlying tool: verify it is installed,
detect its version, show the active compat range, and report any issues.`,
		Example: fmt.Sprintf("  oz run %s doctor", wizardName),
		RunE: func(_ *cobra.Command, _ []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Printf("  %s %s\n", ui.AccentStyle.Render("Wizard:"), w.Name)
			fmt.Printf("  %s %s\n", ui.AccentStyle.Render("Command:"), w.Command)

			if w.Version == nil {
				fmt.Printf("  %s %s\n", ui.AccentStyle.Render("Version detection:"), "not configured")
				fmt.Println()
				return nil
			}

			ver, err := compat.DetectVersion(w.Version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s %s %v\n",
					ui.AccentStyle.Render("Version detection:"),
					ui.MutedStyle.Render("failed —"), err)
				fmt.Println()
				return nil
			}

			fmt.Printf("  %s %s\n", ui.AccentStyle.Render("Detected version:"), ver)
			printDoctorVersionDetails(w, ver)
			printDoctorCompat(w, ver)

			fmt.Println()
			return nil
		},
	}
}

func printDoctorVersionDetails(w *config.Wizard, ver string) {
	if w.Version.CustomVersionCmd != "" {
		fmt.Printf("  %s %s\n", ui.AccentStyle.Render("Version template:"), w.Version.CustomVersionCmd)
		fmt.Printf("  %s %s\n", ui.AccentStyle.Render("Effective command:"), w.EffectiveCommand(ver))
	}
	if w.Version.CustomVersionVerify != "" {
		fmt.Printf("  %s %s\n", ui.AccentStyle.Render("Verify template:"), w.Version.CustomVersionVerify)
	}
	if w.Version.AvailVersionsCmd != "" {
		fmt.Printf("  %s command — %s\n", ui.AccentStyle.Render("Available versions:"), w.Version.AvailVersionsCmd)
	} else if w.Version.AvailVersions != "" {
		fmt.Printf("  %s static — %s\n", ui.AccentStyle.Render("Available versions:"), w.Version.AvailVersions)
	}
}

func printDoctorCompat(w *config.Wizard, ver string) {
	if len(w.Compat) == 0 {
		return
	}
	matched := compat.MatchedRanges(w.Compat, ver)
	if len(matched) > 0 {
		filtered := compat.FilterOptions(w.Options, w.Compat, ver)
		fmt.Printf("  %s %s (%d options)\n", ui.AccentStyle.Render("Compat match:"),
			strings.Join(matched, " + "), len(filtered))
	} else {
		fmt.Printf("  %s %s\n", ui.AccentStyle.Render("Compat match:"), "none (all options shown)")
	}
}

func showCmd(wizardName string) *cobra.Command {
	var forVersion string

	cmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{"s"},
		Short:   "Show all options with descriptions",
		Long: `Display every wizard option with its type, flags, default value,
choices, and visibility conditions. Useful for reviewing a wizard
config without running it.

Use --for-version to see options for a specific version instead of
the detected one.`,
		Example: fmt.Sprintf("  oz run %s show\n  oz run %s show --for-version 8.0",
			wizardName, wizardName),
		RunE: func(_ *cobra.Command, _ []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			version := forVersion
			if version == "" {
				version, _ = compat.DetectVersion(w.Version)
			}
			options := compat.FilterOptions(w.Options, w.Compat, version)

			fmt.Printf("\n  %s\n", ui.Header(w.Name, version, versionLabel(w)))
			effectiveCmd := w.EffectiveCommand(version)
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

	cmd.Flags().StringVar(&forVersion, "for-version", "", "show options for a specific version")

	return cmd
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
		fmt.Printf("         Default: %s\n", o.Default.Display())
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

func printConditions(label string, conds config.Values) {
	if len(conds) == 0 {
		return
	}
	conditions := make([]string, 0, len(conds))
	for k, v := range conds {
		conditions = append(conditions, fmt.Sprintf("%s=%s", k, v.Display()))
	}
	fmt.Printf("         %s: %s\n", label, strings.Join(conditions, ", "))
}

func pinsCmd(wizardName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pins",
		Short: "Manage pinned options",
		Long: `Open an interactive TUI to pin option values. Pinned options are
skipped during the wizard and use their pinned value automatically.
Use "pins list" to view or "pins clear" to remove all pins.`,
		Example: fmt.Sprintf("  oz run %s pins\n  oz run %s pins list", wizardName, wizardName),
		RunE: func(_ *cobra.Command, _ []string) error {
			return runPins(wizardName)
		},
	}

	cmd.AddCommand(pinsListCmd(wizardName))
	cmd.AddCommand(pinsClearCmd(wizardName))

	return cmd
}

func pinsListCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"l", "ls"},
		Short:   "List current pins",
		Long:    "Display all currently pinned option values for this wizard.",
		Example: fmt.Sprintf("  oz run %s pins list", wizardName),
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
				ui.InfoMsgf("No pins set")
				return nil
			}
			fmt.Println()
			if pinnedVer != "" {
				fmt.Printf("  %s: %s\n", ui.AccentStyle.Render("version"), pinnedVer)
			}
			for k, v := range pins {
				fmt.Printf("  %s: %s\n", ui.AccentStyle.Render(k), v.Display())
			}
			fmt.Println()
			return nil
		},
	}
}

func pinsClearCmd(wizardName string) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "clear",
		Short:   "Remove all pins",
		Long:    "Remove all pinned option values and the pinned version for this wizard.",
		Example: fmt.Sprintf("  oz run %s pins clear\n  oz run %s pins clear --force", wizardName, wizardName),
		RunE: func(_ *cobra.Command, _ []string) error {
			w, err := loadWizardConfig(wizardName)
			if err != nil {
				return err
			}

			if !force && !confirmDangerousPrompt("Clear all pins?") {
				return nil
			}

			st := store.New(configDir)
			if err := st.SavePins(w.Name, nil); err != nil {
				return fmt.Errorf("saving pins: %w", err)
			}
			if err := st.SavePinnedVersion(w.Name, ""); err != nil {
				return fmt.Errorf("saving pinned version: %w", err)
			}
			ui.SuccessMsgf("Pins cleared")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")

	return cmd
}

func presetsCmd(wizardName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "presets",
		Short: "Manage presets",
		Long: `Manage named presets for this wizard. Presets store a complete set of
option values that can be replayed with "oz run <wizard> -p <name>".`,
		Example: fmt.Sprintf(`  oz run %s presets list
  oz run %s presets show mypreset
  oz run %s presets save mypreset`, wizardName, wizardName, wizardName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(presetsListCmd(wizardName))
	cmd.AddCommand(presetsShowCmd(wizardName))
	cmd.AddCommand(presetsSaveCmd(wizardName))
	cmd.AddCommand(presetsRemoveCmd(wizardName))

	return cmd
}

func listPresets(wizardName string) error {
	st := store.New(configDir)
	names, err := st.ListPresets(wizardName)
	if err != nil {
		return fmt.Errorf("listing presets: %w", err)
	}
	if len(names) == 0 {
		ui.InfoMsgf("No presets found")
		return nil
	}
	fmt.Println()
	for _, n := range names {
		fmt.Printf("  %s\n", ui.AccentStyle.Render(n))
	}
	fmt.Println()
	return nil
}

func presetsListCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"l", "ls"},
		Short:   "List presets",
		Long:    "List all saved presets for this wizard.",
		Example: fmt.Sprintf("  oz run %s presets list", wizardName),
		RunE: func(_ *cobra.Command, _ []string) error {
			return listPresets(wizardName)
		},
	}
}

func presetsShowCmd(wizardName string) *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:               "show <name>",
		Aliases:           []string{"s"},
		Short:             "Show preset values and generated command",
		Long: `Display stored values for a preset and the command it produces.
Use --verbose to include labels, descriptions, and choice annotations.`,
		Example: fmt.Sprintf("  oz run %s presets show mypreset\n  oz run %s presets show mypreset -v",
			wizardName, wizardName),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePresetNames(wizardName),
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

			fmt.Printf("\n  %s %s\n\n", ui.AccentStyle.Render("Preset:"), args[0])

			if verbose {
				printPresetVerbose(values, w.Options)
			} else {
				for k, v := range values {
					fmt.Printf("  %s: %s\n", ui.AccentStyle.Render(k), v.Display())
				}
			}

			parts := command.Build(w, values)
			command.PrintCommand(parts)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "include labels, descriptions, and choice annotations")

	return cmd
}

func printPresetVerbose(values config.Values, options []config.Option) {
	optMap := make(map[string]config.Option, len(options))
	for _, o := range options {
		optMap[o.Name] = o
	}
	for k, v := range values {
		opt, known := optMap[k]
		if !known {
			fmt.Printf("  %s: %s\n", ui.AccentStyle.Render(k), v.Display())
			continue
		}
		fmt.Printf("  %s: %s\n", ui.TitleStyle.Render(opt.Label), v.Display())
		if opt.Description != "" {
			fmt.Printf("    %s\n", ui.MutedStyle.Render(opt.Description))
		}
		for _, c := range opt.Choices {
			if v.Scalar() == c.Value && c.Description != "" {
				fmt.Printf("    %s\n", ui.MutedStyle.Render(c.Description))
				break
			}
		}
	}
}

func presetsSaveCmd(wizardName string) *cobra.Command {
	return &cobra.Command{
		Use:               "save <name>",
		Short:             "Save last-used values as named preset",
		Long:              "Save the most recent wizard answers as a named preset for later replay.",
		Example:           fmt.Sprintf("  oz run %s presets save fast", wizardName),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePresetNames(wizardName),
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
			ui.SuccessMsgf("Preset %q saved", args[0])
			return nil
		},
	}
}

func presetsRemoveCmd(wizardName string) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:               "remove <name>",
		Aliases:           []string{"rm"},
		Short:             "Remove a preset",
		Long:              "Delete a saved preset by name. Requires confirmation unless --force is set.",
		Example: fmt.Sprintf("  oz run %s presets remove old\n  oz run %s presets rm old -f",
			wizardName, wizardName),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePresetNames(wizardName),
		RunE: func(_ *cobra.Command, args []string) error {
			if !force && !confirmDangerousPrompt(fmt.Sprintf("Remove preset %q?", args[0])) {
				return nil
			}
			st := store.New(configDir)
			if err := st.RemovePreset(wizardName, args[0]); err != nil {
				return fmt.Errorf("removing preset %q: %w", args[0], err)
			}
			ui.SuccessMsgf("Preset %q removed", args[0])
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")

	return cmd
}
