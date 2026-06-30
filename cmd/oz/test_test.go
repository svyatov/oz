package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestHarness writes the "demo" wizard and one fixture (no golden yet) under
// a fresh config dir, returning the dir. The wizard builds a single --name flag.
func setupTestHarness(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	fixDir := filepath.Join(wizDir, "testdata", "demo")
	if err := os.MkdirAll(fixDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	wizard := `name: demo
command: demo
options:
  - name: app
    type: input
    label: App
    flag: --name
`
	if err := os.WriteFile(filepath.Join(wizDir, "demo.yml"), []byte(wizard), 0o644); err != nil {
		t.Fatalf("writing wizard: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixDir, "basic.yml"),
		[]byte("version: \"\"\nanswers:\n  app: blog\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return dir
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	if cerr := w.Close(); cerr != nil {
		t.Fatalf("closing pipe: %v", cerr)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading pipe: %v", err)
	}
	return string(data)
}

func TestTestCmd_PassAfterUpdate(t *testing.T) {
	dir := setupTestHarness(t)

	// --update generates the golden, then a plain run passes and names the case.
	if err := execCmd(t, "--config-dir", dir, "test", "demo", "--update"); err != nil {
		t.Fatalf("test --update: %v", err)
	}
	out := captureStdout(t, func() {
		if err := execCmd(t, "--config-dir", dir, "test", "demo"); err != nil {
			t.Fatalf("test demo: %v", err)
		}
	})
	if !strings.Contains(out, "basic") {
		t.Errorf("output does not name the case:\n%s", out)
	}
}

func TestTestCmd_AllWizards(t *testing.T) {
	dir := setupTestHarness(t)
	if err := execCmd(t, "--config-dir", dir, "test", "--update"); err != nil {
		t.Fatalf("test --update (all): %v", err)
	}
	if err := execCmd(t, "--config-dir", dir, "test"); err != nil {
		t.Fatalf("test (all): %v", err)
	}
}

func TestTestCmd_FailureDiff(t *testing.T) {
	dir := setupTestHarness(t)
	golden := filepath.Join(dir, "wizards", "testdata", "demo", "basic.golden")
	if err := os.WriteFile(golden, []byte("demo --name=stale\n"), 0o644); err != nil {
		t.Fatalf("writing stale golden: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = execCmd(t, "--config-dir", dir, "test", "demo")
	})
	if runErr == nil {
		t.Fatal("expected non-zero exit on golden mismatch")
	}
	if !strings.Contains(out, "expected:") || !strings.Contains(out, "actual:") {
		t.Errorf("output lacks expected/actual diff:\n%s", out)
	}
}

func TestTestCmd_UpdateFixesStaleGolden(t *testing.T) {
	dir := setupTestHarness(t)
	golden := filepath.Join(dir, "wizards", "testdata", "demo", "basic.golden")
	if err := os.WriteFile(golden, []byte("demo --name=stale\n"), 0o644); err != nil {
		t.Fatalf("writing stale golden: %v", err)
	}
	if err := execCmd(t, "--config-dir", dir, "test", "demo", "--update"); err != nil {
		t.Fatalf("test --update: %v", err)
	}
	if err := execCmd(t, "--config-dir", dir, "test", "demo"); err != nil {
		t.Fatalf("re-run after update: %v", err)
	}
}

// TestTestCmd_NoFixtures covers AE3: a wizard with no fixtures fails the gate.
func TestTestCmd_NoFixtures(t *testing.T) {
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wizDir, "nofix.yml"),
		[]byte("name: nofix\ncommand: demo\n"), 0o644); err != nil {
		t.Fatalf("writing wizard: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = execCmd(t, "--config-dir", dir, "test", "nofix")
	})
	if runErr == nil {
		t.Fatal("expected non-zero exit for wizard with no fixtures")
	}
	if !strings.Contains(out, "no fixtures") {
		t.Errorf("output does not report missing fixtures:\n%s", out)
	}
}

func TestTestCmd_UnknownWizard(t *testing.T) {
	dir := setupTestHarness(t)
	if err := execCmd(t, "--config-dir", dir, "test", "ghost"); err == nil {
		t.Fatal("expected error for unknown wizard")
	}
}
