# Roadmap

The live plan. No dates — **gates, not calendars**. Every milestone names the
research gate it depends on; nothing is built ahead of its gate (constitution
IV).

## Where we are

**Phase 1 is complete** (episodes
[0003](../04-JOURNEY/0003-first-boot-is-real.md) /
[0004](../04-JOURNEY/0004-the-realm-remembers.md) /
[0005](../04-JOURNEY/0005-an-agent-runs.md), all 2026-08-02): `soulnode
init` founds a realm in ~0.15 s and `soulnode up` runs it — embedded
operator-mode server, identity plane, memory plane on ordinary loopback
connections; `soulnode workload start` runs a declared agent with a
minted credential under full enforcement. Every §9 exit criterion of
design [`0001-soulnode-composition.md`](../02-DESIGN/0001-soulnode-composition.md)
is measured green in `make test`. Standing exception: three
pseudo-version pins await upstream tags (soulidentity, archivist,
soulrealm). **Next:** Phase 2 — the front door — gated on soulstream's
`018-remote-mcp-node` cycle (in flight upstream, carrying the fourth
embed ask).

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

- **M1.1 — The server and the identity plane.** ✅ **Done** ([episode
  0003](../04-JOURNEY/0003-first-boot-is-real.md);
  `specs/001-init-and-up/`). Measured: `init` founds a realm in 0.15 s
  (17 artifacts, owner-only modes, token printed once, re-init a
  verified no-op); the found→admit→refuse→revoke→restart e2e rides
  `make test` in ~1 s; admission matches the research exactly
  (server-asserted persona, own-prefix confinement, audited refusals).
  One open exception tracked: soulidentity pinned at a pseudo-version of
  main until it tags.
- **M1.2 — The realm joins.** ✅ **Done** ([episode
  0004](../04-JOURNEY/0004-the-realm-remembers.md);
  `specs/002-realm-joins/`). Measured: the owner's full admission path —
  post → kept (author `owner`) → memory answers with attribution and a
  citation — in ~5 s inside `make test`; restart exactly-once; the
  disabled-plane arm clean; the archivist's persona key vault-held.
  Second pseudo-version pin tracked (archivist, above its v0.1.0).
- **M1.3 — An agent runs.** ✅ **Done — Phase 1 complete** ([episode
  0005](../04-JOURNEY/0005-an-agent-runs.md);
  `specs/003-an-agent-runs/`). Measured: upstream's `agent-echo`,
  declared unchanged, runs with a minted TTL-bounded credential under
  full enforcement — turn authored by its persona, lifecycle a completed
  work item owned by `runner`, everything kept, nothing
  credential-shaped lingering. The two-keys split landed in the ceremony
  (plain workload minting key beside the scoped admission key). One
  consumer-proven upstream fix on the first enforcing run (soulrealm
  `3fee11f`: agents need `$JS.API.INFO`). Third pseudo-version pin
  tracked (soulrealm, no tags upstream).

External dependency, tracked openly: soulrealm has no tagged release yet
(it pins soulstream v0.6.0 but is itself consumed at `main` until it
tags); SoulNode pins it the moment it does.

## Phase 2 — The front door — ✅ local mode done 2026-08-02

**Gate met the same day**: upstream 018 landed and tagged (soulstream
v0.7.0), its node module made consumable (soulstream journey 0010).

- **The door plane.** ✅ **Done** ([episode
  0006](../04-JOURNEY/0006-the-door-opens.md);
  `specs/004-the-front-door/`). Measured: MCP client + founding token →
  session, tools, realm-admitted `whoami`; garbage refused; the door
  custodies nothing (state dir untouched); disabled arm identical to
  Phase 1. Fourth pseudo-version pin tracked (`soulstream/node`,
  untagged).
- **Public mode — named, upstream-gated**: the OAuth resource-metadata
  story needs an external authorization server (soulfold is upstream's
  intended AS); `planes.door` grows `public_url`/`auth_issuer`
  additively when it exists. HTTPS today is deployment fronting
  (`tailscale serve` before the loopback door).

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
