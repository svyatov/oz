package wizardtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

func TestLoadFixture_Happy(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "case.yml", `version: "1.2.0"
answers:
  app_name: blog
  api: true
  features:
    - a
    - b
`)

	f, err := LoadFixture(path)
	if err != nil {
		t.Fatalf("LoadFixture: %v", err)
	}

	if f.Name != "case" {
		t.Errorf("Name = %q, want %q", f.Name, "case")
	}
	if f.Version != "1.2.0" {
		t.Errorf("Version = %q, want %q", f.Version, "1.2.0")
	}
	if want := filepath.Join(dir, "case.golden"); f.GoldenPath != want {
		t.Errorf("GoldenPath = %q, want %q", f.GoldenPath, want)
	}

	if got := f.Answers["app_name"]; !got.IsString() || got.String() != "blog" {
		t.Errorf("app_name = %+v, want string blog", got)
	}
	if got := f.Answers["api"]; !got.IsBool() || !got.Bool() {
		t.Errorf("api = %+v, want bool true", got)
	}
	got := f.Answers["features"]
	if !got.IsStrings() || strings.Join(got.Strings(), ",") != "a,b" {
		t.Errorf("features = %+v, want []string{a,b}", got)
	}
}

func TestLoadFixture_EmptyAnswers(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "empty.yml", "version: \"2.0.0\"\n")

	f, err := LoadFixture(path)
	if err != nil {
		t.Fatalf("LoadFixture: %v", err)
	}
	if f.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", f.Version, "2.0.0")
	}
	if len(f.Answers) != 0 {
		t.Errorf("Answers = %v, want empty", f.Answers)
	}
}

func TestLoadFixture_Errors(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name    string
		content string
	}{
		{"malformed", "version: \"1.0.0\"\nanswers: [oops\n"},
		{"missing_version", "answers:\n  app_name: blog\n"},
		{"unparseable_answer", "version: \"1.0.0\"\nanswers:\n  app_name:\n    nested: map\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFile(t, dir, tt.name+".yml", tt.content)
			_, err := LoadFixture(path)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("error %q does not name the file %q", err, path)
			}
		})
	}
}

func TestLoadFixtures_SortedAndSkipsNonYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "b.yml", "version: \"1.0.0\"\n")
	writeFile(t, dir, "a.yml", "version: \"1.0.0\"\n")
	writeFile(t, dir, "a.golden", "ignored\n")
	writeFile(t, dir, "notes.txt", "ignored")

	fixtures, err := LoadFixtures(dir)
	if err != nil {
		t.Fatalf("LoadFixtures: %v", err)
	}
	if len(fixtures) != 2 {
		t.Fatalf("got %d fixtures, want 2", len(fixtures))
	}
	if fixtures[0].Name != "a" || fixtures[1].Name != "b" {
		t.Errorf("order = [%s %s], want [a b]", fixtures[0].Name, fixtures[1].Name)
	}
}

func TestLoadFixtures_MissingDir(t *testing.T) {
	fixtures, err := LoadFixtures(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("LoadFixtures: %v", err)
	}
	if fixtures != nil {
		t.Errorf("got %v, want nil", fixtures)
	}
}

func TestGoldenRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.golden")
	if err := WriteGolden(path, "rails new --name=blog"); err != nil {
		t.Fatalf("WriteGolden: %v", err)
	}
	got, err := ReadGolden(path)
	if err != nil {
		t.Fatalf("ReadGolden: %v", err)
	}
	if got != "rails new --name=blog" {
		t.Errorf("ReadGolden = %q, want %q", got, "rails new --name=blog")
	}
}
