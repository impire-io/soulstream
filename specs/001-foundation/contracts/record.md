# Contract: `record` package

Pure operation-record surface. **Imports nothing from NATS** — it works with header maps and byte
payloads, so callers in `realm` (and later features) adapt to/from `nats.Header`.

## Types

```go
package record

// Version is the only supported wire version.
const Version = 1

// Record is one operation. Immutable after construction; fields are validated by Validate.
type Record struct {
    ID        string            // UUIDv4, lowercase 8-4-4-4-12 (also the Nats-Msg-Id)
    Author    string            // persona slug
    Parents   []string          // ordered op-ids; nil/empty == no parents
    Type      string            // e.g. "turn.post"; non-empty; not enumerated here
    Timestamp time.Time         // author-claimed, RFC 3339 on the wire; informational only
    Signature string            // optional; carried through, never produced/verified here
    Payload   []byte            // pure data (text/references); opaque bytes
    Extras    map[string]string // unknown Soulstream-* headers, preserved verbatim
}
```

## Functions

```go
// NewID returns a fresh UUIDv4 operation identity (lowercase 8-4-4-4-12).
func NewID() string

// Build validates r and returns the wire form: the header set (including Nats-Msg-Id == r.ID and
// Soulstream-Version) and the payload bytes. Returns a *FieldError on invalid input.
func (r Record) Build() (headers map[string][]string, payload []byte, err error)

// Parse reads a wire message back into a Record. It enforces required fields, Version==1, a
// well-formed author slug, RFC 3339 timestamp, and the absent-vs-empty parents rule; it preserves
// unknown Soulstream-* headers into Extras. Returns a *FieldError naming the first violation.
func Parse(headers map[string][]string, payload []byte) (Record, error)

// Validate checks a Record's fields without serialising (used by Build and available standalone).
func (r Record) Validate() error
```

## Canonical form

```go
// Canonical produces the RFC 8785 (JCS) canonical bytes of the record bound to realm+topic.
// Field order in the source is irrelevant: equal content -> byte-identical output. The optional
// "sig" key is omitted when Signature is empty. This is the exact byte sequence any later signing
// must sign; signing itself is out of scope.
func (r Record) Canonical(realm, topic string) ([]byte, error)

// CanonicalRecord is the logical object canonicalised by Canonical (documented for readers /
// future exhibit + sealed-op reuse):
//   { "v":1, "realm":..., "topic":..., "id":..., "author":..., "parents":[...],
//     "ts":..., "type":..., "data":<payload as JSON value> [, "sig":...] }
```

## Errors

```go
// FieldError names the specific field and violation (FR-016). Sentinels allow errors.Is checks.
type FieldError struct { Field, Reason string }
func (e *FieldError) Error() string

var (
    ErrMissingField   = errors.New("record: missing required field")
    ErrBadVersion     = errors.New("record: unsupported version")
    ErrBadTimestamp   = errors.New("record: malformed timestamp")
    ErrBadAuthor      = errors.New("record: invalid author name")
    ErrBadID          = errors.New("record: invalid operation id")
)
```

## Contract guarantees (map to spec)

- **Round-trip inverse** (FR-014, SC-003): `Parse(Build(r))` == `r` across the matrix {0,1,many
  parents} × {sig, no-sig} × {with, without unknown headers}.
- **Absent ⇔ empty parents** (FR-015): empty `Parents` ⇒ no `Soulstream-Parents` header; absent
  header ⇒ empty `Parents`.
- **Unknown headers preserved** (FR-017): any `Soulstream-*` header not in the known set survives a
  round-trip via `Extras`.
- **Determinism** (FR-020, SC-004): `Canonical` is byte-identical for equal content regardless of
  field order.
- **Realm/topic binding** (FR-022): `Canonical` includes `realm` and `topic`.
- **Signature carried, not produced** (FR-023): `Signature` is passed through both directions;
  no signing/verification logic exists here.
- **Payload discipline** (FR-026): documented text/references-only; `Build` does not police size but
  the docs state the contract (attachments belong in the object store — a later feature).
