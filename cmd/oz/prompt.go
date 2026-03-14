package main

import (
	"fmt"
	"os"

	"charm.land/huh/v2"
)

func confirmPrompt(msg string, defaultYes bool) bool {
	confirm := defaultYes
	if err := huh.NewConfirm().Title(msg).Affirmative("Yes").Negative("No").Value(&confirm).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "prompt failed: %v\n", err)
	}
	return confirm
}

func promptPresetSave() string {
	var name string
	if err := huh.NewInput().Title("Save as preset?").Placeholder("name or Enter to skip").Value(&name).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "prompt failed: %v\n", err)
	}
	return name
}
