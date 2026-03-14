// Package store persists last-used state, pins, and presets as YAML files.
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/svyatov/oz/internal/config"
)

func validateName(name string) error {
	if name == "" {
		return errors.New("invalid name: must not be empty")
	}
	if !filepath.IsLocal(name) {
		return fmt.Errorf("invalid name %q: must not contain path separators or '..'", name)
	}
	return nil
}

// Store manages state and preset files for wizards.
type Store struct {
	configDir string
}

// New creates a Store rooted at the given config directory.
func New(configDir string) *Store {
	return &Store{configDir: configDir}
}

// --- State (last-used + pins) ---

// VersionedState is used when version_control is configured.
type VersionedState struct {
	Versions      map[string]*StateEntry `yaml:"versions,omitempty"`
	Pins          config.Values          `yaml:"pins,omitempty"`
	PinnedVersion string                 `yaml:"pinned_version,omitempty"`
}

// StateEntry holds last-used values and pins for a single version (or global).
type StateEntry struct {
	LastUsed config.Values `yaml:"last_used,omitempty"`
	Pins     config.Values `yaml:"pins,omitempty"`
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

func modifyVersionedState(path string, fn func(*VersionedState)) error {
	var vs VersionedState
	existing, readErr := os.ReadFile(path)
	if readErr == nil {
		if err := yaml.Unmarshal(existing, &vs); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: corrupt state file, starting fresh: %v\n", err)
		}
	}
	fn(&vs)
	data, err := yaml.Marshal(&vs)
	if err != nil {
		return fmt.Errorf("marshaling versioned state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing versioned state: %w", err)
	}
	return nil
}

func (s *Store) saveVersionedState(path, version string, entry *StateEntry) error {
	return modifyVersionedState(path, func(vs *VersionedState) {
		if vs.Versions == nil {
			vs.Versions = make(map[string]*StateEntry)
		}
		vs.Versions[version] = entry
	})
}

// LoadPins reads the version-independent pins for a versioned wizard.
func (s *Store) LoadPins(wizard string) (config.Values, error) {
	data, err := os.ReadFile(s.statePath(wizard))
	if os.IsNotExist(err) {
		return make(config.Values), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state: %w", err)
	}
	var vs VersionedState
	if err := yaml.Unmarshal(data, &vs); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	if vs.Pins == nil {
		return make(config.Values), nil
	}
	return vs.Pins, nil
}

// SavePins writes the version-independent pins, preserving other state.
func (s *Store) SavePins(wizard string, pins config.Values) error {
	path := s.statePath(wizard)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}
	return modifyVersionedState(path, func(vs *VersionedState) {
		vs.Pins = pins
	})
}

// LoadPinnedVersion reads the pinned version for a versioned wizard.
func (s *Store) LoadPinnedVersion(wizard string) (string, error) {
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
	return vs.PinnedVersion, nil
}

// SavePinnedVersion writes the pinned version, preserving other state.
func (s *Store) SavePinnedVersion(wizard, version string) error {
	path := s.statePath(wizard)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}
	return modifyVersionedState(path, func(vs *VersionedState) {
		vs.PinnedVersion = version
	})
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
func (s *Store) LoadPreset(wizard, name string) (config.Values, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.presetPath(wizard, name))
	if err != nil {
		return nil, fmt.Errorf("reading preset %q: %w", name, err)
	}
	var values config.Values
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parsing preset %q: %w", name, err)
	}
	return values, nil
}

// SavePreset writes a named preset.
func (s *Store) SavePreset(wizard, name string, values config.Values) error {
	if err := validateName(name); err != nil {
		return err
	}
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

// RenamePreset renames a preset by copying its values to the new name and removing the old one.
func (s *Store) RenamePreset(wizard, oldName, newName string) error {
	if err := validateName(oldName); err != nil {
		return err
	}
	if err := validateName(newName); err != nil {
		return err
	}
	values, err := s.LoadPreset(wizard, oldName)
	if err != nil {
		return fmt.Errorf("loading preset %q for rename: %w", oldName, err)
	}
	if err := s.SavePreset(wizard, newName, values); err != nil {
		return err
	}
	return s.RemovePreset(wizard, oldName)
}

// RemovePreset removes a named preset.
func (s *Store) RemovePreset(wizard, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	path := s.presetPath(wizard, name)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing preset %q: %w", name, err)
	}
	return nil
}
