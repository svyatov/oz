// Package generate parses CLI --help output and scaffolds wizard YAML configs.
package generate

import (
	"regexp"
	"strings"
)

// Flag is a single parsed CLI flag from --help output.
type Flag struct {
	Short       string   // e.g. "-v".
	Long        string   // e.g. "--verbose".
	Placeholder string   // e.g. "FILE", "string"; empty for booleans.
	Description string   // help text.
	Default     string   // extracted default value.
	EnumValues  []string // detected enum values from {a,b,c} or description.
}

// skipFlags are universally useless in wizards.
var skipFlags = map[string]bool{
	"--help": true, "-h": true,
	"--version": true, "-V": true,
}

var (
	ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	bsRe   = regexp.MustCompile(`.\x08`)

	// sectionRe matches flag/option section headers.
	// Primary: "Options:", "Flags:", "Global Flags:", "Compilation Options:", etc.
	// Colon is optional for man pages ("OPTIONS").
	sectionRe = regexp.MustCompile(`(?i)^\s*[\w\s]*(options|flags|optional arguments)\s*:?\s*$`)

	// genericSectionRe matches any short, non-indented, colon-terminated line
	// that looks like a section header (e.g. "Caching:", "Output:", "Scanning options:").
	// Used as a secondary section detector.
	genericSectionRe = regexp.MustCompile(`^[A-Z][\w\s/&-]{0,50}:\s*$`)

	// thorFlagRe matches Thor-style flag lines: [--flag=VALUE]  # description.
	thorFlagRe = regexp.MustCompile(`\[--[\w][\w.-]*`)

	// thorBracketFlagRe extracts flags from [--flag] or [--flag=VALUE] brackets.
	thorBracketFlagRe = regexp.MustCompile(`\[--([\w][\w.-]*(?:=[A-Z_]+)?)\]`)

	// thorPlainLongRe matches non-bracketed long flags like --master.
	thorPlainLongRe = regexp.MustCompile(`--([\w][\w.-]*)`)

	// thorShortFlagRe matches short flags like -p.
	thorShortFlagRe = regexp.MustCompile(`-([A-Za-z])\b`)

	// thorDefaultRe extracts bare Default: value patterns (Thor convention).
	thorDefaultRe = regexp.MustCompile(`\bDefault:\s+(\S+)`)

	// flagPrefixRe captures the flag portion at the start of an indented line.
	// Groups: 1=short (e.g. "-v"), 2=long (e.g. "--verbose").
	flagPrefixRe = regexp.MustCompile(
		`^\s{2,}((-\w),\s+)?(--[\w][\w.-]*)`)

	// shortOnlyRe captures short-flag-only lines.
	shortOnlyRe = regexp.MustCompile(`^\s{2,}(-\w)\b`)

	// kubectlFlagRe matches kubectl-style flag=default lines.
	kubectlFlagRe = regexp.MustCompile(
		`^\s+((-\w),\s+)?(--[\w][\w.-]*)=([^:]*):$`)

	// defaultRe extracts default values from description text.
	defaultRe = regexp.MustCompile(
		`(?i)[\[(]default[:\s]+["']?([^"'\])]*)["']?[\])]`)

	// enumBraceRe matches {a,b,c} placeholders.
	enumBraceRe = regexp.MustCompile(`^\{([^}]+)\}$`)

	// placeholderRe matches a placeholder token (word, braced, or angle-bracketed).
	placeholderRe = regexp.MustCompile(`^(\{[^}]+\}|<[^>]+>|\w+)$`)
)

// Parse extracts flags from CLI --help output text.
func Parse(text string) []Flag {
	text = stripANSI(text)
	lines := strings.Split(text, "\n")

	if isKubectlFormat(lines) {
		return parseKubectl(lines)
	}

	if isThorFormat(lines) {
		lines = preprocessThor(lines)
	}

	return parseGNU(lines)
}

