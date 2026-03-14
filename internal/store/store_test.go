package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestStateRoundTrip(t *testing.T) {
	s := New(t.TempDir())
	entry := &StateEntry{
		LastUsed: config.Values{"lang": config.StringVal("go"), "verbose": config.BoolVal(true)},
		Pins:     config.Values{"lang": config.StringVal("go")},
	}

	if err := s.SaveState("wiz", "", entry); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got, err := s.LoadState("wiz", "")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.LastUsed["lang"].String() != "go" {
		t.Errorf("LastUsed[lang] = %v, want go", got.LastUsed["lang"])
	}
	if got.Pins["lang"].String() != "go" {
		t.Errorf("Pins[lang] = %v, want go", got.Pins["lang"])
	}
}

func TestStateVersionedRoundTrip(t *testing.T) {
	s := New(t.TempDir())

	e1 := &StateEntry{LastUsed: config.Values{"a": config.StringVal("1")}}
	e2 := &StateEntry{LastUsed: config.Values{"a": config.StringVal("2")}}

	if err := s.SaveState("wiz", "1.0", e1); err != nil {
		t.Fatalf("SaveState v1: %v", err)
	}
	if err := s.SaveState("wiz", "2.0", e2); err != nil {
		t.Fatalf("SaveState v2: %v", err)
	}

	got, err := s.LoadState("wiz", "1.0")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.LastUsed["a"].String() != "1" {
		t.Errorf("v1.0 LastUsed[a] = %v, want 1", got.LastUsed["a"])
	}

	got, err = s.LoadState("wiz", "2.0")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.LastUsed["a"].String() != "2" {
		t.Errorf("v2.0 LastUsed[a] = %v, want 2", got.LastUsed["a"])
	}
}

func TestLoadStateMissingFile(t *testing.T) {
	s := New(t.TempDir())
	entry, err := s.LoadState("nonexistent", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry for missing file")
	}
}

func TestPinsRoundTrip(t *testing.T) {
	s := New(t.TempDir())

	pins, err := s.LoadPins("wiz")
	if err != nil {
		t.Fatalf("LoadPins (missing): %v", err)
	}
	if len(pins) != 0 {
		t.Errorf("expected empty pins, got %v", pins)
	}

	err = s.SavePins("wiz", config.Values{"db": config.StringVal("postgres")})
	if err != nil {
		t.Fatalf("SavePins: %v", err)
	}

	pins, err = s.LoadPins("wiz")
	if err != nil {
		t.Fatalf("LoadPins: %v", err)
	}
	if pins["db"].String() != "postgres" {
		t.Errorf("pins[db] = %v, want postgres", pins["db"])
	}
}

func TestPinnedVersionRoundTrip(t *testing.T) {
	s := New(t.TempDir())

	ver, err := s.LoadPinnedVersion("wiz")
	if err != nil {
		t.Fatalf("LoadPinnedVersion (missing): %v", err)
	}
	if ver != "" {
		t.Errorf("expected empty, got %q", ver)
	}

	err = s.SavePinnedVersion("wiz", "7.1.0")
	if err != nil {
		t.Fatalf("SavePinnedVersion: %v", err)
	}
	ver, err = s.LoadPinnedVersion("wiz")
	if err != nil {
		t.Fatalf("LoadPinnedVersion: %v", err)
	}
	if ver != "7.1.0" {
		t.Errorf("got %q, want 7.1.0", ver)
	}
}

func TestPinsPreservesState(t *testing.T) {
	s := New(t.TempDir())

	entry := &StateEntry{LastUsed: config.Values{"a": config.StringVal("1")}}
	if err := s.SaveState("wiz", "7.0", entry); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	if err := s.SavePins("wiz", config.Values{"db": config.StringVal("mysql")}); err != nil {
		t.Fatalf("SavePins: %v", err)
	}
	if err := s.SavePinnedVersion("wiz", "7.1.0"); err != nil {
		t.Fatalf("SavePinnedVersion: %v", err)
	}

	got, err := s.LoadState("wiz", "7.0")
	if err != nil {
		t.Fatalf("LoadState after pins: %v", err)
	}
	if got.LastUsed["a"].String() != "1" {
		t.Errorf("state was not preserved: %v", got.LastUsed)
	}
}

