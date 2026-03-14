package wizard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

// ValuesEditor provides shared value-editing operations for pins and presets TUIs.
// Parents handle cursor navigation and list composition; the editor handles
// field editing, value cycling/toggling, and option row rendering.
type ValuesEditor struct {
	editField    Field
	values       config.Values
	lastUsed     config.Values
	hints        map[string]string
	options      []config.Option
	cursor       int
	editIdx      int
	editing      bool
	showRequired bool // show * marker on required options
}

// NewValuesEditor creates a ValuesEditor for the given options and values.
func NewValuesEditor(
	options []config.Option, values, lastUsed config.Values, hints map[string]string,
) *ValuesEditor {
	if values == nil {
		values = make(config.Values)
	}
	if lastUsed == nil {
		lastUsed = make(config.Values)
	}
	return &ValuesEditor{
		options:  options,
		values:   values,
		lastUsed: lastUsed,
		hints:    hints,
	}
}

// Editing returns true if a field is being edited.
func (e *ValuesEditor) Editing() bool { return e.editing }

// Values returns the current values map.
func (e *ValuesEditor) Values() config.Values { return e.values }

// MaxLabelWidth returns the maximum label width across all options.
func (e *ValuesEditor) MaxLabelWidth() int {
	maxW := 0
	for _, o := range e.options {
		w := len(o.Label)
		if e.showRequired && o.Required {
			w += 2 // " *"
		}
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}

// EnterEdit starts editing the option at the given index.
func (e *ValuesEditor) EnterEdit(idx int) tea.Cmd {
	e.editIdx = idx
	opt := &e.options[idx]
	e.editField = buildField(opt)

	switch f := e.editField.(type) {
	case *SelectField:
		if opt.Default != nil {
			f.SetDefault(*opt.Default)
		}
	case *ConfirmField:
		defVal := config.BoolVal(false)
		if opt.Default != nil {
			defVal = *opt.Default
		}
		f.SetDefault(defVal)
	}

	val := resolveDefault(opt, e.values, e.lastUsed)
	if val != nil {
		e.editField.SetValue(*val)
	}

	e.editing = true
	return e.editField.Init()
}

// UpdateEdit handles a key press during field editing.
// Returns true when editing ends (submitted or cancelled).
func (e *ValuesEditor) UpdateEdit(msg tea.KeyPressMsg) (exited bool, cmd tea.Cmd) {
	if msg.String() == keyEsc || msg.String() == keyCtrlC {
		e.editing = false
		e.editField = nil
		return true, nil
	}

	submitted, cmd := e.editField.Update(msg)
	if submitted {
		opt := &e.options[e.editIdx]
		e.values[opt.Name] = e.editField.Value()
		e.editing = false
		e.editField = nil
		return true, cmd
	}
	return false, cmd
}

// CycleValue cycles the value at the given index in the specified direction.
func (e *ValuesEditor) CycleValue(idx, direction int) {
	opt := &e.options[idx]
	vals := cyclableValues(opt)
	if len(vals) == 0 {
		return
	}

	name := opt.Name
	total := len(vals) + 1 // +1 for the "no value" slot

	pos := len(vals) // default: no value
	if current, has := e.values[name]; has {
		for i, v := range vals {
			if v.Kind() == current.Kind() && v.Scalar() == current.Scalar() {
				pos = i
				break
			}
		}
	}

	newPos := (pos + direction + total) % total
	if newPos == len(vals) {
		delete(e.values, name)
	} else {
		e.values[name] = vals[newPos]
	}
}

// ToggleValue toggles the value at the given index.
// May enter edit mode if the option requires validation.
func (e *ValuesEditor) ToggleValue(idx int) tea.Cmd {
	name := e.options[idx].Name
	if _, has := e.values[name]; has {
		delete(e.values, name)
		return nil
	}

	opt := &e.options[idx]
	val := resolveDefault(opt, e.values, e.lastUsed)
	if opt.Type == config.OptionInput && !e.isValidInputValue(opt, val) {
		return e.EnterEdit(idx)
	}
	if val != nil {
		e.values[name] = *val
	}
	return nil
}

func (e *ValuesEditor) isValidInputValue(opt *config.Option, val *config.FieldValue) bool {
	if val == nil {
		return !opt.Required
	}
	s := val.Scalar()
	if opt.Required && s == "" {
		return false
	}
	if opt.Validate == nil || s == "" {
		return true
	}
	f := NewInputField(config.Option{Validate: opt.Validate, Required: opt.Required})
	f.SetValue(*val)
	return f.validate() == ""
}

// cyclableValues returns the ordered values for cycling through an option.
// Returns nil for option types that don't support cycling (input, multi_select).
func cyclableValues(opt *config.Option) []config.FieldValue {
	switch opt.Type {
	case config.OptionSelect:
		vals := make([]config.FieldValue, 0, len(opt.Choices)+1)
		for _, c := range opt.Choices {
			vals = append(vals, config.StringVal(c.Value))
		}
		if opt.AllowNone {
			vals = append(vals, config.StringVal(config.NoneValue))
		}
		return vals
	case config.OptionConfirm:
		return []config.FieldValue{config.BoolVal(true), config.BoolVal(false)}
	case config.OptionInput, config.OptionMultiSelect:
		return nil
	}
	return nil
}

// ViewOptionRow renders a single option row for the list view.
func (e *ValuesEditor) ViewOptionRow(
	idx int, active bool, maxLabel, gutterWidth, displayNum int,
) string {
	num := ui.NumberGutter(displayNum, gutterWidth, active)
	cursor := cursorBlank
	if active {
		cursor = " " + ui.Cursor() + " "
	}
	o := e.options[idx]
	val, hasVal := e.values[o.Name]
	icon := "  "
	if hasVal {
		icon = ui.PinIcon() + " "
	}
	displayLabel := o.Label
	if e.showRequired && o.Required {
		displayLabel += " *"
	}
	label := ui.ChoiceLabel(displayLabel, active)
	pad := strings.Repeat(" ", maxLabel-len(displayLabel))
	value := ui.MutedStyle.Render("\u2500")
	if hasVal {
		value = ui.CompletedStepAnswer(FormatAnswer(&o, val))
	}
	hint := ""
	if h := e.hints[o.Name]; h != "" {
		hint = " " + ui.NavHintText(h)
	}
	return fmt.Sprintf("   %s%s  %s%s%s  %s%s\n", cursor, num, icon, label, pad, value, hint)
}

// ViewEdit renders the field editing view with the given indicator label.
func (e *ValuesEditor) ViewEdit(indicator string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderFieldWithIndicator(e.editField.View(), indicator))
	b.WriteString("\n" + e.editNavHint() + "\n")
	return b.String()
}

// editNavHint returns the appropriate nav hint for the current edit field type.
func (e *ValuesEditor) editNavHint() string {
	switch e.editField.(type) {
	case *SelectField, *ConfirmField:
		return ui.NavHints(ui.HintUp, ui.HintDown, ui.HintEnter, ui.HintEsc)
	case *MultiSelectField:
		return ui.NavHints(ui.HintUp, ui.HintDown, ui.Hint{Key: "space", Desc: "toggle"}, ui.HintEnter, ui.HintEsc)
	default:
		return ui.NavHints(ui.HintEnter, ui.HintEsc)
	}
}

// renderFieldWithIndicator replaces the step counter placeholder with the given indicator.
func renderFieldWithIndicator(fieldView, indicator string) string {
	placeholder := ui.StepCounter(0, 0)
	fieldView = strings.Replace(fieldView, placeholder, indicator, 1)
	oldIndent := "\n" + strings.Repeat(" ", 2+ui.Width(placeholder)+2)
	newIndent := "\n" + strings.Repeat(" ", 2+ui.Width(indicator)+2)
	return strings.Replace(fieldView, oldIndent, newIndent, 1)
}
