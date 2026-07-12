package identity

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantOK  bool
		reasonC string // substring expected in the rejection reason (when wantOK is false)
	}{
		// Accepted — representative of real personas, realms, and topic ids.
		{"simple", "daan", true, ""},
		{"single-char", "a", true, ""},
		{"digits", "007", true, ""},
		{"hyphenated", "invoice-agent", true, ""},
		{"multi-hyphen", "vat-q2-2026-x7m2", true, ""},
		{"max-length", strings.Repeat("a", MaxNameLen), true, ""},

		// Rejected — each with a specific reason.
		{"empty", "", false, "empty"},
		{"too-long", strings.Repeat("a", MaxNameLen+1), false, "at most"},
		{"uppercase", "Daan", false, "lowercase"},
		{"all-caps", "ARCHITECT", false, "lowercase"},
		{"dot", "a.b", false, "'.'"},
		{"leading-hyphen", "-x", false, "start with a hyphen"},
		{"trailing-hyphen", "x-", false, "end with a hyphen"},
		{"double-hyphen", "a--b", false, "consecutive hyphens"},
		{"space", "a b", false, "whitespace"},
		{"tab", "a\tb", false, "whitespace"},
		{"wildcard-star", "a*", false, "wildcards"},
		{"wildcard-gt", ">", false, "wildcards"},
		{"underscore", "a_b", false, "disallowed character"},
		{"slash", "a/b", false, "disallowed character"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckName(tc.input)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("CheckName(%q) = %v, want nil", tc.input, err)
				}
				if !ValidName(tc.input) {
					t.Fatalf("ValidName(%q) = false, want true", tc.input)
				}
				return
			}

			if err == nil {
				t.Fatalf("CheckName(%q) = nil, want error", tc.input)
			}
			var ne *NameError
			if !errors.As(err, &ne) {
				t.Fatalf("CheckName(%q) error type = %T, want *NameError", tc.input, err)
			}
			if !strings.Contains(ne.Reason, tc.reasonC) {
				t.Fatalf("CheckName(%q) reason = %q, want it to contain %q", tc.input, ne.Reason, tc.reasonC)
			}
			if ValidName(tc.input) {
				t.Fatalf("ValidName(%q) = true, want false", tc.input)
			}
		})
	}
}