func TestPinsClear(t *testing.T) {
	s := New(t.TempDir())

	if err := s.SavePins("wiz", config.Values{"db": config.StringVal("mysql")}); err != nil {
		t.Fatalf("SavePins: %v", err)
	}
	if err := s.SavePinnedVersion("wiz", "7.1.0"); err != nil {
		t.Fatalf("SavePinnedVersion: %v", err)
	}

	if err := s.SavePins("wiz", nil); err != nil {
		t.Fatalf("SavePins (clear): %v", err)
	}
	if err := s.SavePinnedVersion("wiz", ""); err != nil {
		t.Fatalf("SavePinnedVersion (clear): %v", err)
	}

	pins, err := s.LoadPins("wiz")
	if err != nil {
		t.Fatalf("LoadPins: %v", err)
	}
	if len(pins) != 0 {
		t.Errorf("expected empty pins after clear, got %v", pins)
	}
	ver, err := s.LoadPinnedVersion("wiz")
	if err != nil {
		t.Fatalf("LoadPinnedVersion: %v", err)
	}
	if ver != "" {
		t.Errorf("expected empty version after clear, got %q", ver)
	}
}

func TestPresetRoundTrip(t *testing.T) {
	s := New(t.TempDir())
	vals := config.Values{"lang": config.StringVal("go"), "verbose": config.BoolVal(true)}

	if err := s.SavePreset("wiz", "my-preset", vals); err != nil {
		t.Fatalf("SavePreset: %v", err)
	}

	got, err := s.LoadPreset("wiz", "my-preset")
	if err != nil {
		t.Fatalf("LoadPreset: %v", err)
	}
	if got["lang"].String() != "go" {
		t.Errorf("lang = %v, want go", got["lang"])
	}

	names, err := s.ListPresets("wiz")
	if err != nil {
		t.Fatalf("ListPresets: %v", err)
	}
	if len(names) != 1 || names[0] != "my-preset" {
		t.Errorf("ListPresets = %v, want [my-preset]", names)
	}

	err = s.RemovePreset("wiz", "my-preset")
	if err != nil {
		t.Fatalf("RemovePreset: %v", err)
	}
	names, err = s.ListPresets("wiz")
	if err != nil {
		t.Fatalf("ListPresets after delete: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 presets after delete, got %v", names)
	}
}

func TestPathTraversalRejected(t *testing.T) {
	s := New(t.TempDir())
	tests := []struct {
		name    string
		preset  string
		wantErr bool
	}{
		{"dot-dot escape", "../escape", true},
		{"nested slash", "foo/bar", true},
		{"backslash", `foo\bar`, false},
		{"double dot in middle", "foo..bar", false},
		{"empty name", "", true},
		{"valid name", "my-preset", false},
		{"valid with dots", "v1.2.3", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.SavePreset("wiz", tt.preset, config.Values{"k": config.StringVal("v")})
			if (err != nil) != tt.wantErr {
				t.Errorf("SavePreset(%q) error = %v, wantErr %v", tt.preset, err, tt.wantErr)
			}
			_, err = s.LoadPreset("wiz", tt.preset)
			if tt.wantErr && err == nil {
				t.Errorf("LoadPreset(%q) expected error", tt.preset)
			}
			err = s.RemovePreset("wiz", tt.preset)
			if tt.wantErr && err == nil {
				t.Errorf("RemovePreset(%q) expected error", tt.preset)
			}
		})
	}
}

func TestRenamePreset(t *testing.T) {
	s := New(t.TempDir())
	vals := config.Values{"lang": config.StringVal("go")}

	if err := s.SavePreset("wiz", "old-name", vals); err != nil {
		t.Fatalf("SavePreset: %v", err)
	}
	if err := s.RenamePreset("wiz", "old-name", "new-name"); err != nil {
		t.Fatalf("RenamePreset: %v", err)
	}

	// Old name should be gone.
	if _, err := s.LoadPreset("wiz", "old-name"); err == nil {
		t.Error("expected error loading old name after rename")
	}

	// New name should have the values.
	got, err := s.LoadPreset("wiz", "new-name")
	if err != nil {
		t.Fatalf("LoadPreset new name: %v", err)
	}
	if got["lang"].String() != "go" {
		t.Errorf("lang = %v, want go", got["lang"])
	}
}

func TestRenamePresetInvalidNames(t *testing.T) {
	s := New(t.TempDir())
	if err := s.RenamePreset("wiz", "", "new"); err == nil {
		t.Error("expected error for empty old name")
	}
	if err := s.RenamePreset("wiz", "old", "../escape"); err == nil {
		t.Error("expected error for invalid new name")
	}
}

