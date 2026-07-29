# Research: Signer Seam (017)

Decisions resolving every open design point in the plan's Technical Context.
Evidence classes per the working agreement: [measured] — read from the repo
or a run; [mechanism-argument] — reasoned from how the protocol/Go works;
[judgment] — taste.

## R1 — One fallible `Sign`; the local key's own method changes shape

**Decision**: The interface is

```go
type Signer interface {
        PublicKey() string
        Sign(canonical []byte) (string, error)
}
```

and `(*SigningKey).Sign` itself changes from `string` to `(string, error)`
(always `nil` for the local key), so the concrete type satisfies the
interface with no wrapper and there is exactly one way to sign.

**Rationale**: The three shapes considered:

- (a) *chosen* — change `SigningKey.Sign` to return an error. One method
  name, one semantics; the compiler walks us to every call site
  [mechanism-argument]. Cost: compile-time break for importers — accepted
  explicitly in the spec (Assumptions) and cheap at this module's maturity.
- (b) keep `SigningKey.Sign` infallible; interface uses the same name —
  impossible: the method sets would conflict; the local key could not
  satisfy the interface without a wrapper type, which every current caller
  would have to construct. Rejected.
- (c) give the interface a second name (`SignCanonical`) and `SigningKey`
  both methods — two ways to do the same thing, permanent API noise,
  violates Article II. Rejected.

**Alternatives considered**: (b), (c) above.

## R2 — The interface lives in `identity`, named `Signer`

**Decision**: New file `identity/signer.go`; the name `Signer` (the
`realm.Config` field is already called `Signer` [measured:
realm/connect.go:26], so the ecosystem vocabulary does not move).

**Rationale**: `identity` is where signing already lives and is one of the
two packages guaranteed NATS-free — exactly the property FR-009 needs, and
what lets an external custodian client implement the interface by importing
only `identity` [mechanism-argument]. Alternatives: a new package
(`identity/signer`) adds surface for nothing; `realm` would drag NATS into
the contract's import graph for implementers. Rejected.

## R3 — No context parameter on `Sign`

**Decision**: `Sign(canonical []byte) (string, error)` — no `context.Context`.
Timeout/cancellation policy belongs to the implementation (a delegated signer
sets its own request deadline).

**Rationale**: Pinned in the spec (Assumptions) and confirmed against both
sides of the seam [measured]: the day-one implementer, SoulIdentity's
client, exposes `SignRecord(persona string, canonical []byte) (string, error)`
— context-free, deadline owned by the client; and the call site
`buildOpMsg` has no context today (it is a pure builder; its callers hold
the contexts). Threading `ctx` through would ripple into every publish
helper and both responder callbacks for a need no consumer has expressed.
The reversal condition, recorded in the spec: a real consumer proving it
needs caller-side deadline propagation makes this an additive follow-up
(`SignContext` or an interface upgrade), not a redesign.

**Alternatives considered**: `Sign(ctx, canonical)` — rejected as
speculative plumbing (Article II).

## R4 — The publisher is not a verifier; only *empty* is an error

**Decision**: At the chokepoint, a configured signer that returns an error
aborts the publish (FR-004); a returned *empty* signature is converted to an
error (FR-005). Malformed non-empty signature material is NOT checked
publish-side — it travels and fails read-side verification like any corrupt
signature today.

**Rationale**: Empty is special because the wire form spells "unsigned" as
"no sig member" — an empty signature would silently change the record's
class from signed to unsigned, the exact downgrade FR-004 exists to prevent
[mechanism-argument, from the 006 canonical-form rule that an empty
Signature omits `sig`]. Non-empty garbage changes nothing structurally: the
record claims to be signed and readers already answer that claim
(`SigStatus` failed), so a publish-side verify would duplicate the read
side's job and add a second verifier the protocol deliberately doesn't have.

**Alternatives considered**: full publish-side verification against the
signer's `PublicKey()` — rejected: duplicates read-side machinery, costs a
verify per publish, and defends only against a malfunctioning implementation
the read side already exposes loudly.

## R5 — Responder silence needs no new mechanism

**Decision**: FR-012 (a responder that cannot sign stays silent, observably)
is satisfied by the existing error path: once `buildOpMsg` returns an error
on signing failure, `RespondDiscovery`/`RespondDiscoveryWith` and
`RespondMemory` already skip the reply and report `served(query, -1)` /
`served("query"|"fetch", -1)` through their observability callbacks
[measured: topic/discover.go:223–228, topic/memory.go:393–397, 421–426].
The only change these paths need is tests proving the new failure cause
flows through them.

**Rationale**: The protocol's word for "no answer" is already silence
(008/015); the `-1` sentinel already means "heard but could not serve".
Signing failure joins an existing category rather than inventing one.

## R6 — `realm.Config.Signer` becomes the interface; the typed-nil hazard is
handled at the assignment discipline, not with reflection

**Decision**: `Config.Signer identity.Signer` and
`Client.Signer() identity.Signer`. The chokepoint keeps its `!= nil` check.
Call sites that today hold a possibly-nil `*identity.SigningKey` MUST assign
it only when non-nil (a typed-nil pointer in a non-nil interface would pass
the check and panic on use). The hazard and rule are documented on the
`Config.Signer` field; the CLI/MCP wiring is audited for it during
implementation.

**Rationale**: Go's typed-nil-in-interface is a language fact every
interface seam inherits; the honest mitigations are assignment discipline
(chosen — zero runtime cost, enforced by audit + the wiring tests) or
reflection at validate time (rejected — hides the caller's bug behind
runtime magic and still can't distinguish a deliberately nil-receiver-safe
implementation) [judgment].

## R7 — Which surfaces take the interface, and which stay concrete

**Decision**:

| Surface | After 017 | Why |
|---|---|---|
| `realm.Config.Signer` / `Client.Signer()` | `identity.Signer` | the seam itself |
| `topic/wire.go:buildOpMsg` | interface via `c.Signer()` | the one record-signing chokepoint [measured: all six `buildOpMsg` callers, requests and replies alike] |
| `registry.NewAttestationToken` | `identity.Signer` | needs Sign only (FR-007) |
| `registry.Rotate(old, new)` | both params `identity.Signer` | old needs Sign+PublicKey, new needs PublicKey only [measured: registry/kv.go:150–160] |
| `internal/mcpserver` profile publish | interface (uses `PublicKey()` only) | already capability-minimal [measured: tools.go:251–253] |
| `internal/keystore` Save/Load/Replace | `*identity.SigningKey` unchanged | seed custody — a delegated signer has no seed by definition (FR-008) |
| `identity.GenerateSigningKey` / `SigningKeyFromSeed` | unchanged | key material creation is inherently local |

**Rationale**: The rule that decides every row: *does the surface need the
seed, or only the capability?* Capability-only surfaces take the interface;
seed-custody surfaces keep the concrete type so the type system itself
enforces FR-008 [mechanism-argument].

## Confirmed non-changes (scope guard)

- No config file / env / flag learns "delegated signer" (spec assumption 4).
- No second `Signer` implementation ships in this repo — test doubles only.
- `record` package, wire form, canonical form, `SigStatus` semantics:
  untouched (SC-001's "byte-identical" rests on Ed25519 determinism over
  the same canonical bytes [mechanism-argument]).
- Dependency set unchanged (SC-004) — the seam is stdlib-only.
