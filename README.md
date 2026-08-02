# SoulNode

*The whole Soulstream ecosystem in one binary, on a machine you own.*

[Soulstream](../soulstream) is the record — topics as shared workbenches,
operations, baselines, personas. [Soulrealm](../soulrealm) is the room — the
runtime that launches a realm's agents and tools as workloads.
[SoulIdentity](../soulidentity) is the name — the identity plane that mints,
scopes, and admits every persona. Running them today means a NATS deployment,
an identity ceremony, and three daemons.

SoulNode is the house: one `soulnode` binary that embeds the NATS server
(operator mode, JetStream), the identity plane, the archivist, the realm
runtime, and an MCP front door — so `soulnode init && soulnode up` on a
laptop or a home server gives you a working realm, and a Claude client
connects to the URL it prints. The shape is not simplified to fit: a SoulNode
runs the same operator-mode, auth-callout admission as any hosted deployment.
Owning your realm should cost one binary and one command.

## Status

**Phase 1 is complete — the realm founds, runs, remembers, and executes**
(journeys [0003](hq/04-JOURNEY/0003-first-boot-is-real.md),
[0004](hq/04-JOURNEY/0004-the-realm-remembers.md),
[0005](hq/04-JOURNEY/0005-an-agent-runs.md), 2026-08-02):

```sh
soulnode init      # founds a realm in ~0.15s — your token, printed once
soulnode up        # operator-mode server + identity + memory, loopback only
soulnode workload start echo.json   # a declared agent runs, attributed
```

Admission is the full ecosystem shape (sentinel + token through auth
callout, server-asserted personas), memory is the archivist keeping every
op and answering with citations, and workloads run with minted
TTL-bounded credentials under real enforcement. Everything is proven
end-to-end in `make test`. Phase 2 (the MCP front door) is gated on
soulstream's `018-remote-mcp-node` cycle, in flight upstream.

The founding bets, held with recorded reversal conditions:

- **Composition, not invention.** SoulNode wires the components' public,
  tagged surfaces; domain logic lands upstream, never here.
- **Same shape as any deployment.** No dev-mode auth fork — what SoulNode
  proves locally holds hosted.
- **One process, planes by configuration.** Every enabled plane in the
  binary, each on an ordinary loopback NATS connection — repointable or
  removable by configuration alone; workloads always outside, through
  soulrealm's isolation backends.
- **First boot is the product.** The whole key-and-account ceremony is
  `soulnode init`'s job; a manual nkey step is a bug.

## Layout

| Area | What it holds |
|---|---|
| [`hq/00-GENESIS/`](hq/00-GENESIS/README.md) | Vision, constitution, working rules |
| [`hq/01-RESEARCH/`](hq/01-RESEARCH/README.md) | Active investigations (one folder each) |
| [`hq/02-DESIGN/`](hq/02-DESIGN/README.md) | Architecture & feature designs |
| [`hq/03-IMPLEMENTATION/`](hq/03-IMPLEMENTATION/README.md) | The roadmap: gates, not calendars |
| [`hq/04-JOURNEY/`](hq/04-JOURNEY/README.md) | Numbered episodes: the honest log |

Code lives at the repo root as the Go module `github.com/impire-io/soulnode`;
each feature's spec-kit artifacts freeze under `specs/NNN-*/` as it lands.

## License

SoulNode is released under the [MIT License](LICENSE) — © 2026 Daan Gerits.
