# Contract: Library API (new & changed)

## identity (NATS-free)

```go
// sign.go
type SigningKey struct{ /* unexported */ }
func GenerateSigningKey() (*SigningKey, error)            // crypto/rand seed
func SigningKeyFromSeed(seed []byte) (*SigningKey, error) // 32 bytes exactly
func (k *SigningKey) Seed() []byte                        // 32 bytes, caller guards it
func (k *SigningKey) PublicKey() string                   // std base64, 32 bytes
func (k *SigningKey) Sign(canonical []byte) string        // std base64, 64 bytes
func VerifySignature(publicKeyB64 string, canonical []byte, sigB64 string) bool
// malformed key or sig → false, never panic

// RotationProofBytes returns the domain-separated bytes the OLD key signs to endorse
// a new key: "soulstream-key-rotation\n" + persona + "\n" + newPublicKeyB64.
func RotationProofBytes(persona, newPublicKeyB64 string) []byte

// keyring.go
type Keyring struct {
    Keys       map[string][]string // persona → validated chain, oldest→current (base64)
    Distrusted map[string]bool     // persona → substitution suspected
}
// nil *Keyring is valid everywhere: signed ops verify as unknown-key.
```

## registry (NATS-touching; chain logic pure)

```go
const BucketName = "soulstream-personas"

type SigningKeyInfo struct { Ed25519 string; Since time.Time }
type Rotation struct { From, To, Proof string }
type Profile struct {
    Name        string
    DisplayName string
    Kind        string // "human" | "agent" | "service" — presentation only
    Description string
    OperatedBy  string
    CreatedAt   time.Time
    SigningKey  *SigningKeyInfo
    Rotations   []Rotation
}

// pure (chain.go)
func Chain(p Profile) ([]string, error)   // validated chain per data-model; error ⇒ distrust
func BuildKeyring(profiles []Profile, pinned map[string][]string) (*identity.Keyring, map[string][]string)
// returns the keyring plus the updated pin map (extended chains re-pinned);
// diverged/invalid profiles land in Keyring.Distrusted, pins untouched for them

// KV I/O (kv.go)
func Publish(ctx context.Context, c *realm.Client, p Profile) error
// create-or-metadata-update (FR-004): absent persona ⇒ KV Create with p; existing
// persona ⇒ stored signing_key/rotations are authoritative — metadata is updated with
// Update(rev), an incoming nil or equal key preserves the stored key material, and an
// incoming *different* key is refused with ErrKeyConflict (key changes go through
// Rotate). This also implements the second-client-different-key edge case.
func Rotate(ctx context.Context, c *realm.Client, key *identity.SigningKey, oldSeed []byte) (Profile, error)
// reads own profile, appends rotation (proof by old key), Update(rev)
func Lookup(ctx context.Context, c *realm.Client, persona string) (Profile, bool, error)
// (Profile{}, false, nil) when the bucket or key is absent — absence is not an error
func All(ctx context.Context, c *realm.Client) ([]Profile, error)
// empty slice when the bucket is absent
var ErrKeyConflict error
```

## realm

```go
type Config struct {
    ContextName string
    Realm       string
    Persona     string
    Signer      *identity.SigningKey // NEW, optional; nil ⇒ publish unsigned (unchanged)
}
func (c *Client) Signer() *identity.SigningKey // NEW

// Provision gains the third artefact:
const ArtefactPersonas Artefact = "personas" // KV soulstream-personas, history 10
// create-or-report like stream/objects: existing bucket reported, never modified
```

## topic

```go
// verify.go (pure)
type SigStatus string
const (
    SigUnsigned   SigStatus = "unsigned"
    SigVerified   SigStatus = "verified"
    SigFailed     SigStatus = "failed"
    SigUnknownKey SigStatus = "unknown-key"
)
func VerifyRecord(rec record.Record, realmName, binding string, kr *identity.Keyring) SigStatus
func annotate(recs []SeqRecord, realmName, binding string, kr *identity.Keyring) map[string]SigStatus

// view structs gain:  Sig SigStatus   (Contribution, Attachment, Announcement, Notification)

// read paths gain keyring plumbing (nil = today's behaviour, statuses degrade):
func (h *Handle) Materialise(ctx context.Context, ...) // computes Sig when the handle's
// client owner supplies a keyring via Handle.UseKeyring(kr) — one setter, no config struct
func (h *Handle) UseKeyring(kr *identity.Keyring)
func FetchInbox(ctx, c, persona, limit, kr *identity.Keyring) // signature extended
func FollowInbox(ctx, c, persona, kr, handler)                // signature extended

// wire.go: publishOp signs when c.Signer() != nil — no API change, behaviour only.
// Canonical binding rule (subjects.go): OPS/INFO → topic path; NOTIFY → persona.
```

Compatibility notes:

- `record` package: **no changes**. `Signature` field, `HeaderSig`, and canonical
  `sig`/`omitempty` behaviour already exist and are correct.
- Existing callers passing no keyring (nil) compile-break only where signatures change
  (`FetchInbox`, `FollowInbox`); internal callers are updated in the same change. The
  module is pre-1.0 and single-consumer; no deprecation shims (constitution II).
