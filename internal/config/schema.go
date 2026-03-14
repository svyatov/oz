// Package config defines wizard configuration types and handles YAML parsing and validation.
package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// OptionType identifies the kind of wizard option.
type OptionType string

const (
	OptionSelect      OptionType = "select"
	OptionConfirm     OptionType = "confirm"
	OptionInput       OptionType = "input"
	OptionMultiSelect OptionType = "multi_select"
)

// IsValid reports whether t is a recognized option type.
func (t OptionType) IsValid() bool {
	switch t {
	case OptionSelect, OptionConfirm, OptionInput, OptionMultiSelect:
		return true
	default:
		return false
	}
}

// FlagStyle controls how flags are formatted (--flag=value vs --flag value).
type FlagStyle string

const (
	FlagStyleEquals FlagStyle = "equals"
	FlagStyleSpace  FlagStyle = "space"
)

// NoneValue is the sentinel value for "no selection" in select fields.
const NoneValue = "_none"

// Wizard is the top-level YAML config for a wizard.
type Wizard struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Command     string          `yaml:"command"`
	FlagStyle   FlagStyle       `yaml:"flag_style"` // "equals" (default) or "space"
	Version *VersionControl `yaml:"version_control"`
	Options []Option        `yaml:"options"`
}

// EffectiveFlagStyle returns the wizard-level flag style, defaulting to equals.
func (w *Wizard) EffectiveFlagStyle() FlagStyle {
	if w.FlagStyle == FlagStyleSpace {
		return FlagStyleSpace
	}
	return FlagStyleEquals
}

// VersionControl configures version detection and custom version support.
type VersionControl struct {
	Label               string `yaml:"label"`
	Command             string `yaml:"command"`
	Pattern             string `yaml:"pattern"`
	CustomVersionCmd    string `yaml:"custom_version_command"`
	CustomVersionVerify string `yaml:"custom_version_verify_command"`
	AvailVersionsCmd    string `yaml:"available_versions_command"`
	AvailVersions       string `yaml:"available_versions"`
}

// EffectiveCommand returns the command template expanded with version,
// or the wizard's base command if no template is set.
func (w *Wizard) EffectiveCommand(version string) string {
	if version != "" && w.Version != nil && w.Version.CustomVersionCmd != "" {
		return strings.ReplaceAll(w.Version.CustomVersionCmd, "{{version}}", version)
	}
	return w.Command
}

// Option is a single wizard step.
type Option struct {
	Default     *FieldValue `yaml:"default,omitempty"`
	Validate    *InputRule  `yaml:"validate"`
	ShowWhen    Values      `yaml:"show_when,omitempty"`
	HideWhen    Values      `yaml:"hide_when,omitempty"`
	FlagNone    string      `yaml:"flag_none"`
	FlagFalse   string      `yaml:"flag_false"`
	Type        OptionType  `yaml:"type"`
	Label       string      `yaml:"label"`
	Description string      `yaml:"description"`
	Flag        string      `yaml:"flag"`
	FlagTrue    string      `yaml:"flag_true"`
	Name        string      `yaml:"name"`
	Versions    string      `yaml:"versions"`
	Separator   string      `yaml:"separator"`
	FlagStyle   FlagStyle   `yaml:"flag_style"`
	ChoicesFrom string      `yaml:"choices_from"`
	Choices     FlexChoices `yaml:"choices"`
	AllowNone   bool        `yaml:"allow_none"`
	Required    bool        `yaml:"required"`
	Positional  bool        `yaml:"positional"`
}

// EffectiveFlagStyle returns the option-level flag style, falling back to the wizard default.
func (o *Option) EffectiveFlagStyle(wizardDefault FlagStyle) FlagStyle {
	if o.FlagStyle != "" {
		return o.FlagStyle
	}
	return wizardDefault
}

// Choice is a single option in a select or multi_select.
type Choice struct {
	Value       string `yaml:"value"`
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
	Versions    string `yaml:"versions"`
}

// FlexChoices is a []Choice that accepts both string shorthand and full object syntax in YAML.
type FlexChoices []Choice

// UnmarshalYAML parses choices from both string shorthand and full object YAML syntax.
func (fc *FlexChoices) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("choices must be a sequence, got %v", value.Kind)
	}

	choices := make([]Choice, 0, len(value.Content))
	for _, node := range value.Content {
		switch node.Kind { //nolint:exhaustive // only scalar and mapping are valid
		case yaml.ScalarNode:
			s := node.Value
			choices = append(choices, Choice{Value: s, Label: s})
		case yaml.MappingNode:
			var c Choice
			if err := node.Decode(&c); err != nil {
				return fmt.Errorf("decoding choice: %w", err)
			}
			choices = append(choices, c)
		default:
			return fmt.Errorf("choice must be a string or mapping, got %v", node.Kind)
		}
	}
	*fc = choices
	return nil
}

// InputRule defines validation constraints for input fields.
type InputRule struct {
	Pattern   string `yaml:"pattern"`
	Message   string `yaml:"message"`
	MinLength int    `yaml:"min_length"`
	MaxLength int    `yaml:"max_length"`
}
