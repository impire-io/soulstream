# SoulNode — orientation

**Read [`hq/`](hq/README.md) first.** Everything about how this project is run
lives there: the vision and constitution ([`hq/00-GENESIS/`](hq/00-GENESIS/README.md)),
active research ([`hq/01-RESEARCH/`](hq/01-RESEARCH/README.md)), designs
([`hq/02-DESIGN/`](hq/02-DESIGN/README.md)), the roadmap
([`hq/03-IMPLEMENTATION/`](hq/03-IMPLEMENTATION/README.md)), and the honest log
([`hq/04-JOURNEY/`](hq/04-JOURNEY/README.md)).

## What this is

SoulNode is the **single-binary distribution** of the Soulstream ecosystem:
embedded NATS (operator mode, JetStream), the SoulIdentity identity plane,
the archivist, the soulrealm runtime, and an MCP front door in one process —
`soulnode init && soulnode up` on a machine the user owns. Soulstream is the
record, soulrealm is the room, SoulIdentity is the name; SoulNode is the
house.

## Status

**Composition gate met — Phase 1 unblocked** ([journey
0002](hq/04-JOURNEY/0002-the-composition-gate.md), 2026-08-02): all three
bars PASS, constitution ratified 1.0.0, transport decided all-loopback
(decomposition is configuration). Design
[`0001-soulnode-composition.md`](hq/02-DESIGN/0001-soulnode-composition.md)
governs Phase 1; the upstream embed seams exist (soulidentity `embed.Run`,
archivist `keeper`/`archive`, soulrealm pinned to soulstream v0.6.0).
**Next:** the spec-kit pass for M1.1 (`soulnode init` + server + identity
plane).

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan:
`specs/001-init-and-up/plan.md` (spec, research, data model, contracts,
and quickstart live beside it).
<!-- SPECKIT END -->

## The rules that bind every change

- **Composition, not invention** (constitution I, non-negotiable): no domain
  logic in SoulNode; components are consumed as tagged releases through
  public packages — never `internal/` imports, never `replace` on main. What
  needs new behavior lands upstream first.
- **Same shape as any deployment** (constitution II): operator-mode NATS +
  auth-callout admission, identical to hosted; no dev-only auth lane.
- **One process, planes by configuration** (constitution III): enabled
  planes in one process, each on an ordinary loopback NATS connection;
  repointing or disabling a plane is configuration, never a different
  build. Workloads run outside, through soulrealm's backends.
- **Explore → Plan → Code → Commit.** Research goes through `01-RESEARCH/`
  and never through spec-kit; implementation always goes through spec-kit.
- **Quality gate:** `make fmt && make test && make lint` — all green, nothing
  skipped, before any "done" (constitution VI; the hq structural lint,
  `internal/hqlint`, rides `make test`).
- Go module `github.com/impire-io/soulnode`; connect to external NATS via
  `orbit.go/natscontext`, modern `nats.go/jetstream` API; never `nats.ws`.
- Sign every commit. Never commit `.claude/settings.local.json`.
- **The journey duty:** every landed feature, concluded investigation, or
  load-bearing decision gets a numbered episode in `hq/04-JOURNEY/` in the
  same change (`/journey-log`; research via `/research-graduate`). Never
  push; pushing stays a human act.
