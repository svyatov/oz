package main

import (
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/command"
	"github.com/svyatov/oz/internal/compat"
	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/store"
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
		Example: fmt.Sprintf(`  oz run %s
  oz run %s --dry-run
  oz run %s -p fast
  oz run %s doctor
  oz run %s presets list`, name, name, name, name, name),
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWizard(name, presetName, dryRun)
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "print command without executing")
	cmd.Flags().StringVarP(&presetName, "with-preset", "p", "", "run with saved preset (non-interactive)")

	cmd.AddCommand(doctorCmd(name))
	cmd.AddCommand(inspectCmd(name))
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

func resolveVersion(w *config.Wizard, st *store.Store, cached *wizard.VersionResult) (*wizard.VersionResult, bool) {
	pinnedVer, _ := st.LoadPinnedVersion(w.Name)
	vr, err := wizard.RunVersionLoader(w.Name, w.Version, pinnedVer, cached)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: version detection failed: %v\n", err)
		vr = &wizard.VersionResult{}
	}
	overridden := vr.Selected != "" && vr.Selected != vr.Detected
	return vr, overridden
}

type wizardSession struct {
	vr            *wizard.VersionResult
	result        *wizard.Result
	state         *store.StateEntry
	pinnedAnswers wizard.Answers
}

func runWizardLoop(w *config.Wizard, st *store.Store, presetName string, dryRun bool) (*wizardSession, error) {
	var prevResult *wizard.VersionResult
	for {
		vr, overridden := resolveVersion(w, st, prevResult)
		if vr.Aborted {
			return &wizardSession{result: &wizard.Result{Aborted: true}}, nil
		}

		w.Command = w.EffectiveCommand(vr.Selected)
		options := compat.FilterOptions(w.Options, w.Compat, vr.Selected)

		majorVersion := majorVer(vr.Selected)
		state, err := st.LoadState(w.Name, majorVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load state: %v\n", err)
			state = &store.StateEntry{}
		}

		if presetName != "" {
			return nil, runWithPreset(w, st, presetName, dryRun)
		}

		// Load version-independent pins, keep only those matching filtered options.
		allPins, _ := st.LoadPins(w.Name)
		activePins := filterActivePins(allPins, options)

		filteredOptions, pinnedCount := wizard.FilterPinned(options, activePins)
		pinnedAnswers := make(wizard.Answers)
		maps.Copy(pinnedAnswers, activePins)

		result, err := wizard.Run(wizard.RunParams{
			WizardName:    w.Name,
			Version:       vr.Selected,
			VersionLabel:  versionLabel(w),
			Overridden:    overridden,
			Options:       filteredOptions,
			PinnedCount:   pinnedCount,
			Defaults:      state.LastUsed,
			PinnedAnswers: pinnedAnswers,
			CanGoBack:     vr.Interactive,
		})
		if err != nil {
			return nil, fmt.Errorf("running wizard: %w", err)
		}
		if result.GoBack && vr.Interactive {
			prevResult = vr
			continue
		}
		return &wizardSession{vr: vr, result: result, state: state, pinnedAnswers: pinnedAnswers}, nil
	}
}

func runWizard(name string, presetName string, dryRun bool) error {
	w, err := loadWizardConfig(name)
	if err != nil {
		return err
	}

	st := store.New(configDir)
	s, err := runWizardLoop(w, st, presetName, dryRun)
	if err != nil {
		return err
	}
	if s.result.Aborted {
		return nil
	}

	majorVersion := majorVer(s.vr.Selected)

	allAnswers := make(wizard.Answers)
	maps.Copy(allAnswers, s.pinnedAnswers)
	maps.Copy(allAnswers, s.result.Answers)

	parts := command.Build(w, allAnswers)
	saveLastUsed(st, w.Name, majorVersion, s.state, s.result.Answers)
	if dryRun {
		command.PrintCommand(parts)
		return nil
	}
	command.PrintCommand(parts)
	return confirmAndExecute(st, w.Name, parts, allAnswers)
}

func confirmAndExecute(
	st *store.Store, wizardName string,
	parts []command.Part, allAnswers map[string]any,
) error {
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
	presetName string, dryRun bool,
) error {
	values, err := st.LoadPreset(w.Name, presetName)
	if err != nil {
		return fmt.Errorf("loading preset %q: %w", presetName, err)
	}

	parts := command.Build(w, values)
	if dryRun {
		command.PrintCommand(parts)
		return nil
	}
	command.PrintCommand(parts)
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

	allPins, _ := st.LoadPins(w.Name)
	pins := filterActivePins(allPins, w.Options)

	// Collect last-used for defaults.
	detectedVersion, _ := compat.DetectVersion(w.Version)
	majorVersion := majorVer(detectedVersion)
	state, err := st.LoadState(w.Name, majorVersion)
	if err != nil {
		state = &store.StateEntry{}
	}

	hasCustomVersion := w.Version != nil && w.Version.CustomVersionCmd != ""
	hints := compat.OptionHints(w.Compat)
	pinnedVer, _ := st.LoadPinnedVersion(w.Name)

	result, err := wizard.RunPins(
		w.Options, pins, state.LastUsed,
		hints, hasCustomVersion, pinnedVer,
		versionVerifyCmd(w),
	)
	if err != nil {
		return fmt.Errorf("managing pins: %w", err)
	}

	if err := st.SavePins(w.Name, result.Pins); err != nil {
		return fmt.Errorf("saving pins: %w", err)
	}
	if hasCustomVersion {
		if err := st.SavePinnedVersion(w.Name, result.VersionPin); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save pinned version: %v\n", err)
		}
	}

	count := len(result.Pins)
	if result.VersionPin != "" {
		count++
	}
	word := "options"
	if count == 1 {
		word = "option"
	}
	fmt.Printf("  %d %s pinned.\n", count, word)
	return nil
}

func filterActivePins(allPins map[string]any, options []config.Option) map[string]any {
	optionSet := make(map[string]bool, len(options))
	for _, o := range options {
		optionSet[o.Name] = true
	}
	active := make(map[string]any, len(allPins))
	for k, v := range allPins {
		if optionSet[k] {
			active[k] = v
		}
	}
	return active
}

func versionVerifyCmd(w *config.Wizard) string {
	if w.Version != nil {
		return w.Version.CustomVersionVerify
	}
	return ""
}

func versionLabel(w *config.Wizard) string {
	if w.Version != nil {
		return w.Version.Label
	}
	return ""
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

