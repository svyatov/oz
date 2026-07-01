package main

import (
	"strings"
	"testing"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/store"
)

func TestStripSecrets(t *testing.T) {
	w := &config.Wizard{Options: []config.Option{
		{Name: "user", Type: config.OptionInput},
		{Name: "token", Type: config.OptionPassword},
		{Name: "port", Type: config.OptionNumber},
	}}

	in := config.Values{
		"user":  config.StringVal("alice"),
		"token": config.StringVal("s3cr3t"),
		"port":  config.StringVal("443"),
	}
	got := stripSecrets(w, in)

	if _, has := got["token"]; has {
		t.Error("password key must be stripped")
	}
	if got["user"].Scalar() != "alice" || got["port"].Scalar() != "443" {
		t.Errorf("non-password answers altered: %+v", got)
	}
	if _, has := in["token"]; !has {
		t.Error("stripSecrets must not mutate the input map")
	}
}

func TestReconcilePresets_stripsPassword(t *testing.T) {
	dir := t.TempDir()
	st := store.New(dir)
	w := &config.Wizard{Name: "wiz", Options: []config.Option{
		{Name: "token", Type: config.OptionPassword},
		{Name: "user", Type: config.OptionInput},
	}}

	final := map[string]config.Values{
		"p": {"token": config.StringVal("s3cr3t"), "user": config.StringVal("bob")},
	}
	if err := reconcilePresets(st, w, nil, final); err != nil {
		t.Fatalf("reconcilePresets: %v", err)
	}

	vals, err := st.LoadPreset("wiz", "p")
	if err != nil {
		t.Fatalf("loading preset: %v", err)
	}
	if _, has := vals["token"]; has {
		t.Error("password persisted to preset file")
	}
	if vals["user"].Scalar() != "bob" {
		t.Errorf("non-password preset value lost: %+v", vals)
	}
}

func TestPrintOptionDetails_masksPasswordDefault(t *testing.T) {
	def := config.StringVal("topsecret")

	t.Run("password_masked", func(t *testing.T) {
		out := captureStdout(t, func() {
			printOptionDetails(config.Option{Type: config.OptionPassword, Label: "Token", Default: &def})
		})
		if strings.Contains(out, "topsecret") {
			t.Errorf("password default leaked in show output: %q", out)
		}
		if !strings.Contains(out, "****") {
			t.Errorf("expected masked default, got %q", out)
		}
	})

	t.Run("number_verbatim", func(t *testing.T) {
		numDef := config.StringVal("443")
		out := captureStdout(t, func() {
			printOptionDetails(config.Option{Type: config.OptionNumber, Label: "Port", Default: &numDef})
		})
		if !strings.Contains(out, "443") {
			t.Errorf("expected number default shown verbatim, got %q", out)
		}
	})
}
