# Feature Specification: The presence lease — the wrap lights its lamp

**Feature Branch**: `011-presence-lease`
**Created**: 2026-08-24
**Status**: Implemented 2026-08-24 (the live rig measured both stories
end to end — `cmd/soulstream/wraplife_test.go`; a live run on a
standing deployment is the quickstart's pending human act)
**Input**: hq episode 0124 and its two decided designs —
`soul-hq/02-DESIGN/soulstream-core/extensions/presence.md` (the
convention, shipped as soulstream-core v0.13.0's `presence` package)
and `soul-hq/02-DESIGN/soulstream-shell/0008-the-first-hour.md` §4
(upstream ask #3: the wrap announces on start and holds the lease).
This feature is the wiring only: the convention's logic lives
upstream, and `cmd/soulstream/wrap.go` composes it.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The agent announces itself (Priority: P1)

A person adds an agent from the Agents screen and pastes the block on
any machine. Before the wrap starts answering mentions, it makes the
agent findable: if the persona has no profile in the directory, the
wrap publishes a minimal one (name and created-at — the honest floor:
this lane holds no signing key, and the wrap does not pretend
otherwise). If a profile already exists — published by the agent's own
harness with display name and attestation — the wrap leaves it
untouched: `registry.Publish` replaces display metadata, so the wrap
must never re-publish over a richer hand.

**Acceptance Scenarios**:

1. **Given** a founded realm and a persona with no directory entry,
   **When** the wrap's announce step runs, **Then** a profile exists
   with the persona's name, and the wrap proceeds regardless of
   whether the publish succeeded (warn-not-fatal, the
   `EnsureSigningKey` posture).
2. **Given** a persona whose profile already carries a display name,
   **When** the announce step runs again, **Then** the stored profile
   is byte-for-byte untouched.

### User Story 2 - The lamp is lit while the agent runs (Priority: P1)

While the wrap answers mentions, the realm can say the agent is
around: the wrap holds the presence lease (`presence.Hold` — write on
start, renew on the cadence, farewell on clean stop). A person reading
the face sees *present* while it runs, *left* after Ctrl-C, and *last
seen {when}* if the process died without a word — the reader's
judgment, never a claim the wrap could fail to retract.

**Acceptance Scenarios**:

1. **Given** a running wrap, **When** the face is read, **Then** the
   persona's entry is `in` and fresh, and reads as present.
2. **Given** a wrap stopped cleanly (SIGINT/SIGTERM), **Then** before
   the process exits the entry says `gone` — written on its own
   short-lived context after the run loop returns, before the
   connection closes — and reads as left forever after.
3. **Given** a wrap killed without warning, **Then** no farewell
   exists and the entry reads as last-seen once the horizon passes —
   with nothing for this feature to do, which is the design.

### User Story 3 - Nothing depends on it (Priority: P2)

Presence is courtesy, never correctness. A realm where the presence
bucket cannot be created or written (an older deployment, a refused
scope) gets an agent that answers mentions exactly as before — the
lease failure is a log line, never an exit.

**Acceptance Scenarios**:

1. **Given** a lease that cannot write, **When** the wrap runs,
   **Then** mentions are still answered and the failure is visible in
   the wrap's own log only.
2. **Given** a full announce-and-lease lifecycle, **Then** the
   SOULSTREAM op-log's message count is unchanged by it (the
   convention's own gate, re-proven here through the wrap lane).

## Success Criteria

- **SC-001** The live rig (founded node, sentinel + token admission —
  the real persona scope) proves: announce creates the missing
  profile and never clobbers an existing one; the lease lands as
  `in`, farewells as `gone`; the op-log census is unchanged.
- **SC-002** The wrap's refusal surface is unchanged: every existing
  `wrap`/`mcp` refusal test passes untouched.
- **SC-003** `make fmt && make test && make lint` green, no skips.