func TestListPresetsEmpty(t *testing.T) {
	s := New(t.TempDir())
	names, err := s.ListPresets("wiz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestRemoveWizardData(t *testing.T) {
	t.Run("removes state and presets", func(t *testing.T) {
		s := New(t.TempDir())

		// Save state and a preset.
		entry := &StateEntry{LastUsed: config.Values{"a": config.StringVal("1")}}
		if err := s.SaveState("wiz", "", entry); err != nil {
			t.Fatalf("SaveState: %v", err)
		}
		if err := s.SavePreset("wiz", "p1", config.Values{"b": config.StringVal("2")}); err != nil {
			t.Fatalf("SavePreset: %v", err)
		}

		if err := s.RemoveWizardData("wiz"); err != nil {
			t.Fatalf("RemoveWizardData: %v", err)
		}

		// State file should be gone — LoadState returns empty entry (file missing).
		got, err := s.LoadState("wiz", "")
		if err != nil {
			t.Fatalf("LoadState after remove: %v", err)
		}
		if len(got.LastUsed) != 0 {
			t.Errorf("expected empty state, got %v", got.LastUsed)
		}

		// Presets directory should be gone.
		names, err := s.ListPresets("wiz")
		if err != nil {
			t.Fatalf("ListPresets after remove: %v", err)
		}
		if len(names) != 0 {
			t.Errorf("expected no presets, got %v", names)
		}
	})

	t.Run("no files is not an error", func(t *testing.T) {
		s := New(t.TempDir())
		if err := s.RemoveWizardData("nonexistent"); err != nil {
			t.Errorf("RemoveWizardData on empty store: %v", err)
		}
	})
}

func TestPresetExists(t *testing.T) {
	s := New(t.TempDir())

	if s.PresetExists("wiz", "nope") {
		t.Error("PresetExists returned true for missing preset")
	}

	if err := s.SavePreset("wiz", "yes", config.Values{"k": config.StringVal("v")}); err != nil {
		t.Fatalf("SavePreset: %v", err)
	}

	if !s.PresetExists("wiz", "yes") {
		t.Error("PresetExists returned false for existing preset")
	}
}

func TestLoadStateVersionNilVersionsMap(t *testing.T) {
	// Write a VersionedState file that has pins but no versions map.
	dir := t.TempDir()
	s := New(dir)

	if err := s.SavePins("wiz", config.Values{"db": config.StringVal("pg")}); err != nil {
		t.Fatalf("SavePins: %v", err)
	}

	// LoadState with a version should return empty entry because
	// the Versions map is nil.
	got, err := s.LoadState("wiz", "3.0")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(got.LastUsed) != 0 || len(got.Pins) != 0 {
		t.Errorf("expected empty StateEntry, got LastUsed=%v Pins=%v", got.LastUsed, got.Pins)
	}
}

func TestLoadStateVersionMismatch(t *testing.T) {
	s := New(t.TempDir())

	entry := &StateEntry{LastUsed: config.Values{"x": config.StringVal("y")}}
	if err := s.SaveState("wiz", "1.0", entry); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	// Load a different version that was never saved.
	got, err := s.LoadState("wiz", "2.0")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(got.LastUsed) != 0 || len(got.Pins) != 0 {
		t.Errorf("expected empty StateEntry for missing version, got LastUsed=%v Pins=%v",
			got.LastUsed, got.Pins)
	}
}

func TestListPresetsSkipsDirectories(t *testing.T) {
	s := New(t.TempDir())

	// Save a real preset.
	if err := s.SavePreset("wiz", "real", config.Values{"k": config.StringVal("v")}); err != nil {
		t.Fatalf("SavePreset: %v", err)
	}

	// Create a subdirectory inside the presets dir.
	subdir := filepath.Join(s.presetsDir("wiz"), "not-a-preset")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	// Create a non-.yml file to exercise the suffix filter.
	junk := filepath.Join(s.presetsDir("wiz"), "notes.txt")
	if err := os.WriteFile(junk, []byte("hi"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	names, err := s.ListPresets("wiz")
	if err != nil {
		t.Fatalf("ListPresets: %v", err)
	}
	if len(names) != 1 || names[0] != "real" {
		t.Errorf("ListPresets = %v, want [real]", names)
	}
}

func TestModifyVersionedStateCorruptYAML(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	// Write corrupt YAML to the state file.
	stateDir := filepath.Join(dir, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	statePath := filepath.Join(stateDir, "wiz.yml")
	if err := os.WriteFile(statePath, []byte(":\n\t{[bad yaml"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Capture stderr to verify the warning.
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	// SavePins calls modifyVersionedState internally.
	saveErr := s.SavePins("wiz", config.Values{"db": config.StringVal("pg")})

	_ = w.Close()
	os.Stderr = origStderr

	var buf [4096]byte
	n, _ := r.Read(buf[:])
	stderr := string(buf[:n])
	_ = r.Close()

	if saveErr != nil {
		t.Fatalf("SavePins: %v", saveErr)
	}
	if !strings.Contains(stderr, "Warning: corrupt state file") {
		t.Errorf("expected warning on stderr, got %q", stderr)
	}

	// Verify the pins were saved despite the corrupt file.
	pins, err := s.LoadPins("wiz")
	if err != nil {
		t.Fatalf("LoadPins: %v", err)
	}
	if pins["db"].String() != "pg" {
		t.Errorf("pins[db] = %v, want pg", pins["db"])
	}
}
