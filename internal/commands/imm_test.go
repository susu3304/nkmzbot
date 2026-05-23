package commands

import "testing"

func TestNormalizeImmCommandNameTrimsPrefixes(t *testing.T) {
	for input, want := range map[string]string{
		"repeat":    "repeat",
		"?repeat":   "repeat",
		"!repeat":   "repeat",
		"  ?repeat": "repeat",
	} {
		if got := normalizeImmCommandName(input); got != want {
			t.Fatalf("normalizeImmCommandName(%q) = %q, want %q", input, got, want)
		}
	}
}
