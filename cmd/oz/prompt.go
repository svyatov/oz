package main

import (
	"charm.land/huh/v2"

	"github.com/svyatov/oz/internal/ui"
)

func confirmPrompt(msg string, defaultYes bool) bool {
	confirm := defaultYes
	if err := huh.NewConfirm().Title(msg).Affirmative("Yes").Negative("No").Value(&confirm).Run(); err != nil {
		ui.WarnMsgf("prompt failed: %v", err)
	}
	return confirm
}

func promptPresetSave() string {
	var name string
	if err := huh.NewInput().Title("Save as preset?").Placeholder("name or Enter to skip").Value(&name).Run(); err != nil {
		ui.WarnMsgf("prompt failed: %v", err)
	}
	return name
}