// isThorFormat detects Thor/Rails-style help with [--flag] bracket syntax.
func isThorFormat(lines []string) bool {
	matches := 0
	for _, line := range lines {
		if thorFlagRe.MatchString(line) {
			matches++
		}
		if matches >= 3 {
			return true
		}
	}
	return false
}

// preprocessThor converts Thor-style lines to GNU format.
// Handles multi-flag lines by selecting the canonical flag and stripping negation variants.
func preprocessThor(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		switch {
		case thorFlagRe.MatchString(line):
			out = append(out, preprocessThorFlagLine(line))
		case strings.Contains(line, "# "):
			// Continuation lines with # prefix (e.g. "# Default: value").
			out = append(out, strings.Replace(line, "# ", "  ", 1))
		default:
			out = append(out, line)
		}
	}
	return out
}

// preprocessThorFlagLine converts a Thor flag line to clean GNU format.
// It extracts the canonical flag, strips negation variants (--no-X, --skip-X),
// and resolves aliases to the canonical form (e.g. --js → --javascript).
func preprocessThorFlagLine(line string) string {
	// Split on "# " to separate flags from description.
	flagPart, desc, _ := strings.Cut(line, "# ")
	desc = strings.TrimSpace(desc)

	// Extract bracketed flags [--flag] and [--flag=VALUE].
	bracketedFlags := extractRegexFlags(thorBracketFlagRe, flagPart)

	// Remove bracketed patterns to find remaining plain flags.
	remaining := thorBracketFlagRe.ReplaceAllString(flagPart, "")

	// Extract non-bracketed long flags and short flag.
	plainFlags := extractRegexFlags(thorPlainLongRe, remaining)
	var shortFlag string
	if m := thorShortFlagRe.FindStringSubmatch(remaining); m != nil {
		shortFlag = "-" + m[1]
	}

	// Collect base names (flags without no-/skip- prefix).
	baseNames := make(map[string]bool)
	for _, f := range append(bracketedFlags, plainFlags...) {
		name := flagName(f)
		if !strings.HasPrefix(name, "no-") && !strings.HasPrefix(name, "skip-") {
			baseNames[name] = true
		}
	}

	// Filter out negation variants when the base form exists.
	bracketedFlags = filterNegationVariants(bracketedFlags, baseNames)
	plainFlags = filterNegationVariants(plainFlags, baseNames)

	// Pick the best long flag: prefer =PLACEHOLDER, then bracketed, then longest.
	bestLong := selectCanonicalFlag(bracketedFlags, plainFlags)

	return buildGNULine(shortFlag, bestLong, desc)
}

// extractRegexFlags collects all "--<match>" flags from regex submatches.
func extractRegexFlags(re *regexp.Regexp, s string) []string {
	matches := re.FindAllStringSubmatch(s, -1)
	flags := make([]string, 0, len(matches))
	for _, m := range matches {
		flags = append(flags, "--"+m[1])
	}
	return flags
}

// buildGNULine reconstructs a clean GNU-format flag line.
func buildGNULine(shortFlag, longFlag, desc string) string {
	var b strings.Builder
	b.WriteString("  ")
	if shortFlag != "" {
		b.WriteString(shortFlag)
		if longFlag != "" {
			b.WriteString(", ")
		}
	}
	if longFlag != "" {
		if name, placeholder, ok := strings.Cut(longFlag, "="); ok {
			b.WriteString(name)
			b.WriteString(" ")
			b.WriteString(placeholder)
		} else {
			b.WriteString(longFlag)
		}
	}
	if desc != "" {
		b.WriteString("    ")
		b.WriteString(desc)
	}
	return b.String()
}

// flagName strips the -- prefix and any =PLACEHOLDER suffix.
func flagName(f string) string {
	name := strings.TrimPrefix(f, "--")
	if idx := strings.Index(name, "="); idx > 0 {
		name = name[:idx]
	}
	return name
}

