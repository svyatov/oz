package ui

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestStepCounter(t *testing.T) {
	t.Helper()
	tests := []struct {
		name            string
		current, total  int
		wantCur, wantTot string
	}{
		{"single digit", 1, 5, "01", "05"},
		{"double digit", 12, 20, "12", "20"},
		{"zeros", 0, 0, "00", "00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(StepCounter(tt.current, tt.total))
			if !strings.Contains(got, tt.wantCur) {
				t.Errorf("StepCounter(%d, %d) = %q, want current %q", tt.current, tt.total, got, tt.wantCur)
			}
			if !strings.Contains(got, tt.wantTot) {
				t.Errorf("StepCounter(%d, %d) = %q, want total %q", tt.current, tt.total, got, tt.wantTot)
			}
			if !strings.Contains(got, "/") {
				t.Errorf("StepCounter(%d, %d) = %q, want separator /", tt.current, tt.total, got)
			}
		})
	}
}

func TestNumberGutter(t *testing.T) {
	t.Helper()
	active := stripANSI(NumberGutter(1, 2, true))
	inactive := stripANSI(NumberGutter(1, 2, false))

	if active != inactive {
		t.Logf("active=%q inactive=%q (text differs, expected same content)", active, inactive)
	}
	// Both should contain " 1" (right-aligned in width 2).
	if !strings.Contains(active, "1") {
		t.Errorf("NumberGutter(1, 2, true) = %q, want to contain 1", active)
	}

	// Raw output (with ANSI) should differ between active/inactive styles.
	rawActive := NumberGutter(1, 2, true)
	rawInactive := NumberGutter(1, 2, false)
	if rawActive == rawInactive {
		t.Error("NumberGutter active and inactive should have different ANSI styles")
	}
}

func TestNavHints(t *testing.T) {
	t.Helper()
	got := stripANSI(NavHints(HintUp, HintDown))
	if !strings.Contains(got, "up") {
		t.Errorf("NavHints(HintUp, HintDown) = %q, want to contain 'up'", got)
	}
	if !strings.Contains(got, "down") {
		t.Errorf("NavHints(HintUp, HintDown) = %q, want to contain 'down'", got)
	}
}

func TestPinnedInfo(t *testing.T) {
	t.Helper()
	if got := PinnedInfo(0); got != "" {
		t.Errorf("PinnedInfo(0) = %q, want empty", got)
	}
	got := stripANSI(PinnedInfo(3))
	if !strings.Contains(got, "3") {
		t.Errorf("PinnedInfo(3) = %q, want to contain '3'", got)
	}
	if !strings.Contains(got, "pinned") {
		t.Errorf("PinnedInfo(3) = %q, want to contain 'pinned'", got)
	}
}

func TestCompletedStepLine(t *testing.T) {
	t.Helper()
	got := stripANSI(CompletedStepLine(1, "Name", "Alice"))
	if !strings.Contains(got, "01") {
		t.Errorf("CompletedStepLine = %q, want to contain '01'", got)
	}
	if !strings.Contains(got, "Name") {
		t.Errorf("CompletedStepLine = %q, want to contain 'Name'", got)
	}
	if !strings.Contains(got, "Alice") {
		t.Errorf("CompletedStepLine = %q, want to contain 'Alice'", got)
	}
	if !strings.Contains(got, "\u2713") {
		t.Errorf("CompletedStepLine = %q, want to contain checkmark", got)
	}
}

