# Contract: `identity` package

Persona-name validation and attribution enforcement. **Imports nothing from NATS.**

## Name validation

```go
package identity

// ValidName reports whether s is a valid Soulstream slug: ^[a-z0-9]+(-[a-z0-9]+)*$, length 1..64.
// The same grammar validates persona names, realm names, and topic-id slugs.
func ValidName(s string) bool

// CheckName returns nil for a valid name, or a *NameError naming the reason it was rejected
// (empty, too long, uppercase, dot, whitespace, wildcard, leading/trailing/double hyphen).
func CheckName(s string) error

type NameError struct { Name, Reason string }
func (e *NameError) Error() string
```

## Attribution

```go
// EnforceAuthor is the write-side guard (FR-025): it returns an error if the record's Author is not
// the client's own configured persona. A persona-bound client calls this before publishing so it
// can only ever speak as itself.
func EnforceAuthor(ownPersona, recordAuthor string) error

// Resolver optionally supplies the trusted identity that actually delivered a message, when the
// deployment provides one (e.g. an auth-callout stamp). Nil means "no trusted source available".
type Resolver interface {
    // DeliveredBy returns the trusted persona for a delivered message's headers, or ("", false)
    // when it cannot be determined.
    DeliveredBy(headers map[string][]string) (persona string, ok bool)
}

// VerifyAuthor is the read-side check (FR-025): it always validates that claimedAuthor is a
// well-formed persona name; when r != nil and it can resolve a trusted identity, it additionally
// requires the two to match, returning ErrAuthorMismatch otherwise. It never claims to detect
// spoofing when no Resolver is available.
func VerifyAuthor(claimedAuthor string, headers map[string][]string, r Resolver) error

var (
    ErrForeignAuthor  = errors.New("identity: refusing to publish as another persona")
    ErrAuthorMismatch = errors.New("identity: claimed author does not match delivering identity")
)
```

## Contract guarantees (map to spec)

- **Grammar** (FR-024): `ValidName`/`CheckName` accept exactly `^[a-z0-9]+(-[a-z0-9]+)*$` len 1–64;
  every rejection carries a reason.
- **Write-side honesty** (FR-025): `EnforceAuthor` blocks publishing under a foreign name.
- **Read-side best-effort** (FR-025): `VerifyAuthor` validates shape always, cross-checks only when
  a `Resolver` is supplied, and never over-promises detection it cannot perform.
