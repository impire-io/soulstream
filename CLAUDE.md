# SoulNode — orientation

**Read [`../soul-hq/`](../soul-hq/README.md) first.** Everything about how this project is run
lives there: the vision and constitution ([`../soul-hq/00-GENESIS/`](../soul-hq/00-GENESIS/README.md)),
active research ([`../soul-hq/01-RESEARCH/`](../soul-hq/01-RESEARCH/README.md)), designs
([`../soul-hq/02-DESIGN/soulstream/`](../soul-hq/02-DESIGN/soulstream/README.md)), the roadmap
([`../soul-hq/03-IMPLEMENTATION/`](../soul-hq/03-IMPLEMENTATION/README.md)), and the honest log
([`../soul-hq/04-JOURNEY/`](../soul-hq/04-JOURNEY/README.md)).

## What this is

SoulNode is the **single-binary distribution** of the Soulstream ecosystem:
embedded NATS (operator mode, JetStream), the SoulIdentity identity plane,
the archivist, the soulstream-workloads runtime, and an MCP front door in one process —
`soulstream init && soulstream up` on a machine the user owns. Soulstream is the
record, soulstream-workloads is the room, SoulIdentity is the name; SoulNode is the
house.

## Status

**Phases 1 and 2 (local mode) complete** (journeys 0003–0006,
2026-08-02): `init` founds a realm (~0.15 s, token printed once); `up`
runs server + identity + memory + the MCP door on loopback; `workload
start` runs a declared agent under enforcement. An MCP client with the
founding token gets the full tool surface with realm-admitted identity.
All measured green in `make test`. Standing exception: four
pseudo-version pins await upstream tags (soulstream-identity, archivist,
soulstream-workloads, soulstream/node). Public door mode waits on soulstream-idp
upstream; Phase 3 (tsnet) keeps its measurement gate.

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
  build. Workloads run outside, through soulstream-workloads's backends.
- **Explore → Plan → Code → Commit.** Research goes through `01-RESEARCH/`
  and never through spec-kit; implementation always goes through spec-kit.
- **Quality gate:** `make fmt && make test && make lint` — all green, nothing
  skipped, before any "done" (constitution VI; the hq structural lint rides the soul-hq gate).
- Go module `github.com/impire-io/soulstream`; connect to external NATS via
  `orbit.go/natscontext`, modern `nats.go/jetstream` API; never `nats.ws`.
- Sign every commit. Never commit `.claude/settings.local.json`.
- **The journey duty:** every landed feature, concluded investigation, or
  load-bearing decision gets a numbered episode in `../soul-hq/04-JOURNEY/` in the
  same change (`/journey-log`; research via `/research-graduate`). Never
  push; pushing stays a human act.
