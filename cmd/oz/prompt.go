package main

import (
	"fmt"
	"os"

	"charm.land/huh/v2"
)

// confirmPrompt asks for yes/no confirmation. Tests can override this variable.
var confirmPrompt = func(msg string, defaultYes bool) bool {
	confirm := defaultYes
	if err := huh.NewConfirm().Title(msg).Affirmative("Yes").Negative("No").Value(&confirm).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "prompt failed: %v\n", err)
	}
	return confirm
}

// promptPresetSave asks for a preset name. Tests can override this variable.
var promptPresetSave = func() string {
	var name string
	if err := huh.NewInput().Title("Save as preset?").Placeholder("name or Enter to skip").Value(&name).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "prompt failed: %v\n", err)
	}
	return name
}
