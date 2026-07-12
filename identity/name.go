package identity

import (
	"fmt"
	"strings"
)

// MaxNameLen is the maximum length, in characters, of a Soulstream slug.
const MaxNameLen = 64

// NameError explains why a candidate name was rejected.
type NameError struct {
	Name   string
	Reason string
}

func (e *NameError) Error() string {
	return fmt.Sprintf("identity: invalid name %q: %s", e.Name, e.Reason)
}

// ValidName reports whether s is a valid Soulstream slug: it matches
// ^[a-z0-9]+(-[a-z0-9]+)*$ and is between 1 and [MaxNameLen] characters long.
//
// The same grammar validates persona names, realm names, and topic-id slugs.
func ValidName(s string) bool {
	return CheckName(s) == nil
}

// CheckName returns nil if s is a valid slug (see [ValidName]) or a *[NameError]
// naming the first reason it was rejected.
func CheckName(s string) error {
	switch {
	case s == "":
		return &NameError{s, "must not be empty"}
	case len(s) > MaxNameLen:
		return &NameError{s, fmt.Sprintf("must be at most %d characters", MaxNameLen)}
	case strings.HasPrefix(s, "-"):
		return &NameError{s, "must not start with a hyphen"}
	case strings.HasSuffix(s, "-"):
		return &NameError{s, "must not end with a hyphen"}
	case strings.Contains(s, "--"):
		return &NameError{s, "must not contain consecutive hyphens"}
	}

	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			// allowed
		case r >= 'A' && r <= 'Z':
			return &NameError{s, "must be lowercase"}
		case r == '.':
			return &NameError{s, "must not contain '.' (reserved as a subject separator)"}
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			return &NameError{s, "must not contain whitespace"}
		case r == '*' || r == '>':
			return &NameError{s, "must not contain the subject wildcards '*' or '>'"}
		default:
			return &NameError{s, fmt.Sprintf("contains the disallowed character %q", r)}
		}
	}

	return nil
}
