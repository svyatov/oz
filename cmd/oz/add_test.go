package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const remoteWizardYAML = `name: remotewiz
description: A remote wizard
command: echo hello
options:
  - name: greeting
    type: select
    label: Pick greeting
    flag: --greet
    choices:
      - hello
      - world
`

func startRegistryServer(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yml":
			_, _ = w.Write([]byte("wizards:\n  - name: remotewiz\n    description: A remote wizard\n"))
		case "/wizards/remotewiz.yml":
			_, _ = w.Write([]byte(remoteWizardYAML))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OZ_REGISTRY_URL", srv.URL+"/")
}

func TestAddRemote(t *testing.T) {
	dir := t.TempDir()
	startRegistryServer(t)

	if err := execCmd(t, "--config-dir", dir, "add", "remotewiz"); err != nil {
		t.Fatalf("add remote: %v", err)
	}

	path := filepath.Join(dir, "wizards", "remotewiz.yml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected wizard file to exist: %v", err)
	}
}

func TestAddRemoteAlreadyExists(t *testing.T) {
	dir := setupTestConfig(t)
	startRegistryServer(t)

	// Write a wizard named "remotewiz" to conflict.
	dest := filepath.Join(dir, "wizards", "remotewiz.yml")
	if err := os.WriteFile(dest, []byte(remoteWizardYAML), 0o644); err != nil {
		t.Fatalf("writing existing wizard: %v", err)
	}

	err := execCmd(t, "--config-dir", dir, "add", "remotewiz")
	if err == nil {
		t.Fatal("expected error for existing wizard")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestAddRemoteForce(t *testing.T) {
	dir := setupTestConfig(t)
	startRegistryServer(t)

	// Write a wizard named "remotewiz" to overwrite.
	dest := filepath.Join(dir, "wizards", "remotewiz.yml")
	if err := os.WriteFile(dest, []byte(remoteWizardYAML), 0o644); err != nil {
		t.Fatalf("writing existing wizard: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "add", "--force", "remotewiz"); err != nil {
		t.Fatalf("add --force: %v", err)
	}
}

func TestAddRemoteNotFound(t *testing.T) {
	startRegistryServer(t)
	dir := t.TempDir()

	err := execCmd(t, "--config-dir", dir, "add", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing remote wizard")
	}
}

func TestAddRemoteInvalidYAML(t *testing.T) {
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/wizards/badwiz.yml" {
			_, _ = w.Write([]byte("name: badwiz\ncommand: \"\"\noptions: []\n"))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OZ_REGISTRY_URL", srv.URL+"/")

	err := execCmd(t, "--config-dir", dir, "add", "badwiz")
	if err == nil {
		t.Fatal("expected validation error for invalid wizard")
	}

	// Verify no file was written.
	path := filepath.Join(dir, "wizards", "badwiz.yml")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("expected no file to be written for invalid wizard")
	}
}

func TestAddLocal(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(t.TempDir(), "my-wizard.yml")
	if err := os.WriteFile(src, []byte(remoteWizardYAML), 0o644); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "add", src); err != nil {
		t.Fatalf("add local: %v", err)
	}

	// Installed under the YAML name field, not the filename.
	path := filepath.Join(dir, "wizards", "remotewiz.yml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected wizard file to exist: %v", err)
	}
}

func TestAddLocalMissing(t *testing.T) {
	dir := t.TempDir()

	err := execCmd(t, "--config-dir", dir, "add", "/nonexistent/wizard.yml")
	if err == nil {
		t.Fatal("expected error for missing local file")
	}
}

func TestAddLocalAutoDetect(t *testing.T) {
	tests := []struct {
		arg   string
		local bool
	}{
		{"rails-new", false},
		{"./wizard.yml", true},
		{"path/to/wizard.yml", true},
		{"wizard.yml", true},
		{"wizard.yaml", true},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			got := isLocalPath(tt.arg)
			if got != tt.local {
				t.Errorf("isLocalPath(%q) = %v, want %v", tt.arg, got, tt.local)
			}
		})
	}
}

func TestUpdateRemote(t *testing.T) {
	dir := setupTestConfig(t)
	startRegistryServer(t)

	// First add the wizard.
	dest := filepath.Join(dir, "wizards", "remotewiz.yml")
	if err := os.WriteFile(dest, []byte("name: remotewiz\ncommand: old\noptions: []\n"), 0o644); err != nil {
		t.Fatalf("writing wizard: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "update", "remotewiz"); err != nil {
		t.Fatalf("update: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading updated wizard: %v", err)
	}
	if !strings.Contains(string(data), "echo hello") {
		t.Errorf("expected updated content, got: %s", data)
	}
}

func TestListRemote(t *testing.T) {
	dir := setupTestConfig(t)
	startRegistryServer(t)

	if err := execCmd(t, "--config-dir", dir, "list", "--remote"); err != nil {
		t.Fatalf("list --remote: %v", err)
	}
}
