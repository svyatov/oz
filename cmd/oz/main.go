package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/registry"
	"github.com/svyatov/oz/internal/store"
	"github.com/svyatov/oz/internal/ui"
)

var version = "dev"

var configDir string

func main() {
	root := newRootCmd(os.Args[1:])

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}

		os.Exit(1)
	}
}

func newRootCmd(args []string) *cobra.Command {
	root := &cobra.Command{
		Use:   "oz",
		Short: "Config-driven CLI wizard framework",
		Long: `Config-driven CLI wizard framework.

Respects NO_COLOR environment variable to disable colored output.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}
	root.PersistentFlags().StringVar(&configDir, "config-dir", config.DefaultConfigDir(), "config directory")

	run := runCmd()
	root.AddCommand(run)
	root.AddCommand(listCmd())
	root.AddCommand(validateCmd())
	root.AddCommand(editCmd())
	root.AddCommand(removeCmd())
	root.AddCommand(createCmd())
	root.AddCommand(addCmd())
	root.AddCommand(updateCmd())
	root.AddCommand(generateCmd())

	if name := detectWizardName(args); name != "" {
		run.AddCommand(wizardCmd(name))
	}

	return root
}

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run <wizard>",
		Aliases: []string{"r"},
		Short:   "Run a wizard",
		Long: `Launch an interactive wizard that walks through options step by step
and builds a CLI command from your answers.

Wizard subcommands (use after wizard name):
  doctor     Check tool installation and detected version
  show       Show all options with descriptions
  pins       Manage pinned options
  presets    Manage presets`,
		Example: "  oz run myapp\n  oz run myapp --dry-run\n  oz run myapp -p fast",
		ValidArgsFunction: completeWizardNames,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("specify a wizard name — run \"oz list\" to see available wizards")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolP("dry-run", "n", false, "print command without executing")
	cmd.PersistentFlags().StringP("preset", "p", "", "run with saved preset (non-interactive, executes immediately)")

	return cmd
}

// detectWizardName finds the wizard name argument after "run" in os.Args.
// During shell completion the last arg is the word being typed, so we skip it
// to avoid registering partial input (e.g. "cre") as a subcommand.
func detectWizardName(args []string) string {
	completion := false
	foundRun := false
	nameIdx := -1
	name := ""
	for i, a := range args {
		if a == "__complete" || a == "__completeNoDesc" {
			completion = true
			continue
		}
		if a == "" || strings.HasPrefix(a, "-") {
			continue
		}
		if !foundRun {
			if a == "run" || a == "r" {
				foundRun = true
			}
			continue
		}
		name = a
		nameIdx = i
		break
	}
	if completion && nameIdx == len(args)-1 {
		return ""
	}
	return name
}

func completeWizardNames(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	wizards, err := config.ListWizards(configDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(wizards))
	for _, w := range wizards {
		if w.Description != "" {
			names = append(names, fmt.Sprintf("%s\t%s", w.Name, w.Description))
		} else {
			names = append(names, w.Name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func editCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "edit <wizard>",
		Aliases: []string{"e"},
		Short:   "Open wizard config in editor",
		Long:    "Open the wizard YAML config in $VISUAL, $EDITOR, or vi.",
		Example: "  oz edit myapp",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := config.WizardPath(configDir, args[0])
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("wizard config not found: %s", path)
			}
			editor, err := findEditor()
			if err != nil {
				return fmt.Errorf("finding editor: %w", err)
			}
			return syscall.Exec(editor, []string{editor, path}, os.Environ())
		},
		ValidArgsFunction: completeWizardNames,
	}
}

func findEditor() (string, error) {
	name := "vi"
	if v := os.Getenv("VISUAL"); v != "" {
		name = v
	} else if v := os.Getenv("EDITOR"); v != "" {
		name = v
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("looking up %q: %w", name, err)
	}
	return path, nil
}

func listCmd() *cobra.Command {
	var remote bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l", "ls"},
		Args:    cobra.NoArgs,
		Short:   "List available wizards",
		Long: `List all wizard configs found in the config directory.

Use --remote to list wizards available in the remote registry.`,
		Example: "  oz list\n  oz list --remote",
		RunE: func(_ *cobra.Command, _ []string) error {
			if remote {
				return listRemote()
			}
			return listLocal()
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "list wizards from the remote registry")

	return cmd
}

func listLocal() error {
	wizards, err := config.ListWizards(configDir)
	if err != nil {
		return fmt.Errorf("listing wizards: %w", err)
	}
	if len(wizards) == 0 {
		ui.InfoMsgf("No wizards found in %s", config.WizardsDir(configDir))
		return nil
	}
	printWizardList(wizards)
	return nil
}

func listRemote() error {
	client := registry.New(registry.DefaultBaseURL())

	idx, err := client.FetchIndex()
	if err != nil {
		return fmt.Errorf("fetching registry: %w", err)
	}

	if len(idx.Wizards) == 0 {
		ui.InfoMsgf("No wizards found in registry")
		return nil
	}

	local, err := config.ListWizards(configDir)
	if err != nil {
		return fmt.Errorf("listing local wizards: %w", err)
	}
	installed := make(map[string]bool, len(local))
	for _, w := range local {
		installed[w.Name] = true
	}

	t := newListTable()
	for _, e := range idx.Wizards {
		tag := ""
		if installed[e.Name] {
			tag = ui.GreenStyle.Render("(installed)")
		}
		t.Row(e.Name, e.Description, tag)
	}
	fmt.Println("\n" + t.Render())
	return nil
}

func printWizardList(wizards []*config.Wizard) {
	t := newListTable()
	for _, w := range wizards {
		t.Row(w.Name, w.Description)
	}
	fmt.Println("\n" + t.Render())
}

func newListTable() *table.Table {
	return table.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		StyleFunc(func(_, col int) lipgloss.Style {
			switch col {
			case 0:
				return lipgloss.NewStyle().Foreground(ui.Accent).PaddingLeft(1)
			default:
				return lipgloss.NewStyle().Foreground(ui.Muted)
			}
		})
}

func removeCmd() *cobra.Command {
	var force, purge bool

	cmd := &cobra.Command{
		Use:     "remove <wizard>",
		Aliases: []string{"rm"},
		Short:   "Remove a wizard config",
		Long: `Delete a wizard YAML config file. Requires confirmation unless --force is set.

Use --purge to also delete saved state, pins, and presets for the wizard.
Without --purge, this data is kept so it can be reused if the wizard is
reinstalled later.`,
		Example: "  oz remove myapp\n  oz rm myapp -f\n  oz rm myapp --purge",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := config.WizardPath(configDir, args[0])
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("wizard config not found: %s", path)
			}
			if !force && !confirmPrompt(fmt.Sprintf("Remove wizard %q?", args[0]), false) {
				ui.InfoMsgf("Cancelled")
				return nil
			}
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("removing wizard: %w", err)
			}
			if purge {
				st := store.New(configDir)
				if err := st.RemoveWizardData(args[0]); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				}
			}
			ui.SuccessMsgf("Wizard %q removed", args[0])
			return nil
		},
		ValidArgsFunction: completeWizardNames,
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&purge, "purge", false, "also remove saved state, pins, and presets")

	return cmd
}

func createCmd() *cobra.Command {
	var noEdit bool

	cmd := &cobra.Command{
		Use:     "create <wizard>",
		Aliases: []string{"c", "new"},
		Short:   "Create a new wizard config from template",
		Long: `Create a new wizard YAML config from a starter template and open it
in your editor ($VISUAL, $EDITOR, or vi). Use --no-edit to skip the editor.`,
		Example: "  oz create myapp\n  oz create myapp --no-edit",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path := config.WizardPath(configDir, args[0])
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("wizard %q already exists: %s", args[0], path)
			}
			if err := os.MkdirAll(config.WizardsDir(configDir), 0o755); err != nil {
				return fmt.Errorf("creating wizards directory: %w", err)
			}
			if err := os.WriteFile(path, []byte(wizardTemplate(args[0])), 0o644); err != nil {
				return fmt.Errorf("writing wizard config: %w", err)
			}
			ui.SuccessMsgf("Created %s", path)
			if noEdit {
				return nil
			}
			editor, err := findEditor()
			if err != nil {
				return fmt.Errorf("finding editor: %w", err)
			}
			return syscall.Exec(editor, []string{editor, path}, os.Environ())
		},
	}

	cmd.Flags().BoolVar(&noEdit, "no-edit", false, "skip opening editor after creation")

	return cmd
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <wizard|path>",
		Short: "Validate a wizard config file",
		Long:    "Check a wizard config for errors. Accepts a wizard name or a file path.",
		Example: "  oz validate myapp\n  oz validate path/to/config.yml",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeWizardNames,
		RunE: func(_ *cobra.Command, args []string) error {
			path := resolveWizardPath(args[0])
			w, err := config.LoadWizard(path)
			if err != nil {
				return fmt.Errorf("loading wizard: %w", err)
			}
			errs := config.Validate(w)
			if len(errs) > 0 {
				return fmt.Errorf("validation errors:\n%s", config.FormatErrors(errs))
			}
			ui.SuccessMsgf("%s is valid", w.Name)
			return nil
		},
	}
}

func resolveWizardPath(arg string) string {
	if strings.Contains(arg, "/") || strings.HasSuffix(arg, ".yml") || strings.HasSuffix(arg, ".yaml") {
		return arg
	}
	return config.WizardPath(configDir, arg)
}
