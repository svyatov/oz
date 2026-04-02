package wizard

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/svyatov/oz/internal/config"
	"github.com/svyatov/oz/internal/ui"
)

type presetsMode int

const (
	presetsListMode   presetsMode = iota // browsing preset names
	presetsValuesMode                    // editing values within a preset
	presetsNameMode                      // text input for new/rename
	presetsSourceMode                    // select source for new preset
	presetsDeleteMode                    // inline y/n confirmation
)

// PresetsResult is returned by RunPresets.
type PresetsResult struct {
	Presets map[string]config.Values
}

// PresetsModel is a Bubbletea model for interactive preset management.
type PresetsModel struct {
	editor        *ValuesEditor
	options       []config.Option
	lastUsed      config.Values
	hints         map[string]string
	presets       map[string]config.Values
	presetNames   []string
	sourceItems   []string
	nameErr       string
	activeName    string // currently selected preset
	renamingFrom  string // non-empty during rename
	valuesWarning string // missing-values warning text
	nameInput     textinput.Model
	mode          presetsMode
	cursor        int
	sourceCursor  int
	exitWarned    bool // true after first Esc with missing values
	done          bool
}

func newPresetsModel(
	options []config.Option, presets map[string]config.Values,
	lastUsed config.Values, hints map[string]string,
) *PresetsModel {
	if presets == nil {
		presets = make(map[string]config.Values)
	}
	if lastUsed == nil {
		lastUsed = make(config.Values)
	}
	names := sortedPresetNames(presets)

	ti := textinput.New()
	ti.Prompt = "  " + ui.AccentStyle.Render("Name: ")
	ti.CharLimit = 64

	return &PresetsModel{
		options:     options,
		presets:     presets,
		presetNames: names,
		lastUsed:    lastUsed,
		hints:       hints,
		nameInput:   ti,
	}
}

