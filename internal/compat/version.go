// Package compat detects tool versions and filters options by semver range.
package compat

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/svyatov/oz/internal/config"
)

// DetectVersion runs the version_control command and extracts the version string.
// Called once per CLI invocation; the pattern regex is compiled here rather than cached.
func DetectVersion(vc *config.VersionControl) (string, error) {
	if vc == nil {
		return "", nil
	}

	out, err := exec.Command("sh", "-c", vc.Command).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("running %q: %w\n%s", vc.Command, err, out)
	}

	re, err := regexp.Compile(vc.Pattern)
	if err != nil {
		return "", fmt.Errorf("compiling pattern %q: %w", vc.Pattern, err)
	}

	matches := re.FindSubmatch(out)
	if len(matches) < 2 {
		return "", fmt.Errorf("pattern %q did not match output: %s", vc.Pattern, strings.TrimSpace(string(out)))
	}

	return string(matches[1]), nil
}

// FilterOptions returns options whose Versions constraint matches the detected version.
// Options without a Versions field are always included.
// If version is empty, all options are returned.
func FilterOptions(options []config.Option, version string) []config.Option {
	if version == "" {
		return options
	}
	filtered := make([]config.Option, 0, len(options))
	for _, o := range options {
		if o.Versions == "" || matchVersionRange(version, o.Versions) {
			filtered = append(filtered, o)
		}
	}
	return filtered
}

// FilterChoices returns choices whose Versions constraint matches the detected version.
// Choices without a Versions field are always included.
// If version is empty, all choices are returned.
func FilterChoices(choices config.FlexChoices, version string) config.FlexChoices {
	if version == "" {
		return choices
	}
	filtered := make(config.FlexChoices, 0, len(choices))
	for _, c := range choices {
		if c.Versions == "" || matchVersionRange(version, c.Versions) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// matchVersionRange checks if version satisfies a comma-separated constraint string.
// Supports: ">= X.Y", "< X.Y", "> X.Y", "<= X.Y", "= X.Y", "X.Y".
func matchVersionRange(version, constraint string) bool {
	for part := range strings.SplitSeq(constraint, ",") {
		if !matchSingleConstraint(version, strings.TrimSpace(part)) {
			return false
		}
	}
	return true
}

func matchSingleConstraint(version, constraint string) bool {
	constraint = strings.TrimSpace(constraint)

	var op, target string
	for _, prefix := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(constraint, prefix) {
			op = prefix
			target = strings.TrimSpace(constraint[len(prefix):])
			break
		}
	}
	if op == "" {
		op = "="
		target = constraint
	}

	cmp := compareVersions(version, target)
	switch op {
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case "=":
		return cmp == 0
	}
	return false
}

// ExpandTemplate replaces {{version}} in a template string.
func ExpandTemplate(template, version string) string {
	return strings.ReplaceAll(template, "{{version}}", version)
}

// VerifyVersion runs the verify command with the given version and checks exit status.
func VerifyVersion(verifyCmd, version string) error {
	expanded := ExpandTemplate(verifyCmd, version)
	out, err := exec.Command("sh", "-c", expanded).CombinedOutput()
	if err != nil {
		return fmt.Errorf("version %s not available: %w\n%s", version, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// FetchAvailableVersions runs a shell command and parses its output as versions.
// Trailing newlines are stripped before parsing so that single-line output
// (e.g. from echo) stays comma-split rather than triggering newline mode.
func FetchAvailableVersions(cmd string) ([]string, error) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fetching versions: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return ParseAvailableVersions(strings.TrimRight(string(out), "\n")), nil
}

// ParseAvailableVersions splits a version string into a deduplicated slice.
// Multi-line input is split by newlines; single-line input is split by commas.
func ParseAvailableVersions(raw string) []string {
	sep := ","
	if strings.Contains(raw, "\n") {
		sep = "\n"
	}
	seen := make(map[string]bool)
	var versions []string
	for part := range strings.SplitSeq(raw, sep) {
		v := strings.TrimSpace(part)
		if v != "" && !seen[v] {
			seen[v] = true
			versions = append(versions, v)
		}
	}
	return versions
}

// OptionHints returns a map of option name → human-readable version hint.
// Only options with a Versions constraint get a hint.
func OptionHints(options []config.Option) map[string]string {
	hints := make(map[string]string)
	for _, o := range options {
		if o.Versions != "" {
			hints[o.Name] = formatHint(o.Versions)
		}
	}
	return hints
}

// formatHint converts a constraint like ">= 8.0" or ">= 7.0, < 8.0" into "v8.0+" or "v7.0+".
// For comma-separated ranges, uses the first part (lower bound).
func formatHint(constraint string) string {
	// Use only the first part of comma-separated constraints.
	c, _, _ := strings.Cut(constraint, ",")
	c = strings.TrimSpace(c)

	// ">= X.Y" → "vX.Y+"
	if after, ok := strings.CutPrefix(c, ">="); ok {
		return "v" + strings.TrimSpace(after) + "+"
	}

	// "< X.Y" → "< vX.Y"
	if after, ok := strings.CutPrefix(c, "<"); ok {
		return "< v" + strings.TrimSpace(after)
	}

	return c
}

// compareVersions compares two dotted version strings numerically.
// Non-numeric segments (e.g. pre-release suffixes like "-rc1") are ignored;
// only the leading digits of each part are compared.
// Returns -1, 0, or 1.
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := max(len(aParts), len(bParts))

	for i := range maxLen {
		av, bv := 0, 0
		if i < len(aParts) {
			av, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bv, _ = strconv.Atoi(bParts[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}
