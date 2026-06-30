// Package wizardtest is a hermetic snapshot harness for wizard configs: it
// builds the command a wizard's fixture answers produce and checks it against a
// golden file, without detecting tool versions or running shell commands.
package wizardtest

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/svyatov/oz/internal/config"
)

// goldenExt is the extension for expected-command files.
const goldenExt = ".golden"

// Fixture is one golden test case for a wizard: a pinned tool version and the
// answer set that should build the command stored in the sibling golden file.
type Fixture struct {
	Name       string        // case name (input file stem).
	Version    string        // pinned tool version for filtering ("" = no filtering).
	Answers    config.Values // option name → answer value.
	GoldenPath string        // path to the sibling <case>.golden file.
}

// fixtureInput is the on-disk shape of a <case>.yml file. Version is a pointer so
// a missing key (nil) is distinguishable from an explicit empty value.
type fixtureInput struct {
	Version *string       `yaml:"version"`
	Answers config.Values `yaml:"answers"`
}

// LoadFixture parses a single <case>.yml input file into a Fixture.
func LoadFixture(path string) (*Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading fixture %s: %w", path, err)
	}

	var in fixtureInput
	if err := yaml.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("parsing fixture %s: %w", path, err)
	}
	if in.Version == nil {
		return nil, fmt.Errorf("fixture %s: missing required field \"version\"", path)
	}

	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return &Fixture{
		Name:       stem,
		Version:    *in.Version,
		Answers:    in.Answers,
		GoldenPath: strings.TrimSuffix(path, filepath.Ext(path)) + goldenExt,
	}, nil
}

// LoadFixtures returns all <case>.yml fixtures in dir, sorted by name.
// A missing directory yields no fixtures and no error; callers enforce presence.
func LoadFixtures(dir string) ([]*Fixture, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading fixtures dir %s: %w", dir, err)
	}

	var fixtures []*Fixture
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		f, err := LoadFixture(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		fixtures = append(fixtures, f)
	}
	slices.SortFunc(fixtures, func(a, b *Fixture) int {
		return strings.Compare(a.Name, b.Name)
	})
	return fixtures, nil
}

// ReadGolden returns the expected command from a golden file, trailing newline stripped.
func ReadGolden(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading golden %s: %w", path, err)
	}
	return strings.TrimRight(string(data), "\n"), nil
}

// WriteGolden writes the expected command to a golden file with a trailing newline.
func WriteGolden(path, cmd string) error {
	if err := os.WriteFile(path, []byte(cmd+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing golden %s: %w", path, err)
	}
	return nil
}