// filterNegationVariants removes --no-X and --skip-X when --X exists.
func filterNegationVariants(flags []string, baseNames map[string]bool) []string {
	result := make([]string, 0, len(flags))
	for _, f := range flags {
		name := flagName(f)
		if strings.HasPrefix(name, "no-") && baseNames[strings.TrimPrefix(name, "no-")] {
			continue
		}
		if strings.HasPrefix(name, "skip-") && baseNames[strings.TrimPrefix(name, "skip-")] {
			continue
		}
		result = append(result, f)
	}
	return result
}

// selectCanonicalFlag picks the best long flag from bracketed and plain flags.
// Priority: has =PLACEHOLDER (bracketed first) > bracketed > longest.
func selectCanonicalFlag(bracketed, plain []string) string {
	// Prefer flag with =PLACEHOLDER.
	for _, f := range bracketed {
		if strings.Contains(f, "=") {
			return f
		}
	}
	for _, f := range plain {
		if strings.Contains(f, "=") {
			return f
		}
	}
	// Prefer longest bracketed flag.
	var best string
	for _, f := range bracketed {
		if len(f) > len(best) {
			best = f
		}
	}
	if best != "" {
		return best
	}
	// Fall back to longest plain flag.
	for _, f := range plain {
		if len(f) > len(best) {
			best = f
		}
	}
	return best
}

// stripANSI removes ANSI escape codes and man-page backspace formatting.
func stripANSI(s string) string {
	s = ansiRe.ReplaceAllString(s, "")
	s = bsRe.ReplaceAllString(s, "")
	return s
}

// isKubectlFormat detects kubectl-style help by checking if a significant
// portion of flag-like lines match the --flag=value: pattern.
func isKubectlFormat(lines []string) bool {
	var total, kubectl int
	for _, line := range lines {
		if strings.Contains(line, "--") {
			total++
			if kubectlFlagRe.MatchString(line) {
				kubectl++
			}
		}
	}
	return total > 0 && float64(kubectl)/float64(total) > 0.3
}

// parseGNU handles GNU/Cobra/Docker/argparse-style help output.
// Runs both section-aware and full-text scans, returns whichever finds more.
func parseGNU(lines []string) []Flag {
	sectioned := parseGNUSection(lines, false)
	fullScan := parseGNUSection(lines, true)
	if len(fullScan) > len(sectioned) {
		return postProcess(fullScan)
	}
	return postProcess(sectioned)
}

func parseGNUSection(lines []string, scanAll bool) []Flag {
	var flags []Flag
	var current *Flag
	inSection := scanAll
	descCol := 0

	for _, line := range lines {
		if !scanAll && isSectionHeader(line) {
			inSection = true
			continue
		}

		// End section on non-indented, non-empty line.
		if inSection && !scanAll && line != "" && line[0] != ' ' && line[0] != '\t' {
			// Don't exit section if this is another section header.
			if isSectionHeader(line) {
				continue
			}
			inSection = false
			finishFlag(current, &flags)
			current = nil
			continue
		}

		if !inSection {
			continue
		}

		if f, dc := matchGNUFlag(line); f != nil {
			finishFlag(current, &flags)
			current = f
			if dc > 0 {
				descCol = dc
			}
			continue
		}

		// Continuation line: append to current flag's description.
		if current != nil && isContinuation(line, descCol) {
			appendDescription(current, line)
		}
	}

	finishFlag(current, &flags)
	return flags
}

