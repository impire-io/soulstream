# Library Contract Deltas: Signer Seam (017)

The Go API surface this feature changes, and the behavior contracts behind
it. Everything not listed is unchanged.

## `identity` (pure — imports no NATS)

```go
// NEW — identity/signer.go
//
// Signer is the capability to sign canonical bytes as one public identity.
// Implementations must be safe for concurrent use, and must never return
// ("", nil): a signature or an error, nothing between. The contract grants
// no access to secret key material — where the key lives is the
// implementation's business (a local seed, a remote custodian).
type Signer interface {
        // PublicKey returns standard base64 of the 32 raw Ed25519
        // public-key bytes — the encoding profiles and pins use.
        PublicKey() string
        // Sign returns standard base64 of the 64 raw signature bytes over
        // canonical, or an error when no signature could be produced.
        Sign(canonical []byte) (string, error)
}

// CHANGED — identity/sign.go
// Before: func (k *SigningKey) Sign(canonical []byte) string
func (k *SigningKey) Sign(canonical []byte) (string, error) // error always nil
// *SigningKey satisfies Signer. Everything else in the package unchanged.
```

Compile-time impact on importers: any caller of `SigningKey.Sign` gains an
error to handle. Accepted per spec assumption 3; the compiler finds every
site.

## `realm`

```go
// CHANGED — realm/connect.go
type Config struct {
        // …
        // Signer is optional; when set, every op this client publishes carries
        // an Ed25519 signature over its canonical record and a signing failure
        // fails the operation — there is no unsigned fallback. Nil publishes
        // unsigned. Assign a *identity.SigningKey only when it is non-nil: a
        // typed-nil pointer inside a non-nil interface is a caller bug that
        // panics at first use.
        Signer identity.Signer // was *identity.SigningKey
}

func (c *Client) Signer() identity.Signer // was *identity.SigningKey
```

## `topic` (behavior contract at the chokepoint)

`buildOpMsg` (unexported; the contract is observable through every publish
and responder surface):

- signer nil → record unsigned, byte-identical to pre-017 output (FR-006).
- signer returns error → the operation fails with an error wrapping the
  signer's error and naming the op type; nothing is published (FR-004).
- signer returns `("", nil)` → same as an error: "signer returned an empty
  signature" (FR-005).
- signer returns a non-empty signature → it travels verbatim; the publisher
  does not verify (read side remains the only verifier — R4).

Responder surfaces (`RespondDiscovery`, `RespondDiscoveryWith`,
`RespondMemory`): a signing failure while building a reply produces no reply
and reports through the existing observability callback with sent = `-1`
(FR-012, SC-005). No signature: silence is indistinguishable from no-match
to the asker — by design.

## `registry`

```go
// CHANGED — registry/attest.go
// Before: func NewAttestationToken(signer *identity.SigningKey, …)
func NewAttestationToken(signer identity.Signer, operator, operated, operatedKeyB64 string) (string, error)
// signer == nil keeps its current refusal; a signer error propagates wrapped.

// CHANGED — registry/kv.go
// Before: func Rotate(ctx …, oldKey, newKey *identity.SigningKey) (Profile, error)
func Rotate(ctx context.Context, c *realm.Client, oldKey, newKey identity.Signer) (Profile, error)
// oldKey: Sign + PublicKey (proof + current-key match). newKey: PublicKey
// only. A signing failure aborts the rotation before any KV write.
```

## Unchanged by contract (the freeze list)

- `internal/keystore`: `SaveKey/LoadKey/ReplaceKey` keep `*identity.SigningKey`
  (FR-008 enforced by the type system).
- `identity.GenerateSigningKey`, `SigningKeyFromSeed`, `Seed`,
  `VerifySignature`, `RotationProofBytes`, `AttestationBytes`, keyring,
  rotation-chain validation: untouched.
- Wire form, canonical form, `SigStatus` semantics, all read-side paths.
- CLI commands, MCP tools, config files, key file formats (FR-010) — the
  binaries change only internally (they hand their loaded concrete key to an
  interface-typed field).
- `go.mod`: no new dependencies (SC-004).

## Consumer contract (the seam's whole point)

An external custodian client satisfies the seam by importing only
`identity`:

```go
// e.g. in the consumer's codebase, NOT in soulstream:
type custodianSigner struct{ cl *soulidentity.Client; persona string }

func (s custodianSigner) PublicKey() string { /* from the persona directory or custodian */ }
func (s custodianSigner) Sign(canonical []byte) (string, error) {
        return s.cl.SignRecord(s.persona, canonical) // deadline owned by cl
}
```

soulstream never imports the custodian; the custodian imports
`soulstream/identity` (or neither imports the other and the consumer binary
adapts). Cross-service verification of a live custodian-signed record is
SoulIdentity M2's gate, exercised there.
