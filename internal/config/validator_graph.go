package config


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