// matchGNUFlag parses a line for GNU-style flag patterns.
// Returns the flag and the column where the description starts (0 if no description).
func matchGNUFlag(line string) (*Flag, int) {
	// Try long flag (with optional short prefix).
	if m := flagPrefixRe.FindStringSubmatchIndex(line); m != nil {
		sub := flagPrefixRe.FindStringSubmatch(line)
		f := &Flag{
			Short: sub[2],
			Long:  sub[3],
		}
		rest := line[m[1]:]
		parseAfterFlag(rest, f, m[1])
		return f, descColumn(line, m[1])
	}

	// Try short-flag-only.
	if m := shortOnlyRe.FindStringSubmatchIndex(line); m != nil {
		sub := shortOnlyRe.FindStringSubmatch(line)
		f := &Flag{Short: sub[1]}
		rest := line[m[1]:]
		parseAfterFlag(rest, f, m[1])
		return f, descColumn(line, m[1])
	}

	return nil, 0
}

// parseAfterFlag extracts placeholder and description from the text after flags.
func parseAfterFlag(rest string, f *Flag, _ int) {
	// Split on first 2+ space gap to separate placeholder from description.
	parts := splitOnGap(rest)
	switch len(parts) {
	case 0:
		// Nothing after flag.
	case 1:
		// Could be just a placeholder or just a description.
		token := strings.TrimSpace(parts[0])
		if placeholderRe.MatchString(token) {
			f.Placeholder = token
		} else {
			f.Description = token
		}
	default:
		first := strings.TrimSpace(parts[0])
		if placeholderRe.MatchString(first) {
			f.Placeholder = first
			f.Description = strings.TrimSpace(strings.Join(parts[1:], " "))
		} else {
			f.Description = strings.TrimSpace(parts[0] + " " + strings.Join(parts[1:], " "))
		}
	}

	// Handle =placeholder syntax (e.g. --flag=VALUE).
	if f.Placeholder == "" && f.Long != "" {
		if idx := strings.Index(f.Long, "="); idx > 0 {
			f.Placeholder = f.Long[idx+1:]
			f.Long = f.Long[:idx]
		}
	}
}

// splitOnGap splits a string on the first occurrence of 2+ spaces.
func splitOnGap(s string) []string {
	s = strings.TrimLeft(s, " =")
	if s == "" {
		return nil
	}
	// Find first 2+ space gap.
	inSpaces := false
	spaceStart := 0
	for i, c := range s {
		if c == ' ' {
			if !inSpaces {
				inSpaces = true
				spaceStart = i
			}
		} else {
			if inSpaces && i-spaceStart >= 2 {
				return []string{
					strings.TrimSpace(s[:spaceStart]),
					strings.TrimSpace(s[i:]),
				}
			}
			inSpaces = false
		}
	}
	return []string{strings.TrimSpace(s)}
}

// descColumn returns the column where the description starts.
func descColumn(line string, flagEnd int) int {
	rest := line[flagEnd:]
	parts := splitOnGap(rest)
	if len(parts) < 2 {
		return 0
	}
	// Find the description start in the original line.
	lastPart := parts[len(parts)-1]
	idx := strings.LastIndex(line, lastPart)
	if idx >= 0 {
		return idx
	}
	return 0
}

// isContinuation returns true if a line looks like a continuation of
// a previous flag's description based on indentation.
func isContinuation(line string, descCol int) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	indent := countIndent(line)
	if descCol > 0 {
		return indent >= descCol
	}
	// When descCol is unknown (flag had no inline description),
	// accept deeply indented lines as continuations.
	return indent >= minContinuationIndent
}

const minContinuationIndent = 10

// isSectionHeader returns true if a line looks like a section header.
func isSectionHeader(line string) bool {
	return sectionRe.MatchString(line) || genericSectionRe.MatchString(line)
}

// appendDescription appends a trimmed continuation line to a flag's description.
func appendDescription(f *Flag, line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if f.Description == "" {
		f.Description = trimmed
	} else {
		f.Description += " " + trimmed
	}
}

func countIndent(s string) int {
	n := 0
	for _, c := range s {
		switch c {
		case ' ':
			n++
		case '\t':
			n += 8
		default:
			return n
		}
	}
	return n
}

