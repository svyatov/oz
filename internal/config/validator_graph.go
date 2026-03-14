package config

import (
	"strconv"
	"strings"
)

// validateVersionGating checks for version-related inconsistencies:
// - choice versions outside option versions (unreachable choices)
// - default value in a version-gated choice
// - show_when/hide_when referencing a version-gated option.
func validateVersionGating(options []Option, errs *errorCollector) {
	versioned := make(map[string]string, len(options))
	for _, o := range options {
		if o.Versions != "" {
			versioned[o.Name] = o.Versions
		}
	}

	for i, o := range options {
		prefix := optionPrefix(i, o.Name)
		validateChoiceVersionOverlap(o, prefix, errs)
		validateDefaultNotVersionGated(o, prefix, errs)
		validateVisibilityRefsVersionGated(o, prefix, versioned, errs)
	}
}

// validateChoiceVersionOverlap checks that choice versions can overlap with option versions.
func validateChoiceVersionOverlap(o Option, prefix string, errs *errorCollector) {
	if o.Versions == "" {
		return
	}
	for j, c := range o.Choices {
		if c.Versions == "" {
			continue
		}
		if !constraintsOverlap(o.Versions, c.Versions) {
			errs.addf("%s: choices[%d] (%s): versions %q can never match within option versions %q",
				prefix, j, c.Value, c.Versions, o.Versions)
		}
	}
}

// validateDefaultNotVersionGated warns if the default value is in a version-gated choice.
func validateDefaultNotVersionGated(o Option, prefix string, errs *errorCollector) {
	if o.Default == nil || len(o.Choices) == 0 {
		return
	}
	defVal := o.Default.Scalar()
	for _, c := range o.Choices {
		if c.Value == defVal && c.Versions != "" {
			errs.addf("%s: default %q is in a version-gated choice (%s); it won't be available in all versions",
				prefix, defVal, c.Versions)
			return
		}
	}
}

// validateVisibilityRefsVersionGated checks if show_when/hide_when reference version-gated options.
func validateVisibilityRefsVersionGated(o Option, prefix string, versioned map[string]string, errs *errorCollector) {
	if o.Versions != "" {
		// If this option is itself version-gated, skip — both may share the same gate.
		return
	}
	for ref := range o.ShowWhen {
		if v, ok := versioned[ref]; ok {
			errs.addf("%s: show_when references version-gated option %q (%s); condition is unresolvable when %q is filtered out",
				prefix, ref, v, ref)
		}
	}
	for ref := range o.HideWhen {
		if v, ok := versioned[ref]; ok {
			errs.addf("%s: hide_when references version-gated option %q (%s); condition is unresolvable when %q is filtered out",
				prefix, ref, v, ref)
		}
	}
}

// constraintsOverlap checks if two version constraints can ever be simultaneously satisfied.
// Uses probe versions at boundaries and extremes as a heuristic.
func constraintsOverlap(a, b string) bool {
	probes := []string{"0.0.0", "0.1.0", "1.0.0", "2.0.0", "5.0.0", "8.0.0", "9.0.0", "99.0.0"}
	probes = append(probes, extractBoundaryVersions(a)...)
	probes = append(probes, extractBoundaryVersions(b)...)
	for _, v := range probes {
		if matchConstraint(v, a) && matchConstraint(v, b) {
			return true
		}
	}
	return false
}

// extractBoundaryVersions pulls version numbers from a constraint string
// and returns them along with adjacent versions (±0.1).
func extractBoundaryVersions(constraint string) []string {
	var result []string
	for part := range strings.SplitSeq(constraint, ",") {
		p := strings.TrimSpace(part)
		for _, prefix := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(p, prefix) {
				p = strings.TrimSpace(p[len(prefix):])
				break
			}
		}
		if p != "" {
			result = append(result, p)
			result = append(result, incrementMinor(p), decrementMinor(p))
		}
	}
	return result
}

func incrementMinor(v string) string {
	parts := splitVersion(v)
	if len(parts) >= 2 {
		parts[1]++
	}
	return joinVersion(parts)
}

func decrementMinor(v string) string {
	parts := splitVersion(v)
	if len(parts) >= 2 && parts[1] > 0 {
		parts[1]--
	}
	return joinVersion(parts)
}

