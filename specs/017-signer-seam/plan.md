# Implementation Plan: Signer Seam

**Branch**: `017-signer-seam` | **Date**: 2026-07-29 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/017-signer-seam/spec.md`

## Summary

Replace the concrete `*identity.SigningKey` in every signing seam with a small
`identity.Signer` interface — public key + fallible signing over canonical
bytes — so a consumer can delegate signing to an external custodian
(SoulIdentity's `sign.record` NATS service) without soulstream depending on
it. The local key becomes the first implementation; signing failure becomes a
publish failure at the one chokepoint (`topic/wire.go:buildOpMsg`), which the
responder paths already convert to protocol silence plus an observability
callback. Statement-signing surfaces (`registry.NewAttestationToken`,
`registry.Rotate`) accept the interface; seed custody (`internal/keystore`,
key generation) stays concrete. No behavior changes for local-key or unsigned
clients.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire-io/soulstream`)
**Primary Dependencies**: unchanged — nats.go v1.52 + `jetstream`, orbit
`natscontext`, `gowebpki/jcs`, `google/uuid`; test-only embedded
`nats-server/v2`. FR-009/SC-004 forbid additions; the seam is pure Go stdlib.
**Storage**: N/A (no new persistence; existing streams/KV untouched)
**Testing**: `go test ./...` via `make check` (fmt+tidy+build+test+lint);
embedded JetStream through `internal/natstest` where the wire is exercised
**Target Platform**: library (all six release platforms), CLI + MCP binaries
**Project Type**: library seam + mechanical call-site updates
**Performance Goals**: none new — local signing cost unchanged; delegated
latency is the implementation's property (spec assumption)
**Constraints**: `identity` imports no NATS (FR-009); no config surface
learns "delegated signer" (spec assumption 4); CLI/MCP user-visible behavior
frozen (FR-010)
**Scale/Scope**: ~6 non-test files touched + tests; 3 production `.Sign(`
call sites (topic/wire.go:80, registry/attest.go:87, registry/kv.go:158),
1 `.PublicKey()`-only consumer (internal/mcpserver/tools.go:252), 1 config
field + accessor (realm/connect.go:26,125)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. NATS-Native First**: PASS. No new capability is built beside NATS —
  this feature adds zero infrastructure and zero NATS machinery; it reshapes
  a pure-Go seam so that a *future* consumer can put NATS request/reply (the
  custodian's own service) behind it. No server feature is relied upon; no
  minimum server version changes.
- **II. Smallest Viable Implementation**: PASS, with the abstraction argued
  rather than waved through: an interface is normally exactly the
  "speculative abstraction layer" this article prohibits. It is not
  speculative here — the concrete consumer exists today (SoulIdentity M2
  names the soulstream Signer seam as its wiring point, approved as 017),
  and the seam is the *smallest* form of that need: one interface with two
  methods, no options, no registry of signers, no config surface, no second
  implementation shipped here. The genuinely speculative variants were cut
  in research: no context parameter (R3), no publish-side verification (R4),
  no delegated-signer helper types.
- **III. ELI5 Documentation**: PASS — `docs/signing.md` gains a plainly
  worded section ("someone else can hold your pen": the pen stays in a
  vault; you describe what to write; the letter looks exactly the same") in
  the same change (US-level docs task), covering delegation, the
  fail-loudly rule, and responder silence.

*Post-design re-check (after Phase 1)*: still PASS — the contracts add no
surface beyond the two-method interface and the signature changes it forces;
the data model introduces no new stored state.

## Project Structure

### Documentation (this feature)

```text
specs/017-signer-seam/
├── plan.md              # This file
├── research.md          # Phase 0: decisions R1–R7
├── data-model.md        # Phase 1: contract entities + invariants
├── quickstart.md        # Phase 1: wiring a delegated signer (consumer view)
├── contracts/
│   └── library.md       # Phase 1: Go API + behavior contract deltas
└── tasks.md             # Phase 2 (/speckit-tasks — not created here)
```

### Source Code (repository root)

```text
identity/
├── sign.go              # SigningKey.Sign → (string, error); Signer docs anchor
├── signer.go            # NEW: the Signer interface (pure, no NATS)
└── sign_test.go         # local key satisfies Signer; determinism; nil-error law

realm/
└── connect.go           # Config.Signer identity.Signer; Client.Signer() returns it

topic/
├── wire.go              # buildOpMsg: fallible sign, empty-sig = error (FR-004/005)
├── sign_test.go         # delegated-double publish/verify; failure injection
├── discover_test.go     # responder silence on signing failure (FR-012/SC-005)
└── memory_test.go       # witness silence on signing failure (FR-012/SC-005)

registry/
├── attest.go            # NewAttestationToken(signer identity.Signer, …)
├── kv.go                # Rotate(ctx, c, old, new identity.Signer)
├── attest_test.go       # delegated attestation verifies (US3)
└── chain_test.go        # delegated rotation proof validates (US3)

internal/cli/            # compile-only: wiring passes *SigningKey as Signer
internal/mcpserver/      # compile-only: PublicKey() through the interface
internal/keystore/       # UNTOUCHED — seed custody stays concrete (FR-008)

docs/
└── signing.md           # ELI5: delegated signing, fail-loudly, responder silence
```

**Structure Decision**: no new packages beyond one new file in `identity`.
The seam reuses the existing chokepoint architecture: every record that
signs already flows through `topic/wire.go:buildOpMsg`, and every reply-
signing responder already treats a build error as silence + `served(-1)`
(`topic/discover.go:225`, `topic/memory.go:395,423`) — so FR-004/FR-012 are
enforced at two existing sites, not new machinery.

## Complexity Tracking

No constitution violations to justify. The one debatable addition — the
interface itself — is argued under Article II above and R1/R2 in research.md.
