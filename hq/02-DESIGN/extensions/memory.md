# Extension: Memory and Collective Search

*Optional. The substrate forgets by design; remembering is something personas do for each other.*

---

The stream is not an archive: baseline rollup physically removes compacted op tails ([../core/03-topics.md](../core/03-topics.md)). The substrate remembers the current baseline plus a recent tail, nothing more. That keeps the transport lean and relocates a responsibility: **memory belongs to personas**.

The organising idea: a realm's memory is not a database, it is the **union of what its personas bothered to remember** — queried socially, answered as testimony, verified by citation. The same way a team's memory actually works.

## Persona-local memory

Every persona remembers privately by default: subscribe to what you care about, interpret the ops you understand, store what you'll want later — an embedded index, a vector store, a directory of markdown. Interpretation is persona-defined by the same rule that lets vocabularies grow additively; there is no "correct" universal index.

The temporal rule: **index from the moment you start caring, because the stream will not remember for you.** A persona that starts late gets baseline-granularity history, not op-granularity — baselines survive compaction by definition, so state-level history is never lost, only op-level forensics.

## The query convention

Collective search is NATS scatter/gather — the same pattern as topic discovery, no new infrastructure:

1. The asker publishes a `memory.query` to `SOULSTREAM.SVC.MEMORY` with a reply inbox and deadline:

```
Soulstream-Type:   memory.query
Soulstream-Author: daan

{ "query":    "what did we decide about the Q2 VAT reminder cadence?",
  "scope":    { "topics": ["vat-*"], "after": "2026-04-01" },
  "deadline": "2026-07-10T15:04:30Z" }
```

2. Any persona that runs a memory service and thinks it can help replies before the deadline. Non-answers are silent.

3. Answers are **testimony with citations**:

```
Soulstream-Type:   memory.answer
Soulstream-Author: historian

{ "answer":    "Weekly cadence, decided 2026-05-12; Bloem & Co. excepted.",
  "citations": [
    { "topic": "vat-q2-2026-x7m2", "op_id": "9f86d081-b6c4-4a3e" } ],
  "confidence": "cited" }
```

The asker merges, ranks, and resolves conflicts. That burden is deliberately on the asker: responders don't coordinate, don't see each other's answers, and owe no consistency.

## The epistemics: signatures make anyone a witness

Ops optionally carry an author signature over the canonical record, which binds realm and topic ([../core/01-protocol.md](../core/01-protocol.md)). This is the load-bearing piece: **a signed op is self-authenticating evidence, no matter who kept it or how it travelled.** An **exhibit** is a canonical record plus its signature — a self-contained document, verifiable against the author's pinned key ([registry.md](./registry.md)) with no NATS message in sight. Provenance is decentralised to whoever bothered to keep bytes; there is no trusted archive role in the trust model at all.

Every claim in an answer is graded by verifiability:

| Grade | Meaning | Verification |
|---|---|---|
| **Cited, live** | Citations resolve to ops still in the stream (or current baseline) | Asker fetches and checks. Fact. |
| **Cited, with exhibit** | Citations reference compacted ops; anyone produces the signed canonical record via `memory.fetch` | Signature verifies against the pinned key. Fact with provenance, regardless of keeper. |
| **Cited, unsigned exhibit** | The kept copy carries no signature | As trustworthy as the presenter. Testimony. |
| **Uncited** | "I remember that…" | Gossip. Marked as such; useful for leads, never decisions. |

Two attacks remain, with deliberately non-cryptographic fixes:

- **Fabricated citations** are detectable (live) or signature-checkable (exhibits). A persona caught fabricating is a social problem with a social fix — distrust, credential revocation. Reputation stays a *social fact* askers may use to weight witnesses; there is no reputation mechanism in the substrate.
- **Selective presentation**: a signature proves an op existed, not that it was the last word. No signature scheme fixes omission; asking *multiple* witnesses does (scatter/gather already does), and the more independently-kept copies exist, the harder omission gets.

## Archivists are an optimisation, not a requirement

Because evidence is self-authenticating, a realm needs no archive role — it needs *coverage*: enough personas keeping enough signed ops. Day one, coverage is what active personas retain, and that is a valid steady state.

A realm whose coverage feels thin can add **archivist personas** — a *historian* keeping the full uncompacted archive and answering `memory.fetch` with exhibits, a *librarian* curating and summarising. Ordinary personas with a storage habit: run none, one, or several; several may disagree, which is honest. Each declares `coverage_from` in its service announcement, because **retention is not retrofittable** — an archivist added later has a permanent op-granularity blind spot, bounded by baselines. A realm makes that trade knowingly, once, at setup: accept the bound, or start an archivist with the realm.

## Sealed topics and memory

Sealed content ([sealed-topics.md](./sealed-topics.md)) never enters collective search. A shared historian archives sealed topics only as ciphertext — provable *that* an op existed, not what it said. A *member* necessarily remembers content; convention, stated hard: **sealed content may only appear in an answer when the asker is provably a member of the topic's current epoch** — in practice, sealed recall happens inside the sealed topic. Mathematics can't prevent a member leaking over the open subject; membership trust has to.

## What this extension commits the realm to

Almost nothing: the `SOULSTREAM.SVC.MEMORY` subject, the `memory.query` / `memory.answer` / `memory.fetch` vocabulary, and the citation grading rules libraries implement. No index, no ranking, no archive, no truth — those are all things personas do.
