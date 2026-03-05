package main

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"strings"

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
	)

	cmd := &cobra.Command{
		Use:                name,
		Short:              fmt.Sprintf("Run %s wizard", name),
		DisableFlagParsing: false,
		// Accept any args (positional args for the wizard)
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWizard(name, args, presetName, dryRun)
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "print command without executing")
	cmd.Flags().StringVarP(&presetName, "with-preset", "p", "", "run with saved preset (non-interactive)")

	cmd.AddCommand(doctorCmd(name))
	cmd.AddCommand(explainCmd(name))
	cmd.AddCommand(pinsCmd(name))
	cmd.AddCommand(presetsCmd(name))

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

func resolveVersion(w *config.Wizard, st *store.Store) (*wizard.VersionResult, string) {
	versionPin, _ := st.LoadVersionPin(w.Name)
	vr, err := wizard.RunVersionLoader(w.Name, w.Version, versionPin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: version detection failed: %v\n", err)
		vr = &wizard.VersionResult{}
	}

	displayVersion := vr.Selected
	if vr.Selected != "" && vr.Selected != vr.Detected {
		displayVersion = vr.Selected + " (override)"
	}
	return vr, displayVersion
}

func runWizard(name string, args []string, presetName string, dryRun bool) error {
	w, err := loadWizardConfig(name)
	if err != nil {
		return err
	}

	st := store.New(configDir)
	vr, displayVersion := resolveVersion(w, st)
	if vr.Aborted {
		return nil
	}

	w.Command = w.EffectiveCommand(vr.Selected)
	majorVersion := majorVer(vr.Selected)
	options := compat.FilterOptions(w.Options, w.Compat, vr.Selected)

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
		w.Name, displayVersion, filteredOptions,
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
	saveLastUsed(st, w.Name, majorVersion, state, result.Answers)
	return confirmAndExecute(st, w.Name, parts, allAnswers, dryRun)
}

func confirmAndExecute(
	st *store.Store, wizardName string,
	parts []command.Part, allAnswers map[string]any, dryRun bool,
) error {
	if dryRun {
		return nil
	}
	if !confirmPrompt("  Execute?") {
		return nil
	}
	promptAndSavePreset(st, wizardName, allAnswers)
	fmt.Println()
	if err := command.Run(command.PlainParts(parts)); err != nil {
		return fmt.Errorf("executing command: %w", err)
	}
	return nil
}

func saveLastUsed(
	st *store.Store, wizardName, majorVersion string,
	state *store.StateEntry, answers wizard.Answers,
) {
	if state.LastUsed == nil {
		state.LastUsed = make(map[string]any)
	}
	maps.Copy(state.LastUsed, answers)
	if err := st.SaveState(wizardName, majorVersion, state); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
	}
}

func promptAndSavePreset(st *store.Store, wizardName string, allAnswers map[string]any) {
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

	detectedVersion, _ := compat.DetectVersion(w.Version)
	majorVersion := majorVer(detectedVersion)
	options := compat.FilterOptions(w.Options, w.Compat, detectedVersion)

	state, err := st.LoadState(w.Name, majorVersion)
	if err != nil {
		state = &store.StateEntry{}
	}
	if state.Pins == nil {
		state.Pins = make(map[string]any)
	}

	hasCustomVersion := w.Version != nil && w.Version.CustomVersionCmd != ""
	versionPin, _ := st.LoadVersionPin(w.Name)
	result, err := wizard.RunPins(options, state.Pins, state.LastUsed, hasCustomVersion, versionPin)
	if err != nil {
		return fmt.Errorf("managing pins: %w", err)
	}

	state.Pins = result.Pins
	if err := st.SaveState(w.Name, majorVersion, state); err != nil {
		return fmt.Errorf("saving pins: %w", err)
	}
	if hasCustomVersion {
		if err := st.SaveVersionPin(w.Name, result.VersionPin); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save version pin: %v\n", err)
		}
	}

	fmt.Printf("  %s pinned.\n", ui.Plural(len(state.Pins), "option"))
	return nil
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
