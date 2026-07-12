package record

import (
	"errors"
	"fmt"
)

// Sentinel errors for the specific ways a record can be malformed. Each [FieldError]
// wraps exactly one of these, so callers can test with errors.Is.
var (
	ErrMissingField = errors.New("record: missing required field")
	ErrBadVersion   = errors.New("record: unsupported version")
	ErrBadTimestamp = errors.New("record: malformed timestamp")
	ErrBadAuthor    = errors.New("record: invalid author name")
	ErrBadID        = errors.New("record: invalid operation id")
)

// FieldError names the specific field and the reason it was rejected. It wraps one of
// the sentinel errors above.
type FieldError struct {
	Field  string
	Reason string
	err    error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("record: field %q: %s", e.Field, e.Reason)
}

// Unwrap exposes the wrapped sentinel for errors.Is.
func (e *FieldError) Unwrap() error { return e.err }

func missingField(field string) *FieldError {
	return &FieldError{Field: field, Reason: "is required but missing", err: ErrMissingField}
}

func badVersion(field, reason string) *FieldError {
	return &FieldError{Field: field, Reason: reason, err: ErrBadVersion}
}

func badTimestamp(field, reason string) *FieldError {
	return &FieldError{Field: field, Reason: reason, err: ErrBadTimestamp}
}

func badAuthor(field, reason string) *FieldError {
	return &FieldError{Field: field, Reason: reason, err: ErrBadAuthor}
}

func badID(field, reason string) *FieldError {
	return &FieldError{Field: field, Reason: reason, err: ErrBadID}
}
