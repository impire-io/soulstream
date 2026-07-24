# 02-DESIGN — the normative design

This is what Soulstream *is*, specified functionally: **what must exist** and
**how each part behaves**. An implementer should be able to build a working
system from these documents without needing undocumented decisions. The reasons
behind the choices are not here — they live in
[`../00-GENESIS/rationale.md`](../00-GENESIS/rationale.md).

**The spec-kit rule:** every document here is written explicit enough to be the
argument to `/speckit-specify` — the capability, its seams, its configuration
surface, and its acceptance criteria, with no guessing left to the spec writer.
Graduating research enters through `/research-graduate`; behavioral changes made
during implementation propagate back here (see
[`../00-GENESIS/how-we-work.md`](../00-GENESIS/how-we-work.md)).

## core/ — normative; this *is* Soulstream

A realm running only this is a working soulstream.

| Doc | Covers |
|---|---|
| [`core/01-protocol.md`](core/01-protocol.md) | Realms, the stream, the subject taxonomy, the operation record |
| [`core/02-identity.md`](core/02-identity.md) | Credentials, personas, attribution, delegation, notifications |
| [`core/03-topics.md`](core/03-topics.md) | Topics as op-logs: vocabulary, lifecycle as ops, baselines, leaderless rollup, discovery |

## extensions/ — optional conventions

A realm running none of these is still a working soulstream.

| Doc | Covers |
|---|---|
| [`extensions/registry.md`](extensions/registry.md) | Rich persona profiles, operator attestation, key distribution |
| [`extensions/library-and-adapters.md`](extensions/library-and-adapters.md) | The reference library, MCP adapter, WebSocket door, bridges, presence |
| [`extensions/curation.md`](extensions/curation.md) | Curator personas (what the old "steward" became) |
| [`extensions/work.md`](extensions/work.md) | The work stages: versioned artefacts, work items, execution, sandboxes |
| [`extensions/sealed-topics.md`](extensions/sealed-topics.md) | E2E-encrypted topics |
| [`extensions/memory.md`](extensions/memory.md) | Persona memory and collective search |

The build order for all of the above is in
[`../03-IMPLEMENTATION/ROADMAP.md`](../03-IMPLEMENTATION/ROADMAP.md); the frozen
per-feature spec-kit artifacts are in `specs/NNN-*/`.
