package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const fakeToolName = "gentool"

// fakeTool creates a script in a temp dir that prints parseable --help output
// and prepends that dir to PATH so generate can find it.
func fakeTool(t *testing.T) {
	t.Helper()

	dir := t.TempDir()

	helpOutput := "Usage: " + fakeToolName + ` [options]

Options:
  -v, --verbose   Enable verbose output
  -q, --quiet     Suppress output
  -o, --output    Output file path
`

	if runtime.GOOS == "windows" {
		script := filepath.Join(dir, fakeToolName+".bat")
		content := "@echo off\necho " + strings.ReplaceAll(helpOutput, "\n", "\necho ")
		if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
			t.Fatalf("writing fake tool: %v", err)
		}
	} else {
		script := filepath.Join(dir, fakeToolName)
		content := "#!/bin/sh\ncat <<'HELP'\n" + helpOutput + "HELP\n"
		if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
			t.Fatalf("writing fake tool: %v", err)
		}
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestGenerateDefault(t *testing.T) {
	fakeTool(t)
	dir := t.TempDir()

	if err := execCmd(t, "--config-dir", dir, "generate", fakeToolName); err != nil {
		t.Fatalf("generate: %v", err)
	}

	path := filepath.Join(dir, "wizards", "gentool.yml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected wizard file at %s: %v", path, err)
	}
}

func TestGenerateAlreadyExists(t *testing.T) {
	fakeTool(t)
	dir := t.TempDir()

	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}

	dest := filepath.Join(wizDir, "gentool.yml")
	if err := os.WriteFile(dest, []byte("existing"), 0o644); err != nil {
		t.Fatalf("writing existing file: %v", err)
	}

	err := execCmd(t, "--config-dir", dir, "generate", fakeToolName)
	if err == nil {
		t.Fatal("expected error for existing file")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestGenerateForce(t *testing.T) {
	fakeTool(t)
	dir := t.TempDir()

	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}

	dest := filepath.Join(wizDir, "gentool.yml")
	if err := os.WriteFile(dest, []byte("old"), 0o644); err != nil {
		t.Fatalf("writing existing file: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "generate", "--force", fakeToolName); err != nil {
		t.Fatalf("generate --force: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	if string(data) == "old" {
		t.Error("file was not overwritten")
	}
}

func TestGenerateOutput(t *testing.T) {
	fakeTool(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "custom.yml")

	if err := execCmd(t, "--config-dir", dir, "generate", "-o", out, fakeToolName); err != nil {
		t.Fatalf("generate -o: %v", err)
	}

	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected output file at %s: %v", out, err)
	}

	// Wizard dir should not be created when using -o.
	wizDir := filepath.Join(dir, "wizards")
	if _, err := os.Stat(wizDir); err == nil {
		t.Error("wizards dir should not be created when using -o")
	}
}

func TestGenerateName(t *testing.T) {
	fakeTool(t)
	dir := t.TempDir()

	if err := execCmd(t, "--config-dir", dir, "generate", "--name", "my-wizard", fakeToolName); err != nil {
		t.Fatalf("generate --name: %v", err)
	}

	path := filepath.Join(dir, "wizards", "my-wizard.yml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at custom name path: %v", err)
	}
}

func TestGenerateNoArgs(t *testing.T) {
	err := execCmd(t, "generate")
	if err == nil {
		t.Fatal("expected error for no args")
	}
}
