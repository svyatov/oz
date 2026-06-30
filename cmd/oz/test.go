package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
	"github.com/svyatov/oz/internal/wizardtest"
)

func testCmd() *cobra.Command {
	var update bool

	cmd := &cobra.Command{
		Use:   "test [wizard]",
		Short: "Run wizard fixtures and check built commands",
		Long: `Build each wizard's golden fixtures hermetically and check the result
against its expected command. With no argument, every wizard is tested.

Fixtures live in <wizards>/testdata/<wizard>/<case>.yml with a sibling
<case>.golden. Execution is hermetic: the fixture's pinned version drives
option filtering (no detection) and dynamic-choice answers are taken as
literal values (no shell runs). A wizard with no fixtures fails.

Use --update to (re)write goldens from the current build output.`,
		Example: "  oz test\n  oz test rails-new\n  oz test rails-new --update",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeWizardNames,
		RunE: func(_ *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return runTest(name, update)
		},
	}

	cmd.Flags().BoolVar(&update, "update", false, "rewrite golden files from current output")

	return cmd
}

func runTest(name string, update bool) error {
	dir := resolveTestDir(configDir)

	names, err := testTargets(dir, name)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("no wizards found in %s", dir)
	}

	fmt.Println()
	failed := 0
	for _, n := range names {
		wizardPath := filepath.Join(dir, n+".yml")
		fixturesDir := filepath.Join(dir, "testdata", n)
		r := wizardtest.TestWizard(n, wizardPath, fixturesDir, update)
		printWizardResult(r, update)
		if !r.OK() {
			failed++
		}
	}
	fmt.Println()

	if failed > 0 {
		return fmt.Errorf("%d of %d wizard(s) failed", failed, len(names))
	}
	return nil
}

// testTargets returns the wizard names to test: one named wizard, or all wizards in dir.
func testTargets(dir, name string) ([]string, error) {
	if name != "" {
		if _, err := os.Stat(filepath.Join(dir, name+".yml")); err != nil {
			return nil, fmt.Errorf("wizard %q not found in %s", name, dir)
		}
		return []string{name}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading wizards dir %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".yml"))
	}
	return names, nil
}

// resolveTestDir locates the directory holding wizard YAMLs. Normally that is the
// "wizards" subdir of the config dir (as every other command resolves it). As a
// convenience for pointing the gate straight at a wizards directory (e.g.
// --config-dir ./wizards in CI), configDir is used directly when it holds wizard
// YAMLs and its "wizards" subdir does not.
func resolveTestDir(configDir string) string {
	sub := config.WizardsDir(configDir)
	if dirHasYAML(sub) {
		return sub
	}
	if dirHasYAML(configDir) {
		return configDir
	}
	return sub
}

func dirHasYAML(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yml") {
			return true
		}
	}
	return false
}

func printWizardResult(r wizardtest.WizardResult, update bool) {
	fmt.Printf("  %s\n", ui.TitleStyle.Render(r.Wizard))
	if r.NoFixtures {
		fmt.Printf("    %s no fixtures — every wizard must carry at least one\n",
			ui.WarningStyle.Render("✗"))
		return
	}
	for _, c := range r.Cases {
		printCaseResult(c, update)
	}
}

func printCaseResult(c wizardtest.CaseResult, update bool) {
	switch {
	case c.Err != nil:
		fmt.Printf("    %s %s — %v\n", ui.WarningStyle.Render("✗"), c.Name, c.Err)
	case update && c.Updated:
		fmt.Printf("    %s %s %s\n", ui.GreenStyle.Render("✓"), c.Name,
			ui.MutedStyle.Render("(updated)"))
	case c.Pass:
		fmt.Printf("    %s %s\n", ui.GreenStyle.Render("✓"), c.Name)
	default:
		fmt.Printf("    %s %s\n", ui.WarningStyle.Render("✗"), c.Name)
		fmt.Printf("      %s %s\n", ui.MutedStyle.Render("expected:"), c.Expected)
		fmt.Printf("      %s %s\n", ui.MutedStyle.Render("actual:  "), c.Actual)
	}
}
