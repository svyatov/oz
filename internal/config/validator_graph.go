package config

import (
	"regexp"
	"slices"

	semver "github.com/Masterminds/semver/v3"
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

// versionRe matches version-like numbers in constraint strings.
var versionRe = regexp.MustCompile(`\d+(?:\.\d+)*`)

// constraintsOverlap checks if two version constraints can ever be simultaneously satisfied.
// Checks each constraint independently (not concatenated) so OR (||) in either
// constraint is handled correctly.
func constraintsOverlap(a, b string) bool {
	ca, errA := semver.NewConstraint(a)
	cb, errB := semver.NewConstraint(b)
	if errA != nil || errB != nil {
		return false
	}
	return slices.ContainsFunc(probeVersions(a, b), func(v *semver.Version) bool {
		return ca.Check(v) && cb.Check(v)
	})
}

// probeVersions extracts version numbers from constraint strings
// and generates ±1 neighbors to catch strict inequality boundaries.
func probeVersions(constraints ...string) []*semver.Version {
	seen := make(map[string]bool)
	var probes []*semver.Version
	add := func(v *semver.Version) {
		if k := v.String(); !seen[k] {
			seen[k] = true
			probes = append(probes, v)
		}
	}
	for _, cs := range constraints {
		for _, m := range versionRe.FindAllString(cs, -1) {
			v, err := semver.NewVersion(m)
			if err != nil {
				continue
			}
			add(v)
			add(semver.New(v.Major()+1, 0, 0, "", ""))
			add(semver.New(v.Major(), v.Minor()+1, 0, "", ""))
			add(semver.New(v.Major(), v.Minor(), v.Patch()+1, "", ""))
			if v.Major() > 0 {
				add(semver.New(v.Major()-1, 0, 0, "", ""))
			}
			if v.Minor() > 0 {
				add(semver.New(v.Major(), v.Minor()-1, 0, "", ""))
			}
			if v.Patch() > 0 {
				add(semver.New(v.Major(), v.Minor(), v.Patch()-1, "", ""))
			}
		}
	}
	return probes
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
