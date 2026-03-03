package compat

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/svyatov/oz/internal/config"
)

// DetectVersion runs the detect_version command and extracts the version string.
func DetectVersion(dv *config.DetectVersion) (string, error) {
	if dv == nil {
		return "", nil
	}

	parts := strings.Fields(dv.Command)
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("running %q: %w\n%s", dv.Command, err, out)
	}

	re, err := regexp.Compile(dv.Pattern)
	if err != nil {
		return "", fmt.Errorf("compiling pattern %q: %w", dv.Pattern, err)
	}

	matches := re.FindSubmatch(out)
	if len(matches) < 2 {
		return "", fmt.Errorf("pattern %q did not match output: %s", dv.Pattern, strings.TrimSpace(string(out)))
	}

	return string(matches[1]), nil
}

// MatchedRange returns the version constraint string that matches, or "" if none.
func MatchedRange(entries []config.CompatEntry, version string) string {
	if version == "" {
		return ""
	}
	for _, c := range entries {
		if matchVersionRange(version, c.Versions) {
			return c.Versions
		}
	}
	return ""
}

// FilterOptions returns the subset of options allowed for the detected version.
// If no compat entries exist, all options are returned.
func FilterOptions(options []config.Option, compat []config.CompatEntry, version string) []config.Option {
	if len(compat) == 0 || version == "" {
		return options
	}

	var allowed map[string]bool
	for _, c := range compat {
		if matchVersionRange(version, c.Versions) {
			allowed = make(map[string]bool, len(c.Options))
			for _, name := range c.Options {
				allowed[name] = true
			}
			break
		}
	}

	if allowed == nil {
		return options
	}

	filtered := make([]config.Option, 0, len(options))
	for _, o := range options {
		if allowed[o.Name] {
			filtered = append(filtered, o)
		}
	}
	return filtered
}

// matchVersionRange checks if version satisfies a comma-separated constraint string.
// Supports: ">= X.Y", "< X.Y", "> X.Y", "<= X.Y", "= X.Y", "X.Y"
func matchVersionRange(version, constraint string) bool {
	parts := strings.Split(constraint, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !matchSingleConstraint(version, part) {
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

// compareVersions compares two dotted version strings numerically.
// Returns -1, 0, or 1.
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
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
