package wizard

import (
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestEvalShowWhen(t *testing.T) {
	answers := config.Values{"lang": config.StringVal("go"), "verbose": config.BoolVal(true)}

	tests := []struct {
		name     string
		showWhen config.Values
		want     bool
	}{
		{"empty_conditions", nil, true},
		{"all_met", config.Values{"lang": config.StringVal("go"), "verbose": config.BoolVal(true)}, true},
		{"one_unmet", config.Values{"lang": config.StringVal("rust")}, false},
		{"missing_answer", config.Values{"missing": config.StringVal("x")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EvalShowWhen(tt.showWhen, answers); got != tt.want {
				t.Errorf("EvalShowWhen(%v) = %v, want %v", tt.showWhen, got, tt.want)
			}
		})
	}
}

func TestEvalHideWhen(t *testing.T) {
	answers := config.Values{"lang": config.StringVal("go"), "verbose": config.BoolVal(true)}

	tests := []struct {
		name     string
		hideWhen config.Values
		want     bool
	}{
		{"empty_conditions", nil, false},
		{"all_met", config.Values{"lang": config.StringVal("go")}, true},
		{"not_met", config.Values{"lang": config.StringVal("rust")}, false},
		{"missing_answer", config.Values{"missing": config.StringVal("x")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EvalHideWhen(tt.hideWhen, answers); got != tt.want {
				t.Errorf("EvalHideWhen(%v) = %v, want %v", tt.hideWhen, got, tt.want)
			}
		})
	}
}

func TestIsVisible(t *testing.T) {
	answers := config.Values{"lang": config.StringVal("go"), "skip": config.BoolVal(true)}

	tests := []struct {
		name string
		opt  config.Option
		want bool
	}{
		{"no_conditions", config.Option{Name: "a"}, true},
		{"show_when_met", config.Option{Name: "a", ShowWhen: config.Values{"lang": config.StringVal("go")}}, true},
		{"show_when_not_met", config.Option{Name: "a", ShowWhen: config.Values{"lang": config.StringVal("rust")}}, false},
		{"hide_when_met", config.Option{Name: "a", HideWhen: config.Values{"skip": config.BoolVal(true)}}, false},
		{"hide_when_not_met", config.Option{Name: "a", HideWhen: config.Values{"skip": config.BoolVal(false)}}, true},
		{"show_met_hide_not_met", config.Option{
			Name:     "a",
			ShowWhen: config.Values{"lang": config.StringVal("go")},
			HideWhen: config.Values{"skip": config.BoolVal(false)},
		}, true},
		{"show_met_hide_met", config.Option{
			Name:     "a",
			ShowWhen: config.Values{"lang": config.StringVal("go")},
			HideWhen: config.Values{"skip": config.BoolVal(true)},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVisible(tt.opt, answers); got != tt.want {
				t.Errorf("IsVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValuesMatch(t *testing.T) {
	tests := []struct {
		name           string
		actual, expect config.FieldValue
		want           bool
	}{
		{"string_string", config.StringVal("foo"), config.StringVal("foo"), true},
		{"mismatch", config.StringVal("a"), config.StringVal("b"), false},
		// Multi-select membership: actual is a list
		{"membership_match", config.StringsVal("auth", "api"), config.StringVal("auth"), true},
		{"membership_no_match", config.StringsVal("auth", "api"), config.StringVal("logging"), false},
		// Both lists: OR + membership
		{"both_lists_match", config.StringsVal("auth", "api"), config.StringsVal("api", "logging"), true},
		{"both_lists_no_match", config.StringsVal("auth", "api"), config.StringsVal("logging", "cache"), false},
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
		pins      config.Values
		wantNames []string
		wantCount int
	}{
		{"no_pins", config.Values{}, []string{"a", "b", "c"}, 0},
		{"some_pinned", config.Values{"b": config.StringVal("val")}, []string{"a", "c"}, 1},
		{"all_pinned", config.Values{
			"a": config.StringVal("1"),
			"b": config.StringVal("2"),
			"c": config.StringVal("3"),
		}, nil, 3},
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
		{Name: "b", ShowWhen: config.Values{"a": config.StringVal("yes")}},
		{Name: "c"},
		{Name: "d", HideWhen: config.Values{"a": config.StringVal("yes")}},
	}

	t.Run("all_visible", func(t *testing.T) {
		answers := config.Values{"a": config.StringVal("yes")}
		got := VisibleSteps(opts, answers)
		want := []int{0, 1, 2}
		assertIntSlice(t, got, want)
	})

	t.Run("some_hidden", func(t *testing.T) {
		answers := config.Values{"a": config.StringVal("no")}
		got := VisibleSteps(opts, answers)
		want := []int{0, 2, 3}
		assertIntSlice(t, got, want)
	})
}

func TestFormatAnswer(t *testing.T) {
	tests := []struct {
		name string
		opt  config.Option
		val  config.FieldValue
		want string
	}{
		{"confirm_true", config.Option{Type: config.OptionConfirm}, config.BoolVal(true), "Yes"},
		{"confirm_false", config.Option{Type: config.OptionConfirm}, config.BoolVal(false), "No"},
		{"select_label_lookup", config.Option{
			Type:    config.OptionSelect,
			Choices: config.FlexChoices{{Value: "go", Label: "Go"}},
		}, config.StringVal("go"), "Go"},
		{"select_none", config.Option{Type: config.OptionSelect}, config.StringVal(config.NoneValue), "None"},
		{"select_fallback", config.Option{Type: config.OptionSelect}, config.StringVal("unknown"), "unknown"},
		{"multi_select_labels", config.Option{
			Type:    config.OptionMultiSelect,
			Choices: config.FlexChoices{{Value: "a", Label: "Alpha"}, {Value: "b", Label: "Beta"}},
		}, config.StringsVal("a", "b"), "Alpha, Beta"},
		{"input_fallback", config.Option{Type: config.OptionInput}, config.StringVal("hello"), "hello"},
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
