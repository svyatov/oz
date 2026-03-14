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
	rule            *config.InputRule
	compiledPattern *regexp.Regexp
	label           string
	description     string
	errMsg          string
	ti              textinput.Model
	required        bool
}

// NewInputField creates an InputField from a config option.
func NewInputField(opt config.Option) *InputField {
	ti := textinput.New()
	f := &InputField{
		label:       opt.Label,
		description: opt.Description,
		ti:          ti,
		rule:        opt.Validate,
		required:    opt.Required,
	}
	if opt.Validate != nil && opt.Validate.Pattern != "" {
		// Pattern already validated by config.Validate; compile error is impossible here.
		f.compiledPattern, _ = regexp.Compile(opt.Validate.Pattern)
	}
	return f
}

func (f *InputField) Init() tea.Cmd {
	return f.ti.Focus()
}

func (f *InputField) Update(msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case keyEnter, keyTab:
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

	b.WriteString(fieldHeader(f.label, f.description))
	b.WriteString("    " + f.ti.View() + "\n")

	if f.errMsg != "" {
		b.WriteString("    " + ui.WarningText(f.errMsg) + "\n")
	}

	return b.String()
}

func (f *InputField) Value() config.FieldValue { return config.StringVal(f.ti.Value()) }

func (f *InputField) SetValue(v config.FieldValue) {
	f.ti.SetValue(v.Scalar())
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
