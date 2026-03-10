package store

import (
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

	if err := s.SavePins("wiz", config.Values{"db": config.StringVal("postgres")}); err != nil {
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

	if err := s.SavePinnedVersion("wiz", "7.1.0"); err != nil {
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

	if err := s.DeletePreset("wiz", "my-preset"); err != nil {
		t.Fatalf("DeletePreset: %v", err)
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
		{"backslash", `foo\bar`, true},
		{"double dot in middle", "foo..bar", true},
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
			err = s.DeletePreset("wiz", tt.preset)
			if tt.wantErr && err == nil {
				t.Errorf("DeletePreset(%q) expected error", tt.preset)
			}
		})
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