func sortedPresetNames(presets map[string]config.Values) []string {
	names := make([]string, 0, len(presets))
	for k := range presets {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func (m *PresetsModel) Init() tea.Cmd { return nil }

func (m *PresetsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch m.mode {
	case presetsListMode:
		return m.updateList(keyMsg)
	case presetsValuesMode:
		return m.updateValues(keyMsg)
	case presetsNameMode:
		return m.updateName(keyMsg)
	case presetsSourceMode:
		return m.updateSource(keyMsg)
	case presetsDeleteMode:
		return m.updateDelete(keyMsg)
	}
	return m, nil
}

func (m *PresetsModel) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	n := len(m.presetNames)

	switch msg.String() {
	case keyUp, "k":
		if n > 0 {
			m.cursor = (m.cursor - 1 + n) % n
		}
	case keyDown, "j":
		if n > 0 {
			m.cursor = (m.cursor + 1) % n
		}
	case keyEnter:
		if n > 0 {
			return m.enterPreset()
		}
	case "n":
		return m.startNameInput("")
	case "r":
		if n > 0 {
			return m.startNameInput(m.presetNames[m.cursor])
		}
	case "d":
		if n > 0 {
			m.mode = presetsDeleteMode
		}
	case keyEsc, keyCtrlC:
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m *PresetsModel) enterPreset() (tea.Model, tea.Cmd) {
	name := m.presetNames[m.cursor]
	m.activeName = name
	values := m.presets[name]
	if values == nil {
		values = make(config.Values)
	}
	m.editor = NewValuesEditor(m.options, values, m.lastUsed, m.hints)
	m.editor.showRequired = true
	m.mode = presetsValuesMode
	return m, nil
}

func (m *PresetsModel) startNameInput(renamingFrom string) (tea.Model, tea.Cmd) {
	m.renamingFrom = renamingFrom
	m.nameInput.SetValue("")
	if renamingFrom != "" {
		m.nameInput.SetValue(renamingFrom)
	}
	m.nameErr = ""
	m.mode = presetsNameMode
	return m, m.nameInput.Focus()
}

func (m *PresetsModel) updateValues(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.editor.Editing() {
		_, cmd := m.editor.UpdateEdit(msg)
		return m, cmd
	}
	return m.updateValuesNav(msg)
}

func (m *PresetsModel) updateValuesNav(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	n := len(m.options)

	switch msg.String() {
	case keyUp, "k":
		if n > 0 {
			m.editor.cursor = (m.editor.cursor - 1 + n) % n
		}
	case keyDown, "j":
		if n > 0 {
			m.editor.cursor = (m.editor.cursor + 1) % n
		}
	case keyLeft, "h":
		m.editor.CycleValue(m.editor.cursor, -1)
	case keyRight, "l":
		m.editor.CycleValue(m.editor.cursor, 1)
	case keyEnter:
		if n > 0 {
			return m, m.editor.EnterEdit(m.editor.cursor)
		}
	case keySpace:
		if n > 0 {
			return m, m.editor.ToggleValue(m.editor.cursor)
		}
	case keyEsc, keyCtrlC:
		return m.tryExitValues()
	}

	if msg.Code >= '1' && msg.Code <= '9' {
		idx := int(msg.Code-'0') - 1
		if idx < n {
			m.editor.cursor = idx
			return m, m.editor.EnterEdit(idx)
		}
	}

	// Clear the warning after any value-changing action.
	m.exitWarned = false
	m.valuesWarning = ""
	return m, nil
}

func (m *PresetsModel) tryExitValues() (tea.Model, tea.Cmd) {
	if m.exitWarned {
		// Second Esc — force exit without saving.
		m.exitWarned = false
		m.valuesWarning = ""
		m.editor = nil
		m.mode = presetsListMode
		return m, nil
	}

	missing := m.missingRequiredOptions()
	if len(missing) == 0 {
		m.presets[m.activeName] = m.editor.Values()
		m.editor = nil
		m.mode = presetsListMode
		return m, nil
	}

	m.exitWarned = true
	m.valuesWarning = "Required options not set: " + strings.Join(missing, ", ") + " (esc again to leave)"
	return m, nil
}

// missingRequiredOptions returns labels of visible required options that have no value.
func (m *PresetsModel) missingRequiredOptions() []string {
	return MissingRequired(m.options, m.editor.Values())
}

func (m *PresetsModel) updateName(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc, keyCtrlC:
		m.mode = presetsListMode
		m.nameErr = ""
		return m, nil
	case keyEnter:
		return m.submitName()
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	m.nameErr = ""
	return m, cmd
}

func (m *PresetsModel) submitName() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.nameInput.Value())
	if name == "" {
		m.nameErr = "Name must not be empty"
		return m, nil
	}
	if !filepath.IsLocal(name) {
		m.nameErr = "Name must not contain path separators or '..'"
		return m, nil
	}

	if m.renamingFrom != "" {
		return m.finishRename(name)
	}

	// Check for duplicate.
	if _, exists := m.presets[name]; exists {
		m.nameErr = fmt.Sprintf("Preset %q already exists", name)
		return m, nil
	}

	m.activeName = name
	m.buildSourceItems()
	if len(m.sourceItems) == 1 {
		// Only "Empty" — skip source selection.
		return m.createPreset(0)
	}
	m.sourceCursor = 0
	m.mode = presetsSourceMode
	return m, nil
}

func (m *PresetsModel) finishRename(newName string) (tea.Model, tea.Cmd) {
	if newName == m.renamingFrom {
		m.mode = presetsListMode
		return m, nil
	}
	if _, exists := m.presets[newName]; exists {
		m.nameErr = fmt.Sprintf("Preset %q already exists", newName)
		return m, nil
	}

	values := m.presets[m.renamingFrom]
	delete(m.presets, m.renamingFrom)
	m.presets[newName] = values
	m.presetNames = sortedPresetNames(m.presets)
	if idx := slices.Index(m.presetNames, newName); idx >= 0 {
		m.cursor = idx
	}
	m.mode = presetsListMode
	return m, nil
}

