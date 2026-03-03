package config

import (
	"fmt"
	"strings"
)

// Validate checks a wizard config for errors.
func Validate(w *Wizard) []error {
	var errs []error
	add := func(msg string, args ...any) {
		errs = append(errs, fmt.Errorf(msg, args...))
	}

	if w.Name == "" {
		add("name is required")
	}
	if w.Command == "" {
		add("command is required")
	}
	if w.FlagStyle != "" && w.FlagStyle != "equals" && w.FlagStyle != "space" {
		add("flag_style must be 'equals' or 'space', got %q", w.FlagStyle)
	}

	// Validate args
	for i, a := range w.Args {
		if a.Name == "" {
			add("args[%d]: name is required", i)
		}
		if a.Position < 1 {
			add("args[%d] (%s): position must be >= 1", i, a.Name)
		}
	}

	// Validate detect_version
	if w.Detect != nil {
		if w.Detect.Command == "" {
			add("detect_version.command is required")
		}
		if w.Detect.Pattern == "" {
			add("detect_version.pattern is required")
		}
	}

	// Validate compat requires detect_version
	if len(w.Compat) > 0 && w.Detect == nil {
		add("compat requires detect_version to be set")
	}

	// Build option name set for reference validation
	optionNames := make(map[string]bool)
	validTypes := map[string]bool{
		"select": true, "confirm": true, "input": true, "multi_select": true,
	}

	for i, o := range w.Options {
		prefix := fmt.Sprintf("options[%d]", i)
		if o.Name != "" {
			prefix = fmt.Sprintf("options[%d] (%s)", i, o.Name)
		}

		if o.Name == "" {
			add("%s: name is required", prefix)
		} else if optionNames[o.Name] {
			add("%s: duplicate option name", prefix)
		}
		optionNames[o.Name] = true

		if !validTypes[o.Type] {
			add("%s: type must be one of select, confirm, input, multi_select; got %q", prefix, o.Type)
		}
		if o.Label == "" {
			add("%s: label is required", prefix)
		}
		if o.FlagStyle != "" && o.FlagStyle != "equals" && o.FlagStyle != "space" {
			add("%s: flag_style must be 'equals' or 'space', got %q", prefix, o.FlagStyle)
		}

		// select / multi_select need choices
		if (o.Type == "select" || o.Type == "multi_select") && len(o.Choices) == 0 {
			add("%s: choices are required for type %q", prefix, o.Type)
		}

		// Validate choices
		for j, c := range o.Choices {
			if c.Value == "" {
				add("%s: choices[%d]: value is required", prefix, j)
			}
		}
	}

	// Validate show_when references
	for i, o := range w.Options {
		for ref := range o.ShowWhen {
			if !optionNames[ref] {
				add("options[%d] (%s): show_when references unknown option %q", i, o.Name, ref)
			}
		}
	}

	// Validate compat option references
	for i, c := range w.Compat {
		for _, name := range c.Options {
			if !optionNames[name] {
				add("compat[%d]: references unknown option %q", i, name)
			}
		}
	}

	return errs
}

// FormatErrors formats validation errors as a single string.
func FormatErrors(errs []error) string {
	if len(errs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, e := range errs {
		fmt.Fprintf(&b, "  - %s\n", e)
	}
	return b.String()
}
