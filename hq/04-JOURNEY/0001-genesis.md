# Episode 0001 — Genesis: SoulNode gets an HQ (2026-07-31)

SoulNode was founded as the fourth sibling of the Soulstream ecosystem —
soulstream is the record, soulrealm is the room, SoulIdentity is the name;
SoulNode is the house: the **single-binary distribution** that puts embedded
NATS, the identity plane, the archivist, the runtime, and an MCP front door
on a machine the user owns, behind `soulnode init && soulnode up`. The
founding bet: owning your realm should cost one binary and one command,
*without* simplifying the ecosystem's shape — the laptop runs operator-mode
NATS, auth-callout admission, and a sealed vault exactly as a hosted
deployment does (constitution II).

The idea entered feasible, not speculative. Same-day readings across the
sibling repos: all three components are Go modules on identical
infrastructure versions (Go 1.26, nats-server v2.14.3, nats.go v1.52)
[measured]; all three already embed the NATS server in their test rigs,
including an *operator-mode* embedded server in soulrealm's natstest
[measured]; and soulstream's `remote-mcp-node` research measured Bars 1–3
PASS on a consumer-position node that wires soulstream + SoulIdentity through
callout admission on the wire, naming tailscale for its HTTPS story
[measured, that topic's journal]. The obstacles are equally concrete:
SoulIdentity's entire serve path lives under `internal/` (unimportable by a
consumer module) [measured]; soulrealm's main depends on soulstream via a
`replace` directive (untaggable for a consumer) [measured]; and the
first-boot ceremony — operator, accounts, keys, sentinel, vault first key,
buckets — exists today only as manual steps and test-rig code
[mechanism-argument]. Hence the constitution's two load-bearing articles:
**composition, not invention** (no domain logic in SoulNode, components
consumed only through public tagged surfaces) and **first boot is the
product**.

The hq was seeded from soulrealm's proven structure (GENESIS / 01-RESEARCH /
02-DESIGN / 03-IMPLEMENTATION / 04-JOURNEY, the research→design→spec-kit→
journey pipeline, the anti-drift working agreement, the hq structural lint
riding `make test`, the vendored spec-kit flow with the constitution
symlink). No code exists yet — deliberately, per constitution IV: nothing was
decided about the composition itself. Whether in-process admission behaves
identically to the wire rig, what minimal embed surface each component needs,
and the exact ceremony inventory are the opening research topic
(`single-binary-composition`, roadmap Phase 0) — named, not yet opened, so
its bars can be pre-registered with the maintainer per how-we-work. The
tailscale question is phased deliberately: host `tailscale serve` first,
embedded tsnet only behind a Phase 3 gate that must measure the host path
insufficient.

Refuted/reversed: nothing yet — this is the first entry. The single-process
bet is held as the founding direction with its reversal condition below;
"SoulNode" itself is a working name, cheap to change while no code exists.

Opened: the composition gate (Phase 0). Phases 1–3 (the bundle, the front
door, the tailnet inside) are sketched in the roadmap but designed only once
their gates close.

Reversal condition: if the composition rig measures that in-process admission
cannot match wire behavior without forking a component — or an embed surface
upstream is refused — SoulNode becomes a supervisor of separate component
processes shipped as one installer (the distribution promise survives; the
one-process bet does not).

Trail: `hq/` (GENESIS, roadmap); sibling readings in soulstream
(`hq/01-RESEARCH/remote-mcp-node`, its journal 2026-07-30/31), soulidentity
(`cmd/soulidentity/main.go`, `internal/`), soulrealm (`go.mod`,
`internal/natstest/operator.go`). Commits <pending>.
