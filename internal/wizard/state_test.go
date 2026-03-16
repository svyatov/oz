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

func TestFormatAnswerMultiSelectEmpty(t *testing.T) {
	opt := config.Option{
		Type:    config.OptionMultiSelect,
		Choices: config.FlexChoices{{Value: "a", Label: "Alpha"}},
	}
	got := FormatAnswer(&opt, config.StringsVal())
	if got != "" {
		t.Errorf("expected empty string for empty multi_select, got %q", got)
	}
}

func TestFormatAnswerMultiSelectUnknownValue(t *testing.T) {
	opt := config.Option{
		Type:    config.OptionMultiSelect,
		Choices: config.FlexChoices{{Value: "a", Label: "Alpha"}},
	}
	got := FormatAnswer(&opt, config.StringsVal("a", "unknown"))
	if got != "Alpha, unknown" {
		t.Errorf("expected 'Alpha, unknown', got %q", got)
	}
}

func TestResolveDefaultFromSources(t *testing.T) {
	dbChoices := []config.Choice{{Value: "pg"}}
	sqliteDefault := new(config.FieldValue)
	*sqliteDefault = config.StringVal("sqlite")

	t.Run("found_in_first_source", func(t *testing.T) {
		opt := config.Option{Name: "db", Type: config.OptionSelect, Choices: dbChoices}
		got := resolveDefault(&opt, config.Values{"db": config.StringVal("mysql")})
		assertResolvedScalar(t, got, "mysql")
	})
	t.Run("found_in_second_source", func(t *testing.T) {
		opt := config.Option{Name: "db", Type: config.OptionSelect, Choices: dbChoices}
		got := resolveDefault(&opt, config.Values{}, config.Values{"db": config.StringVal("pg")})
		assertResolvedScalar(t, got, "pg")
	})
	t.Run("falls_back_to_opt_default", func(t *testing.T) {
		opt := config.Option{
			Name: "db", Type: config.OptionSelect,
			Default: sqliteDefault, Choices: dbChoices,
		}
		got := resolveDefault(&opt, config.Values{})
		assertResolvedScalar(t, got, "sqlite")
	})
	t.Run("select_first_choice_fallback", func(t *testing.T) {
		twoChoices := []config.Choice{{Value: "pg"}, {Value: "mysql"}}
		opt := config.Option{Name: "db", Type: config.OptionSelect, Choices: twoChoices}
		got := resolveDefault(&opt, config.Values{})
		assertResolvedScalar(t, got, "pg")
	})
}

func TestResolveDefaultTypeFallbacks(t *testing.T) {
	empty := config.Values{}

	t.Run("confirm_defaults_false", func(t *testing.T) {
		opt := config.Option{Name: "flag", Type: config.OptionConfirm}
		assertResolvedScalar(t, resolveDefault(&opt, empty), "false")
	})
	t.Run("input_defaults_empty", func(t *testing.T) {
		opt := config.Option{Name: "name", Type: config.OptionInput}
		assertResolvedScalar(t, resolveDefault(&opt, empty), "")
	})
	t.Run("multi_select_no_default", func(t *testing.T) {
		opt := config.Option{Name: "features", Type: config.OptionMultiSelect}
		if got := resolveDefault(&opt, empty); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
	t.Run("select_no_choices_no_default", func(t *testing.T) {
		opt := config.Option{Name: "empty", Type: config.OptionSelect}
		if got := resolveDefault(&opt, empty); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func assertResolvedScalar(t *testing.T, got *config.FieldValue, want string) {
	t.Helper()
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.Scalar() != want {
		t.Errorf("expected %q, got %q", want, got.Scalar())
	}
}

func TestMissingRequired(t *testing.T) {
	opts := []config.Option{
		{Name: "name", Label: "Name", Required: true},
		{Name: "database", Label: "Database", Required: true},
		{Name: "optional", Label: "Optional"},
		{Name: "hidden", Label: "Hidden", Required: true,
			HideWhen: config.Values{"name": config.StringVal("skip")}},
	}

	tests := []struct {
		name   string
		values config.Values
		want   []string
	}{
		{"all_filled", config.Values{
			"name":     config.StringVal("app"),
			"database": config.StringVal("pg"),
			"hidden":   config.StringVal("val"),
		}, nil},
		{"one_missing", config.Values{
			"name":   config.StringVal("app"),
			"hidden": config.StringVal("val"),
		}, []string{"Database"}},
		{"hidden_required_skipped", config.Values{
			"name":     config.StringVal("skip"),
			"database": config.StringVal("pg"),
		}, nil},
		{"none_filled", config.Values{}, []string{"Name", "Database", "Hidden"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MissingRequired(opts, tt.values)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValuesMatchExpectedListActualScalar(t *testing.T) {
	// Expected is a list, actual is scalar: match if actual is in expected.
	actual := config.StringVal("go")
	expected := config.StringsVal("go", "rust")
	if !valuesMatch(actual, expected) {
		t.Error("expected match: scalar 'go' should be in list [go, rust]")
	}

	actual = config.StringVal("python")
	if valuesMatch(actual, expected) {
		t.Error("expected no match: scalar 'python' should not be in list [go, rust]")
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
