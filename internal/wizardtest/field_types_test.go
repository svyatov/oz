package wizardtest

import (
	"strings"
	"testing"

	"github.com/svyatov/oz/internal/config"
)

// TestBuildCommand_SecretEnvPassword covers AE1 through the production build path:
// a secret_env password emits no argv token (the secret rides the env channel).
func TestBuildCommand_SecretEnvPassword(t *testing.T) {
	yaml := `name: demo
command: gh auth
options:
  - name: token
    type: password
    label: Token
    secret_env: GH_TOKEN
`
	answers := config.Values{"token": config.StringVal("abc123")}
	got := BuildCommand(mustParse(t, yaml), &Fixture{Version: "", Answers: answers})

	if got != "gh auth" {
		t.Errorf("got %q, want %q", got, "gh auth")
	}
	if strings.Contains(got, "abc123") {
		t.Errorf("secret leaked into built command: %q", got)
	}
}

// TestBuildCommand_FlagPassword covers AE2: a flag password (no env channel)
// passes its real value in argv; the fidelity renderer is never redacted.
func TestBuildCommand_FlagPassword(t *testing.T) {
	yaml := `name: demo
command: gh auth
options:
  - name: token
    type: password
    label: Token
    flag: --token
`
	answers := config.Values{"token": config.StringVal("abc123")}
	got := BuildCommand(mustParse(t, yaml), &Fixture{Version: "", Answers: answers})

	if got != "gh auth --token=abc123" {
		t.Errorf("got %q, want %q", got, "gh auth --token=abc123")
	}
}

// TestBuildCommand_Number covers R6/R8: an in-range number emits its flag;
// a blank number omits it.
func TestBuildCommand_Number(t *testing.T) {
	yaml := `name: demo
command: serve
options:
  - name: port
    type: number
    label: Port
    flag: --port
    min: 1
    max: 65535
`
	w := mustParse(t, yaml)

	got := BuildCommand(w, &Fixture{Version: "", Answers: config.Values{"port": config.StringVal("443")}})
	if got != "serve --port=443" {
		t.Errorf("got %q, want %q", got, "serve --port=443")
	}

	blank := BuildCommand(w, &Fixture{Version: "", Answers: config.Values{"port": config.StringVal("")}})
	if blank != "serve" {
		t.Errorf("blank number: got %q, want %q", blank, "serve")
	}
}
