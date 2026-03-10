package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testWizardYAML = `name: testwiz
description: A test wizard
command: echo
options:
  - name: greeting
    type: select
    label: Pick greeting
    flag: --greet
    choices:
      - hello
      - world
  - name: verbose
    type: confirm
    label: Verbose?
    flag: --verbose
`

func setupTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wizDir, "testwiz.yml"), []byte(testWizardYAML), 0o644); err != nil {
		t.Fatalf("writing test wizard: %v", err)
	}
	return dir
}

func execCmd(t *testing.T, args ...string) error {
	t.Helper()
	configDir = ""
	cmd := newRootCmd(args)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("executing command: %w", err)
	}
	return nil
}

func TestListCmd(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "list"); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestListCmdEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := execCmd(t, "--config-dir", dir, "list"); err != nil {
		t.Fatalf("list empty: %v", err)
	}
}

func TestValidateCmd(t *testing.T) {
	dir := setupTestConfig(t)
	path := filepath.Join(dir, "wizards", "testwiz.yml")

	if err := execCmd(t, "validate", path); err != nil {
		t.Fatalf("validate valid: %v", err)
	}
}

func TestValidateCmdByName(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "validate", "testwiz"); err != nil {
		t.Fatalf("validate by name: %v", err)
	}
}

func TestValidateCmdAlias(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "v", "testwiz"); err != nil {
		t.Fatalf("validate alias: %v", err)
	}
}

func TestValidateCmdInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yml")
	if err := os.WriteFile(path, []byte("name: \"\"\ncommand: \"\"\noptions: []\n"), 0o644); err != nil {
		t.Fatalf("writing bad wizard: %v", err)
	}

	if err := execCmd(t, "validate", path); err == nil {
		t.Fatal("expected validation error for invalid wizard")
	}
}

func TestValidateCmdMissing(t *testing.T) {
	if err := execCmd(t, "validate", "/nonexistent/path.yml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRemoveCmdForce(t *testing.T) {
	dir := setupTestConfig(t)
	path := filepath.Join(dir, "wizards", "testwiz.yml")
	if err := execCmd(t, "--config-dir", dir, "remove", "--force", "testwiz"); err != nil {
		t.Fatalf("remove --force: %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("expected wizard file to be removed")
	}
}

func TestRemoveCmdMissing(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "remove", "nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent wizard")
	}
}

func TestAliases(t *testing.T) {
	dir := setupTestConfig(t)
	tests := []struct {
		name string
		args []string
	}{
		{"run alias r", []string{"--config-dir", dir, "r", "testwiz", "doctor"}},
		{"list alias l", []string{"--config-dir", dir, "l"}},
		{"list alias ls", []string{"--config-dir", dir, "ls"}},
		{"remove alias rm", []string{"--config-dir", dir, "rm", "nonexistent"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := execCmd(t, tt.args...)
			// rm with nonexistent is expected to fail; others should succeed
			if tt.name == "remove alias rm" {
				if err == nil {
					t.Fatal("expected error for nonexistent wizard")
				}
				return
			}
			if err != nil {
				t.Fatalf("alias %s failed: %v", tt.name, err)
			}
		})
	}
}

func TestRunBareError(t *testing.T) {
	err := execCmd(t, "run")
	if err == nil {
		t.Fatal("expected error for bare run")
	}
	if !strings.Contains(err.Error(), "oz list") {
		t.Fatalf("expected helpful error mentioning 'oz list', got: %v", err)
	}
}

func TestCreateNoEdit(t *testing.T) {
	dir := t.TempDir()
	if err := execCmd(t, "--config-dir", dir, "create", "newwiz", "--no-edit"); err != nil {
		t.Fatalf("create --no-edit: %v", err)
	}
	path := filepath.Join(dir, "wizards", "newwiz.yml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected wizard file to exist: %v", err)
	}
}

func TestDoctorCmd(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "doctor"); err != nil {
		t.Fatalf("doctor: %v", err)
	}
}

func TestShowCmd(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "show"); err != nil {
		t.Fatalf("show: %v", err)
	}
}

func TestDoctorCmdMissingWizard(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "wizards"), 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}
	if err := execCmd(t, "--config-dir", dir, "run", "nonexistent", "doctor"); err == nil {
		t.Fatal("expected error for missing wizard")
	}
}

func TestPresetsBareHelp(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets"); err != nil {
		t.Fatalf("presets bare help: %v", err)
	}
}

func TestPresetsListEmpty(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets", "list"); err != nil {
		t.Fatalf("presets list: %v", err)
	}
}

func TestPinsListEmpty(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "pins", "list"); err != nil {
		t.Fatalf("pins list: %v", err)
	}
}

func TestPinsClear(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "pins", "clear", "--force"); err != nil {
		t.Fatalf("pins clear: %v", err)
	}
}
