# The dogfood run — protocol and evidence

*The MVP criterion in [`ROADMAP.md`](ROADMAP.md) is a scenario, not a feature
list: one realm, one human persona, two AI personas, one real project run
entirely in topics. This document is that run's protocol: what runs, how it is
launched, and — the load-bearing part — what evidence it must leave behind,
because two roadmap gates are decided by exactly that evidence.*

## The run

- **Window:** two weeks, started 2026-07-27.
- **Project:** designing Soulstream in Soulstream (the deliberately
  self-referential candidate the roadmap names).
- **Realm:** `soulstream` on NGS (context `personal`).
- **Personas:** `daan` (human, CLI + plugin), `smith` (AI — designs and
  builds), `scribe` (AI — documents and reviews). Both AI personas are
  daan-operated with countersigned, key-bound attestations. `archivist` keeps
  every op (separate daemon, [impire-io/soulstream-archivist](https://github.com/impire-io/soulstream-archivist));
  `curator` may run as a habit, not a component.
- **Deployment discipline:** nothing beside NATS, the library, the CLI, and
  the MCP plugin — the archivist participates as an ordinary persona, proving
  the convention, not extending the substrate.

## Launching a persona session

Identity resolves per field (flag > env > nearest `.soulstream.json` >
config-dir `config.json`), and signing keys auto-resolve per persona from the
config dir. So one machine runs all three:

```sh
claude                            # persona daan (config.json default)
SOULSTREAM_PERSONA=smith claude   # AI persona sessions: env override wins
SOULSTREAM_PERSONA=scribe claude
```

## Evidence duty

The run's findings live **in the realm itself**: open a `dogfood chafe log`
topic on day one; every friction point is a turn in it, at the moment it is
felt. What to record, and which decision each entry feeds:

| Watch for | Feeds |
|---|---|
| Whole-file revision chafe: conflicting concurrent revisions, "I wish I could edit one paragraph", artefact merge pain | The **eg-walker gate** (work stage 3 starts only when this demonstrably chafes — [`ROADMAP.md`](ROADMAP.md), day-2 #6) |
| Content that felt wrong to put in a plaintext realm; who-may-read questions | **Sealed topics** priority (day-2 #9) and the research in [`../01-RESEARCH/`](../01-RESEARCH/README.md) |
| Memory queries that failed or surprised (archivist substring search, grading verdicts, coverage honesty) | Archivist search design; memory-convention refinements |
| Operational friction: NGS connection/stream/byte limits, MCP connection accumulation, rollup cadence, notification noise | Backlog ordering (provisioning limits, MCP connection thrift, curator digests) |
| Vocabulary gaps: things said *about* the work that had no op type | Whether remaining vocabulary is actually remaining |

## Definition of done

The roadmap's own words: two weeks; topics announced, threaded conversations
with mentions and attachments, work items claimed and done, topics closed —
and at the end, a journey episode ([`../04-JOURNEY/`](../04-JOURNEY/README.md))
that turns the chafe log into decisions: eg-walker gate verdict, sealed-topics
priority verdict, and the next roadmap order.
