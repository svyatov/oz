package ui

import "fmt"

// Plural returns "N word" with correct pluralization.
// With one form, appends "s" for plural: Plural(2, "option") → "2 options".
// With two forms, uses the second for plural: Plural(2, "entry", "entries") → "2 entries".
func Plural(n int, forms ...string) string {
	singular := forms[0]
	plural := singular + "s"
	if len(forms) > 1 {
		plural = forms[1]
	}
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}
