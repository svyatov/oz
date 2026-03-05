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
func DetectVersion(vc *config.VersionControl) (string, error) {
	if vc == nil {
		return "", nil
	}

	parts := strings.Fields(vc.Command)
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
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
	parts := strings.Fields(expanded)
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("version %s not available: %w\n%s", version, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// FetchAvailableVersions runs a command that returns comma-separated versions.
func FetchAvailableVersions(cmd string) ([]string, error) {
	parts := strings.Fields(cmd)
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fetching versions: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return ParseAvailableVersions(string(out)), nil
}

// ParseAvailableVersions splits a comma-separated version string into a deduplicated slice.
func ParseAvailableVersions(csv string) []string {
	seen := make(map[string]bool)
	var versions []string
	for part := range strings.SplitSeq(csv, ",") {
		v := strings.TrimSpace(part)
		if v != "" && !seen[v] {
			seen[v] = true
			versions = append(versions, v)
		}
	}
	return versions
}

// compareVersions compares two dotted version strings numerically.
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
