package ui

import "testing"

func TestPlural(t *testing.T) {
	tests := []struct {
		name  string
		n     int
		forms []string
		want  string
	}{
		{"zero_regular", 0, []string{"option"}, "0 options"},
		{"one_regular", 1, []string{"option"}, "1 option"},
		{"many_regular", 5, []string{"option"}, "5 options"},
		{"zero_irregular", 0, []string{"entry", "entries"}, "0 entries"},
		{"one_irregular", 1, []string{"entry", "entries"}, "1 entry"},
		{"many_irregular", 3, []string{"entry", "entries"}, "3 entries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Plural(tt.n, tt.forms...)
			if got != tt.want {
				t.Errorf("Plural(%d, %v) = %q, want %q", tt.n, tt.forms, got, tt.want)
			}
		})
	}
}
