# SoulNode

*The whole Soulstream ecosystem in one binary, on a machine you own.*

**New here? [The getting-started guide](docs/getting-started.md)** takes
you from nothing to a realm your Claude talks to, in about five minutes.

[Soulstream](../soulstream) is the record — topics as shared workbenches,
operations, baselines, personas. [Soulrealm](../soulstream-workloads) is the room — the
runtime that launches a realm's agents and tools as workloads.
[SoulIdentity](../soulstream-identity) is the name — the identity plane that mints,
scopes, and admits every persona. Running them today means a NATS deployment,
an identity ceremony, and three daemons.

SoulNode is the house: one `soulstream` binary that embeds the NATS server
(operator mode, JetStream), the identity plane, the archivist, the realm
runtime, and an MCP front door — so `soulstream init && soulstream up` on a
laptop or a home server gives you a working realm, and a Claude client
connects to the URL it prints. The shape is not simplified to fit: a SoulNode
runs the same operator-mode, auth-callout admission as any hosted deployment.
Owning your realm should cost one binary and one command.

## Status

**Phase 1 is complete — the realm founds, runs, remembers, and executes**
(journeys [0003](../soul-hq/04-JOURNEY/0042-soulstream-first-boot-is-real.md),
[0004](../soul-hq/04-JOURNEY/0044-soulstream-the-realm-remembers.md),
[0005](../soul-hq/04-JOURNEY/0045-soulstream-an-agent-runs.md), 2026-08-02):

```sh
soulstream init      # founds a realm in ~0.15s — your token, printed once
soulstream up        # server + identity + memory + the MCP door, loopback only
soulstream workload start echo.json   # a declared agent runs, attributed
```

Then point an MCP client (Claude Code, a desktop client) at
`http://127.0.0.1:8080` with the printed token as its bearer — the
session's `whoami` is the persona the *realm* admitted, never what the
client claims ([journey 0006](../soul-hq/04-JOURNEY/0049-soulstream-the-door-opens.md)).
Admission is the full ecosystem shape (sentinel + token through auth
callout), memory is the archivist keeping every op and answering with
citations, and workloads run with minted TTL-bounded credentials under
real enforcement. Everything is proven end-to-end in `make test`. Public
(OAuth) mode waits on soulstream-idp upstream; HTTPS today is `tailscale
serve` in front of the loopback door.

The founding bets, held with recorded reversal conditions:

- **Composition, not invention.** SoulNode wires the components' public,
  tagged surfaces; domain logic lands upstream, never here.
- **Same shape as any deployment.** No dev-mode auth fork — what SoulNode
  proves locally holds hosted.
- **One process, planes by configuration.** Every enabled plane in the
  binary, each on an ordinary loopback NATS connection — repointable or
  removable by configuration alone; workloads always outside, through
  soulstream-workloads's isolation backends.
- **First boot is the product.** The whole key-and-account ceremony is
  `soulstream init`'s job; a manual nkey step is a bug.

## Layout

| Area | What it holds |
|---|---|
| [`../soul-hq/00-GENESIS/`](../soul-hq/00-GENESIS/README.md) | Vision, constitution, working rules |
| [`../soul-hq/01-RESEARCH/`](../soul-hq/01-RESEARCH/README.md) | Active investigations (one folder each) |
| [`../soul-hq/02-DESIGN/soulstream/`](../soul-hq/02-DESIGN/soulstream/README.md) | Architecture & feature designs |
| [`../soul-hq/03-IMPLEMENTATION/`](../soul-hq/03-IMPLEMENTATION/README.md) | The roadmap: gates, not calendars |
| [`../soul-hq/04-JOURNEY/`](../soul-hq/04-JOURNEY/README.md) | Numbered episodes: the honest log |

Code lives at the repo root as the Go module `github.com/impire-io/soulstream`;
each feature's spec-kit artifacts freeze under `specs/NNN-*/` as it lands.

## License

SoulNode is [fair-code](https://faircode.io) licensed under the
[Sustainable Use License](LICENSE) — © 2026 Daan Gerits. Free to use, modify,
and self-host for internal or non-commercial use; offering it to others as a
paid product or service requires an agreement — see
[impire.io/license](https://impire.io/license/). Versions released before this
change remain MIT.
