# Roadmap

The live plan. No dates — **gates, not calendars**. Every milestone names the
research gate it depends on; nothing is built ahead of its gate (constitution
IV).

## Where we are

**The composition gate is met — Phase 1 is unblocked** ([episode
0002](../04-JOURNEY/0002-the-composition-gate.md), 2026-08-02): all three
pre-registered bars measured PASS, three of the four upstream embed asks
delivered before graduation, the transport decided all-loopback by the
maintainer (decomposition is configuration), and the constitution ratified
at 1.0.0. Design
[`0001-soulnode-composition.md`](../02-DESIGN/0001-soulnode-composition.md)
governs Phase 1. **Next:** the spec-kit pass for M1.1.

## Phase 0 — Composition (research) — ✅ closed 2026-08-02

**Gate met.** `single-binary-composition` graduated to design (episode
0002). Decided: five planes in one process, every plane on an ordinary
loopback NATS connection (constitution III as ratified); the first-boot
ceremony is code, provisioning through public surfaces only; the
in-process pipe transport is a finding of record (fixed ~10 s mute
refusals — candidate upstream issue), not the product shape.

## Phase 1 — The bundle (design → build) — *unblocked*

Runs the spec-kit flow against design 0001 (§9 acceptance criteria). Exit
criteria made precise per feature in `specs/NNN-*/`:

- **M1.1 — The server and the identity plane.** `soulnode init` (the full
  persisted ceremony, idempotent, zero manual steps) + `soulnode up`:
  embedded operator-mode server on loopback, SoulIdentity in-process via
  its public `embed.Run`, admission proven (printed token admits scoped;
  garbage/revoked refused, audited).
- **M1.2 — The realm joins.** Realm provisioned; the archivist keeps ops
  and answers memory through its public `keeper`/`archive` seam.
- **M1.3 — An agent runs.** A declared workload launches through
  soulrealm's public packages (native backend), posts an attributed turn,
  lifecycle as work ops — soulrealm's own M1.1 proof re-run inside
  SoulNode.

External dependency, tracked openly: soulrealm has no tagged release yet
(it pins soulstream v0.6.0 but is itself consumed at `main` until it
tags); SoulNode pins it the moment it does.

## Phase 2 — The front door (design → build) — gated

The MCP edge in-process: streamable HTTP, static-bearer admission through
the callout, per-user pooled **loopback** connections, corpse eviction.
Gate: soulstream's `remote-mcp-node` topic (the maintainer's open
investigation — Bars 1–3 PASS there, Bar 4 in flight) and the public MCP
surface it graduates into — the fourth upstream ask, deliberately held for
that vehicle.

## Phase 3 — The tailnet inside — gated

An embedded tailnet node (tsnet) behind a flag: a stable MagicDNS name and
HTTPS certs with zero host setup. Built only if Phase 2 measures the
host-`tailscale serve` path insufficient for the audience — the dependency
is heavy, and the gate exists to keep it honest.

## Later horizons (named, not planned)

Each will get its own research gate when it approaches:

- **BYO NATS.** Design 0001 §4 carries the [O]: the ceremony subset
  against a user-supplied server. Ships behind its own pass, not with the
  bundle.
- **Day 2.** Upgrade in place, backup/restore of the state dir, moving a
  realm to a new machine as a copy.
- **Multi-node.** Deferred to soulrealm's Fleet work; SoulNode stays
  single-node until the upstream node supervisor exists and a second node
  is a measured need.

## Discipline

Exit criteria are written before the work and amended only openly with the
raw findings recorded. Landing a feature updates this file, writes a
journey episode, and propagates design changes — in the same merge
(constitution VI).
