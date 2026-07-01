package wizard

import (
	"math"
	"strconv"

	"github.com/svyatov/oz/internal/config"
)

// NumberField is an InputField that accepts only numeric input, with optional
// inclusive min/max bounds. It reuses InputField for rendering and value
// handling; only entry validation differs.
type NumberField struct {
	*InputField
	min *float64
	max *float64
}

// NewNumberField creates a numeric field from a config option. It reuses
// InputField's Update loop, routing entry validation back to numeric validate
// via the validateFn hook (Go embedding has no virtual dispatch).
func NewNumberField(opt config.Option) *NumberField {
	f := &NumberField{InputField: NewInputField(opt), min: opt.Min, max: opt.Max}
	f.validateFn = f.validate
	return f
}

func (f *NumberField) validate() string {
	val := f.ti.Value()

	if f.required && val == "" {
		return f.ruleMessage("This field is required")
	}
	if val == "" {
		return "" // optional blank omits the flag.
	}

	n, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return f.ruleMessage("Must be a number")
	}
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return f.ruleMessage("Must be a finite number") // NaN/Inf slip past min/max comparisons.
	}
	if f.min != nil && n < *f.min {
		return f.ruleMessage("Must be at least " + formatBound(*f.min))
	}
	if f.max != nil && n > *f.max {
		return f.ruleMessage("Must be at most " + formatBound(*f.max))
	}

	return f.InputField.validate() // honor length/pattern rules if set.
}

// formatBound renders a bound without a trailing ".0" for whole numbers.
func formatBound(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
