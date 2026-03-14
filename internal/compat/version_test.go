package compat

import (
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestVersionMatchesConstraint(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		constraint string
		want       bool
	}{
		// Basic operators.
		{"gte_match", "1.0.0", ">= 1.0.0", true},
		{"gte_no_match", "0.9.0", ">= 1.0.0", false},
		{"lt_match", "0.9.0", "< 1.0.0", true},
		{"lt_no_match", "1.0.0", "< 1.0.0", false},
		{"eq_match", "1.0.0", "= 1.0.0", true},
		{"eq_no_match", "1.0.1", "= 1.0.0", false},
		{"ne_match", "1.0.1", "!= 1.0.0", true},
		{"ne_no_match", "1.0.0", "!= 1.0.0", false},

		// Comma-separated AND.
		{"range_match", "1.5.0", ">= 1.0.0, < 2.0.0", true},
		{"range_no_match", "2.1.0", ">= 1.0.0, < 2.0.0", false},

		// Tilde (~) — patch-level range.
		{"tilde_match", "1.2.5", "~1.2.3", true},
		{"tilde_no_match", "1.3.0", "~1.2.3", false},

		// Caret (^) — major-level range.
		{"caret_match", "1.9.0", "^1.2.3", true},
		{"caret_no_match", "2.0.0", "^1.2.3", false},

		// Wildcards.
		{"wildcard_match", "1.2.9", "1.2.x", true},
		{"wildcard_no_match", "1.3.0", "1.2.x", false},

		// OR (||).
		{"or_first_match", "1.5.0", ">= 1.0.0 < 2.0.0 || >= 3.0.0", true},
		{"or_second_match", "3.5.0", ">= 1.0.0 < 2.0.0 || >= 3.0.0", true},
		{"or_no_match", "2.5.0", ">= 1.0.0 < 2.0.0 || >= 3.0.0", false},

		// Two-segment version (coercion).
		{"two_segment", "8.0", ">= 8.0", true},

		// Invalid inputs.
		{"invalid_version", "not-a-version", ">= 1.0.0", false},
		{"invalid_constraint", "1.0.0", ">>> bad", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionMatchesConstraint(tt.version, tt.constraint)
			if got != tt.want {
				t.Errorf("versionMatchesConstraint(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
			}
		})
	}
}

func TestExpandTemplate(t *testing.T) {
	tests := []struct {
		name, template, version, want string
	}{
		{"basic", "rails _{{version}}_ new", "7.1.0", "rails _7.1.0_ new"},
		{"multiple", "{{version}} and {{version}}", "1.0", "1.0 and 1.0"},
		{"empty_version", "rails _{{version}}_ new", "", "rails __ new"},
		{"no_placeholder", "rails new", "7.1.0", "rails new"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandTemplate(tt.template, tt.version)
			if got != tt.want {
				t.Errorf("ExpandTemplate(%q, %q) = %q, want %q", tt.template, tt.version, got, tt.want)
			}
		})
	}
}