func TestHeader(t *testing.T) {
	t.Helper()
	tests := []struct {
		name, version, versionLabel string
		wantParts                   []string
	}{
		{"myapp", "", "", []string{"myapp"}},
		{"myapp", "1.2.3", "", []string{"myapp", "1.2.3"}},
		{"myapp", "1.2.3", "Go", []string{"myapp", "Go v1.2.3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/"+tt.version, func(t *testing.T) {
			got := stripANSI(Header(tt.name, tt.version, tt.versionLabel))
			for _, want := range tt.wantParts {
				if !strings.Contains(got, want) {
					t.Errorf("Header(%q, %q, %q) = %q, want to contain %q",
						tt.name, tt.version, tt.versionLabel, got, want)
				}
			}
		})
	}
}

func TestDefaultTag(t *testing.T) {
	t.Helper()
	got := stripANSI(DefaultTag())
	if got != "(default)" {
		t.Errorf("DefaultTag() = %q, want '(default)'", got)
	}
}

func TestWarningText(t *testing.T) {
	t.Helper()
	got := stripANSI(WarningText("oops"))
	if got != "oops" {
		t.Errorf("WarningText('oops') = %q, want 'oops'", got)
	}
}

func TestWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"plain text", "hello", 5},
		{"styled text", GreenStyle.Render("hi"), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Width(tt.input)
			if got != tt.want {
				t.Errorf("Width(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestSuccessMsgf(t *testing.T) {
	var buf bytes.Buffer
	Output = &buf
	t.Cleanup(func() { Output = nil })

	SuccessMsgf("done %d items", 3)

	got := buf.String()
	plain := stripANSI(got)
	if !strings.Contains(plain, "done 3 items") {
		t.Errorf("SuccessMsgf output = %q, want to contain 'done 3 items'", plain)
	}
}

func TestInfoMsgf(t *testing.T) {
	var buf bytes.Buffer
	Output = &buf
	t.Cleanup(func() { Output = nil })

	InfoMsgf("found %s", "nothing")

	got := buf.String()
	plain := stripANSI(got)
	if !strings.Contains(plain, "found nothing") {
		t.Errorf("InfoMsgf output = %q, want to contain 'found nothing'", plain)
	}
}

func TestCompletedStepAnswer(t *testing.T) {
	tests := []struct {
		name, answer string
	}{
		{"simple", "yes"},
		{"with spaces", "hello world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(CompletedStepAnswer(tt.answer))
			if !strings.Contains(got, tt.answer) {
				t.Errorf("CompletedStepAnswer(%q) = %q, want to contain %q", tt.answer, got, tt.answer)
			}
		})
	}
}

func TestFieldTitle(t *testing.T) {
	got := stripANSI(FieldTitle("Project Name"))
	if !strings.Contains(got, "Project Name") {
		t.Errorf("FieldTitle = %q, want to contain 'Project Name'", got)
	}
}

func TestFieldDesc(t *testing.T) {
	got := stripANSI(FieldDesc("Enter your name"))
	if !strings.Contains(got, "Enter your name") {
		t.Errorf("FieldDesc = %q, want to contain 'Enter your name'", got)
	}
}

func TestChoiceDesc(t *testing.T) {
	got := stripANSI(ChoiceDesc("a fine choice"))
	if !strings.Contains(got, "a fine choice") {
		t.Errorf("ChoiceDesc = %q, want to contain 'a fine choice'", got)
	}
}

func TestCursor(t *testing.T) {
	got := stripANSI(Cursor())
	if !strings.Contains(got, "\u203a") {
		t.Errorf("Cursor() = %q, want to contain '\u203a'", got)
	}
}

func TestChoiceLabel(t *testing.T) {
	tests := []struct {
		name   string
		label  string
		active bool
	}{
		{"active", "Option A", true},
		{"inactive", "Option A", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(ChoiceLabel(tt.label, tt.active))
			if !strings.Contains(got, tt.label) {
				t.Errorf("ChoiceLabel(%q, %v) = %q, want to contain %q", tt.label, tt.active, got, tt.label)
			}
		})
	}

	// Raw output should differ between active and inactive.
	rawActive := ChoiceLabel("X", true)
	rawInactive := ChoiceLabel("X", false)
	if rawActive == rawInactive {
		t.Error("ChoiceLabel active and inactive should have different ANSI styles")
	}
}

func TestVersionVerifyingTag(t *testing.T) {
	got := stripANSI(VersionVerifyingTag("\u2807"))
	if !strings.Contains(got, "verifying") {
		t.Errorf("VersionVerifyingTag = %q, want to contain 'verifying'", got)
	}
}

func TestVersionOverrideTag(t *testing.T) {
	got := stripANSI(VersionOverrideTag())
	if !strings.Contains(got, "override") {
		t.Errorf("VersionOverrideTag = %q, want to contain 'override'", got)
	}
}

func TestPinIcon(t *testing.T) {
	got := stripANSI(PinIcon())
	if !strings.Contains(got, "\u25cf") {
		t.Errorf("PinIcon() = %q, want to contain '\u25cf'", got)
	}
}

func TestPinEditIndicator(t *testing.T) {
	got := stripANSI(PinEditIndicator())
	if !strings.Contains(got, "pin") {
		t.Errorf("PinEditIndicator() = %q, want to contain 'pin'", got)
	}
}

func TestPresetEditIndicator(t *testing.T) {
	got := stripANSI(PresetEditIndicator())
	if !strings.Contains(got, "preset") {
		t.Errorf("PresetEditIndicator() = %q, want to contain 'preset'", got)
	}
}

func TestNavHintText(t *testing.T) {
	got := stripANSI(NavHintText("press enter"))
	if !strings.Contains(got, "press enter") {
		t.Errorf("NavHintText = %q, want to contain 'press enter'", got)
	}
}