func (m *PresetsModel) buildSourceItems() {
	m.sourceItems = []string{"Empty"}
	if len(m.lastUsed) > 0 {
		m.sourceItems = append(m.sourceItems, "Last-used values")
	}
	for _, name := range m.presetNames {
		m.sourceItems = append(m.sourceItems, "Copy: "+name)
	}
}

func (m *PresetsModel) updateSource(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	n := len(m.sourceItems)

	switch msg.String() {
	case keyUp, "k":
		m.sourceCursor = (m.sourceCursor - 1 + n) % n
	case keyDown, "j":
		m.sourceCursor = (m.sourceCursor + 1) % n
	case keyEnter:
		return m.createPreset(m.sourceCursor)
	case keyEsc, keyCtrlC:
		m.mode = presetsListMode
		return m, nil
	}

	if msg.Code >= '1' && msg.Code <= '9' {
		idx := int(msg.Code-'0') - 1
		if idx < n {
			return m.createPreset(idx)
		}
	}

	return m, nil
}

func (m *PresetsModel) createPreset(sourceIdx int) (tea.Model, tea.Cmd) {
	var values config.Values

	switch item := m.sourceItems[sourceIdx]; item {
	case "Empty":
		values = make(config.Values)
	case "Last-used values":
		values = make(config.Values, len(m.lastUsed))
		maps.Copy(values, m.lastUsed)
	default:
		sourceName := strings.TrimPrefix(item, "Copy: ")
		src := m.presets[sourceName]
		values = make(config.Values, len(src))
		maps.Copy(values, src)
	}

	m.presets[m.activeName] = values
	m.presetNames = sortedPresetNames(m.presets)
	if idx := slices.Index(m.presetNames, m.activeName); idx >= 0 {
		m.cursor = idx
	}

	// Enter values editing for the new preset.
	m.editor = NewValuesEditor(m.options, values, m.lastUsed, m.hints)
	m.editor.showRequired = true
	m.mode = presetsValuesMode
	return m, nil
}

func (m *PresetsModel) updateDelete(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		name := m.presetNames[m.cursor]
		delete(m.presets, name)
		m.presetNames = sortedPresetNames(m.presets)
		if m.cursor >= len(m.presetNames) && m.cursor > 0 {
			m.cursor--
		}
		m.mode = presetsListMode
	case "n", keyEsc, keyCtrlC:
		m.mode = presetsListMode
	}
	return m, nil
}

// --- Views ---

func (m *PresetsModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	switch m.mode {
	case presetsListMode:
		return tea.NewView(m.viewList())
	case presetsValuesMode:
		return tea.NewView(m.viewValues())
	case presetsNameMode:
		return tea.NewView(m.viewName())
	case presetsSourceMode:
		return tea.NewView(m.viewSource())
	case presetsDeleteMode:
		return tea.NewView(m.viewDelete())
	}
	return tea.NewView("")
}

func (m *PresetsModel) viewList() string {
	var b strings.Builder
	b.WriteString("\n  " + ui.TitleStyle.Render("Manage presets") + "\n\n")

	if len(m.presetNames) == 0 {
		b.WriteString("  " + ui.MutedStyle.Render("No presets yet. Press n to create one.") + "\n")
		b.WriteString("\n" + ui.NavHints(ui.Hint{Key: "n", Desc: "new"}, ui.HintEscDone) + "\n")
		return b.String()
	}

	n := len(m.presetNames)
	gutterWidth := len(strconv.Itoa(n))
	for i, name := range m.presetNames {
		active := i == m.cursor
		b.WriteString(m.viewPresetRow(i, name, active, gutterWidth))
	}

	b.WriteString("\n" + ui.NavHints(
		ui.HintUp, ui.HintDown, ui.HintEdit,
		ui.Hint{Key: "n", Desc: "new"}, ui.Hint{Key: "r", Desc: "rename"},
		ui.Hint{Key: "d", Desc: "delete"}, ui.HintEscDone,
	) + "\n")
	return b.String()
}

