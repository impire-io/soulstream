# Episode 0001 — Genesis to v0.3: the protocol, and the library that proves it (2026-07-11 → 2026-07-24)

A founding retrospective, composed from [`../00-GENESIS/rationale.md`](../00-GENESIS/rationale.md),
the decision log in [`../../README.md`](../../README.md), the CLAUDE.md "Done"
list, and git history. It records where Soulstream came from and what it shipped
before this journal existed — so later episodes have a floor to stand on. Load-
bearing claims carry their evidence class; the retrospective is honest about one
gap in the record (the missing `012` spec folder).

## What happened

**The founding move was a subtraction (2026-07-11, pre-repo).** Earlier drafts
had grown into "a platform" — vision, substrate, personas, topics, sealed
topics, memory, search, a whitepaper — with no small answer to "what is
Soulstream?". The restructure cut it back to one sentence, *a stream on which
humans and AI collaborate through operations applied to topics*, and a
deliberately short "what is needed for a working soulstream" list: a NATS server
with JetStream, a `SOULSTREAM` stream, a credential per persona, the protocol on
the stream, and baselines. Everything else moved to `extensions/`
[mechanism-argument]. The load-bearing consequence: the steward — described as an
ordinary persona but load-bearing in discovery and lifecycle — was removed. *A
component you can't turn off without degrading core flows is plumbing, whatever
the docs call it* [judgment]. Coordination was rebuilt with no coordinator **and**
no consensus: deterministic rules + idempotent ops + optimistic concurrency
(`Nats-Expected-Last-Subject-Sequence`, first writer wins) [mechanism-argument].

**The other founding decisions**, all in the decision log with their reasons:
identity is a NATS credential + a name, with the registry as an extension and no
`on_behalf_of` (attribution laundering refused by design); the wire is
`SOULSTREAM.TOPICS.INFO/OPS.<path>` / `SOULSTREAM.PERSONA.NOTIFY.<id>` /
`SOULSTREAM.SVC.*` with `Soulstream-*` headers (fixed tokens uppercase,
identifiers lowercase — normative, because subjects are case-sensitive); the
stream has no `MaxAge` (the stream carries operations, not state; never expire
pointers independently of the objects they reference); blobs live in the
JetStream object store; the record lives in headers with a canonical JCS form for
signing; a topic is a **shared workbench** (baseline + ops), not a chat room.
Invented vocabulary is a budget — *persona*, *realm*, *baseline* earned their
place; "head" became **client** and "rung" became **stage** the same day
[judgment].

**Then the library was built to prove the protocol.** Git begins 2026-07-12
(`Initial commit from Specify template`); 116 commits later the reference library
(Go, `github.com/impire-io/soulstream`) had shipped, feature by feature through
the spec-kit flow: foundation + op-log engine + participation (`001`–`003`), the
CLI and MCP clients (`004`, `005`), signing (`006`), rollup/manifest/`archived`
(`007`), scatter-gather discovery (`008`), the curator (`009`), work stages 1–2
(`010`, `011`), distribution (`012`), config-file identity (`013`), and persona
accountability (`014`). Releases, from git tags [measured]: **`v0.1.0`
(2026-07-21)** through `012-distribution`, **`v0.2.0` (2026-07-21)** for
`013-config`, **`v0.3.0` (2026-07-23)** for `014-persona-accountability`, and
**`v0.3.1` (2026-07-24)** a registry legacy-profile republish fix. The suite is
green with no external NATS — `record`/`identity` import no NATS at all, and the
NATS-touching packages test against an in-process JetStream server [measured].

## What was refuted or reversed

**Persona `kind` was reversed out of the protocol entirely (`014`, v0.3.0).** It
began structural, was demoted to presentation metadata ("behaviour may never
branch on it"), and was finally removed: a persona is a voice with a key, and
accountability is a countersigned `operated_by` operator attestation, never a
human/agent label. The reasoning is a mechanism argument that closed the debate:
*the protocol cannot verify what controls a key, so it refuses to record the
claim* [mechanism-argument] — the peer principle made testable, with no field
left to branch on. Earlier the same discipline retired a separate
`soulstream.life.<topic>` lifecycle subject: folding `life.transition` onto the
topic's own op-log restored one invariant shape (baseline first, ops after,
everything compacts into baselines), and the removed subject's only real consumer
had been the now-deleted steward.

## What it taught / what it opened

The through-line is constitution II (Smallest Viable Implementation) doing real
work: growth is expressed as new vocabulary over the existing log (`edit`,
`work.*`, `revise`, attestations) rather than new machinery, and the "what is
needed" list never grew. What is *not* yet built is the forward plan in
[`../03-IMPLEMENTATION/ROADMAP.md`](../03-IMPLEMENTATION/ROADMAP.md): eg-walker
live co-editing, a memory convention, sealed topics, a browser client.

**One honest gap in the record:** `012-distribution` shipped in `v0.1.0` (the tag
is `Merge 012-distribution`) but has **no `specs/012-*` folder** in the tree —
its spec-kit artifacts were never committed. Recorded here as a fact rather than
reconstructed after the event.

Reversal condition: the central architectural bet is leaderless coordination
(no coordinator, no consensus). It reopens if a realm under real concurrent load
produces divergent topic state — a measured reading of two personas'
materialisations disagreeing after a rollup, or an un-rolled-up tail that
optimistic concurrency fails to keep race-safe. Absent such a reading, elections
and consensus stay banned in the protocol.

Trail: [`../00-GENESIS/rationale.md`](../00-GENESIS/rationale.md); the decision
log in [`../../README.md`](../../README.md); `specs/001-*` … `specs/014-*`
(bodies frozen, `Status` records the shipping version); git tags `v0.1.0`
`v0.2.0` `v0.3.0` `v0.3.1`; commits `42f310a` (genesis) … `034f7ca` (v0.3.1).
