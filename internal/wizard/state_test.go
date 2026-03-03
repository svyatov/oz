package wizard

import (
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestEvalShowWhen(t *testing.T) {
	answers := Answers{"lang": "go", "verbose": true}

	tests := []struct {
		name     string
		showWhen map[string]any
		want     bool
	}{
		{"empty_conditions", nil, true},
		{"all_met", map[string]any{"lang": "go", "verbose": true}, true},
		{"one_unmet", map[string]any{"lang": "rust"}, false},
		{"missing_answer", map[string]any{"missing": "x"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EvalShowWhen(tt.showWhen, answers); got != tt.want {
				t.Errorf("EvalShowWhen(%v) = %v, want %v", tt.showWhen, got, tt.want)
			}
		})
	}
}

func TestValuesMatch(t *testing.T) {
	tests := []struct {
		name           string
		actual, expect any
		want           bool
	}{
		{"string_string", "foo", "foo", true},
		{"int_int", 42, 42, true},
		{"string_int_coerce", "42", 42, true},
		{"mismatch", "a", "b", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := valuesMatch(tt.actual, tt.expect); got != tt.want {
				t.Errorf("valuesMatch(%v, %v) = %v, want %v", tt.actual, tt.expect, got, tt.want)
			}
		})
	}
}

func TestFilterPinned(t *testing.T) {
	opts := []config.Option{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}

	tests := []struct {
		name      string
		pins      map[string]any
		wantNames []string
		wantCount int
	}{
		{"no_pins", map[string]any{}, []string{"a", "b", "c"}, 0},
		{"some_pinned", map[string]any{"b": "val"}, []string{"a", "c"}, 1},
		{"all_pinned", map[string]any{"a": 1, "b": 2, "c": 3}, nil, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, count := FilterPinned(opts, tt.pins)
			if count != tt.wantCount {
				t.Errorf("pinCount = %d, want %d", count, tt.wantCount)
			}
			var names []string
			for _, o := range filtered {
				names = append(names, o.Name)
			}
			if len(names) != len(tt.wantNames) {
				t.Fatalf("got names %v, want %v", names, tt.wantNames)
			}
			for i, n := range names {
				if n != tt.wantNames[i] {
					t.Errorf("[%d] = %q, want %q", i, n, tt.wantNames[i])
				}
			}
		})
	}
}

func TestVisibleSteps(t *testing.T) {
	opts := []config.Option{
		{Name: "a"},
		{Name: "b", ShowWhen: map[string]any{"a": "yes"}},
		{Name: "c"},
	}

	t.Run("all_visible", func(t *testing.T) {
		answers := Answers{"a": "yes"}
		got := VisibleSteps(opts, answers)
		want := []int{0, 1, 2}
		assertIntSlice(t, got, want)
	})

	t.Run("some_hidden", func(t *testing.T) {
		answers := Answers{"a": "no"}
		got := VisibleSteps(opts, answers)
		want := []int{0, 2}
		assertIntSlice(t, got, want)
	})
}

func TestFormatAnswer(t *testing.T) {
	tests := []struct {
		name string
		opt  config.Option
		val  any
		want string
	}{
		{"confirm_true", config.Option{Type: "confirm"}, true, "Yes"},
		{"confirm_false", config.Option{Type: "confirm"}, false, "No"},
		{"select_label_lookup", config.Option{
			Type:    "select",
			Choices: []config.Choice{{Value: "go", Label: "Go"}},
		}, "go", "Go"},
		{"select_none", config.Option{Type: "select"}, "_none", "None"},
		{"select_fallback", config.Option{Type: "select"}, "unknown", "unknown"},
		{"multi_select_labels", config.Option{
			Type:    "multi_select",
			Choices: []config.Choice{{Value: "a", Label: "Alpha"}, {Value: "b", Label: "Beta"}},
		}, []string{"a", "b"}, "Alpha, Beta"},
		{"input_fallback", config.Option{Type: "input"}, "hello", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAnswer(&tt.opt, tt.val)
			if got != tt.want {
				t.Errorf("FormatAnswer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func assertIntSlice(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}
