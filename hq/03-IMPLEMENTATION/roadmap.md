# Roadmap

The live plan. No dates — **gates, not calendars**. Every milestone names the
research gate it depends on; nothing is built ahead of its gate (constitution
IV).

## Where we are

**The project was founded 2026-07-31** ([episode
0001](../04-JOURNEY/0001-genesis.md)): SoulNode is the single-binary
distribution of the Soulstream ecosystem — embedded NATS, SoulIdentity, the
archivist, soulrealm, and an MCP front door in one process, first boot as a
product. No code exists yet — deliberately, per constitution IV: the
composition gate comes first. **Next:** open the `single-binary-composition`
research topic (Phase 0) with `/research-start`.

## Phase 0 — Composition (research) — *next*

Open `single-binary-composition` via `/research-start` (bars are drafted with
the maintainer at open, per how-we-work; the areas below scope the topic, they
are not the bars):

- **In-process admission.** The SoulIdentity service and callout issuer,
  running against an *embedded* operator-mode NATS server, admit and scope
  personas identically to the wire rig. The reference measurement is
  soulstream's `remote-mcp-node` rig (Bars 1–3 measured PASS on the wire,
  2026-07-30; see `../../../soulstream/hq/01-RESEARCH/` while the topic is
  open, git history after).
- **Embed surfaces.** The minimal public surface each component needs so
  SoulNode can wire it without `internal/` imports (constitution I). Known
  today [measured]: SoulIdentity's serve path lives entirely under
  `internal/`; soulrealm's natstest operator rig is `internal/` too.
- **The ceremony, enumerated.** Every artifact `soulnode init` must generate
  — operator, system/AUTH/realm accounts, signing keys, sentinel, vault first
  key, buckets — written down end-to-end with nothing implicit.

**Gate for Phase 1.** The outcome decides whether the bundle is one process
(the founding bet) or a supervised multi-process fallback (the named
reversal, episode 0001).

## Phase 1 — The bundle (design → build) — gated on Phase 0

Runs the spec-kit flow against the design Phase 0 graduates into. Exit
criteria made precise per feature in `specs/NNN-*/`:

- **M1.1 — The server and the identity plane in one process.** Embedded
  operator-mode NATS (JetStream on a state dir) + the SoulIdentity service
  and callout issuer, wired in-process; admission proven with the same
  observations as the wire rig.
- **M1.2 — `soulnode init`.** The full first-boot ceremony, zero manual key
  steps, idempotent on an existing state dir (constitution V).
- **M1.3 — The realm joins.** The soulrealm node (native backend) and the
  archivist run in-process; an agent launches, posts a turn attributed to its
  persona, and memory answers.

External dependencies, tracked openly: the public embed surfaces land
upstream (SoulIdentity, soulrealm, archivist — constitution I forbids
`internal/` reaches); soulstream↔soulrealm release tagging (soulrealm's main
carries a `replace` directive today [measured], which SoulNode cannot consume).

## Phase 2 — The front door (design → build) — gated

The MCP edge in-process: streamable HTTP, static-bearer admission through the
callout, corpse-evicting connection pool. HTTPS documented via the host's
`tailscale serve` — no embedded tailnet yet. External reference: soulstream's
`remote-mcp-node` topic (Bars 1–3 PASS; Bar 4 — the no-install client — in
progress there). Gate: that topic's outcome, plus a design pass on what of
the prototype node graduates where.

## Phase 3 — The tailnet inside — gated

An embedded tailnet node (tsnet) behind a flag: a stable MagicDNS name and
HTTPS certs with zero host setup. Built only if Phase 2 measures the
host-`tailscale serve` path insufficient for the audience — the dependency is
heavy, and the gate exists to keep it honest.

## Later horizons (named, not planned)

Each will get its own research gate when it approaches:

- **Day 2.** Upgrade in place, backup/restore of the state dir, moving a
  realm to a new machine as a copy.
- **Multi-node.** Deferred to soulrealm's Fleet work; SoulNode stays
  single-node until Fleet is real and a second node is a measured need.

## Discipline

Exit criteria are written before the work and amended only openly with the raw
findings recorded. Landing a feature updates this file, writes a journey
episode, and propagates design changes — in the same merge (constitution VI).
