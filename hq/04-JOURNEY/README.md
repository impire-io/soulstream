# 04-JOURNEY — the narrative record

What was built, what was measured, what was believed and then refuted, and
what each episode taught. Specs say what the system *is*; these episodes say
how we *got here* — including the dead ends, because the refuted hypotheses
are as load-bearing as the shipped code.

> **Keeping this log alive:** whenever a feature lands, a research
> investigation concludes, or a load-bearing decision is made, add a numbered
> episode with `/journey-log` (research topics get theirs via
> `/research-graduate`). Follow [`TEMPLATE.md`](TEMPLATE.md) — including its
> required Reversal-condition line and evidence-class tags. Honesty rules
> apply here as everywhere: record what actually happened, including failures,
> reversals, and findings that contradicted expectations. This duty is
> anchored in `../00-GENESIS/how-we-work.md`.

## Where things stand (2026-07-31)

**The project was founded** ([episode 0001](0001-genesis.md)): SoulNode is
the single-binary distribution of the Soulstream ecosystem — embedded NATS,
SoulIdentity, the archivist, soulrealm, and an MCP front door in one process
on a machine the user owns, with the first boot (`soulnode init`) treated as
the product. The hq is bootstrapped from soulrealm's proven structure. The
constitution's load-bearing articles are **composition, not invention**
(SoulNode wires public tagged surfaces; domain logic lands upstream) and
**same shape as any deployment** (operator-mode NATS + auth-callout locally,
no dev fork). Feasibility entered measured (identical infra versions across
components, operator-mode embedded server already proven in sibling test
rigs); the obstacles too (SoulIdentity's serve path is `internal/`,
soulrealm's `replace` directive). Nothing about the composition itself is
decided yet. **Next:** open the `single-binary-composition` research topic
(roadmap Phase 0) with `/research-start`, bars pre-registered with the
maintainer.

## Episode index

| # | Episode |
|---|---|
| 0001 | [Genesis: SoulNode gets an HQ](0001-genesis.md) |
