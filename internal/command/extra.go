package command

import (
	"strings"

	"github.com/svyatov/oz/internal/config"
)

// flagEntry maps a CLI flag to a wizard option and an action.
type flagEntry struct {
	name    string // option name
	setBool bool   // value to set for confirm options
}

type extraMaps struct {
	valueFlags   map[string]string     // flag → option name (value-bearing)
	confirmFlags map[string]flagEntry  // flag → confirm entry
	positionals  []string              // ordered positional option names
}

// ParseExtra matches CLI args (after --) against wizard option definitions.
// Matched args are returned as values to merge with preset/wizard answers.
// Unmatched args are returned separately for raw append to the command.
func ParseExtra(options []config.Option, args []string) (config.Values, []string) {
	if len(args) == 0 {
		return nil, nil
	}

	m := buildExtraMaps(options)
	values := make(config.Values)
	var raw []string
	posIdx := 0

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if !strings.HasPrefix(arg, "-") {
			if posIdx < len(m.positionals) {
				values[m.positionals[posIdx]] = config.StringVal(arg)
				posIdx++
			} else {
				raw = append(raw, arg)
			}
			continue
		}

		flag, val, hasEq := strings.Cut(arg, "=")

		if entry, ok := m.confirmFlags[flag]; ok && !hasEq {
			values[entry.name] = config.BoolVal(entry.setBool)
			continue
		}

		if optName, ok := m.valueFlags[flag]; ok {
			if !hasEq && i+1 < len(args) {
				i++
				val = args[i]
			}
			appendValue(values, optName, val, optionType(options, optName))
			continue
		}

		raw = append(raw, arg)
	}

	return values, raw
}

func buildExtraMaps(options []config.Option) extraMaps {
	m := extraMaps{
		valueFlags:   make(map[string]string),
		confirmFlags: make(map[string]flagEntry),
	}
	for i := range options {
		opt := &options[i]
		if opt.Positional {
			m.positionals = append(m.positionals, opt.Name)
			continue
		}
		if opt.Type == config.OptionConfirm {
			registerConfirmFlags(m.confirmFlags, opt)
			continue
		}
		if opt.Flag != "" {
			m.valueFlags[opt.Flag] = opt.Name
		}
	}
	return m
}

// registerConfirmFlags adds flag_true, flag (as shorthand), and flag_false entries.
func registerConfirmFlags(flags map[string]flagEntry, opt *config.Option) {
	if opt.FlagTrue != "" {
		flags[opt.FlagTrue] = flagEntry{name: opt.Name, setBool: true}
	} else if opt.Flag != "" {
		// Shorthand: flag acts as flag_true when flag_true is empty.
		flags[opt.Flag] = flagEntry{name: opt.Name, setBool: true}
	}
	if opt.FlagFalse != "" {
		flags[opt.FlagFalse] = flagEntry{name: opt.Name, setBool: false}
	}
}

// appendValue sets or accumulates a value for an option.
// Multi-select options accumulate repeated flags into a StringsVal.
func appendValue(values config.Values, name, val string, typ config.OptionType) {
	if typ == config.OptionMultiSelect {
		if existing, ok := values[name]; ok && existing.IsStrings() {
			values[name] = config.StringsVal(append(existing.Strings(), val)...)
		} else {
			values[name] = config.StringsVal(val)
		}
		return
	}
	values[name] = config.StringVal(val)
}

// optionType returns the type of an option by name.
func optionType(options []config.Option, name string) config.OptionType {
	for i := range options {
		if options[i].Name == name {
			return options[i].Type
		}
	}
	return config.OptionInput
}
