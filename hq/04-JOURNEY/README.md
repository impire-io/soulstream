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

## Where things stand (2026-07-29)

The reference library has shipped the MVP and most of day-2 — **`v0.5.0`**
(2026-07-28) is current: foundation + op-log engine, CLI + MCP clients,
signing, rollup, scatter-gather discovery, the curator, work stages 1–2,
distribution, config-file identity, persona accountability
([episode 0001](0001-genesis-and-the-reference-library.md), the founding
retrospective), the **memory convention**
([episode 0003](0003-memory-convention-and-exhibits.md)): collective search as
graded testimony, portable self-authenticating exhibits, a public witness
surface — with the first archivist a separate repository,
[impire-io/soulstream-archivist](https://github.com/impire-io/soulstream-archivist),
**verified live against the NGS realm** (2026-07-26) — and **provisioning
byte limits** ([episode 0004](0004-provisioning-byte-limits.md)): limit-
enforced accounts provision out of the box. Merged after `v0.5.0`,
unreleased: the **signer seam** ([episode 0006](0006-the-signer-seam.md)) —
signing delegated through `identity.Signer` to an external custodian
(SoulIdentity's M2 wiring point), local keys the first implementation, a
failing signer failing the publish loudly. The **two-week dogfood run
started 2026-07-27** (protocol:
[`../03-IMPLEMENTATION/DOGFOOD.md`](../03-IMPLEMENTATION/DOGFOOD.md))
— daan, smith, and scribe on the NGS realm, the archivist keeping; its chafe
log feeds the eg-walker and sealed-topics gates. The central architectural
bet — leaderless coordination, no coordinator and no consensus — stands,
un-refuted.

**The project's working structure lives in `hq/`**
([episode 0002](0002-adopting-the-hq-way.md)): GENESIS holds the vision, the
constitution (v1.1.0, wired into every spec-kit plan via the
`.specify/memory/constitution.md` symlink, now carrying the anti-drift Working
Agreement), and how-we-work; research runs a `/research-start` →
`/research-graduate` lifecycle; this journal is numbered episodes with the
structure enforced by `internal/hqlint` on the standard gate. The first
research topic has concluded: **sealed-topics**
([episode 0005](0005-sealed-topics.md)) ran its four pre-registered bars in
a day and graduated to design — the sealed design survives the shipped
substrate, with amendments now folded into
[`../02-DESIGN/extensions/sealed-topics.md`](../02-DESIGN/extensions/sealed-topics.md)
(the `{"ct"}` payload wrapper, signature-covered epoch/nonce, signing-chain
endorsement of sealing keys, key-carrying sealed baselines); the build's
priority stays gated on the dogfood chafe log (to 2026-08-10). What is
*not* yet built is the forward plan in
[`../03-IMPLEMENTATION/ROADMAP.md`](../03-IMPLEMENTATION/ROADMAP.md):
eg-walker live co-editing, sealed topics, and a browser/WebSocket client.

## Episode index

| # | Episode |
|---|---|
| 0001 | [Genesis to v0.3: the protocol, and the library that proves it](0001-genesis-and-the-reference-library.md) |
| 0002 | [Adopting the hq way: the process gets a constitution](0002-adopting-the-hq-way.md) |
| 0003 | [The memory convention: the realm learns to be asked](0003-memory-convention-and-exhibits.md) |
| 0004 | [Provisioning byte limits: the strict landlord gets a one-command realm](0004-provisioning-byte-limits.md) |
| 0005 | [Sealed topics survive the substrate: four bars, one encoding amendment](0005-sealed-topics.md) |
| 0006 | [The signer seam: signing learns to be delegated](0006-the-signer-seam.md) |