func (m *PresetsModel) viewPresetRow(
	i int, name string, active bool, gutterWidth int,
) string {
	num := ui.NumberGutter(i+1, gutterWidth, active)
	cursor := cursorBlank
	if active {
		cursor = " " + ui.Cursor() + " "
	}
	label := ui.ChoiceLabel(name, active)
	count := len(m.presets[name])
	suffix := ui.MutedStyle.Render(fmt.Sprintf("(%d values)", count))
	return fmt.Sprintf("   %s%s  %s  %s\n", cursor, num, label, suffix)
}

func (m *PresetsModel) viewValues() string {
	if m.editor.Editing() {
		return m.editor.ViewEdit(ui.PresetEditIndicator())
	}

	var b strings.Builder
	title := "Editing: " + m.activeName
	b.WriteString("\n  " + ui.TitleStyle.Render(title) + "\n\n")

	maxLabel := m.editor.MaxLabelWidth()
	n := len(m.options)
	gutterWidth := len(strconv.Itoa(n))
	for i := range n {
		active := i == m.editor.cursor
		b.WriteString(m.editor.ViewOptionRow(i, active, maxLabel, gutterWidth, i+1))
	}

	if m.valuesWarning != "" {
		b.WriteString("\n  " + ui.WarningText(m.valuesWarning) + "\n")
	}
	b.WriteString("\n" + ui.NavHints(
		ui.HintUp, ui.HintDown, ui.HintCycle,
		ui.HintEdit, ui.HintSpace, ui.HintEscBack,
	) + "\n")
	return b.String()
}

func (m *PresetsModel) viewName() string {
	var b strings.Builder
	action := "New preset"
	if m.renamingFrom != "" {
		action = "Rename: " + m.renamingFrom
	}
	b.WriteString("\n  " + ui.TitleStyle.Render(action) + "\n\n")
	b.WriteString(m.nameInput.View() + "\n")
	if m.nameErr != "" {
		b.WriteString("  " + ui.WarningText(m.nameErr) + "\n")
	}
	b.WriteString("\n" + ui.NavHints(ui.HintEnter, ui.HintEsc) + "\n")
	return b.String()
}

func (m *PresetsModel) viewSource() string {
	var b strings.Builder
	b.WriteString("\n  " + ui.TitleStyle.Render("New preset: "+m.activeName) + "\n\n")
	b.WriteString("  " + ui.MutedStyle.Render("Start from:") + "\n\n")

	n := len(m.sourceItems)
	gutterWidth := len(strconv.Itoa(n))
	for i, item := range m.sourceItems {
		active := i == m.sourceCursor
		num := ui.NumberGutter(i+1, gutterWidth, active)
		cursor := cursorBlank
		if active {
			cursor = " " + ui.Cursor() + " "
		}
		label := ui.ChoiceLabel(item, active)
		fmt.Fprintf(&b, "   %s%s  %s\n", cursor, num, label)
	}

	b.WriteString("\n" + ui.NavHints(ui.HintNav, ui.HintSelect, ui.HintEsc) + "\n")
	return b.String()
}

func (m *PresetsModel) viewDelete() string {
	var b strings.Builder
	name := m.presetNames[m.cursor]
	b.WriteString("\n  " + ui.TitleStyle.Render("Manage presets") + "\n\n")
	b.WriteString("  " + ui.WarningText(fmt.Sprintf("Delete %q?", name)) +
		" " + ui.MutedStyle.Render("y/n") + "\n")
	return b.String()
}

// RunPresets shows the interactive preset management UI and returns updated presets.
func RunPresets(
	options []config.Option,
	presets map[string]config.Values,
	lastUsed config.Values,
	hints map[string]string,
) (*PresetsResult, error) {
	// Deep copy presets so we don't mutate the caller's map.
	working := make(map[string]config.Values, len(presets))
	for k, v := range presets {
		cp := make(config.Values, len(v))
		maps.Copy(cp, v)
		working[k] = cp
	}

	model := newPresetsModel(options, working, lastUsed, hints)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("presets UI error: %w", err)
	}

	final, ok := finalModel.(*PresetsModel)
	if !ok {
		return nil, errors.New("unexpected model type")
	}
	return &PresetsResult{Presets: final.presets}, nil
}
