package main

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/command"
	"github.com/svyatov/oz/internal/compat"
	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/store"
	"github.com/svyatov/oz/internal/ui"
	"github.com/svyatov/oz/internal/wizard"
)

func wizardCmd(name string) *cobra.Command {
	var (
		dryRun     bool
		presetName string
		pins       bool
	)

	cmd := &cobra.Command{
		Use:                name,
		Short:              fmt.Sprintf("Run %s wizard", name),
		DisableFlagParsing: false,
		// Accept any args (positional args for the wizard)
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if pins {
				return runPins(name)
			}
			return runWizard(name, args, presetName, dryRun)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print command without executing")
	cmd.Flags().StringVar(&presetName, "preset", "", "run with saved preset (non-interactive)")
	cmd.Flags().BoolVar(&pins, "pins", false, "interactive multi-select to toggle pins")

	cmd.AddCommand(doctorCmd(name))
	cmd.AddCommand(explainCmd(name))
	cmd.AddCommand(presetCmd(name))

	return cmd
}

func loadWizardConfig(name string) (*config.Wizard, error) {
	w, err := config.FindWizard(configDir, name)
	if err != nil {
		return nil, fmt.Errorf("wizard %q not found in %s: %w", name, config.WizardsDir(configDir), err)
	}
	errs := config.Validate(w)
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid wizard config:\n%s", config.FormatErrors(errs))
	}
	return w, nil
}

func runWizard(name string, args []string, presetName string, dryRun bool) error {
	w, err := loadWizardConfig(name)
	if err != nil {
		return err
	}

	st := store.New(configDir)
	detectedVersion, err := compat.DetectVersion(w.Detect)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: version detection failed: %v\n", err)
	}
	majorVersion := majorVer(detectedVersion)
	options := compat.FilterOptions(w.Options, w.Compat, detectedVersion)

	state, err := st.LoadState(w.Name, majorVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load state: %v\n", err)
		state = &store.StateEntry{}
	}

	positionalArgs := buildPositionalArgs(w.Args, args)
	if presetName != "" {
		return runWithPreset(w, st, positionalArgs, presetName, dryRun)
	}

	filteredOptions, pinnedCount := wizard.FilterPinned(options, state.Pins)
	pinnedAnswers := make(map[string]any)
	maps.Copy(pinnedAnswers, state.Pins)

	result, err := wizard.Run(
		w.Name, detectedVersion, filteredOptions,
		pinnedCount, state.LastUsed, pinnedAnswers,
	)
	if err != nil {
		return fmt.Errorf("running wizard: %w", err)
	}
	if result.Aborted {
		return nil
	}

	allAnswers := make(map[string]any)
	maps.Copy(allAnswers, pinnedAnswers)
	maps.Copy(allAnswers, result.Answers)

	parts := command.Build(w, positionalArgs, allAnswers)
	command.PrintCommand(parts)

	if dryRun {
		return nil
	}
	if !confirmPrompt("  Execute?") {
		return nil
	}

	saveRunState(st, w.Name, majorVersion, state, result.Answers, allAnswers)
	fmt.Println()
	if err := command.Run(command.PlainParts(parts)); err != nil {
		return fmt.Errorf("executing command: %w", err)
	}
	return nil
}

func saveRunState(
	st *store.Store, wizardName, majorVersion string,
	state *store.StateEntry, answers wizard.Answers, allAnswers map[string]any,
) {
	if state.LastUsed == nil {
		state.LastUsed = make(map[string]any)
	}
	maps.Copy(state.LastUsed, answers)
	if err := st.SaveState(wizardName, majorVersion, state); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
	}

	presetSaveName := promptPresetSave()
	if presetSaveName != "" {
		if err := st.SavePreset(wizardName, presetSaveName, allAnswers); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save preset: %v\n", err)
		} else {
			fmt.Printf("  Preset %q saved.\n", presetSaveName)
		}
	}
}

func runWithPreset(
	w *config.Wizard, st *store.Store,
	positionalArgs map[string]string, presetName string, dryRun bool,
) error {
	values, err := st.LoadPreset(w.Name, presetName)
	if err != nil {
		return fmt.Errorf("loading preset %q: %w", presetName, err)
	}

	parts := command.Build(w, positionalArgs, values)
	command.PrintCommand(parts)

	if dryRun {
		return nil
	}

	if err := command.Run(command.PlainParts(parts)); err != nil {
		return fmt.Errorf("executing command: %w", err)
	}
	return nil
}

func runPins(name string) error {
	w, err := loadWizardConfig(name)
	if err != nil {
		return err
	}

	st := store.New(configDir)

	detectedVersion, _ := compat.DetectVersion(w.Detect)
	majorVersion := majorVer(detectedVersion)
	options := compat.FilterOptions(w.Options, w.Compat, detectedVersion)

	state, err := st.LoadState(w.Name, majorVersion)
	if err != nil {
		state = &store.StateEntry{}
	}
	if state.Pins == nil {
		state.Pins = make(map[string]any)
	}

	selected, err := promptPinSelection(options, state.Pins)
	if err != nil {
		return err
	}

	state.Pins = resolvePinValues(options, selected, state.LastUsed)
	if err := st.SaveState(w.Name, majorVersion, state); err != nil {
		return fmt.Errorf("saving pins: %w", err)
	}

	fmt.Printf("  %d option(s) pinned.\n", len(state.Pins))
	return nil
}

func promptPinSelection(
	options []config.Option, currentPins map[string]any,
) ([]string, error) {
	var optionNames []huh.Option[string]
	var currentlyPinned []string
	for _, o := range options {
		label := o.Label
		if o.Description != "" {
			label += "  " + o.Description
		}
		optionNames = append(optionNames, huh.NewOption(label, o.Name))
		if _, ok := currentPins[o.Name]; ok {
			currentlyPinned = append(currentlyPinned, o.Name)
		}
	}

	selected := currentlyPinned
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Pinned options").
				Description("Pinned options use their last-used value and are skipped in the wizard").
				Options(optionNames...).
				Value(&selected),
		),
	).WithTheme(ui.HuhTheme())

	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("running pin selection: %w", err)
	}
	return selected, nil
}

func resolvePinValues(
	options []config.Option, selected []string, lastUsed map[string]any,
) map[string]any {
	selectedSet := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedSet[s] = true
	}

	pins := make(map[string]any)
	for _, o := range options {
		if !selectedSet[o.Name] {
			continue
		}
		v, ok := lastUsed[o.Name]
		switch {
		case ok:
			pins[o.Name] = v
		case o.Default != nil:
			pins[o.Name] = o.Default
		case o.Type == "confirm":
			pins[o.Name] = false
		default:
			pins[o.Name] = ""
		}
	}
	return pins
}

func buildPositionalArgs(argDefs []config.Arg, cliArgs []string) map[string]string {
	result := make(map[string]string)
	argIdx := 0
	for _, a := range argDefs {
		if argIdx < len(cliArgs) {
			result[a.Name] = cliArgs[argIdx]
			argIdx++
		}
	}
	return result
}

func majorVer(version string) string {
	if version == "" {
		return ""
	}
	parts := strings.SplitN(version, ".", 2)
	if len(parts) >= 2 {
		return parts[0] + "." + strings.SplitN(parts[1], ".", 2)[0]
	}
	return parts[0]
}

func confirmPrompt(msg string) bool {
	fmt.Printf("%s [Y/n] ", msg)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "" || line == "y" || line == "yes"
}

func promptPresetSave() string {
	fmt.Print("  Save as preset? (name or Enter to skip): ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
