package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Store manages state and preset files for wizards.
type Store struct {
	configDir string
}

func New(configDir string) *Store {
	return &Store{configDir: configDir}
}

// --- State (last-used + pins) ---

// VersionedState is used when version_control is configured.
type VersionedState struct {
	Versions   map[string]*StateEntry `yaml:"versions"`
	VersionPin string                 `yaml:"version_pin,omitempty"`
}

// StateEntry holds last-used values and pins for a single version (or global).
type StateEntry struct {
	LastUsed map[string]any `yaml:"last_used,omitempty"`
	Pins     map[string]any `yaml:"pins,omitempty"`
}

func (s *Store) statePath(wizard string) string {
	return filepath.Join(s.configDir, "state", wizard+".yml")
}

// LoadState reads the state file. Returns nil StateEntry (not error) if file doesn't exist.
func (s *Store) LoadState(wizard, version string) (*StateEntry, error) {
	data, err := os.ReadFile(s.statePath(wizard))
	if os.IsNotExist(err) {
		return &StateEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state: %w", err)
	}

	if version != "" {
		var vs VersionedState
		if err := yaml.Unmarshal(data, &vs); err != nil {
			return nil, fmt.Errorf("parsing versioned state: %w", err)
		}
		if vs.Versions == nil {
			return &StateEntry{}, nil
		}
		entry := vs.Versions[version]
		if entry == nil {
			return &StateEntry{}, nil
		}
		return entry, nil
	}

	var entry StateEntry
	if err := yaml.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	return &entry, nil
}

// SaveState writes the state file, merging with existing data.
func (s *Store) SaveState(wizard, version string, entry *StateEntry) error {
	path := s.statePath(wizard)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	if version != "" {
		return s.saveVersionedState(path, version, entry)
	}

	data, err := yaml.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	return nil
}

func (s *Store) saveVersionedState(path, version string, entry *StateEntry) error {
	var vs VersionedState

	existing, err := os.ReadFile(path)
	if err == nil {
		_ = yaml.Unmarshal(existing, &vs)
	}
	if vs.Versions == nil {
		vs.Versions = make(map[string]*StateEntry)
	}

	vs.Versions[version] = entry

	data, err := yaml.Marshal(&vs)
	if err != nil {
		return fmt.Errorf("marshaling versioned state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing versioned state: %w", err)
	}
	return nil
}

// LoadVersionPin reads the version pin for a wizard.
func (s *Store) LoadVersionPin(wizard string) (string, error) {
	data, err := os.ReadFile(s.statePath(wizard))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading state: %w", err)
	}
	var vs VersionedState
	if err := yaml.Unmarshal(data, &vs); err != nil {
		return "", fmt.Errorf("parsing state: %w", err)
	}
	return vs.VersionPin, nil
}

// SaveVersionPin writes the version pin, preserving existing state.
func (s *Store) SaveVersionPin(wizard, pin string) error {
	path := s.statePath(wizard)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	var vs VersionedState
	existing, err := os.ReadFile(path)
	if err == nil {
		_ = yaml.Unmarshal(existing, &vs)
	}
	vs.VersionPin = pin

	data, err := yaml.Marshal(&vs)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	return nil
}

// --- Presets ---

func (s *Store) presetsDir(wizard string) string {
	return filepath.Join(s.configDir, "presets", wizard)
}

func (s *Store) presetPath(wizard, name string) string {
	return filepath.Join(s.presetsDir(wizard), name+".yml")
}

// ListPresets returns names of all presets for a wizard.
func (s *Store) ListPresets(wizard string) ([]string, error) {
	dir := s.presetsDir(wizard)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading presets directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".yml"))
	}
	return names, nil
}

// LoadPreset reads a named preset, returning a map of option name → value.
func (s *Store) LoadPreset(wizard, name string) (map[string]any, error) {
	data, err := os.ReadFile(s.presetPath(wizard, name))
	if err != nil {
		return nil, fmt.Errorf("reading preset %q: %w", name, err)
	}
	var values map[string]any
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parsing preset %q: %w", name, err)
	}
	return values, nil
}

// SavePreset writes a named preset.
func (s *Store) SavePreset(wizard, name string, values map[string]any) error {
	dir := s.presetsDir(wizard)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating presets directory: %w", err)
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshaling preset %q: %w", name, err)
	}
	if err := os.WriteFile(s.presetPath(wizard, name), data, 0o644); err != nil {
		return fmt.Errorf("writing preset %q: %w", name, err)
	}
	return nil
}

// DeletePreset removes a named preset.
func (s *Store) DeletePreset(wizard, name string) error {
	path := s.presetPath(wizard, name)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting preset %q: %w", name, err)
	}
	return nil
}
