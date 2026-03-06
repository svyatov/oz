package wizard

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// InputField wraps bubbles/textinput for free-text entry.
type InputField struct {
	label           string
	description     string
	ti              textinput.Model
	rule            *config.InputRule
	compiledPattern *regexp.Regexp
	required        bool
	errMsg          string
}

func NewInputField(label, description string, rule *config.InputRule, required bool) *InputField {
	ti := textinput.New()
	f := &InputField{
		label:       label,
		description: description,
		ti:          ti,
		rule:        rule,
		required:    required,
	}
	if rule != nil && rule.Pattern != "" {
		f.compiledPattern, _ = regexp.Compile(rule.Pattern)
	}
	return f
}

func (f *InputField) Init() tea.Cmd {
	return f.ti.Focus()
}

func (f *InputField) Update(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "enter", "tab":
		if err := f.validate(); err != "" {
			f.errMsg = err
			return false, nil
		}
		return true, nil
	}

	f.errMsg = ""
	var cmd tea.Cmd
	f.ti, cmd = f.ti.Update(msg)
	return false, cmd
}

func (f *InputField) View() string {
	var b strings.Builder

	b.WriteString("  " + ui.StepCounter(0, 0) + "  ")
	b.WriteString(ui.FieldTitle(f.label) + "\n")
	if f.description != "" {
		b.WriteString("         " + ui.FieldDesc(f.description) + "\n")
	}
	b.WriteString("\n")
	b.WriteString("    " + f.ti.View() + "\n")

	if f.errMsg != "" {
		b.WriteString("    " + ui.WarningText(f.errMsg) + "\n")
	}

	return b.String()
}

func (f *InputField) Value() any { return f.ti.Value() }

func (f *InputField) SetValue(v any) {
	if v != nil {
		f.ti.SetValue(fmt.Sprintf("%v", v))
	}
}

func (f *InputField) validate() string {
	val := f.ti.Value()

	if f.required && val == "" {
		return f.ruleMessage("This field is required")
	}

	if f.rule == nil || val == "" {
		return ""
	}

	if f.rule.MinLength > 0 && len(val) < f.rule.MinLength {
		return f.ruleMessage(fmt.Sprintf("Must be at least %d characters", f.rule.MinLength))
	}

	if f.rule.MaxLength > 0 && len(val) > f.rule.MaxLength {
		return f.ruleMessage(fmt.Sprintf("Must be at most %d characters", f.rule.MaxLength))
	}

	return f.validatePattern(val)
}

func (f *InputField) validatePattern(val string) string {
	if f.compiledPattern == nil {
		return ""
	}
	if !f.compiledPattern.MatchString(val) {
		return f.ruleMessage("Must match pattern: " + f.rule.Pattern)
	}
	return ""
}

func (f *InputField) ruleMessage(fallback string) string {
	if f.rule != nil && f.rule.Message != "" {
		return f.rule.Message
	}
	return fallback
}
