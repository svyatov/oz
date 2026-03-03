package command

import (
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestFormatFlag(t *testing.T) {
	tests := []struct {
		name, flag, value, style, want string
	}{
		{"equals", "--flag", "val", "equals", "--flag=val"},
		{"space", "--flag", "val", "space", "--flag val"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFlag(tt.flag, tt.value, tt.style)
			if got != tt.want {
				t.Errorf("formatFlag(%q, %q, %q) = %q, want %q", tt.flag, tt.value, tt.style, got, tt.want)
			}
		})
	}
}

func TestBuildConfirmFlags(t *testing.T) {
	opt := config.Option{FlagTrue: "--yes", FlagFalse: "--no"}

	tests := []struct {
		name string
		val  any
		want []string
	}{
		{"true_with_flag_true", true, []string{"--yes"}},
		{"false_with_flag_false", false, []string{"--no"}},
		{"non_bool", "string", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildConfirmFlags(opt, tt.val)
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
		{"none_with_flag_none", "_none", []string{"--no-lang"}},
		{"empty_with_flag_none", "", []string{"--no-lang"}},
		{"no_flag", "go", nil}, // tested below with empty flag
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
	opt := config.Option{Flag: "--feature"}

	tests := []struct {
		name string
		val  any
		want []string
	}{
		{"multiple", []string{"a", "b"}, []string{"--feature=a", "--feature=b"}},
		{"empty_slice", []string{}, nil},
		{"non_slice", "bad", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildMultiSelectFlags(opt, tt.val, "equals")
			assertStringSlice(t, got, tt.want)
		})
	}
}

func TestBuild(t *testing.T) {
	w := &config.Wizard{
		Command: "docker run",
		Args: []config.Arg{
			{Name: "image", Position: 2},
			{Name: "tag", Position: 1},
		},
		Options: []config.Option{
			{Name: "verbose", Type: "confirm", FlagTrue: "-v"},
			{Name: "port", Type: "input", Flag: "-p", Label: "Port"},
		},
	}
	posArgs := map[string]string{"image": "nginx", "tag": "latest"}
	answers := map[string]any{"verbose": true, "port": "8080"}

	parts := Build(w, posArgs, answers)
	plain := PlainParts(parts)
	formatted := FormatCommand(parts)

	wantPlain := []string{"docker", "run", "latest", "nginx", "-v", "-p=8080"}
	assertStringSlice(t, plain, wantPlain)

	if formatted != "docker run latest nginx -v -p=8080" {
		t.Errorf("FormatCommand = %q", formatted)
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v (len %d), want %v (len %d)", got, len(got), want, len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, got[i], want[i])
		}
	}
}
