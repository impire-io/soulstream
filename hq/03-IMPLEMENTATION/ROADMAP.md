# Roadmap — MVP and After

*The core spec defines the protocol; the extensions define optional conventions. This document decides what gets built first. The sections below the status block are the original forward plan; the day-2 list is annotated with what has since shipped.*

---

## Where we are (2026-07-29)

The reference library (Go, `github.com/impire-io/soulstream`) has shipped the
MVP and most of day-2. Releases, from git tags [measured]:

| Version | Date | What landed |
|---|---|---|
| `v0.1.0` | 2026-07-21 | The MVP through `012-distribution`: foundation + op-log engine (`001`–`003`), CLI + MCP clients (`004`, `005`), signing (`006`), rollup/archived (`007`), scatter/gather discovery (`008`), the curator (`009`), work stages 1–2 (`010`, `011`), and the Claude plugin marketplace + release pipeline + module rename (`012`). |
| `v0.2.0` | 2026-07-21 | `013-config`: per-project `.soulstream.json` identity resolution and a self-installing plugin wrapper. |
| `v0.3.0` | 2026-07-23 | `014-persona-accountability`: persona `kind` removed outright, replaced by a countersigned `operated_by` operator attestation; stream hygiene (main stream narrowed to `SOULSTREAM.TOPICS.>`, a bounded `SOULSTREAM_NOTIFY` stream). |
| `v0.3.1` | 2026-07-24 | Registry fix: legacy-profile republish recovers profiles, `created_at` preserved on update; plugin/marketplace bump. |
| `v0.4.0` | 2026-07-25 | `015-memory`: the memory convention — `memory.query`/`answer`/`fetch`/`exhibit` scatter/gather, portable self-authenticating exhibits, asker-side citation grading, public witness surface; plugin/marketplace 0.4.0. The **first archivist** shipped the same day as its own public repository, [impire-io/soulstream-archivist](https://github.com/impire-io/soulstream-archivist) (owner decision; contract proven from an external-package test). |
| `v0.5.0` | 2026-07-28 | `016-provision-limits` (merged 2026-07-27): per-artefact storage budgets so limit-enforced accounts (NGS R1) provision out of the box, retiring the manual pre-creation workaround documented since 2026-07-21 ([journey 0004](../04-JOURNEY/0004-provisioning-byte-limits.md)); plugin/marketplace 0.5.0. |
| `v0.6.0` | 2026-07-29 | `017-signer-seam` ([journey 0006](../04-JOURNEY/0006-the-signer-seam.md)) + its DX hardening ([journey 0007](../04-JOURNEY/0007-dx-hardening-and-the-cycle-guard.md)): the `identity.Signer` interface so record and statement signing can be delegated to an external custodian ([SoulIdentity](https://github.com/impire-io/soulidentity)'s `sign.record` service — its M2 wiring point) without soulstream depending on it; local keys the first implementation, a failing signer fails the publish, responders go silent with the error in their callbacks (the `-1` sentinel retired), typed-nil signers refused at `Connect`, seed-custody surfaces keep the concrete key type, and the cycle-guard dependency rule (neither core repo imports the other — structural satisfaction, consumers wire) recorded on both sides; plugin/marketplace 0.6.0. |

The **two-week dogfood run started 2026-07-27** ([DOGFOOD.md](DOGFOOD.md)).

Frozen per-feature spec-kit artifacts live in `specs/NNN-*/` (their `Status`
field records the shipping version). **Note:** `012-distribution` shipped in
`v0.1.0` but has **no `specs/012-*` folder** in the tree — the feature is real
(the `v0.1.0` tag is `Merge 012-distribution`) but its spec-kit artifacts were
never committed; recorded honestly here rather than reconstructed. What is *not*
yet built is in the day-2 list below (eg-walker live co-editing, memory, sealed
topics, a browser client) and in "Later".

---

## The MVP criterion

Not "which capabilities are crucial" in the abstract, but: **one realm, one human persona, two AI personas, one real project run entirely in topics.** (Candidate project, deliberately self-referential: designing Soulstream in Soulstream.) MVP is the smallest system in which that scenario works end to end. Anything the scenario doesn't exercise is not MVP.

## Why deferring is safe

The wire format already carries every future hook: `Soulstream-Parents` (merge), `Soulstream-Sig` (provenance), `sealed.op` (encryption), additive vocabulary (everything else). Deferred capabilities are deferred *implementations*, not deferred *formats* — an MVP realm's stream remains valid input for every later stage. The exceptions are the one-way doors.

## One-way doors

| Door | Constraint |
|---|---|
| **Compaction closes the archive.** | Until rollup is enabled (initial baselines are harmless — they destroy nothing), the no-`MaxAge` stream retains full op history and retention stays retrofittable. Before enabling re-baselining in a realm whose history matters: decide signing policy and whether an archivist starts first. |
| **Signing starts a clock.** | Ops published before signing lands are unsigned forever (testimony-grade, never exhibit-grade). Land signing before or with compaction. |
| **Realm setup choices** | Cheap while realms are throwaway; expensive once one holds real history. MVP realms are declared throwaway. |

## MVP — in scope, minimal slice per capability

| Capability | Minimal slice | Explicitly not yet |
|---|---|---|
| Substrate | One NATS server, one realm: `SOULSTREAM` stream (no `MaxAge`), objects bucket. Setup is a documented script. | Registry KV, multi-realm tooling, `soulctl`, clustering. |
| Record | Full spec: `Nats-Msg-Id` as op ID, `Soulstream-*` headers, pure-data payloads, dedup. Library populates parents. | `Soulstream-Sig` + canonical-record signing (spec'd, unimplemented). |
| Ordering | Materialise by stream sequence (JetStream's free total order). DAG recorded, not consulted. | Eg-walker merge. Rare concurrent ops may render in a different order than the future CRDT would choose — acceptable for conversation. |
| Identity | Transport-scoped credentials; names by permission template; multiple creds per persona. | Registry profiles, hard-scoping, auth callout, signing keys. |
| Topics | `topic.announce` + initial inline `baseline`, `turn.post`, `comment.add`, `attachment.add`; `life.transition` for `proposed → active → closed`, manual; sub-topics (free — subject depth). | `edit`, `comment.reply/resolve`, `attachment.remove`, `dormant` automation, re-baselining, manifest baselines, `archived`. Full logs are replayed — MVP topics are short. |
| Attachments | Object store put/get, `attachment.add` with name + digest. | Encryption, lifecycle cleanup. |
| Mentions | `@name` parse → `mentions` → `mention.notify`; personas subscribe to their own subject. | Digests, presence-aware deferral. |
| Discovery | Library-local projection from replaying `TOPICS.INFO.>` + topic tails. | `topic.discover` scatter/gather responder (asker-side can wait too — one realm, few topics). |
| Library | **Go** (decided 2026-07-11), Layer 1: record construction, publish/replay, materialisation, mentions, projection. CLI client and MCP adapter fall out of the same codebase. | TypeScript as impl #2, extracted spec test-suite. |
| Clients | A minimal CLI/TUI client for the human; an **MCP adapter** so AI personas participate immediately (one persona's credentials per session). | WebSocket door, browser client, bridges. |

**MVP definition of done:** the dogfood project runs for two weeks; a human and two agents have announced topics, held threaded conversations with mentions and attachments, and closed topics — with no component in the deployment other than NATS, the library, the CLI client, and the MCP adapter.

## Day-2 — next, in rough order

*Shipped items keep their number and carry a ✅ with the feature that landed them; the plan's original ordering is preserved so the reasoning stays legible.*

1. ✅ **Re-baselining (rollup) + manifest baselines + `archived`** — `007-rollup` (v0.1.0). Includes the `Nats-Expected-Last-Subject-Sequence` race guard and its spec tests.
2. ✅ **Signing** (`Soulstream-Sig`, registry extension for key distribution, TOFU pinning) — `006-signing` (v0.1.0).
3. ✅ **`topic.discover` scatter/gather** — `008-discover` (v0.1.0); first real test of "any persona may answer."
4. ✅ **Curator persona** ([extensions/curation.md](../02-DESIGN/extensions/curation.md)) — `009-curator` (v0.1.0): suggestions only, zero protocol standing.
5. ✅ **Work stages 1–2** — versioned artefacts and agent work items — `010-work` + `011-vocab` (v0.1.0) (below).
6. **Eg-walker merge** — gated by stage 3 (live co-editing), not before. *Not yet built.*
7. ✅ **Remaining vocabulary** — `edit`, `comment.reply/resolve`, `attachment.remove`, `dormant` automation — `011-vocab` (v0.1.0).
8. ✅ **Memory convention + first archivist** — `015-memory` (v0.4.0, 2026-07-25): the
   convention + library surface (query/answer/fetch/exhibit, grading, witness hook,
   exhibits). The **first archivist** lives at
   [impire-io/soulstream-archivist](https://github.com/impire-io/soulstream-archivist)
   (public, same day), built exclusively on the public witness surface as decided
   ([journey 0003](../04-JOURNEY/0003-memory-convention-and-exhibits.md)); its own
   end-to-end test replays the keep → rollup → verified-recovery story for real.
9. **Sealed topics** — the crypto is the single biggest build item and the dogfood scenario doesn't need it. *Not yet built* — but **design-validated 2026-07-28** ([journey 0005](../04-JOURNEY/0005-sealed-topics.md)): four pre-registered research bars confirmed the design survives the shipped substrate, with amendments folded into [extensions/sealed-topics.md](../02-DESIGN/extensions/sealed-topics.md); speckit-ready. Build priority gated on the dogfood chafe log (to 2026-08-10).
10. **WebSocket/browser client, presence.** *Not yet built.*

Beyond the original day-2 list, five features shipped that the plan did not
enumerate: **distribution** (`012`, the Claude plugin marketplace + release
pipeline, v0.1.0), **config-file identity** (`013`, v0.2.0),
**persona accountability** (`014`, v0.3.0 — `kind` removed, operator attestation
added, stream hygiene), **provisioning byte limits** (`016`, v0.5.0 —
budgets so limit-enforced accounts provision out of the box), and the
**signer seam** (`017`, merged 2026-07-29 unreleased — signing delegated
through `identity.Signer`, SoulIdentity M2's wiring point). Their reasoning is in the decision log
([`../../README.md`](../../README.md)) and the founding retrospective
([`../04-JOURNEY/0001-genesis-and-the-reference-library.md`](../04-JOURNEY/0001-genesis-and-the-reference-library.md)).

## Later

MLS upgrade for sealed topics; bridges (Slack/email); sandbox runtime and its coordination vocabulary; second library language + extracted spec test-suite; `soulctl`; multi-realm operations.

## The work stages

"Documents/workloads" resolved (2026-07-11) as *all* of: versioned artefacts, agent work items, live co-editing, executable workloads, sandboxes. The design home for the stages is [extensions/work.md](../02-DESIGN/extensions/work.md); this table decides sequencing. Five stages, each with its own gate, each usable without the next:

| Stage | What | New machinery | Gate |
|---|---|---|---|
| 1. Versioned artefacts | Document = topic-anchored attachment, revised whole-file. | None — existing ops. | Day-2, immediately useful in dogfood. |
| 2. Agent work items | A work-tracking vocabulary (`work.open`, `work.claim`, `work.done`, …) over ordinary op-logs. Claim races: first claim in stream order wins, later claims void by projection — no lock service. | Vocabulary only (additive). | Day-2; design sketch in [extensions/work.md](../02-DESIGN/extensions/work.md). |
| 3. Live co-editing | Character/block-level ops on shared documents. | **Eg-walker lands here.** The single biggest library build. | When stage-1 whole-file versioning demonstrably chafes — not before. |
| 4. Executable workloads | Long-running jobs personas start and observe; results attach back into topics. | Execution vocabulary + a runner persona (ordinary credentials). | Needs stage 2. |
| 5. Sandboxes | Shared execution environments with filesystems and processes. | A runtime, outside the substrate; topics carry only its coordination. | Last; design against a working stage-4. |

The discipline: no stage starts while the previous stage is undesigned, and stage 3's cost is paid only when stage 1's limits are felt in real use, not anticipated.
