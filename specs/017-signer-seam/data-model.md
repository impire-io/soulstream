# Data Model: Signer Seam (017)

No stored state changes: no wire-form, stream, KV, or object-store shape
moves. The "model" of this feature is the capability contract and the
invariants around it.

## Signer (the contract) — `identity.Signer`

| Member | Type | Meaning |
|---|---|---|
| `PublicKey()` | `string` | standard base64 of the 32 raw Ed25519 public-key bytes — the same encoding profiles and pins already use |
| `Sign(canonical []byte)` | `(string, error)` | standard base64 of the 64 raw signature bytes over the given canonical bytes, or an error when no signature could be produced |

**Contract rules** (documented on the interface, tested where testable):

- Implementations MUST be safe for concurrent use (FR-011).
- `Sign` MUST return either a non-empty signature or an error — never
  `("", nil)`. The chokepoint defends anyway (FR-005), but the contract
  states it so implementations fail their own tests, not integration.
- `PublicKey` identifies the key the implementation signs *as*; whether that
  key is the persona's published key is the read side's question (the
  persona directory stays the authority — spec Edge Cases).
- The interface implies no access to secret key material (FR-008): nothing
  in the contract can return, name, or require a seed.

## Local signing key — `identity.SigningKey` (existing, reshaped)

- `Sign` changes shape: `(canonical []byte) (string, error)`, error always
  nil. Determinism (same key + same bytes ⇒ same signature) is unchanged —
  it is what SC-001's byte-identical criterion measures.
- Satisfies `Signer` with no wrapper. All other members (`Seed`,
  `PublicKey`, generation/reconstruction) untouched.
- Remains the only type the keystore ever stores or loads.

## Delegated signer (external; represented here by test doubles)

Not a shipped type. In tests, a double implements `Signer` by holding a real
`SigningKey` behind the interface (proving transparency: SC-001) and by
returning injected errors or an empty string (proving FR-004/FR-005/FR-012:
SC-002/SC-005). The double's shape mirrors what SoulIdentity's client will
be for a fixed persona: capability in, signature or error out.

## State transitions

None. A publish attempt with a failing signer produces *no* record —
there is no partial state to model (the failure happens before
`record.Build`, before any NATS publish). Responder failure produces no
reply — the asker's state machine already models silence.

## Relationships

```text
realm.Config.Signer ──(is a)── identity.Signer
identity.SigningKey ──(satisfies)── identity.Signer
topic buildOpMsg ──(consumes)── Client.Signer()      [records + replies]
registry.NewAttestationToken ──(consumes)── Signer   [statement: attestation]
registry.Rotate ──(consumes old+new)── Signer        [statement: rotation proof]
internal/keystore ──(stores/loads)── *SigningKey     [never the interface]
```

## Validation rules carried from the spec

| Rule | Where enforced |
|---|---|
| signer error ⇒ publish error naming the cause (FR-004) | `buildOpMsg` |
| empty signature ⇒ same as signer error (FR-005) | `buildOpMsg` |
| nil signer ⇒ unsigned publish, byte-identical to today (FR-006) | `buildOpMsg` (`!= nil` unchanged) |
| responder cannot sign ⇒ silence + `served(-1)` (FR-012) | existing `berr` paths in discover/memory responders |
| concrete keys only in seed custody (FR-008) | type system: keystore signatures unchanged |
| no NATS in the contract (FR-009) | `identity` import graph (existing test conventions guard it) |
