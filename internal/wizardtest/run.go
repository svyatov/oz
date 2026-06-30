package wizardtest

import (
	"fmt"

	"github.com/svyatov/oz/internal/command"
	"github.com/svyatov/oz/internal/compat"
	"github.com/svyatov/oz/internal/config"
)

// BuildCommand renders the command a fixture's answers produce for w, exactly as
// a user would get it — but hermetically. Options and choices are filtered by the
// fixture's pinned version (never detected), the command template is expanded
// with that version, and the production builder renders the result. No shell
// runs: dynamic-choice answers are taken as the literal values in the fixture.
//
// BuildCommand mutates w (Command and Options), mirroring the real run path; call
// it with a freshly loaded wizard per case.
func BuildCommand(w *config.Wizard, f *Fixture) string {
	w.Command = w.EffectiveCommand(f.Version)
	w.Options = compat.FilterOptions(w.Options, f.Version)
	for i := range w.Options {
		if len(w.Options[i].Choices) > 0 {
			w.Options[i].Choices = compat.FilterChoices(w.Options[i].Choices, f.Version)
		}
	}
	return command.FormatCommand(command.Build(w, f.Answers))
}

// CaseResult is the outcome of building and checking one fixture case.
type CaseResult struct {
	Err      error  // load/build error, if any.
	Name     string // case name.
	Expected string // golden command (or written command under --update).
	Actual   string // built command.
	Pass     bool
	Updated  bool // golden was rewritten (--update).
}

// WizardResult aggregates a wizard's fixture outcomes.
type WizardResult struct {
	Wizard     string
	Cases      []CaseResult
	NoFixtures bool // true when the wizard ships no fixtures (a gate failure).
}

// OK reports whether the wizard passed: it has fixtures and every case passed.
func (r WizardResult) OK() bool {
	if r.NoFixtures {
		return false
	}
	for _, c := range r.Cases {
		if !c.Pass {
			return false
		}
	}
	return true
}

// TestWizard discovers the fixtures under fixturesDir and builds + checks each
// case for the wizard at wizardPath. With update set, goldens are rewritten
// instead of compared. A wizard with no fixtures yields NoFixtures = true.
func TestWizard(name, wizardPath, fixturesDir string, update bool) WizardResult {
	res := WizardResult{Wizard: name}

	fixtures, err := LoadFixtures(fixturesDir)
	if err != nil {
		res.Cases = append(res.Cases, CaseResult{Err: err})
		return res
	}
	if len(fixtures) == 0 {
		res.NoFixtures = true
		return res
	}
	for _, f := range fixtures {
		res.Cases = append(res.Cases, checkCase(wizardPath, f, update))
	}
	return res
}

// checkCase loads a fresh wizard, builds the fixture, and compares or updates.
func checkCase(wizardPath string, f *Fixture, update bool) CaseResult {
	cr := CaseResult{Name: f.Name}

	w, err := config.LoadWizard(wizardPath)
	if err != nil {
		cr.Err = fmt.Errorf("loading wizard: %w", err)
		return cr
	}
	cr.Actual = BuildCommand(w, f)

	if update {
		if werr := WriteGolden(f.GoldenPath, cr.Actual); werr != nil {
			cr.Err = werr
			return cr
		}
		cr.Pass, cr.Updated, cr.Expected = true, true, cr.Actual
		return cr
	}

	expected, err := ReadGolden(f.GoldenPath)
	if err != nil {
		cr.Err = err
		return cr
	}
	cr.Expected = expected
	cr.Pass = cr.Actual == expected
	return cr
}