func TestParseAvailableVersions(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"comma_basic", "7.2.1, 7.1.0, 7.0.8", []string{"7.2.1", "7.1.0", "7.0.8"}},
		{"comma_whitespace", " 7.2.1 , 7.1.0 ", []string{"7.2.1", "7.1.0"}},
		{"comma_empty_entries", "7.2.1,,7.1.0,", []string{"7.2.1", "7.1.0"}},
		{"comma_duplicates", "7.2.1, 7.1.0, 7.2.1", []string{"7.2.1", "7.1.0"}},
		{"newline_basic", "7.2.1\n7.1.0\n7.0.8", []string{"7.2.1", "7.1.0", "7.0.8"}},
		{"newline_trailing", "7.2.1\n7.1.0\n", []string{"7.2.1", "7.1.0"}},
		{"newline_blank_line", "7.2.1\n\n7.1.0", []string{"7.2.1", "7.1.0"}},
		{"newline_duplicates", "7.2.1\n7.1.0\n7.2.1", []string{"7.2.1", "7.1.0"}},
		{"empty", "", nil},
		{"only_commas", ",,,", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAvailableVersions(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestFetchAvailableVersions(t *testing.T) {
	versions, err := FetchAvailableVersions("printf '7.2.1, 7.1.0'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 2 || versions[0] != "7.2.1" || versions[1] != "7.1.0" {
		t.Errorf("got %v, want [7.2.1, 7.1.0]", versions)
	}
}

func TestOptionHints(t *testing.T) {
	tests := []struct {
		name    string
		options []config.Option
		want    map[string]string
	}{
		{
			"no_versions",
			[]config.Option{
				{Name: "a", Type: config.OptionInput, Label: "A"},
			},
			map[string]string{},
		},
		{
			"single_versioned",
			[]config.Option{
				{Name: "a", Type: config.OptionInput, Label: "A", Versions: ">= 8.0"},
				{Name: "b", Type: config.OptionInput, Label: "B", Versions: ">= 8.0"},
			},
			map[string]string{"a": "v8.0+", "b": "v8.0+"},
		},
		{
			"mixed_versioned_and_unversioned",
			[]config.Option{
				{Name: "a", Type: config.OptionInput, Label: "A"},
				{Name: "b", Type: config.OptionInput, Label: "B", Versions: ">= 8.0"},
				{Name: "c", Type: config.OptionInput, Label: "C", Versions: "< 8.0"},
			},
			map[string]string{"b": "v8.0+", "c": "< v8.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OptionHints(tt.options)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("hints[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestFilterOptions(t *testing.T) {
	opts := []config.Option{
		{Name: "shared", Type: config.OptionInput, Label: "Shared"},
		{Name: "old_only", Type: config.OptionInput, Label: "Old", Versions: ">= 1.0.0, < 2.0.0"},
		{Name: "new_only", Type: config.OptionInput, Label: "New", Versions: ">= 2.0.0"},
		{Name: "newer_only", Type: config.OptionInput, Label: "Newer", Versions: ">= 3.0.0"},
		{Name: "tilde_gated", Type: config.OptionInput, Label: "Tilde", Versions: "~1.2.0"},
		{Name: "caret_gated", Type: config.OptionInput, Label: "Caret", Versions: "^2.0.0"},
	}

	tests := []struct {
		name      string
		version   string
		wantNames []string
	}{
		{"old_version", "1.5.0", []string{"shared", "old_only"}},
		{"new_version", "2.5.0", []string{"shared", "new_only", "caret_gated"}},
		{"newer_version", "3.0.0", []string{"shared", "new_only", "newer_only"}},
		{"no_match_returns_ungated", "0.1.0", []string{"shared"}},
		{"empty_version_returns_all", "",
			[]string{"shared", "old_only", "new_only", "newer_only", "tilde_gated", "caret_gated"}},
		{"tilde_match", "1.2.5", []string{"shared", "old_only", "tilde_gated"}},
		{"caret_match", "2.9.0", []string{"shared", "new_only", "caret_gated"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterOptions(opts, tt.version)
			if len(got) != len(tt.wantNames) {
				t.Fatalf("got %v, want %v", optNames(got), tt.wantNames)
			}
			for i, o := range got {
				if o.Name != tt.wantNames[i] {
					t.Errorf("option[%d].Name = %q, want %q", i, o.Name, tt.wantNames[i])
				}
			}
		})
	}
}

func TestFilterChoices(t *testing.T) {
	choices := config.FlexChoices{
		{Value: "sqlite3", Label: "SQLite"},
		{Value: "postgresql", Label: "PostgreSQL"},
		{Value: "mariadb-mysql", Label: "MariaDB (mysql2)", Versions: ">= 8.0"},
		{Value: "mariadb-trilogy", Label: "MariaDB (trilogy)", Versions: ">= 8.0"},
	}

	tests := []struct {
		name       string
		version    string
		wantValues []string
	}{
		{"empty_version_returns_all", "", []string{"sqlite3", "postgresql", "mariadb-mysql", "mariadb-trilogy"}},
		{"old_version_filters_gated", "7.2.0", []string{"sqlite3", "postgresql"}},
		{"new_version_includes_gated", "8.0.0", []string{"sqlite3", "postgresql", "mariadb-mysql", "mariadb-trilogy"}},
		{"newer_version_includes_gated", "9.0.0", []string{"sqlite3", "postgresql", "mariadb-mysql", "mariadb-trilogy"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterChoices(choices, tt.version)
			if len(got) != len(tt.wantValues) {
				t.Fatalf("got %v, want %v", choiceValues(got), tt.wantValues)
			}
			for i, c := range got {
				if c.Value != tt.wantValues[i] {
					t.Errorf("choice[%d].Value = %q, want %q", i, c.Value, tt.wantValues[i])
				}
			}
		})
	}
}

func optNames(opts []config.Option) []string {
	names := make([]string, len(opts))
	for i, o := range opts {
		names[i] = o.Name
	}
	return names
}

func choiceValues(choices config.FlexChoices) []string {
	values := make([]string, len(choices))
	for i, c := range choices {
		values[i] = c.Value
	}
	return values
}
