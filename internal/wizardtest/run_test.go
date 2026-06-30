package wizardtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func mustParse(t *testing.T, yaml string) *config.Wizard {
	t.Helper()
	w, err := config.ParseWizard([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseWizard: %v", err)
	}
	return w
}

// TestBuildCommand_VersionGating covers AE1: options are selected by the pinned
// version, with no detection of the (absent) tool.
func TestBuildCommand_VersionGating(t *testing.T) {
	yaml := `name: demo
command: demo run
options:
  - name: legacy
    type: confirm
    label: Legacy
    flag: --legacy
    versions: "< 8.0"
  - name: modern
    type: confirm
    label: Modern
    flag: --modern
    versions: ">= 8.0"
`
	answers := config.Values{
		"legacy": config.BoolVal(true),
		"modern": config.BoolVal(true),
	}

	old := BuildCommand(mustParse(t, yaml), &Fixture{Version: "7.2.0", Answers: answers})
	if old != "demo run --legacy" {
		t.Errorf("v7.2.0 = %q, want %q", old, "demo run --legacy")
	}

	recent := BuildCommand(mustParse(t, yaml), &Fixture{Version: "8.0.0", Answers: answers})
	if recent != "demo run --modern" {
		t.Errorf("v8.0.0 = %q, want %q", recent, "demo run --modern")
	}
}

// TestBuildCommand_EffectiveCommand checks the version is woven into a custom
// command template, matching what a user pinning that version gets.
func TestBuildCommand_EffectiveCommand(t *testing.T) {
	yaml := `name: demo
command: rails new
version_control:
  custom_version_command: "rails _{{version}}_ new"
options:
  - name: app
    type: input
    label: App
    flag: --name
`
	answers := config.Values{"app": config.StringVal("blog")}
	got := BuildCommand(mustParse(t, yaml), &Fixture{Version: "7.1.0", Answers: answers})
	if got != "rails _7.1.0_ new --name=blog" {
		t.Errorf("got %q, want %q", got, "rails _7.1.0_ new --name=blog")
	}
}

// TestBuildCommand_DynamicChoiceLiteral covers AE2: a choices_from option uses
// the literal fixture answer and the shell command never runs.
func TestBuildCommand_DynamicChoiceLiteral(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	yaml := `name: demo
command: demo
options:
  - name: branch
    type: select
    label: Branch
    flag: --branch
    choices_from: "touch ` + marker + `; echo main"
`
	answers := config.Values{"branch": config.StringVal("feature")}
	got := BuildCommand(mustParse(t, yaml), &Fixture{Version: "", Answers: answers})

	if got != "demo --branch=feature" {
		t.Errorf("got %q, want %q", got, "demo --branch=feature")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("choices_from shell command ran (marker %s exists)", marker)
	}
}

// TestBuildCommand_MultiSelect checks a multi-select renders repeated flags.
func TestBuildCommand_MultiSelect(t *testing.T) {
	yaml := `name: demo
command: demo
options:
  - name: tags
    type: multi_select
    label: Tags
    flag: --tag
    choices: [a, b, c]
`
	answers := config.Values{"tags": config.StringsVal("a", "c")}
	got := BuildCommand(mustParse(t, yaml), &Fixture{Version: "", Answers: answers})
	if got != "demo --tag=a --tag=c" {
		t.Errorf("got %q, want %q", got, "demo --tag=a --tag=c")
	}
}

// TestBuildCommand_Deterministic checks two runs produce identical output.
func TestBuildCommand_Deterministic(t *testing.T) {
	yaml := `name: demo
command: demo
options:
  - name: tags
    type: multi_select
    label: Tags
    flag: --tag
`
	f := &Fixture{Version: "", Answers: config.Values{"tags": config.StringsVal("x", "y", "z")}}
	first := BuildCommand(mustParse(t, yaml), f)
	second := BuildCommand(mustParse(t, yaml), f)
	if first != second {
		t.Errorf("non-deterministic: %q != %q", first, second)
	}
}

func TestTestWizard(t *testing.T) {
	dir := t.TempDir()
	wizardPath := filepath.Join(dir, "demo.yml")
	if err := os.WriteFile(wizardPath, []byte(`name: demo
command: demo
options:
  - name: app
    type: input
    label: App
    flag: --name
`), 0o644); err != nil {
		t.Fatalf("writing wizard: %v", err)
	}

	fixturesDir := filepath.Join(dir, "testdata", "demo")
	if err := os.MkdirAll(fixturesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, fixturesDir, "basic.yml", "version: \"\"\nanswers:\n  app: blog\n")

	// No golden yet → update writes it, then a plain run passes.
	if r := TestWizard("demo", wizardPath, fixturesDir, true); !r.OK() || !r.Cases[0].Updated {
		t.Fatalf("update run: OK=%v case=%+v", r.OK(), r.Cases)
	}
	if r := TestWizard("demo", wizardPath, fixturesDir, false); !r.OK() {
		t.Fatalf("compare run failed: %+v", r.Cases)
	}

	// A stale golden fails with a diff.
	if err := WriteGolden(filepath.Join(fixturesDir, "basic.golden"), "demo --name=stale"); err != nil {
		t.Fatalf("WriteGolden: %v", err)
	}
	r := TestWizard("demo", wizardPath, fixturesDir, false)
	if r.OK() {
		t.Error("expected failure on stale golden")
	}
	if r.Cases[0].Expected == r.Cases[0].Actual {
		t.Error("want Expected != Actual on a golden mismatch")
	}
}

func TestTestWizard_NoFixtures(t *testing.T) {
	dir := t.TempDir()
	wizardPath := filepath.Join(dir, "demo.yml")
	if err := os.WriteFile(wizardPath, []byte("name: demo\ncommand: demo\n"), 0o644); err != nil {
		t.Fatalf("writing wizard: %v", err)
	}
	r := TestWizard("demo", wizardPath, filepath.Join(dir, "testdata", "demo"), false)
	if !r.NoFixtures {
		t.Error("expected NoFixtures = true")
	}
	if r.OK() {
		t.Error("a wizard with no fixtures must not pass")
	}
}
