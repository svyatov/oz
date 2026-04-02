package main

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/command"
	"github.com/svyatov/oz/internal/compat"
	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/store"
	"github.com/svyatov/oz/internal/ui"
	"github.com/svyatov/oz/internal/wizard"
)

func wizardCmd(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                name,
		Short:              fmt.Sprintf("Run %s wizard", name),
		DisableFlagParsing: false,
		Example: fmt.Sprintf(`  oz run %s
  oz run %s --dry-run
  oz run %s -p fast
  oz run %s -p fast -- myproject
  oz run %s -- --extra-flag
  oz run %s doctor
  oz run %s show
  oz run %s presets list`, name, name, name, name, name, name, name, name),
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			presetName, _ := cmd.Flags().GetString("preset")
			return runWizard(name, presetName, dryRun, args)
		},
	}

	cmd.AddCommand(doctorCmd(name))
	cmd.AddCommand(showCmd(name))
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
		ui.WarnMsgf("version detection failed: %v", err)
		vr = &wizard.VersionResult{}
	}
	overridden := vr.Selected != "" && vr.Selected != vr.Detected
	return vr, overridden
}

type wizardSession struct {
	vr            *wizard.VersionResult
	result        *wizard.Result
	state         *store.StateEntry
	pinnedValues  config.Values
}

func runWizardLoop(
	w *config.Wizard, st *store.Store,
	presetName string, dryRun bool, extraArgs []string,
) (*wizardSession, error) {
	var prevResult *wizard.VersionResult
	for {
		vr, overridden := resolveVersion(w, st, prevResult)
		if vr.Aborted {
			return &wizardSession{result: &wizard.Result{Aborted: true}}, nil
		}

		w.Command = w.EffectiveCommand(vr.Selected)
		options := compat.FilterOptions(w.Options, vr.Selected)
		for i := range options {
			if len(options[i].Choices) > 0 {
				options[i].Choices = compat.FilterChoices(options[i].Choices, vr.Selected)
			}
		}

		majorVersion := majorVer(vr.Selected)
		state, err := st.LoadState(w.Name, majorVersion)
		if err != nil {
			ui.WarnMsgf("failed to load state: %v", err)
			state = &store.StateEntry{}
		}

		if presetName != "" {
			return nil, runWithPreset(w, st, presetName, dryRun, extraArgs)
		}

		// Load version-independent pins, keep only those matching filtered options.
		allPins, _ := st.LoadPins(w.Name)
		activePins := filterActivePins(allPins, options)

		filteredOptions, pinnedCount := wizard.FilterPinned(options, activePins)
		pinnedValues := make(config.Values)
		maps.Copy(pinnedValues, activePins)

		result, err := wizard.Run(wizard.RunParams{
			WizardName:   w.Name,
			Version:      vr.Selected,
			VersionLabel: versionLabel(w),
			Overridden:   overridden,
			Options:      filteredOptions,
			PinnedCount:  pinnedCount,
			Defaults:     state.LastUsed,
			PinnedValues: pinnedValues,
			CanGoBack:    vr.Interactive,
		})
		if err != nil {
			return nil, fmt.Errorf("running wizard: %w", err)
		}
		if result.GoBack && vr.Interactive {
			prevResult = vr
			continue
		}
		return &wizardSession{vr: vr, result: result, state: state, pinnedValues: pinnedValues}, nil
	}
}

func runWizard(name string, presetName string, dryRun bool, extraArgs []string) error {
	w, err := loadWizardConfig(name)
	if err != nil {
		return err
	}

	st := store.New(configDir)
	s, err := runWizardLoop(w, st, presetName, dryRun, extraArgs)
	if err != nil {
		return err
	}
	if s == nil {
		return nil // preset path already handled
	}
	if s.result.Aborted {
		return nil
	}

	majorVersion := majorVer(s.vr.Selected)

	allAnswers := make(config.Values)
	maps.Copy(allAnswers, s.pinnedValues)
	maps.Copy(allAnswers, s.result.Values)

	matched, rawExtra := command.ParseExtra(w.Options, extraArgs)
	maps.Copy(allAnswers, matched)

	if missing := wizard.MissingRequired(w.Options, allAnswers); len(missing) > 0 {
		return fmt.Errorf("missing required options: %s", strings.Join(missing, ", "))
	}

	parts := command.Build(w, allAnswers)
	parts = command.AppendExtra(parts, rawExtra)
	saveLastUsed(st, w.Name, majorVersion, s.state, s.result.Values)
	if dryRun {
		command.PrintCommand(parts)
		return nil
	}
	command.PrintCommand(parts)
	return confirmAndExecute(st, w.Name, parts, allAnswers)
}

