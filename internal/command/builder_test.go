package command

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestFormatFlag(t *testing.T) {
	tests := []struct {
		name  string
		flag  string
		value string
		style config.FlagStyle
		want  string
	}{
		{"equals", "--flag", "val", config.FlagStyleEquals, "--flag=val"},
		{"space", "--flag", "val", config.FlagStyleSpace, "--flag val"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFlag(tt.flag, tt.value, tt.style)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildConfirmFlags(t *testing.T) {
	tests := []struct {
		name string
		opt  config.Option
		val  config.FieldValue
		want []string
	}{
		{"true_with_flag_true",
			config.Option{FlagTrue: "--yes", FlagFalse: "--no"},
			config.BoolVal(true), []string{"--yes"}},
		{"false_with_flag_false",
			config.Option{FlagTrue: "--yes", FlagFalse: "--no"},
			config.BoolVal(false), []string{"--no"}},
		{"non_bool",
			config.Option{FlagTrue: "--yes"}, config.StringVal("string"), nil},
		{"flag_shorthand_true",
			config.Option{Flag: "--verbose"}, config.BoolVal(true), []string{"--verbose"}},
		{"flag_shorthand_false",
			config.Option{Flag: "--verbose"}, config.BoolVal(false), nil},
		{"flag_true_precedence",
			config.Option{Flag: "--verbose", FlagTrue: "--yes"},
			config.BoolVal(true), []string{"--yes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildConfirmFlags(tt.opt, tt.val)
			assertStringSlice(t, got, tt.want)
		})
	}
}

func TestBuildSelectFlags(t *testing.T) {
	opt := config.Option{Flag: "--lang", FlagNone: "--no-lang"}

	tests := []struct {
		name string
		val  config.FieldValue
		want []string
	}{
		{"normal_value", config.StringVal("go"), []string{"--lang=go"}},
		{"none_with_flag_none", config.StringVal(config.NoneValue), []string{"--no-lang"}},
		{"empty_with_flag_none", config.StringVal(""), []string{"--no-lang"}},
		{"no_flag", config.StringVal("go"), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := opt
			if tt.name == "no_flag" {
				o = config.Option{}
			}
			got := buildSelectFlags(o, tt.val, "equals")
			assertStringSlice(t, got, tt.want)
		})
	}
}

func TestBuildInputFlags(t *testing.T) {
	tests := []struct {
		name string
		opt  config.Option
		val  config.FieldValue
		want []string
	}{
		{"normal", config.Option{Flag: "--name"}, config.StringVal("foo"), []string{"--name=foo"}},
		{"empty_value", config.Option{Flag: "--name"}, config.StringVal(""), nil},
		{"no_flag", config.Option{}, config.StringVal("foo"), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildInputFlags(tt.opt, tt.val, "equals")
			assertStringSlice(t, got, tt.want)
		})
	}
}

func TestBuildMultiSelectFlags(t *testing.T) {
	tests := []struct {
		name string
		opt  config.Option
		val  config.FieldValue
		want []string
	}{
		{"repeated",
			config.Option{Flag: "--feature"},
			config.StringsVal("a", "b"), []string{"--feature=a", "--feature=b"}},
		{"empty_slice",
			config.Option{Flag: "--feature"}, config.StringsVal(), nil},
		{"non_slice",
			config.Option{Flag: "--feature"}, config.StringVal("bad"), nil},
		{"separator_comma",
			config.Option{Flag: "--features", Separator: ","},
			config.StringsVal("auth", "api"), []string{"--features=auth,api"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildMultiSelectFlags(tt.opt, tt.val, "equals")
			assertStringSlice(t, got, tt.want)
		})
	}
}

func TestBuild(t *testing.T) {
	t.Run("flags_only", func(t *testing.T) {
		w := &config.Wizard{
			Command: "docker run",
			Options: []config.Option{
				{Name: "verbose", Type: config.OptionConfirm, FlagTrue: "-v"},
				{Name: "port", Type: config.OptionInput, Flag: "-p", Label: "Port"},
			},
		}
		answers := config.Values{"verbose": config.BoolVal(true), "port": config.StringVal("8080")}
		parts := Build(w, answers)
		if got := FormatCommand(parts); got != "docker run -v -p=8080" {
			t.Errorf("FormatCommand = %q", got)
		}
	})

	t.Run("positional_options", func(t *testing.T) {
		w := &config.Wizard{
			Command: "task",
			Options: []config.Option{{
				Name: "task_name", Type: config.OptionSelect, Label: "Task",
				Positional: true,
				Choices:    config.FlexChoices{{Value: "build", Label: "build"}},
			}},
		}
		answers := config.Values{"task_name": config.StringVal("build")}
		assertStringSlice(t, PlainParts(Build(w, answers)), []string{"task", "build"})
	})

	t.Run("positional_before_flags", func(t *testing.T) {
		w := &config.Wizard{
			Command: "docker run",
			Options: []config.Option{
				{
					Name: "image", Type: config.OptionSelect, Label: "Image",
					Positional: true,
					Choices:    config.FlexChoices{{Value: "nginx", Label: "nginx"}},
				},
				{Name: "detach", Type: config.OptionConfirm, Flag: "-d"},
			},
		}
		answers := config.Values{"image": config.StringVal("nginx"), "detach": config.BoolVal(true)}
		want := []string{"docker", "run", "nginx", "-d"}
		assertStringSlice(t, PlainParts(Build(w, answers)), want)
	})
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// stripANSI removes ANSI escape codes from a string.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestRun(t *testing.T) {
	t.Run("empty_parts", func(t *testing.T) {
		err := Run(nil)
		if err == nil || !strings.Contains(err.Error(), "empty command") {
			t.Errorf("got %v, want error containing %q", err, "empty command")
		}
	})
}

func TestFormatCommandColored(t *testing.T) {
	parts := []Part{
		{Text: "git", Kind: PartCommand},
		{Text: "main", Kind: PartArg},
		{Text: "--force", Kind: PartFlag},
		{Text: "--branch=dev", Kind: PartFlag},
	}

	got := stripANSI(formatCommandColored(parts))

	for _, want := range []string{"git", "main", "--force", "--branch=", "dev"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestAppendExtra(t *testing.T) {
	parts := []Part{{Text: "git", Kind: PartCommand}}

	t.Run("appends_as_PartExtra", func(t *testing.T) {
		got := AppendExtra(parts, []string{"--force", "file.txt"})
		if len(got) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(got))
		}
		if got[1].Kind != PartExtra || got[1].Text != "--force" {
			t.Errorf("[1] = %+v, want PartExtra --force", got[1])
		}
		if got[2].Kind != PartExtra || got[2].Text != "file.txt" {
			t.Errorf("[2] = %+v, want PartExtra file.txt", got[2])
		}
	})

	t.Run("empty_noop", func(t *testing.T) {
		got := AppendExtra(parts, nil)
		if len(got) != 1 {
			t.Fatalf("expected 1 part, got %d", len(got))
		}
	})
}

func TestFormatCommandColored_extra(t *testing.T) {
	parts := []Part{
		{Text: "rails", Kind: PartCommand},
		{Text: "myapp", Kind: PartExtra},
	}
	got := stripANSI(formatCommandColored(parts))
	for _, want := range []string{"rails", "myapp"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestBuild_with_extra(t *testing.T) {
	w := &config.Wizard{
		Command: "rails new",
		Options: []config.Option{
			{Name: "database", Type: config.OptionSelect, Flag: "--database"},
		},
	}
	answers := config.Values{"database": config.StringVal("pg")}
	parts := Build(w, answers)
	parts = AppendExtra(parts, []string{"myapp", "--force"})
	want := []string{"rails", "new", "--database=pg", "myapp", "--force"}
	assertStringSlice(t, PlainParts(parts), want)
}

func TestBuildOptionFlags_unknown_type(t *testing.T) {
	opt := config.Option{
		Name: "bad",
		Type: config.OptionType("bogus"),
		Flag: "--bad",
	}
	got := buildOptionFlags(opt, config.StringVal("x"), config.FlagStyleEquals)
	if got != nil {
		t.Errorf("got %v, want nil for unknown type", got)
	}
}

func TestBuild_skip_missing_answer(t *testing.T) {
	w := &config.Wizard{
		Command: "echo",
		Options: []config.Option{
			{Name: "missing", Type: config.OptionInput, Flag: "--flag"},
		},
	}
	parts := Build(w, config.Values{})
	assertStringSlice(t, PlainParts(parts), []string{"echo"})
}

func TestBuild_positional_none_value(t *testing.T) {
	w := &config.Wizard{
		Command: "task",
		Options: []config.Option{{
			Name:       "target",
			Type:       config.OptionSelect,
			Positional: true,
			Choices:    config.FlexChoices{{Value: "build"}},
		}},
	}
	parts := Build(w, config.Values{"target": config.StringVal(config.NoneValue)})
	assertStringSlice(t, PlainParts(parts), []string{"task"})
}
