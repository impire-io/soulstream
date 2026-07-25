# 04-JOURNEY — the narrative record

What was built, what was measured, what was believed and then refuted, and what
each episode taught. The specs in `specs/` and the design docs in `02-DESIGN/`
say what the system *is*; these episodes say how we *got here* — including the
reversals, because a refuted assumption is as load-bearing as the shipped code.

> **Keeping this log alive:** whenever a feature lands, a research investigation
> concludes, or a load-bearing decision is made, add a numbered episode with
> `/journey-log` (research topics get theirs via `/research-graduate`). Follow
> [`TEMPLATE.md`](TEMPLATE.md) — including its required Reversal-condition line
> and evidence-class tags. Honesty rules apply here as everywhere: record what
> actually happened, including failures, reversals, and findings that
> contradicted expectations. This duty is anchored in
> [`../00-GENESIS/how-we-work.md`](../00-GENESIS/how-we-work.md); the numbering
> and index are enforced by `internal/hqlint`.

## Where things stand (2026-07-25)

The reference library has shipped the MVP and most of day-2 —
**`v0.3.1`** is current ([episode 0001](0001-genesis-and-the-reference-library.md), the founding
retrospective): foundation + op-log engine, CLI + MCP clients, signing, rollup,
scatter-gather discovery, the curator, work stages 1–2, distribution, config-file
identity, and persona accountability (`kind` removed in favour of a countersigned
operator attestation). The **memory convention** has now landed on `main`
([episode 0003](0003-memory-convention-and-exhibits.md)) and shipped as
**`v0.4.0`**: collective search as graded testimony, portable self-authenticating
exhibits, and a public witness surface — with the first archivist deliberately a
separate repository, [impire-io/soulstream-archivist](https://github.com/impire-io/soulstream-archivist)
(live the same day), its contract proven from outside. The central architectural
bet — leaderless coordination, no coordinator and no consensus — stands,
un-refuted.

**The project's working structure lives in `hq/`**
([episode 0002](0002-adopting-the-hq-way.md)): GENESIS holds the vision, the
constitution (v1.1.0, wired into every spec-kit plan via the
`.specify/memory/constitution.md` symlink, now carrying the anti-drift Working
Agreement), and how-we-work; research runs a `/research-start` →
`/research-graduate` lifecycle; this journal is numbered episodes with the
structure enforced by `internal/hqlint` on the standard gate. What is *not* yet
built is the forward plan in
[`../03-IMPLEMENTATION/ROADMAP.md`](../03-IMPLEMENTATION/ROADMAP.md): the
archivist (separate repo), eg-walker live co-editing, sealed topics, and a
browser/WebSocket client. There are no active research topics yet.

## Episode index

| # | Episode |
|---|---|
| 0001 | [Genesis to v0.3: the protocol, and the library that proves it](0001-genesis-and-the-reference-library.md) |
| 0002 | [Adopting the hq way: the process gets a constitution](0002-adopting-the-hq-way.md) |
| 0003 | [The memory convention: the realm learns to be asked](0003-memory-convention-and-exhibits.md) |
