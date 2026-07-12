# Rationale

*How we got here. Nothing in this document is normative; it exists so future changes argue against the real reasons, not guesses.*

---

## The gap Soulstream fills

Collaboration platforms — Notion, Google Workspace, Slack — were built for humans, then had AI bolted on: an assistant in a sidebar, a bot with a special API, a copilot outside the document model. The AI is always a second-class citizen with a different door into the building. Soulstream inverts this: one protocol, one identity model, one attribution model, for every persona. There is no bot API and no human API. There is one protocol.

## From platform back to protocol (2026-07-11)

Earlier drafts grew into "a platform": vision, substrate, personas, topics, protocol surface, sealed topics, memory — presented as one system, plus a whitepaper that interleaved what/MVP/day-2/why. Two failures followed:

1. **Scope creep in the definition.** Memory, search, curation, and encryption sat at the same level as the stream and the op record, so "what is Soulstream?" had no small answer. The original idea — *a stream on which humans and AI collaborate through operations applied to topics* — was buried under its own elaborations.
2. **A load-bearing "optional" component.** The steward was described as an ordinary persona, but discovery pointed at "the steward's projection" and lifecycle at "steward-suggested" transitions. A component you can't turn off without degrading core flows is plumbing, whatever the docs call it.

The restructure answers one question first — **what is needed for a working soulstream?** — and the answer is deliberately short: a NATS server with JetStream; a `SOULSTREAM` stream; an identity per persona (NATS credential); the protocol on the stream (subjects, op record, topic lifecycle, discovery); baselines and rollups. That is a protocol with a reference library, not a platform. Everything else moved to `extensions/`.

## Why no coordinator — and why no consensus either

Removing the steward raised the obvious question: who rolls up, who marks topics dormant, who answers discovery? The first instinct — "personas decide by consensus" — is a trap: distributed consensus among unreliable, heterogeneous peers (humans who close laptops, agents that crash mid-turn) is a harder moving part than the process it replaces. The core avoids *both* coordinator and consensus by construction:

- **Rollup** is optional for correctness (an un-rolled-up topic just has a longer tail) and race-safe by optimistic concurrency (`Nats-Expected-Last-Subject-Sequence`; first writer wins, loser discards). No election, nothing to agree on.
- **Lifecycle** transitions are idempotent ops any persona posts under deterministic rules; concurrent duplicates merge harmlessly. Human-judgment transitions (close, archive) are decided socially *in the topic*, recorded as ops — a conversation, not a protocol mechanism.
- **Discovery** replays the durable topic-info board (`TOPICS.INFO.>`), plus scatter/gather where *any* persona may answer from its local projection. No responder is required for the base flow to work.

The design rule that fell out: every coordination problem must be solved with deterministic rules + idempotent ops + optimistic concurrency. If a proposal needs an election, a lock service, or agreement among live personas, it is misdesigned for this substrate.

## Why the stream has no `MaxAge`

An earlier (Impire-era) design aged messages out of the stream, then needed three cooperating mechanisms — synchronised retention timers, explicit tombstoning on supersede, advisory-driven reconciliation — to stop externally-stored state from being orphaned, because JetStream evicts silently and the storage layer could never know when a reference died. The root cause was letting the stream expire pointers independently of the objects they point to. The fix removes the cause: the stream never ages messages out; baseline rollup keeps per-topic history physically small; object-store chunks are written before the baseline that references them and deleted only after a superseding baseline publishes. There is no timing window in which a live message references a dead object. Reclaiming closed-topic history is an explicit, loud archival act by a persona — never a silent janitor.

## Why the subjects and headers are named the way they are

The header prefix was originally `Ss-`. Dropped (2026-07-11): "SS" carries a connotation nobody wants in their protocol; the replacement `Soulstream-` is the full word, mirroring the `Nats-` prefix pattern, with nothing left to abbreviate badly later. Subjects moved to `SOULSTREAM.TOPICS.INFO/OPS.<topic-path>`, `SOULSTREAM.PERSONA.NOTIFY.<persona-id>`, and `SOULSTREAM.SVC.*` — fixed tokens uppercase, identifiers lowercase, stated as a normative rule because NATS subjects are case-sensitive and a mixed convention would fork the wire. Moving from one shared `announce` subject to per-topic `INFO` subjects bought something concrete: announcement revisions can use per-subject rollup, so the realm's topic board compacts to at most one message per topic with no janitor. `NOTIFY` replaced `mention` because a per-persona inbox subject is more general than its first use.

Two invented terms were retired the same day (2026-07-11): "head" (a UI or process embedding the library) became plain **client**, and "rung" / "the work ladder" became **stage** / "the work stages." The rule that emerged: invented vocabulary is a budget, spent only on concepts that have no plain-word equivalent — *persona* (an identity that may be operated, shared, or multi-credentialed), *realm* (account-enforced tenancy), *baseline* (the rollup zero-point) earn their place; a "head" is just a client and a "rung" is just a stage. If the plain word works, the plain word wins.

