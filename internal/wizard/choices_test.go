package wizard

import (
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestParseChoicesOutput_Values(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []config.Choice
	}{
		{
			"simple_values",
			"alpha\nbeta\ngamma\n",
			[]config.Choice{
				{Value: "alpha", Label: "alpha"},
				{Value: "beta", Label: "beta"},
				{Value: "gamma", Label: "gamma"},
			},
		},
		{
			"empty_lines_skipped",
			"\nalpha\n\nbeta\n\n",
			[]config.Choice{
				{Value: "alpha", Label: "alpha"},
				{Value: "beta", Label: "beta"},
			},
		},
		{"empty_output", "", nil},
		{"whitespace_only", "  \n  \n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseChoicesOutput(tt.output)
			assertChoices(t, got, tt.want)
		})
	}
}

func TestParseChoicesOutput_Tabs(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []config.Choice
	}{
		{
			"tab_separated_label",
			"mysql\tMySQL 8\npostgres\tPostgreSQL\n",
			[]config.Choice{
				{Value: "mysql", Label: "MySQL 8"},
				{Value: "postgres", Label: "PostgreSQL"},
			},
		},
		{
			"tab_separated_description",
			"mysql\tMySQL 8\tMost popular\n",
			[]config.Choice{
				{Value: "mysql", Label: "MySQL 8", Description: "Most popular"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseChoicesOutput(tt.output)
			assertChoices(t, got, tt.want)
		})
	}
}

func TestInterpolateCommand(t *testing.T) {
	answers := config.Values{"profile": config.StringVal("default"), "region": config.StringVal("us-east-1")}

	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{"no_interpolation", "ls *.txt", "ls *.txt"},
		{"single", "cmd --profile={{profile}}", "cmd --profile='default'"},
		{"multiple",
			"cmd --profile={{profile}} --region={{region}}",
			"cmd --profile='default' --region='us-east-1'"},
		{"missing_answer", "cmd --x={{unknown}}", "cmd --x={{unknown}}"},
		{"dot_syntax_preserved",
			"docker images --format '{{.Names}}'",
			"docker images --format '{{.Names}}'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interpolateCommand(tt.cmd, answers)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"simple", "'simple'"},
		{"it's", `'it'\''s'`},
		{"", "''"},
		{"a b", "'a b'"},
		{"$HOME", "'$HOME'"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := shellEscape(tt.input); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveChoices(t *testing.T) {
	t.Run("printf_command", func(t *testing.T) {
		choices, err := ResolveChoices("printf 'alpha\\nbeta\\n'", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(choices) != 2 {
			t.Fatalf("got %d choices, want 2", len(choices))
		}
		if choices[0].Value != "alpha" || choices[1].Value != "beta" {
			t.Errorf("got %+v", choices)
		}
	})

	t.Run("with_interpolation", func(t *testing.T) {
		answers := config.Values{"name": config.StringVal("world")}
		choices, err := ResolveChoices("echo {{name}}", answers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(choices) != 1 || choices[0].Value != "world" {
			t.Errorf("got %+v", choices)
		}
	})

	t.Run("failing_command", func(t *testing.T) {
		_, err := ResolveChoices("false", nil)
		if err == nil {
			t.Fatal("expected error for failing command")
		}
	})
}

func assertChoices(t *testing.T, got, want []config.Choice) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d choices, want %d", len(got), len(want))
	}
	for i, c := range got {
		if c != want[i] {
			t.Errorf("[%d] got %+v, want %+v", i, c, want[i])
		}
	}
}
