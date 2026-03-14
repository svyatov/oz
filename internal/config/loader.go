package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

// DefaultConfigDir returns the default oz config directory.
// Respects $OZ_CONFIG_DIR, then $XDG_CONFIG_HOME/oz, then ~/.config/oz.
func DefaultConfigDir() string {
	if dir, ok := os.LookupEnv("OZ_CONFIG_DIR"); ok {
		return dir
	}
	return filepath.Join(xdg.ConfigHome, "oz")
}

// WizardsDir returns the wizards subdirectory.
func WizardsDir(configDir string) string {
	return filepath.Join(configDir, "wizards")
}

// LoadWizard loads and parses a wizard YAML file.
func LoadWizard(path string) (*Wizard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading wizard config: %w", err)
	}
	return ParseWizard(data)
}

// ParseWizard parses wizard YAML bytes.
func ParseWizard(data []byte) (*Wizard, error) {
	var w Wizard
	if err := yaml.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("parsing wizard YAML: %w", err)
	}
	return &w, nil
}

// WizardPath returns the full path to a wizard config file.
func WizardPath(configDir, name string) string {
	return filepath.Join(WizardsDir(configDir), name+".yml")
}

// FindWizard looks up a wizard by name in the config directory.
func FindWizard(configDir, name string) (*Wizard, error) {
	return LoadWizard(WizardPath(configDir, name))
}

// ListWizards returns all wizard configs found in the config directory.
func ListWizards(configDir string) ([]*Wizard, error) {
	dir := WizardsDir(configDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading wizards directory: %w", err)
	}

	var wizards []*Wizard
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		w, err := LoadWizard(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", e.Name(), err)
		}
		wizards = append(wizards, w)
	}
	return wizards, nil
}
