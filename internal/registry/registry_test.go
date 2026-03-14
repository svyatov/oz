package registry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testIndexYAML = `wizards:
  - name: rails-new
    description: Rails app generator
  - name: docker-compose
    description: Docker Compose builder
`

const testWizardYAML = `name: rails-new
description: Rails app generator
command: rails new
options: []
`

func newTestServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL)
}

func TestFetchIndex(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.yml" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(testIndexYAML))
	})

	idx, err := client.FetchIndex()
	if err != nil {
		t.Fatalf("FetchIndex: %v", err)
	}
	if len(idx.Wizards) != 2 {
		t.Fatalf("expected 2 wizards, got %d", len(idx.Wizards))
	}
	if idx.Wizards[0].Name != "rails-new" {
		t.Errorf("expected first wizard name %q, got %q", "rails-new", idx.Wizards[0].Name)
	}
	if idx.Wizards[1].Description != "Docker Compose builder" {
		t.Errorf("expected second description %q, got %q", "Docker Compose builder", idx.Wizards[1].Description)
	}
}

func TestFetchIndexNotFound(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	_, err := client.FetchIndex()
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention 404, got: %v", err)
	}
}

func TestFetchWizard(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wizards/rails-new.yml" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(testWizardYAML))
	})

	data, err := client.FetchWizard("rails-new")
	if err != nil {
		t.Fatalf("FetchWizard: %v", err)
	}
	if !strings.Contains(string(data), "rails-new") {
		t.Errorf("expected YAML to contain wizard name, got: %s", data)
	}
}

func TestFetchWizardNotFound(t *testing.T) {
	client := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	_, err := client.FetchWizard("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing wizard")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention wizard name, got: %v", err)
	}
}

func TestDefaultBaseURL(t *testing.T) {
	got := DefaultBaseURL()
	if got != defaultBaseURL {
		t.Errorf("expected default %q, got %q", defaultBaseURL, got)
	}

	t.Setenv("OZ_REGISTRY_URL", "https://example.com/custom/")
	got = DefaultBaseURL()
	if got != "https://example.com/custom/" {
		t.Errorf("expected env override, got %q", got)
	}
}

func TestNewAddsTrailingSlash(t *testing.T) {
	client := New("https://example.com")
	if client.baseURL != "https://example.com/" {
		t.Errorf("expected trailing slash, got %q", client.baseURL)
	}
}