// parseKubectl handles kubectl/Go-pflag-style help output.
func parseKubectl(lines []string) []Flag {
	var flags []Flag
	var current *Flag
	inSection := false

	for _, line := range lines {
		if sectionRe.MatchString(line) {
			inSection = true
			continue
		}

		if inSection && line != "" && line[0] != ' ' && line[0] != '\t' {
			inSection = false
			finishFlag(current, &flags)
			current = nil
			continue
		}

		if !inSection {
			continue
		}

		if m := kubectlFlagRe.FindStringSubmatch(line); m != nil {
			finishFlag(current, &flags)
			current = &Flag{
				Short:   m[2],
				Long:    m[3],
				Default: strings.TrimSpace(m[4]),
			}

			// Infer placeholder from default value.
			if current.Default != "" && current.Default != "false" && current.Default != "true" {
				current.Placeholder = "value"
			}
			continue
		}

		// Description continuation lines.
		if current != nil {
			appendDescription(current, line)
		}
	}

	finishFlag(current, &flags)
	return postProcess(flags)
}

// finishFlag saves the current flag to the list if non-nil and not skipped.
func finishFlag(current *Flag, flags *[]Flag) {
	if current == nil {
		return
	}

	// Skip help/version flags.
	if skipFlags[current.Long] || skipFlags[current.Short] {
		return
	}

	*flags = append(*flags, *current)
}

// postProcess extracts defaults, detects enums, and infers boolean flags.
func postProcess(flags []Flag) []Flag {
	result := make([]Flag, 0, len(flags))
	for _, f := range flags {
		extractDefault(&f)
		detectEnum(&f)
		result = append(result, f)
	}
	return result
}

// extractDefault pulls default values from description text.
func extractDefault(f *Flag) {
	if f.Default == "" {
		m := defaultRe.FindStringSubmatch(f.Description)
		if m != nil {
			f.Default = strings.TrimSpace(m[1])
		}
	}
	// Try bare Default: value pattern (Thor convention).
	if f.Default == "" {
		if m := thorDefaultRe.FindStringSubmatch(f.Description); m != nil {
			f.Default = strings.TrimRight(strings.TrimSpace(m[1]), ".")
		}
	}
	// Strip surrounding quotes from default values (common in kubectl output).
	f.Default = strings.Trim(f.Default, "'\"")
}

// detectEnum detects enumeration values from {a,b,c} placeholders
// or description patterns like "one of: a, b, c" or "must be a, b, or c".
func detectEnum(f *Flag) {
	// Check placeholder for {a,b,c} pattern.
	if m := enumBraceRe.FindStringSubmatch(f.Placeholder); m != nil {
		f.EnumValues = splitEnum(m[1])
		return
	}

	// Check description for enum patterns.
	for _, p := range enumDescPatterns {
		if m := p.FindStringSubmatch(f.Description); m != nil {
			vals := splitEnum(m[1])
			if len(vals) >= 2 {
				f.EnumValues = vals
				return
			}
		}
	}
}

var enumDescPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)one of[:\s]+([^.)\]]+)`),
	regexp.MustCompile(`(?i)must be[:\s]+([^.)\]]+)`),
	regexp.MustCompile(`(?i)valid values?[:\s]+([^.)\]]+)`),
	regexp.MustCompile(`(?i)possible values?[:\s]+([^)\]]+)`),
}

// splitEnum splits comma/or-separated enum values, stripping quotes.
func splitEnum(s string) []string {
	s = strings.NewReplacer(`"`, "", `'`, "").Replace(s)
	// Normalize "a, b or c" → "a, b, c" before splitting.
	s = strings.ReplaceAll(s, ", or ", ", ")
	s = strings.ReplaceAll(s, " or ", ", ")
	s = strings.ReplaceAll(s, ", and ", ", ")
	s = strings.ReplaceAll(s, " and ", ", ")
	parts := strings.Split(s, ",")
	var vals []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			vals = append(vals, p)
		}
	}
	return vals
}
