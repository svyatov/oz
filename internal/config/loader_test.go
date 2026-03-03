package config

import "testing"

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
