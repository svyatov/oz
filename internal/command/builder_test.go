package command

import (
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
		val  any
		want []string
	}{
		{"true_with_flag_true",
			config.Option{FlagTrue: "--yes", FlagFalse: "--no"},
			true, []string{"--yes"}},
		{"false_with_flag_false",
			config.Option{FlagTrue: "--yes", FlagFalse: "--no"},
			false, []string{"--no"}},
		{"non_bool",
			config.Option{FlagTrue: "--yes"}, "string", nil},
		{"flag_shorthand_true",
			config.Option{Flag: "--verbose"}, true, []string{"--verbose"}},
		{"flag_shorthand_false",
			config.Option{Flag: "--verbose"}, false, nil},
		{"flag_true_precedence",
			config.Option{Flag: "--verbose", FlagTrue: "--yes"},
			true, []string{"--yes"}},
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
		val  any
		want []string
	}{
		{"normal_value", "go", []string{"--lang=go"}},
		{"none_with_flag_none", config.NoneValue, []string{"--no-lang"}},
		{"empty_with_flag_none", "", []string{"--no-lang"}},
		{"no_flag", "go", nil},
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
		val  any
		want []string
	}{
		{"normal", config.Option{Flag: "--name"}, "foo", []string{"--name=foo"}},
		{"empty_value", config.Option{Flag: "--name"}, "", nil},
		{"no_flag", config.Option{}, "foo", nil},
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
		val  any
		want []string
	}{
		{"repeated",
			config.Option{Flag: "--feature"},
			[]string{"a", "b"}, []string{"--feature=a", "--feature=b"}},
		{"empty_slice",
			config.Option{Flag: "--feature"}, []string{}, nil},
		{"non_slice",
			config.Option{Flag: "--feature"}, "bad", nil},
		{"separator_comma",
			config.Option{Flag: "--features", Separator: ","},
			[]string{"auth", "api"}, []string{"--features=auth,api"}},
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
		answers := map[string]any{"verbose": true, "port": "8080"}
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
		answers := map[string]any{"task_name": "build"}
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
		answers := map[string]any{"image": "nginx", "detach": true}
		want := []string{"docker", "run", "nginx", "-d"}
		assertStringSlice(t, PlainParts(Build(w, answers)), want)
	})
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v (len %d), want %v (len %d)",
			got, len(got), want, len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}
