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

**Founded 2026-07-31** ([journey 0001](hq/04-JOURNEY/0001-genesis.md)). The
hq is bootstrapped from the sibling projects' proven structure; no code
exists yet — deliberately. The opening research topic
(`single-binary-composition`, [roadmap](hq/03-IMPLEMENTATION/roadmap.md)
Phase 0) gates the first build: in-process admission parity with the wire
rig, the minimal public embed surface each component needs, and the
first-boot ceremony enumerated end-to-end.

The founding bets, held with recorded reversal conditions:

- **Composition, not invention.** SoulNode wires the components' public,
  tagged surfaces; domain logic lands upstream, never here.
- **Same shape as any deployment.** No dev-mode auth fork — what SoulNode
  proves locally holds hosted.
- **One process, workloads apart.** Everything in the binary except
  workloads, which run outside through soulrealm's isolation backends.
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
