// Package registry fetches wizard configs from a remote registry.
package registry

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultBaseURL = "https://raw.githubusercontent.com/svyatov/oz-wizards/main/"
	maxBodySize    = 1 << 20 // 1 MB.
	httpTimeout    = 30 * time.Second
)

// Entry is a single wizard in the remote registry index.
type Entry struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Index is the parsed registry index.
type Index struct {
	Wizards []Entry `yaml:"wizards"`
}

// Client fetches wizard configs from a remote registry.
type Client struct {
	http    *http.Client
	baseURL string
}

// DefaultBaseURL returns the registry URL from OZ_REGISTRY_URL or the compiled-in default.
func DefaultBaseURL() string {
	if url, ok := os.LookupEnv("OZ_REGISTRY_URL"); ok {
		return url
	}
	return defaultBaseURL
}

// New creates a registry client. baseURL must end with "/".
func New(baseURL string) *Client {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: httpTimeout},
	}
}

// FetchIndex downloads and parses the registry index.
func (c *Client) FetchIndex() (*Index, error) {
	data, err := c.get("index.yml")
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}

	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}
	return &idx, nil
}

// FetchWizard downloads a single wizard YAML by name.
func (c *Client) FetchWizard(name string) ([]byte, error) {
	data, err := c.get("wizards/" + name + ".yml")
	if err != nil {
		return nil, fmt.Errorf("fetching wizard %q: %w", name, err)
	}
	return data, nil
}

func (c *Client) get(path string) ([]byte, error) {
	resp, err := c.http.Get(c.baseURL + path) //nolint:noctx // no context needed for simple CLI fetches.
	if err != nil {
		return nil, fmt.Errorf("connecting to registry: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	return data, nil
}
