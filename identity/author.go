package identity

import (
	"errors"
	"fmt"
)

// Attribution errors.
var (
	ErrForeignAuthor  = errors.New("identity: refusing to publish as another persona")
	ErrAuthorMismatch = errors.New("identity: claimed author does not match delivering identity")
)

// EnforceAuthor is the write-side guard: it returns [ErrForeignAuthor] if recordAuthor
// is not ownPersona. A persona-bound client calls this before publishing so it can
// only ever speak as itself.
func EnforceAuthor(ownPersona, recordAuthor string) error {
	if recordAuthor != ownPersona {
		return fmt.Errorf("%w: own=%q record=%q", ErrForeignAuthor, ownPersona, recordAuthor)
	}
	return nil
}

// Resolver optionally supplies the trusted identity that actually delivered a message,
// when the deployment provides one (for example, an auth-callout stamp). A nil
// Resolver means no trusted source is available.
type Resolver interface {
	// DeliveredBy returns the trusted persona for a delivered message's headers, or
	// ("", false) when it cannot be determined.
	DeliveredBy(headers map[string][]string) (persona string, ok bool)
}

// VerifyAuthor is the read-side check. It always validates that claimedAuthor is a
// well-formed persona name. When r is non-nil and resolves a trusted identity, it
// additionally requires the two to match, returning [ErrAuthorMismatch] otherwise. It
// never claims to detect spoofing when no Resolver is available — that is the
// operator's auth-callout or the (deferred) signature's role.
func VerifyAuthor(claimedAuthor string, headers map[string][]string, r Resolver) error {
	if err := CheckName(claimedAuthor); err != nil {
		return err
	}
	if r == nil {
		return nil
	}
	trusted, ok := r.DeliveredBy(headers)
	if !ok {
		return nil
	}
	if trusted != claimedAuthor {
		return fmt.Errorf("%w: claimed=%q delivered=%q", ErrAuthorMismatch, claimedAuthor, trusted)
	}
	return nil
}
