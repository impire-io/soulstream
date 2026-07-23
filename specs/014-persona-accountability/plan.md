# Implementation Plan: Persona Accountability & Stream Hygiene

**Branch**: `014-persona-accountability` | **Date**: 2026-07-23 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/014-persona-accountability/spec.md`

## Summary

Remove the unverifiable persona classification (`kind`) outright and replace the
taxonomy with a verifiable accountability model: `operated_by` claims may carry an
operator countersignature (same Ed25519 material and chain rule as op signing),
reported everywhere as attested / unverified / failed, with operator chains resolving
to a principal. Fix two retention defects with NATS-native means: narrow the op-log
stream to `SOULSTREAM.TOPICS.>` so transient service traffic is never stored, and give
mention inboxes their own tiny stream with `MaxMsgsPerSubject: 100` so inbox reads
stay flat forever. Provisioning converges legacy realms; no backwards compatibility
anywhere (sole-user codebase). Ships as v0.3.0 with plugin + marketplace bumps.

## Technical Context

**Language/Version**: Go 1.26
**Primary Dependencies**: nats.go v1.52 + `jetstream`, orbit `natscontext`, `gowebpki/jcs`, `google/uuid` (no new deps)
**Storage**: JetStream only — streams `SOULSTREAM` + new `SOULSTREAM_NOTIFY`, KV `soulstream-personas`, object store `soulstream-objects`
**Testing**: `go test` with embedded `nats-server/v2` via `internal/natstest.StartJetStream(t)`; pure logic server-free
**Target Platform**: same six goreleaser platforms as v0.2.0
**Project Type**: library + two thin clients (CLI, MCP adapter)
**Performance Goals**: inbox check cost flat w.r.t. lifetime mentions (SC-005); chain walk O(chain length)
**Constraints**: profiles strict-decoded (fail loud); no backwards compatibility; provisioning converges only the exactly-recognised legacy stream shape
**Scale/Scope**: dogfood realm (NGS) + local realms; directory of tens of personas

## Constitution Check

- **I. NATS-Native First**: PASS. Every mechanism is a built-in primitive: a second
  stream with `MaxMsgsPerSubject` + `DiscardOld` (the exact "last N per subject"
  server feature), purge-by-subject for residue, KV for profiles, Ed25519 via stdlib.
  Newer server features evaluated in research.md D4: per-message TTLs rejected
  (time-based, spec bound is count-based), subject transforms rejected (move, don't
  bound), rollup rejected (voluntary, bounds nothing). Minimum server version:
  unchanged from constitution's 2.12+ target; nothing newer is required.
- **II. Smallest Viable Implementation**: PASS. The second stream is the one
  unavoidable addition (per-subject limits are stream-wide — research.md D4); no new
  op types, no new tools beyond one CLI subcommand (`profile attest`) and one flag;
  attestation reuses the existing key type, signature encoding, and chain-membership
  rule; chain walk and status are pure functions. Removals (kind, its flags/params)
  shrink the surface. No configuration knobs added — the inbox window (100) and
  notify MaxBytes are mandated constants like every other realm setting.
- **III. ELI5 Documentation**: PASS — every story carries its docs task:
  `persona-and-attribution.md` + `persona-directory.md` rewrites (voice-with-a-key),
  NEW `docs/operators.md` (principal chains + attestation — "a parent co-signing the
  permission slip"), `mentions.md` (inbox window — "the fridge holds the newest 100
  notes; the letters stay in the mailbox"), `realm.md`/`provisioning.md` (two
  streams), `discovery.md` (nothing stored), `cli.md`/`mcp.md` surface updates.

*Post-design re-check (after Phase 1)*: no violations introduced; Complexity Tracking
stays empty.

## Project Structure

### Documentation (this feature)

```text
specs/014-persona-accountability/
├── plan.md              # This file
├── research.md          # Phase 0 — decisions D1–D6
├── data-model.md        # Phase 1 — profile/attestation/stream shapes
├── quickstart.md        # Phase 1 — attest flow + realm convergence walkthrough
├── contracts/
│   └── library.md       # Phase 1 — API deltas (identity/registry/realm/topic/CLI/MCP)
└── tasks.md             # Phase 2 (/speckit-tasks — not created by /speckit-plan)
```

### Source Code (repository root)

```text
identity/                  # +AttestationBytes (pure, no NATS)
registry/                  # -Kind; +OperatorAttestation, strict decode, token,
                           #  AttestationStatus, OperatorChain, All warnings
realm/                     # spec.go two streams; conformance per-stream;
                           #  provision converge legacy + migrate + purge; OutcomeUpdated
topic/                     # notify.go reads SOULSTREAM_NOTIFY
internal/cli/              # profile.go: attest, --attestation, show status/chain; -kind
internal/mcpserver/        # tools.go: publish_profile -kind +attestation
docs/                      # operators.md NEW; persona-*, mentions, realm,
                           #  provisioning, discovery, cli, mcp updated
plugins/soulstream/        # 0.3.0 bump (plugin.json, README if it mentions kind)
.claude-plugin/            # marketplace.json 0.3.0
```

**Structure Decision**: existing flat library layout; no new packages. Pure logic
(attestation statement/status, chain walk, strict decode) lands beside its package's
existing pure code so it unit-tests server-free; NATS-touching changes stay in
`realm`/`topic`/`registry` kv paths.

## Complexity Tracking

*(empty — no constitutional violations to justify)*
