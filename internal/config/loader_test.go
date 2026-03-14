package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWizard(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		yaml := []byte(`
name: test
command: echo
options:
  - name: opt
    type: input
    label: Opt
    flag: --opt
`)
		w, err := ParseWizard(yaml)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.Name != "test" {
			t.Errorf("Name = %q, want test", w.Name)
		}
		if w.Command != "echo" {
			t.Errorf("Command = %q, want echo", w.Command)
		}
		if len(w.Options) != 1 {
			t.Fatalf("got %d options, want 1", len(w.Options))
		}
		if w.Options[0].Flag != "--opt" {
			t.Errorf("Flag = %q, want --opt", w.Options[0].Flag)
		}
	})

	t.Run("invalid_yaml", func(t *testing.T) {
		_, err := ParseWizard([]byte("\t- :\n\t\t- invalid"))
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})

	t.Run("minimal", func(t *testing.T) {
		w, err := ParseWizard([]byte("name: x\ncommand: y"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.Name != "x" || w.Command != "y" {
			t.Errorf("got Name=%q Command=%q", w.Name, w.Command)
		}
		if len(w.Options) != 0 {
			t.Errorf("expected 0 options, got %d", len(w.Options))
		}
	})
}

func TestDefaultConfigDir(t *testing.T) {
	t.Run("with_env", func(t *testing.T) {
		t.Setenv("OZ_CONFIG_DIR", "/custom/path")
		got := DefaultConfigDir()
		if got != "/custom/path" {
			t.Errorf("DefaultConfigDir() = %q, want /custom/path", got)
		}
	})

	t.Run("without_env", func(t *testing.T) {
		// t.Setenv restores the original value on cleanup, but we need
		// the variable fully unset for this subtest. Setting it first
		// ensures cleanup restores whatever was there before.
		t.Setenv("OZ_CONFIG_DIR", "placeholder")
		if err := os.Unsetenv("OZ_CONFIG_DIR"); err != nil {
			t.Fatalf("unsetting OZ_CONFIG_DIR: %v", err)
		}
		got := DefaultConfigDir()
		if !strings.HasSuffix(got, "/oz") {
			t.Errorf("DefaultConfigDir() = %q, want suffix /oz", got)
		}
	})
}

func TestWizardsDir(t *testing.T) {
	got := WizardsDir("/base")
	want := filepath.Join("/base", "wizards")
	if got != want {
		t.Errorf("WizardsDir(/base) = %q, want %q", got, want)
	}
}

func TestWizardPath(t *testing.T) {
	got := WizardPath("/base", "myapp")
	want := filepath.Join("/base", "wizards", "myapp.yml")
	if got != want {
		t.Errorf("WizardPath(/base, myapp) = %q, want %q", got, want)
	}
}

func TestLoadWizard(t *testing.T) {
	t.Run("valid_file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.yml")
		data := []byte("name: loaded\ncommand: echo hello")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("writing temp file: %v", err)
		}

		w, err := LoadWizard(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.Name != "loaded" {
			t.Errorf("Name = %q, want loaded", w.Name)
		}
		if w.Command != "echo hello" {
			t.Errorf("Command = %q, want %q", w.Command, "echo hello")
		}
	})

	t.Run("missing_file", func(t *testing.T) {
		_, err := LoadWizard("/nonexistent/path/wizard.yml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func TestFindWizard(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		dir := t.TempDir()
		wizDir := filepath.Join(dir, "wizards")
		if err := os.MkdirAll(wizDir, 0o755); err != nil {
			t.Fatalf("creating wizards dir: %v", err)
		}
		data := []byte("name: found\ncommand: ls")
		if err := os.WriteFile(filepath.Join(wizDir, "mywiz.yml"), data, 0o644); err != nil {
			t.Fatalf("writing wizard file: %v", err)
		}

		w, err := FindWizard(dir, "mywiz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w.Name != "found" {
			t.Errorf("Name = %q, want found", w.Name)
		}
	})

	t.Run("missing", func(t *testing.T) {
		dir := t.TempDir()
		_, err := FindWizard(dir, "nope")
		if err == nil {
			t.Fatal("expected error for missing wizard")
		}
	})
}

func TestListWizards(t *testing.T) {
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}
	files := map[string]string{
		"alpha.yml": "name: alpha\ncommand: a",
		"beta.yml":  "name: beta\ncommand: b",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(wizDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	wizards, err := ListWizards(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wizards) != 2 {
		t.Fatalf("got %d wizards, want 2", len(wizards))
	}

	names := map[string]bool{}
	for _, w := range wizards {
		names[w.Name] = true
	}
	for _, want := range []string{"alpha", "beta"} {
		if !names[want] {
			t.Errorf("missing wizard %q", want)
		}
	}
}

func TestListWizardsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}

	wizards, err := ListWizards(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wizards != nil {
		t.Errorf("got %v, want nil", wizards)
	}
}

func TestListWizardsMissingDir(t *testing.T) {
	dir := t.TempDir()
	wizards, err := ListWizards(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wizards != nil {
		t.Errorf("got %v, want nil", wizards)
	}
}

func TestListWizardsSkipsNonYAML(t *testing.T) {
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}
	// Write a valid YAML wizard.
	if err := os.WriteFile(
		filepath.Join(wizDir, "good.yml"),
		[]byte("name: good\ncommand: g"),
		0o644,
	); err != nil {
		t.Fatalf("writing good.yml: %v", err)
	}
	// Write a .txt file that should be skipped.
	if err := os.WriteFile(
		filepath.Join(wizDir, "ignore.txt"),
		[]byte("not a wizard"),
		0o644,
	); err != nil {
		t.Fatalf("writing ignore.txt: %v", err)
	}
	// Create a subdirectory that should be skipped.
	if err := os.MkdirAll(filepath.Join(wizDir, "subdir"), 0o755); err != nil {
		t.Fatalf("creating subdir: %v", err)
	}

	wizards, err := ListWizards(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wizards) != 1 {
		t.Fatalf("got %d wizards, want 1", len(wizards))
	}
	if wizards[0].Name != "good" {
		t.Errorf("Name = %q, want good", wizards[0].Name)
	}
}