func confirmAndExecute(
	st *store.Store, wizardName string,
	parts []command.Part, allAnswers config.Values,
) error {
	if !confirmPrompt("Execute?", true) {
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
	state *store.StateEntry, answers config.Values,
) {
	if state.LastUsed == nil {
		state.LastUsed = make(config.Values)
	}
	maps.Copy(state.LastUsed, answers)
	if err := st.SaveState(wizardName, majorVersion, state); err != nil {
		ui.WarnMsgf("failed to save state: %v", err)
	}
}

func promptAndSavePreset(st *store.Store, wizardName string, allAnswers config.Values) {
	presetSaveName := promptPresetSave()
	if presetSaveName != "" {
		if err := st.SavePreset(wizardName, presetSaveName, allAnswers); err != nil {
			ui.WarnMsgf("failed to save preset: %v", err)
		} else {
			ui.SuccessMsgf("Preset %q saved", presetSaveName)
		}
	}
}

func runWithPreset(
	w *config.Wizard, st *store.Store,
	presetName string, dryRun bool, extraArgs []string,
) error {
	values, err := st.LoadPreset(w.Name, presetName)
	if err != nil {
		return fmt.Errorf("loading preset %q: %w", presetName, err)
	}

	matched, rawExtra := command.ParseExtra(w.Options, extraArgs)
	maps.Copy(values, matched)

	if missing := wizard.MissingRequired(w.Options, values); len(missing) > 0 {
		return fmt.Errorf("missing required options: %s", strings.Join(missing, ", "))
	}

	parts := command.Build(w, values)
	parts = command.AppendExtra(parts, rawExtra)
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

func runPresets(name string) error {
	w, err := loadWizardConfig(name)
	if err != nil {
		return err
	}

	st := store.New(configDir)

	presetNames, err := st.ListPresets(w.Name)
	if err != nil {
		return fmt.Errorf("listing presets: %w", err)
	}

	presets := make(map[string]config.Values, len(presetNames))
	for _, pn := range presetNames {
		vals, loadErr := st.LoadPreset(w.Name, pn)
		if loadErr != nil {
			ui.WarnMsgf("failed to load preset %q: %v", pn, loadErr)
			continue
		}
		presets[pn] = vals
	}

	// Load last-used for "copy from last-used" source.
	detectedVersion, _ := compat.DetectVersion(w.Version)
	majorVersion := majorVer(detectedVersion)
	state, err := st.LoadState(w.Name, majorVersion)
	if err != nil {
		state = &store.StateEntry{}
	}

	hints := compat.OptionHints(w.Options)
	result, err := wizard.RunPresets(w.Options, presets, state.LastUsed, hints)
	if err != nil {
		return fmt.Errorf("managing presets: %w", err)
	}

	return reconcilePresets(st, w.Name, presetNames, result.Presets)
}

func reconcilePresets(
	st *store.Store, wizardName string,
	originalNames []string, final map[string]config.Values,
) error {
	// Save all presets in final state.
	for name, values := range final {
		if err := st.SavePreset(wizardName, name, values); err != nil {
			return fmt.Errorf("saving preset %q: %w", name, err)
		}
	}
	// Remove presets that no longer exist.
	for _, name := range originalNames {
		if _, exists := final[name]; !exists {
			if err := st.RemovePreset(wizardName, name); err != nil {
				ui.WarnMsgf("failed to remove preset %q: %v", name, err)
			}
		}
	}

	count := len(final)
	word := "presets"
	if count == 1 {
		word = "preset"
	}
	ui.SuccessMsgf("%d %s saved", count, word)
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
	hints := compat.OptionHints(w.Options)
	pinnedVer, _ := st.LoadPinnedVersion(w.Name)

	result, err := wizard.RunPins(wizard.PinsParams{
		Options:             w.Options,
		Pins:                pins,
		LastUsed:            state.LastUsed,
		Hints:               hints,
		HasCustomVersion:    hasCustomVersion,
		VersionPin:          pinnedVer,
		CustomVersionVerify: versionVerifyCmd(w),
	})
	if err != nil {
		return fmt.Errorf("managing pins: %w", err)
	}

	if err := st.SavePins(w.Name, result.Pins); err != nil {
		return fmt.Errorf("saving pins: %w", err)
	}
	if hasCustomVersion {
		if err := st.SavePinnedVersion(w.Name, result.VersionPin); err != nil {
			ui.WarnMsgf("failed to save pinned version: %v", err)
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
	ui.SuccessMsgf("%d %s pinned", count, word)
	return nil
}

func filterActivePins(allPins config.Values, options []config.Option) config.Values {
	optionSet := make(map[string]bool, len(options))
	for _, o := range options {
		optionSet[o.Name] = true
	}
	active := make(config.Values, len(allPins))
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
	v, err := semver.NewVersion(version)
	if err != nil {
		// Fallback for non-semver strings.
		parts := strings.SplitN(version, ".", 2)
		if len(parts) >= 2 {
			return parts[0] + "." + strings.SplitN(parts[1], ".", 2)[0]
		}
		return parts[0]
	}
	return strconv.FormatUint(v.Major(), 10) + "." + strconv.FormatUint(v.Minor(), 10)
}