The identity noun was also standardised (2026-07-11): drafts drifted between *persona*, *participant*, and *member*. **Persona** won — it is the term the design already leaned on, and it correctly suggests an identity that may be operated, multi-credentialed, or team-shared. *Member* was deliberately not chosen: it already means something precise — a key-holder of a sealed topic's epoch, the one place membership is enforced rather than hinted — and overloading it there would blur the spec exactly where it must be sharp. *Participant* is plain English, never a defined term. The notify subject follows the noun: `SOULSTREAM.PERSONA.NOTIFY.<persona-id>`.

## Why the record lives in headers

A NATS message is already an envelope: subject, headers, payload. Wrapping payloads in a JSON envelope duplicated what the transport provides (notably an ID next to `Nats-Msg-Id`). Headers carry the record; payloads are pure data; sealed payloads become raw ciphertext with no wrapper to leak structure. A canonical JSON record (JCS) still exists — signatures and portable exhibits need a serialisation that lives outside a NATS message — but it is derived mechanically from the wire form, defined once.

## Why identity is a credential, and the registry is an extension

"Every persona needs an identity" bottoms out in what NATS already enforces: a user credential (nkey) whose permissions are the ACL, plus a persona name bound by the permission template. Profiles, `kind`, `operated_by`, and published keys are useful — but a realm without the registry KV is still a working soulstream, so the registry is an extension. The one identity rule that *is* core: no `on_behalf_of`. Delegation is scoped credentials or a separately-named persona; attribution laundering is refused by design.

## Why lifecycle moved onto the ops subject

Earlier drafts gave lifecycle its own subject (`soulstream.life.<topic>`), which bought a cheap "all lifecycle everywhere" wildcard at the cost of a second subscription per topic, lifecycle ops outside the DAG, and transitions surviving outside baselines. Folding `life.transition` into the topic's own op-log restored one invariant shape — baseline first, ops after, everything in the DAG, everything compacted into baselines. The wildcard's only consumer was the steward, which no longer exists; a curator extension can afford to subscribe to `SOULSTREAM.TOPICS.OPS.>` and filter.

## Why the runtime is deferred but the artefact is not (2026-07-11)

A fair worry, raised by Daan after the restructure: by paring core down to subjects, records, and topics, does Soulstream become communication-only? Humans collaborate *around* concrete things — a file, an invoice, a codebase; a persona doing work needs a workbench. If the protocol only carries talk, it loses collaboration itself.

The resolution was a reframe, not a scope change, because the design already contained the answer without naming it. A topic has state — the baseline, a materialised artefact that persists and outlives the conversation that produced it — and operations that change that state. "Baseline + ops → new baseline" *is* "modifying things together and seeing where they end up." Conversation was never the essence; `turn.post` is one operation type. The docs, however, framed topics as "conversations," which invited exactly the communication-only misreading. The framing was fixed ([core/03-topics.md](./core/03-topics.md)): a topic is a shared workbench; the baseline is the thing on the bench.

What a sandbox adds on top is precisely two things, both runtime concerns: a filesystem/process *view* of topic artefacts so ordinary tools can touch them, and an *execution site* with shared visible state. What it must never own is the artefact itself — authoritative bytes, history, and current state stay in the topic, and sandbox outputs flow back as ops. That principle, plus the staged vocabularies that lead there (versioned artefacts → work items → co-editing → executable workloads → sandboxes), now lives in [extensions/work.md](./extensions/work.md). The runtime is still designed last, against a working execution stage, because a coordination vocabulary invented before a real runtime is speculation — but *deferred* now demonstrably means "the room is built last," not "the work doesn't matter."

## Decisions inherited from the Impire-era redesign

Still standing, with their original reasons:

| Decision | Why |
|---|---|
| Vocabulary: personas / realm / topics (not imps / keepers / tenant) | Humans and AIs share one noun by design. |
| Blobs in the JetStream object store, not an external service | Single-dependency deployment; swappable behind the same name+digest convention. |
| `kind` is presentation metadata; behaviour may never branch on it | The peer principle, made testable. |
| Sealed topics for confidentiality; realm stays read-open by default | The threat model includes the operator; permissions alone only exclude peers. |
| Memory is persona-local + scatter/gather testimony with citation grades | No privileged plumbing; a realm's memory is the union of what personas bothered to remember. |
| Optional Ed25519 signatures; any kept signed op is self-authenticating | Anyone can be a witness; archivists are a coverage optimisation, not a trust anchor. No reputation mechanism in the substrate. |
| The sandbox *runtime* is designed last; artefact-centred topics are core framing | A sandbox contributes a filesystem/process view and an execution site — runtime concerns. The artefact, its state, and its history are already the substrate's job. See below. |

## Principles that survived the diet

One protocol, no second door. Protocol symmetry, attention asymmetry (the scarce resource is human attention; mentions and digests exist for the slow reader). No canonical UI — the substrate is the product; anything that only exists in one client isn’t part of the platform. No privileged plumbing — above NATS there are no special services, and now also no special *dependencies*. Convention over enforcement — subject permissions are nearly the whole security model; unknown op types are ignored with a warning. Lean on NATS, don't wrap it.
