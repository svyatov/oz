package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/store"
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

func TestPresetsBareRequiresTTY(t *testing.T) {
	dir := setupTestConfig(t)
	// Bare "presets" now launches the TUI which requires a TTY.
	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets"); err == nil {
		t.Fatal("expected error running presets TUI without TTY")
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

// --- Pure function tests ---

func TestMajorVer(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"1.2.3", "1.2"},
		{"v1.2.3", "1.2"},
		{"1.2", "1.2"},
		{"1", "1.0"},
		{"not-semver", "not-semver"},
		{"8.0.3-rc1", "8.0"},
		{"abc.def", "abc.def"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := majorVer(tt.input)
			if got != tt.want {
				t.Errorf("majorVer(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFilterActivePins(t *testing.T) {
	opts := func(names ...string) []config.Option {
		out := make([]config.Option, len(names))
		for i, n := range names {
			out[i] = config.Option{Name: n}
		}
		return out
	}

	tests := []struct {
		name    string
		pins    config.Values
		options []config.Option
		want    map[string]bool
	}{
		{
			"all match",
			config.Values{"a": config.StringVal("1"), "b": config.StringVal("2")},
			opts("a", "b"),
			map[string]bool{"a": true, "b": true},
		},
		{
			"partial match",
			config.Values{"a": config.StringVal("1"), "c": config.StringVal("3")},
			opts("a", "b"),
			map[string]bool{"a": true},
		},
		{
			"none match",
			config.Values{"x": config.StringVal("1")},
			opts("a", "b"),
			map[string]bool{},
		},
		{
			"empty pins",
			config.Values{},
			opts("a"),
			map[string]bool{},
		},
		{
			"empty options",
			config.Values{"a": config.StringVal("1")},
			nil,
			map[string]bool{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterActivePins(tt.pins, tt.options)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d pins, want %d", len(got), len(tt.want))
			}
			for k := range tt.want {
				if _, ok := got[k]; !ok {
					t.Errorf("expected key %q in result", k)
				}
			}
		})
	}
}

func TestVersionVerifyCmd(t *testing.T) {
	tests := []struct {
		name    string
		wizard  *config.Wizard
		want    string
	}{
		{
			"nil version",
			&config.Wizard{},
			"",
		},
		{
			"with verify",
			&config.Wizard{Version: &config.VersionControl{
				CustomVersionVerify: "ruby --version",
			}},
			"ruby --version",
		},
		{
			"without verify",
			&config.Wizard{Version: &config.VersionControl{}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionVerifyCmd(tt.wizard)
			if got != tt.want {
				t.Errorf("versionVerifyCmd() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionLabel(t *testing.T) {
	tests := []struct {
		name   string
		wizard *config.Wizard
		want   string
	}{
		{
			"nil version",
			&config.Wizard{},
			"",
		},
		{
			"with label",
			&config.Wizard{Version: &config.VersionControl{Label: "Ruby"}},
			"Ruby",
		},
		{
			"without label",
			&config.Wizard{Version: &config.VersionControl{}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionLabel(tt.wizard)
			if got != tt.want {
				t.Errorf("versionLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- Store-dependent tests ---

func TestReconcilePresets(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)

	if err := st.SavePreset("wiz", "a", config.Values{"x": config.StringVal("old")}); err != nil {
		t.Fatalf("saving preset a: %v", err)
	}
	if err := st.SavePreset("wiz", "b", config.Values{"y": config.StringVal("2")}); err != nil {
		t.Fatalf("saving preset b: %v", err)
	}

	final := map[string]config.Values{
		"a": {"x": config.StringVal("updated")},
		"c": {"z": config.StringVal("new")},
	}

	if err := reconcilePresets(st, "wiz", []string{"a", "b"}, final); err != nil {
		t.Fatalf("reconcilePresets: %v", err)
	}

	// "a" should be updated.
	vals, err := st.LoadPreset("wiz", "a")
	if err != nil {
		t.Fatalf("loading preset a: %v", err)
	}
	if vals["x"].String() != "updated" {
		t.Errorf("preset a/x = %q, want %q", vals["x"].String(), "updated")
	}

	// "c" should be added.
	vals, err = st.LoadPreset("wiz", "c")
	if err != nil {
		t.Fatalf("loading preset c: %v", err)
	}
	if vals["z"].String() != "new" {
		t.Errorf("preset c/z = %q, want %q", vals["z"].String(), "new")
	}

	// "b" should be removed.
	if st.PresetExists("wiz", "b") {
		t.Error("expected preset b to be removed")
	}
}

// --- CLI integration tests ---

func TestPresetsShow(t *testing.T) {
	dir := setupTestConfig(t)
	st := store.New(dir)
	if err := st.SavePreset("testwiz", "mypreset", config.Values{
		"greeting": config.StringVal("hello"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets", "show", "mypreset"); err != nil {
		t.Fatalf("presets show: %v", err)
	}
}

func TestPresetsShowVerbose(t *testing.T) {
	dir := setupTestConfig(t)
	st := store.New(dir)
	if err := st.SavePreset("testwiz", "mypreset", config.Values{
		"greeting": config.StringVal("hello"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets", "show", "mypreset", "-v"); err != nil {
		t.Fatalf("presets show -v: %v", err)
	}
}

func TestPresetsSaveForce(t *testing.T) {
	dir := setupTestConfig(t)
	st := store.New(dir)

	// Save state so there's last-used data.
	state := &store.StateEntry{
		LastUsed: config.Values{"greeting": config.StringVal("world")},
	}
	if err := st.SaveState("testwiz", "", state); err != nil {
		t.Fatalf("saving state: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets", "save", "mypreset", "--force"); err != nil {
		t.Fatalf("presets save --force: %v", err)
	}

	if !st.PresetExists("testwiz", "mypreset") {
		t.Error("expected preset to exist after save")
	}
}

func TestPresetsRemoveForce(t *testing.T) {
	dir := setupTestConfig(t)
	st := store.New(dir)
	if err := st.SavePreset("testwiz", "mypreset", config.Values{
		"greeting": config.StringVal("hello"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets", "remove", "mypreset", "--force"); err != nil {
		t.Fatalf("presets remove --force: %v", err)
	}

	if st.PresetExists("testwiz", "mypreset") {
		t.Error("expected preset to be removed")
	}
}

func TestPinsListWithPins(t *testing.T) {
	dir := setupTestConfig(t)
	st := store.New(dir)
	if err := st.SavePins("testwiz", config.Values{
		"greeting": config.StringVal("hello"),
	}); err != nil {
		t.Fatalf("saving pins: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "pins", "list"); err != nil {
		t.Fatalf("pins list with pins: %v", err)
	}
}

func TestRemovePurge(t *testing.T) {
	dir := setupTestConfig(t)
	st := store.New(dir)

	// Save state and preset.
	state := &store.StateEntry{
		LastUsed: config.Values{"greeting": config.StringVal("world")},
	}
	if err := st.SaveState("testwiz", "", state); err != nil {
		t.Fatalf("saving state: %v", err)
	}
	if err := st.SavePreset("testwiz", "fast", config.Values{
		"greeting": config.StringVal("hello"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "remove", "--force", "--purge", "testwiz"); err != nil {
		t.Fatalf("remove --force --purge: %v", err)
	}

	// Wizard file should be gone.
	wizPath := filepath.Join(dir, "wizards", "testwiz.yml")
	if _, err := os.Stat(wizPath); err == nil {
		t.Error("expected wizard file to be removed")
	}

	// State file should be gone.
	statePath := filepath.Join(dir, "state", "testwiz.yml")
	if _, err := os.Stat(statePath); err == nil {
		t.Error("expected state file to be removed")
	}

	// Preset should be gone.
	if st.PresetExists("testwiz", "fast") {
		t.Error("expected preset to be removed")
	}
}

func TestUpdateAll(t *testing.T) {
	dir := setupTestConfig(t)
	startRegistryServer(t)

	// Install a wizard that exists in the registry.
	dest := filepath.Join(dir, "wizards", "remotewiz.yml")
	if err := os.WriteFile(dest, []byte("name: remotewiz\ncommand: old\noptions: []\n"), 0o644); err != nil {
		t.Fatalf("writing wizard: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "update", "--all"); err != nil {
		t.Fatalf("update --all: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading updated wizard: %v", err)
	}
	if !strings.Contains(string(data), "echo hello") {
		t.Errorf("expected updated content, got: %s", data)
	}
}

func TestUpdateAllArgs(t *testing.T) {
	err := execCmd(t, "update", "--all", "foo")
	if err == nil {
		t.Fatal("expected error for --all with args")
	}
}

func TestUpdateBare(t *testing.T) {
	err := execCmd(t, "update")
	if err == nil {
		t.Fatal("expected error for bare update")
	}
}

func TestFindEditor(t *testing.T) {
	t.Setenv("VISUAL", "/usr/bin/true")
	t.Setenv("EDITOR", "")

	path, err := findEditor()
	if err != nil {
		t.Fatalf("findEditor: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestCompleteWizardNames(t *testing.T) {
	t.Run("with args returns nil", func(t *testing.T) {
		names, _ := completeWizardNames(nil, []string{"already"}, "")
		if names != nil {
			t.Errorf("expected nil, got %v", names)
		}
	})

	t.Run("no args returns wizard names", func(t *testing.T) {
		dir := setupTestConfig(t)
		configDir = dir
		t.Cleanup(func() { configDir = "" })

		names, _ := completeWizardNames(nil, nil, "")
		if len(names) == 0 {
			t.Fatal("expected at least one wizard name")
		}
		found := false
		for _, n := range names {
			if strings.HasPrefix(n, "testwiz") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected testwiz in names, got %v", names)
		}
	})
}

func TestCompletePresetNames(t *testing.T) {
	dir := setupTestConfig(t)
	st := store.New(dir)
	if err := st.SavePreset("testwiz", "fast", config.Values{
		"greeting": config.StringVal("hello"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	configDir = dir
	t.Cleanup(func() { configDir = "" })

	fn := completePresetNames("testwiz")
	names, _ := fn(nil, nil, "")
	if len(names) == 0 {
		t.Fatal("expected at least one preset name")
	}
	if !slices.Contains(names, "fast") {
		t.Errorf("expected 'fast' in names, got %v", names)
	}

	// With args should return nil.
	names, _ = fn(nil, []string{"already"}, "")
	if names != nil {
		t.Errorf("expected nil with args, got %v", names)
	}
}

func TestDetectWizardName(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"basic", []string{"run", "mywiz"}, "mywiz"},
		{"with_flags", []string{"run", "--dry-run", "mywiz"}, "mywiz"},
		{"alias", []string{"r", "mywiz"}, "mywiz"},
		{"with_double_dash", []string{"run", "mywiz", "--", "extra"}, "mywiz"},
		{"flags_and_double_dash", []string{"run", "mywiz", "--dry-run", "--", "foo", "--bar"}, "mywiz"},
		{"no_run", []string{"list"}, ""},
		{"empty", nil, ""},
		{"run_only", []string{"run"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectWizardName(tt.args)
			if got != tt.want {
				t.Errorf("detectWizardName(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestPresetWithExtraArgs(t *testing.T) {
	dir := setupTestConfig(t)
	st := store.New(dir)

	// Save a preset for testwiz.
	if err := st.SavePreset("testwiz", "fast", config.Values{
		"greeting": config.StringVal("hello"),
		"verbose":  config.BoolVal(true),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	// Run with preset + extra args (dry-run to avoid execution).
	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "-p", "fast", "-n"); err != nil {
		t.Fatalf("preset with dry-run: %v", err)
	}
}

const versionWizardYAML = `name: verwiz
description: Version wizard
command: echo
version_control:
  command: "echo 1.2.3"
  pattern: '(\d+\.\d+\.\d+)'
options:
  - name: greeting
    type: select
    label: Pick greeting
    flag: --greet
    choices:
      - hello
`

func setupVersionWizard(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wizDir, "verwiz.yml"), []byte(versionWizardYAML), 0o644); err != nil {
		t.Fatalf("writing version wizard: %v", err)
	}
	return dir
}

func TestDoctorWithVersion(t *testing.T) {
	dir := setupVersionWizard(t)
	if err := execCmd(t, "--config-dir", dir, "run", "verwiz", "doctor"); err != nil {
		t.Fatalf("doctor with version: %v", err)
	}
}

func TestShowForVersion(t *testing.T) {
	dir := setupTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "show", "--for-version", "1.0"); err != nil {
		t.Fatalf("show --for-version: %v", err)
	}
}

// --- Rich wizard for comprehensive print function coverage ---

const richWizardYAML = `name: richwiz
description: A rich test wizard
command: echo
version_control:
  command: "echo 1.2.3"
  pattern: '(\d+\.\d+\.\d+)'
  custom_version_command: "echo v{{version}}"
  custom_version_verify_command: "echo {{version}}"
  available_versions: "1.0, 1.1, 1.2.3"
options:
  - name: greeting
    type: select
    label: Pick greeting
    flag: --greet
    default: hello
    choices:
      - value: hello
        label: hello
        description: say hi
      - world
  - name: verbose
    type: confirm
    label: Verbose?
    flag_true: --verbose
    flag_false: --quiet
  - name: name
    type: input
    label: Your name
    flag: --name
    required: true
    validate:
      pattern: "^[a-z]+$"
      min_length: 2
      max_length: 20
  - name: features
    type: multi_select
    label: Features
    flag: --feature
    separator: ","
    choices:
      - auth
      - api
  - name: advanced
    type: input
    label: Advanced
    flag: --adv
    show_when:
      verbose: true
  - name: hidden
    type: input
    label: Hidden
    flag: --hidden
    hide_when:
      verbose: false
  - name: positional_arg
    type: select
    label: Target
    positional: true
    choices:
      - build
      - test
  - name: dynamic
    type: select
    label: Dynamic
    flag: --dyn
    choices_from: "echo one two three"
`

func setupRichTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wizDir, "richwiz.yml"), []byte(richWizardYAML), 0o644); err != nil {
		t.Fatalf("writing rich wizard: %v", err)
	}
	return dir
}

// 1. printValidateInfo — exercises pattern, min_length, max_length.
func TestShowRichWizard(t *testing.T) {
	dir := setupRichTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "richwiz", "show"); err != nil {
		t.Fatalf("show rich wizard: %v", err)
	}
}

// 2. printDoctorVersionDetails — exercises custom_version_cmd, custom_version_verify, avail_versions.
func TestDoctorRichWizard(t *testing.T) {
	dir := setupRichTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "richwiz", "doctor"); err != nil {
		t.Fatalf("doctor rich wizard: %v", err)
	}
}

// 3. printDoctorVersionDetails with avail_versions_cmd instead of static.
func TestDoctorAvailVersionsCmd(t *testing.T) {
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}
	yaml := `name: avcmdwiz
description: Wizard with avail_versions_cmd
command: echo
version_control:
  command: "echo 2.0.0"
  pattern: '(\d+\.\d+\.\d+)'
  available_versions_command: "echo 1.0 2.0"
options:
  - name: x
    type: input
    label: X
    flag: --x
`
	if err := os.WriteFile(filepath.Join(wizDir, "avcmdwiz.yml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("writing wizard: %v", err)
	}
	if err := execCmd(t, "--config-dir", dir, "run", "avcmdwiz", "doctor"); err != nil {
		t.Fatalf("doctor avail_versions_cmd: %v", err)
	}
}

// 4. printPresetVerbose with unknown key (not in options).
func TestPresetsShowVerboseUnknownKey(t *testing.T) {
	dir := setupRichTestConfig(t)
	st := store.New(dir)
	if err := st.SavePreset("richwiz", "mypreset", config.Values{
		"greeting":    config.StringVal("hello"),
		"unknown_key": config.StringVal("mystery"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "richwiz", "presets", "show", "mypreset", "-v"); err != nil {
		t.Fatalf("presets show -v with unknown key: %v", err)
	}
}

// 5. listPresets with saved presets (non-empty).
func TestPresetsListWithPresets(t *testing.T) {
	dir := setupTestConfig(t)
	st := store.New(dir)
	if err := st.SavePreset("testwiz", "fast", config.Values{
		"greeting": config.StringVal("hello"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets", "list"); err != nil {
		t.Fatalf("presets list with presets: %v", err)
	}
}

func TestPresetsSaveNoState(t *testing.T) {
	dir := setupTestConfig(t)
	err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets", "save", "mypreset", "--force")
	if err == nil {
		t.Fatal("expected error for no last-used state")
	}
	if !strings.Contains(err.Error(), "no last-used values") {
		t.Errorf("expected 'no last-used values' error, got: %v", err)
	}
}

// createCmd when wizard already exists — should error.
func TestCreateAlreadyExists(t *testing.T) {
	dir := setupTestConfig(t)
	err := execCmd(t, "--config-dir", dir, "create", "testwiz", "--no-edit")
	if err == nil {
		t.Fatal("expected error for existing wizard")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

// 16. editCmd when wizard does not exist — should error.
func TestEditMissing(t *testing.T) {
	dir := setupTestConfig(t)
	err := execCmd(t, "--config-dir", dir, "edit", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent wizard")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// 17. presetsShowCmd with verbose — exercise choice description annotation path.
func TestPresetsShowVerboseWithChoiceMatch(t *testing.T) {
	dir := setupRichTestConfig(t)
	st := store.New(dir)
	// "hello" matches a choice with description "say hi".
	if err := st.SavePreset("richwiz", "detailed", config.Values{
		"greeting": config.StringVal("hello"),
		"verbose":  config.BoolVal(true),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "richwiz", "presets", "show", "detailed", "-v"); err != nil {
		t.Fatalf("presets show -v choice match: %v", err)
	}
}

// 18. pinsListCmd with pinned version.
func TestPinsListWithPinnedVersion(t *testing.T) {
	dir := setupRichTestConfig(t)
	st := store.New(dir)
	if err := st.SavePinnedVersion("richwiz", "1.2.3"); err != nil {
		t.Fatalf("saving pinned version: %v", err)
	}
	if err := st.SavePins("richwiz", config.Values{
		"greeting": config.StringVal("hello"),
	}); err != nil {
		t.Fatalf("saving pins: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "richwiz", "pins", "list"); err != nil {
		t.Fatalf("pins list with pinned version: %v", err)
	}
}

// 19. presetsShowCmd non-verbose — simple key=value display.
func TestPresetsShowRich(t *testing.T) {
	dir := setupRichTestConfig(t)
	st := store.New(dir)
	if err := st.SavePreset("richwiz", "basic", config.Values{
		"greeting": config.StringVal("world"),
		"name":     config.StringVal("alice"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "richwiz", "presets", "show", "basic"); err != nil {
		t.Fatalf("presets show rich: %v", err)
	}
}

// 20. presetsShowCmd missing preset — should error.
func TestPresetsShowMissing(t *testing.T) {
	dir := setupTestConfig(t)
	err := execCmd(t, "--config-dir", dir, "run", "testwiz", "presets", "show", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing preset")
	}
}

// 21. show with rich wizard and --for-version to exercise effective command display.
func TestShowRichForVersion(t *testing.T) {
	dir := setupRichTestConfig(t)
	if err := execCmd(t, "--config-dir", dir, "run", "richwiz", "show", "--for-version", "1.2.3"); err != nil {
		t.Fatalf("show rich --for-version: %v", err)
	}
}

// 22. doctor with version detection failure (bad command).
func TestDoctorVersionDetectFail(t *testing.T) {
	dir := t.TempDir()
	wizDir := filepath.Join(dir, "wizards")
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("creating wizards dir: %v", err)
	}
	yaml := `name: badver
description: Bad version detection
command: echo
version_control:
  command: "false"
  pattern: '(\d+)'
options:
  - name: x
    type: input
    label: X
    flag: --x
`
	if err := os.WriteFile(filepath.Join(wizDir, "badver.yml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("writing wizard: %v", err)
	}
	// Doctor should still succeed (prints warning, not error).
	if err := execCmd(t, "--config-dir", dir, "run", "badver", "doctor"); err != nil {
		t.Fatalf("doctor bad version: %v", err)
	}
}

// 23. presetsListCmd with rich wizard.
func TestPresetsListRich(t *testing.T) {
	dir := setupRichTestConfig(t)
	st := store.New(dir)
	if err := st.SavePreset("richwiz", "one", config.Values{
		"greeting": config.StringVal("hello"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}
	if err := st.SavePreset("richwiz", "two", config.Values{
		"greeting": config.StringVal("world"),
	}); err != nil {
		t.Fatalf("saving preset: %v", err)
	}

	if err := execCmd(t, "--config-dir", dir, "run", "richwiz", "presets", "list"); err != nil {
		t.Fatalf("presets list rich: %v", err)
	}
}

