package compat

import (
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1", "1.0.0", 0},
		{"1.10", "1.9", 1},
		{"1.9", "1.10", -1},
		{"0.1.0", "0.2.0", -1},
		{"3.2.1", "3.2.1", 0},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := compareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMatchSingleConstraint(t *testing.T) {
	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"1.0.0", ">= 1.0.0", true},
		{"0.9.0", ">= 1.0.0", false},
		{"2.0.0", ">= 1.0.0", true},
		{"1.0.0", "<= 1.0.0", true},
		{"1.1.0", "<= 1.0.0", false},
		{"1.1.0", "> 1.0.0", true},
		{"1.0.0", "> 1.0.0", false},
		{"0.9.0", "< 1.0.0", true},
		{"1.0.0", "< 1.0.0", false},
		{"1.0.0", "= 1.0.0", true},
		{"1.0.1", "= 1.0.0", false},
		{"1.0.0", "1.0.0", true},
		{"1.0.1", "1.0.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.version+"_"+tt.constraint, func(t *testing.T) {
			got := matchSingleConstraint(tt.version, tt.constraint)
			if got != tt.want {
				t.Errorf("matchSingleConstraint(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
			}
		})
	}
}

func TestMatchVersionRange(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		constraint string
		want       bool
	}{
		{"comma_separated_match", "1.5.0", ">= 1.0.0, < 2.0.0", true},
		{"comma_separated_no_match", "2.1.0", ">= 1.0.0, < 2.0.0", false},
		{"single_constraint", "1.0.0", ">= 1.0.0", true},
		{"empty_constraint", "1.0.0", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchVersionRange(tt.version, tt.constraint)
			if got != tt.want {
				t.Errorf("matchVersionRange(%q, %q) = %v, want %v", tt.version, tt.constraint, got, tt.want)
			}
		})
	}
}

func TestMatchedRange(t *testing.T) {
	entries := []config.CompatEntry{
		{Versions: ">= 1.0.0, < 2.0.0", Options: []string{"a"}},
		{Versions: ">= 2.0.0", Options: []string{"b"}},
	}

	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"match_first", "1.5.0", ">= 1.0.0, < 2.0.0"},
		{"match_second", "2.1.0", ">= 2.0.0"},
		{"no_match", "0.5.0", ""},
		{"empty_version", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchedRange(entries, tt.version)
			if got != tt.want {
				t.Errorf("MatchedRange(..., %q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestFilterOptions(t *testing.T) {
	opts := []config.Option{
		{Name: "a", Type: "input", Label: "A"},
		{Name: "b", Type: "input", Label: "B"},
		{Name: "c", Type: "input", Label: "C"},
	}
	compat := []config.CompatEntry{
		{Versions: ">= 1.0.0, < 2.0.0", Options: []string{"a", "c"}},
		{Versions: ">= 2.0.0", Options: []string{"b"}},
	}

	tests := []struct {
		name      string
		compat    []config.CompatEntry
		version   string
		wantNames []string
	}{
		{"filters_correctly", compat, "1.5.0", []string{"a", "c"}},
		{"empty_compat_returns_all", nil, "1.5.0", []string{"a", "b", "c"}},
		{"no_match_returns_all", compat, "0.1.0", []string{"a", "b", "c"}},
		{"empty_version_returns_all", compat, "", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterOptions(opts, tt.compat, tt.version)
			if len(got) != len(tt.wantNames) {
				t.Fatalf("got %d options, want %d", len(got), len(tt.wantNames))
			}
			for i, o := range got {
				if o.Name != tt.wantNames[i] {
					t.Errorf("option[%d].Name = %q, want %q", i, o.Name, tt.wantNames[i])
				}
			}
		})
	}
}
