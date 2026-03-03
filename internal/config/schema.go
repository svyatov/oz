package config

// Wizard is the top-level YAML config for a wizard.
type Wizard struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Command     string          `yaml:"command"`
	FlagStyle   string          `yaml:"flag_style"` // "equals" (default) or "space"
	Args        []Arg           `yaml:"args"`
	Detect      *DetectVersion  `yaml:"detect_version"`
	Compat      []CompatEntry   `yaml:"compat"`
	Options     []Option        `yaml:"options"`
}

func (w *Wizard) EffectiveFlagStyle() string {
	if w.FlagStyle == "space" {
		return "space"
	}
	return "equals"
}

// Arg is a positional argument for the command.
type Arg struct {
	Name     string `yaml:"name"`
	Label    string `yaml:"label"`
	Required bool   `yaml:"required"`
	Position int    `yaml:"position"`
}

// DetectVersion configures automatic version detection.
type DetectVersion struct {
	Command string `yaml:"command"`
	Pattern string `yaml:"pattern"`
}

// CompatEntry maps a version range to allowed option names.
type CompatEntry struct {
	Versions string   `yaml:"versions"`
	Options  []string `yaml:"options"`
}

// Option is a single wizard step.
type Option struct {
	Name        string            `yaml:"name"`
	Type        string            `yaml:"type"` // select, confirm, input, multi_select
	Label       string            `yaml:"label"`
	Description string            `yaml:"description"`
	Flag        string            `yaml:"flag"`
	FlagTrue    string            `yaml:"flag_true"`
	FlagFalse   string            `yaml:"flag_false"`
	FlagNone    string            `yaml:"flag_none"`
	FlagStyle   string            `yaml:"flag_style"` // per-option override
	Default     any               `yaml:"default"`
	AllowNone   bool              `yaml:"allow_none"`
	Required    bool              `yaml:"required"`
	ShowWhen    map[string]any    `yaml:"show_when"`
	Choices     []Choice          `yaml:"choices"`
}

func (o *Option) EffectiveFlagStyle(wizardDefault string) string {
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
}