// matchConstraint checks if version satisfies a comma-separated constraint.
// Duplicates compat.matchVersionRange to avoid circular imports.
func matchConstraint(version, constraint string) bool {
	for part := range strings.SplitSeq(constraint, ",") {
		p := strings.TrimSpace(part)
		var op, target string
		for _, prefix := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(p, prefix) {
				op = prefix
				target = strings.TrimSpace(p[len(prefix):])
				break
			}
		}
		if op == "" {
			op = "="
			target = p
		}
		cmp := compareVer(version, target)
		ok := false
		switch op {
		case ">=":
			ok = cmp >= 0
		case "<=":
			ok = cmp <= 0
		case ">":
			ok = cmp > 0
		case "<":
			ok = cmp < 0
		case "=":
			ok = cmp == 0
		}
		if !ok {
			return false
		}
	}
	return true
}

func splitVersion(v string) []int {
	var parts []int
	for seg := range strings.SplitSeq(v, ".") {
		n, _ := strconv.Atoi(seg)
		parts = append(parts, n)
	}
	return parts
}

func joinVersion(parts []int) string {
	strs := make([]string, len(parts))
	for i, p := range parts {
		strs[i] = strconv.Itoa(p)
	}
	return strings.Join(strs, ".")
}

func compareVer(a, b string) int {
	ap := splitVersion(a)
	bp := splitVersion(b)
	maxLen := max(len(ap), len(bp))
	for i := range maxLen {
		av, bv := 0, 0
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
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

// validateVisibilityGraph checks show_when/hide_when/choices_from for
// self-references, forward references, and conflicting visibility conditions.
func validateVisibilityGraph(options []Option, errs *errorCollector) {
	positionOf := make(map[string]int, len(options))
	for i, o := range options {
		if o.Name != "" {
			positionOf[o.Name] = i
		}
	}

	for i, o := range options {
		prefix := optionPrefix(i, o.Name)
		checkVisibilityRefs(o.ShowWhen, "show_when", i, o.Name, prefix, positionOf, errs)
		checkVisibilityRefs(o.HideWhen, "hide_when", i, o.Name, prefix, positionOf, errs)
		checkChoicesFromForwardRefs(o, i, prefix, positionOf, errs)
		checkVisibilityConflict(o, prefix, errs)
	}
}

func checkVisibilityRefs(
	cond Values, kind string,
	idx int, name, prefix string,
	positionOf map[string]int, errs *errorCollector,
) {
	for ref := range cond {
		if ref == name {
			errs.addf("%s: %s references itself", prefix, kind)
			continue
		}
		if pos, known := positionOf[ref]; known && pos >= idx {
			errs.addf("%s: %s references option %q which appears later (index %d); wizard steps are sequential",
				prefix, kind, ref, pos)
		}
	}
}

func checkChoicesFromForwardRefs(
	o Option, idx int, prefix string,
	positionOf map[string]int, errs *errorCollector,
) {
	if o.ChoicesFrom == "" {
		return
	}
	for _, match := range ChoicesFromInterpolationRe.FindAllStringSubmatch(o.ChoicesFrom, -1) {
		ref := match[1]
		if ref == o.Name {
			errs.addf("%s: choices_from interpolation references itself", prefix)
			continue
		}
		if pos, known := positionOf[ref]; known && pos > idx {
			errs.addf("%s: choices_from interpolation references option %q which appears later (index %d)",
				prefix, ref, pos)
		}
	}
}

// checkVisibilityConflict detects when an option can never be visible.
// Both show_when and hide_when use AND semantics (all keys must match).
// The option is never visible when every hide_when condition is implied by
// show_when — i.e., every hide_when key appears in show_when and show_when's
// accepted values are a subset of hide_when's accepted values.
func checkVisibilityConflict(o Option, prefix string, errs *errorCollector) {
	if len(o.ShowWhen) == 0 || len(o.HideWhen) == 0 {
		return
	}
	for key, hideVal := range o.HideWhen {
		showVal, inShow := o.ShowWhen[key]
		if !inShow || !isSubset(showVal, hideVal) {
			return
		}
	}
	// All hide_when keys covered by show_when with matching values.
	for key := range o.HideWhen {
		errs.addf("%s: show_when and hide_when conflict on key %q — option can never be visible", prefix, key)
		return
	}
}

// isSubset returns true if every value in a's set is also in b's set.
func isSubset(a, b FieldValue) bool {
	setA := toStringSet(a)
	setB := toStringSet(b)
	for v := range setA {
		if !setB[v] {
			return false
		}
	}
	return true
}

func toStringSet(v FieldValue) map[string]bool {
	if v.IsStrings() {
		m := make(map[string]bool, len(v.Strings()))
		for _, s := range v.Strings() {
			m[s] = true
		}
		return m
	}
	return map[string]bool{v.Scalar(): true}
}
