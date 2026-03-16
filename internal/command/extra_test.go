package command

import (
	"testing"

	"github.com/svyatov/oz/internal/config"
)

func TestParseExtra_empty(t *testing.T) {
	vals, raw := ParseExtra(nil, nil)
	if vals != nil {
		t.Errorf("expected nil values, got %v", vals)
	}
	if raw != nil {
		t.Errorf("expected nil raw, got %v", raw)
	}
}

func TestParseExtra_positional(t *testing.T) {
	opts := []config.Option{
		{Name: "app_name", Type: config.OptionInput, Positional: true},
		{Name: "target", Type: config.OptionInput, Positional: true},
	}

	t.Run("single", func(t *testing.T) {
		vals, raw := ParseExtra(opts, []string{"myapp"})
		if vals["app_name"].String() != "myapp" {
			t.Errorf("app_name = %q, want %q", vals["app_name"].String(), "myapp")
		}
		if raw != nil {
			t.Errorf("expected no raw args, got %v", raw)
		}
	})

	t.Run("multiple", func(t *testing.T) {
		vals, raw := ParseExtra(opts, []string{"myapp", "prod"})
		if vals["app_name"].String() != "myapp" {
			t.Errorf("app_name = %q, want %q", vals["app_name"].String(), "myapp")
		}
		if vals["target"].String() != "prod" {
			t.Errorf("target = %q, want %q", vals["target"].String(), "prod")
		}
		if raw != nil {
			t.Errorf("expected no raw args, got %v", raw)
		}
	})

	t.Run("overflow_to_raw", func(t *testing.T) {
		vals, raw := ParseExtra(opts, []string{"a", "b", "c"})
		if len(vals) != 2 {
			t.Fatalf("expected 2 matched values, got %d", len(vals))
		}
		assertStringSlice(t, raw, []string{"c"})
	})
}

func TestParseExtra_flag_equals(t *testing.T) {
	opts := []config.Option{
		{Name: "database", Type: config.OptionSelect, Flag: "--database"},
	}
	vals, raw := ParseExtra(opts, []string{"--database=mysql"})
	if vals["database"].String() != "mysql" {
		t.Errorf("database = %q, want %q", vals["database"].String(), "mysql")
	}
	if raw != nil {
		t.Errorf("expected no raw args, got %v", raw)
	}
}

func TestParseExtra_flag_space(t *testing.T) {
	opts := []config.Option{
		{Name: "database", Type: config.OptionSelect, Flag: "--database"},
	}
	vals, raw := ParseExtra(opts, []string{"--database", "mysql"})
	if vals["database"].String() != "mysql" {
		t.Errorf("database = %q, want %q", vals["database"].String(), "mysql")
	}
	if raw != nil {
		t.Errorf("expected no raw args, got %v", raw)
	}
}

func TestParseExtra_confirm_flag_true(t *testing.T) {
	opts := []config.Option{
		{Name: "verbose", Type: config.OptionConfirm, FlagTrue: "--verbose", FlagFalse: "--quiet"},
	}

	t.Run("true", func(t *testing.T) {
		vals, _ := ParseExtra(opts, []string{"--verbose"})
		if !vals["verbose"].Bool() {
			t.Error("expected verbose=true")
		}
	})

	t.Run("false", func(t *testing.T) {
		vals, _ := ParseExtra(opts, []string{"--quiet"})
		if vals["verbose"].Bool() {
			t.Error("expected verbose=false")
		}
	})
}

func TestParseExtra_confirm_flag_shorthand(t *testing.T) {
	// When flag_true is empty, flag acts as flag_true.
	opts := []config.Option{
		{Name: "api", Type: config.OptionConfirm, Flag: "--api"},
	}
	vals, _ := ParseExtra(opts, []string{"--api"})
	if !vals["api"].Bool() {
		t.Error("expected api=true")
	}
}

func TestParseExtra_multi_select(t *testing.T) {
	opts := []config.Option{
		{Name: "features", Type: config.OptionMultiSelect, Flag: "--feature"},
	}
	vals, _ := ParseExtra(opts, []string{"--feature=auth", "--feature=api"})
	got := vals["features"].Strings()
	assertStringSlice(t, got, []string{"auth", "api"})
}

func TestParseExtra_unmatched(t *testing.T) {
	opts := []config.Option{
		{Name: "database", Type: config.OptionSelect, Flag: "--database"},
	}
	vals, raw := ParseExtra(opts, []string{"--force", "--database=mysql"})
	if vals["database"].String() != "mysql" {
		t.Errorf("database = %q, want %q", vals["database"].String(), "mysql")
	}
	assertStringSlice(t, raw, []string{"--force"})
}

func TestParseExtra_mixed(t *testing.T) {
	opts := []config.Option{
		{Name: "app_name", Type: config.OptionInput, Positional: true},
		{Name: "database", Type: config.OptionSelect, Flag: "--database"},
		{Name: "api", Type: config.OptionConfirm, Flag: "--api"},
	}
	vals, raw := ParseExtra(opts, []string{
		"myapp", "--database=postgresql", "--api", "--force",
	})
	if vals["app_name"].String() != "myapp" {
		t.Errorf("app_name = %q, want %q", vals["app_name"].String(), "myapp")
	}
	if vals["database"].String() != "postgresql" {
		t.Errorf("database = %q, want %q", vals["database"].String(), "postgresql")
	}
	if !vals["api"].Bool() {
		t.Error("expected api=true")
	}
	assertStringSlice(t, raw, []string{"--force"})
}

func TestParseExtra_no_positionals(t *testing.T) {
	opts := []config.Option{
		{Name: "database", Type: config.OptionSelect, Flag: "--database"},
	}
	vals, raw := ParseExtra(opts, []string{"extra"})
	if len(vals) != 0 {
		t.Errorf("expected no matched values, got %v", vals)
	}
	assertStringSlice(t, raw, []string{"extra"})
}
