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
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWizard(name, presetName, dryRun)
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

func resolveVersion(w *config.Wizard, st *store.Store, preselect string) (*wizard.VersionResult, bool) {
	versionPin, _ := st.LoadVersionPin(w.Name)
	vr, err := wizard.RunVersionLoader(w.Name, w.Version, versionPin, preselect)
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
	pinnedAnswers map[string]any
}

func runWizardLoop(w *config.Wizard, st *store.Store, presetName string, dryRun bool) (*wizardSession, error) {
	var prevSelected string
	for {
		vr, overridden := resolveVersion(w, st, prevSelected)
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

		filteredOptions, pinnedCount := wizard.FilterPinned(options, state.Pins)
		pinnedAnswers := make(map[string]any)
		maps.Copy(pinnedAnswers, state.Pins)

		result, err := wizard.Run(
			w.Name, vr.Selected, versionLabel(w), overridden, filteredOptions,
			pinnedCount, state.LastUsed, pinnedAnswers, vr.Interactive,
		)
		if err != nil {
			return nil, fmt.Errorf("running wizard: %w", err)
		}
		if result.GoBack && vr.Interactive {
			prevSelected = vr.Selected
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

	allAnswers := make(map[string]any)
	maps.Copy(allAnswers, s.pinnedAnswers)
	maps.Copy(allAnswers, s.result.Answers)

	parts := command.Build(w, allAnswers)
	command.PrintCommand(parts)
	saveLastUsed(st, w.Name, majorVersion, s.state, s.result.Answers)
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
	presetName string, dryRun bool,
) error {
	values, err := st.LoadPreset(w.Name, presetName)
	if err != nil {
		return fmt.Errorf("loading preset %q: %w", presetName, err)
	}

	parts := command.Build(w, values)
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
