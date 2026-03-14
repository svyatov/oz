package main

import "charm.land/huh/v2"

func confirmPrompt(msg string) bool {
	confirm := true
	_ = huh.NewConfirm().Title(msg).Affirmative("Yes").Negative("No").Value(&confirm).Run()
	return confirm
}

func confirmDangerousPrompt(msg string) bool {
	var confirm bool
	_ = huh.NewConfirm().Title(msg).Affirmative("Yes").Negative("No").Value(&confirm).Run()
	return confirm
}

func promptPresetSave() string {
	var name string
	_ = huh.NewInput().Title("Save as preset?").Placeholder("name or Enter to skip").Value(&name).Run()
	return name
}
