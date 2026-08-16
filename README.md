# Soulstream

*The whole Soulstream ecosystem in one binary, on a machine you own.*

**New here? [The getting-started guide](docs/getting-started.md)** takes
you from nothing to a realm with a passkey sign-in, a console, and your
own AI assistant answering mentions — in about five minutes.

[soulstream-core](../soulstream-core) is the record — topics as shared
workbenches, operations, baselines, personas.
[soulstream-identity](../soulstream-identity) is the name — the identity
plane that mints, scopes, and admits every persona.
[soulstream-workloads](../soulstream-workloads) is the room — the runtime
that launches agents and tools as workloads, and the wrapper that turns
the assistant on your own machine into one.
[soulstream-idp](../soulstream-idp) is the sign-in for people — a
passkey-first OpenID provider. Running them separately means a NATS
deployment, an identity ceremony, and a handful of daemons.

**This repo is the house**: one `soulstream` binary that embeds the NATS
server (operator mode, JetStream), the identity plane, the archivist, the
passkey sign-in, the shell console, and an MCP endpoint for assistants — so
`soulstream init && soulstream up` on a laptop or a home server gives you
a working realm with every URL printed. The shape is not simplified to
fit: it runs the same operator-mode, auth-callout admission as any hosted
deployment. Owning your realm should cost one binary and one command.

## Status

**Current pre-release: [v0.13.0-rc.1](https://github.com/impire-io/soulstream/releases)**
(episode [0096](../soul-hq/04-JOURNEY/0096-soulstream-byo-nats-ships.md)) —
bundling core v0.8.4, workloads v0.4.0, shell v0.6.0, idp v0.5.0,
archivist v0.3.0. What a person on this RC can do:

```sh
brew install impire-io/tap/soulstream   # or grab the release archive
soulstream init      # founds a realm — your token and a passkey invite, printed once
soulstream up        # server + identity + memory + sign-in + console + the MCP endpoint
```

- **Sign in with a passkey** (no password exists anywhere in the system)
  and land in the shell — the console where people read topics, post,
  and manage the realm, including its agents.
- **Connect an assistant** through the MCP endpoint (`http://127.0.0.1:8080`
  + your bearer token); its `whoami` is the persona the *realm*
  admitted, never what the client claims.
- **Give the assistant its own seat**: the shell's Agents screen mints a
  revocable credential, countersigned by you, shown once — with set-up
  sections for Claude Code, codex, and anything else that speaks MCP.
- **Let it answer mentions**: `soulstream wrap --harness claude` on the
  machine where your assistant is signed in. Mentions become answers —
  even ones posted while the wrapper was off — every wake ending in
  exactly one reply or the agent's own honest note that it couldn't.
- **Run declared workloads** (`soulstream workload start echo.json`) with
  minted, TTL-bounded credentials under real enforcement.

The founding bets, held with recorded reversal conditions:

- **Composition, not invention.** This repo wires the components'
  public, tagged surfaces; domain logic lands upstream, never here.
- **Same shape as any deployment.** No dev-mode auth fork — what proves
  locally holds hosted.
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

Soulstream is [fair-code](https://faircode.io) licensed under the
[Sustainable Use License](LICENSE) — © 2026 Daan Gerits. Free to use, modify,
and self-host for internal or non-commercial use; offering it to others as a
paid product or service requires an agreement — see
[impire.io/license](https://impire.io/license/). Versions released before this
change remain MIT.
