package store

import (
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	s := New(t.TempDir())
	entry := &StateEntry{
		LastUsed: map[string]any{"lang": "go", "verbose": true},
		Pins:     map[string]any{"lang": "go"},
	}

	if err := s.SaveState("wiz", "", entry); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got, err := s.LoadState("wiz", "")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.LastUsed["lang"] != "go" {
		t.Errorf("LastUsed[lang] = %v, want go", got.LastUsed["lang"])
	}
	if got.Pins["lang"] != "go" {
		t.Errorf("Pins[lang] = %v, want go", got.Pins["lang"])
	}
}

func TestStateVersionedRoundTrip(t *testing.T) {
	s := New(t.TempDir())

	e1 := &StateEntry{LastUsed: map[string]any{"a": "1"}}
	e2 := &StateEntry{LastUsed: map[string]any{"a": "2"}}

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
	if got.LastUsed["a"] != "1" {
		t.Errorf("v1.0 LastUsed[a] = %v, want 1", got.LastUsed["a"])
	}

	got, err = s.LoadState("wiz", "2.0")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.LastUsed["a"] != "2" {
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

func TestPresetRoundTrip(t *testing.T) {
	s := New(t.TempDir())
	vals := map[string]any{"lang": "go", "verbose": true}

	if err := s.SavePreset("wiz", "my-preset", vals); err != nil {
		t.Fatalf("SavePreset: %v", err)
	}

	got, err := s.LoadPreset("wiz", "my-preset")
	if err != nil {
		t.Fatalf("LoadPreset: %v", err)
	}
	if got["lang"] != "go" {
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
