package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func confirmPrompt(msg string) bool {
	fmt.Printf("%s [Y/n] ", msg)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "" || line == "y" || line == "yes"
}

func confirmDangerousPrompt(msg string) bool {
	fmt.Printf("%s [y/N] ", msg)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func promptPresetSave() string {
	fmt.Print("  Save as preset? (name or Enter to skip): ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
