package command

import (
	"slices"
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestSecretEnv(t *testing.T) {
	envPass := config.Option{Name: "tok", Type: config.OptionPassword, SecretEnv: "GH_TOKEN", Flag: "--token"}
	flagPass := config.Option{Name: "tok", Type: config.OptionPassword, Flag: "--token"}

	tests := []struct {
		name string
		opts []config.Option
		vals config.Values
		want []string
	}{
		{"env_channel", []config.Option{envPass},
			config.Values{"tok": config.StringVal("abc123")}, []string{"GH_TOKEN=abc123"}},
		{"blank_value", []config.Option{envPass}, config.Values{"tok": config.StringVal("")}, nil},
		{"missing_value", []config.Option{envPass}, config.Values{}, nil},
		{"no_secret_env", []config.Option{flagPass},
			config.Values{"tok": config.StringVal("abc123")}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &config.Wizard{Options: tt.opts}
			got := SecretEnv(w, tt.vals)
			if !slices.Equal(got, tt.want) {
				t.Errorf("SecretEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildOptionFlagsPassword(t *testing.T) {
	envPass := config.Option{Type: config.OptionPassword, SecretEnv: "GH_TOKEN", Flag: "--token"}
	assertStringSlice(t, buildOptionFlags(envPass, config.StringVal("abc123"), "equals"), nil)

	flagPass := config.Option{Type: config.OptionPassword, Flag: "--token"}
	assertStringSlice(t, buildOptionFlags(flagPass, config.StringVal("abc123"), "equals"), []string{"--token=abc123"})
}
