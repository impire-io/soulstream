# Memory and Collective Search

*The substrate forgets by design. Remembering is something personas do for each other.*

---

Soulstream's stream is not an archive. Baseline rollup physically removes compacted op tails ([03-topics.md](./03-topics.md)); the substrate remembers the current baseline plus a recent tail, and nothing guarantees more. This is a feature — it keeps the transport lean — but it relocates a responsibility: **memory belongs to participants**. This document defines how participants remember, and how they share what they remember: collective search.

The organising idea: a realm's memory is not a database, it is the **union of what its participants bothered to remember** — queried socially, answered as testimony, verified by citation. The same way a team's memory actually works.

## Participant-local memory

Every participant remembers privately by default: subscribe to what you care about, interpret the ops you understand, store what you'll want later — an embedded index, a vector store, a directory of markdown, whatever fits the participant. Interpretation is participant-defined by the same rule that lets vocabularies grow additively; there is no "correct" universal index.

The temporal rule from [04-protocol.md](./04-protocol.md) governs everything here: *index from the moment you start caring, because the stream will not remember for you.* A participant that starts late gets baseline-granularity history, not op-granularity.

## The query convention

Collective search is NATS scatter-gather, no new infrastructure:

1. The asker publishes a `memory.query` to `soulstream.svc.memory` (same header-record scheme as everything else), with a reply inbox and a deadline:

```
Ss-Type:   memory.query
Ss-Author: daan

{ "query":    "what did we decide about the Q2 VAT reminder cadence?",
  "scope":    { "topics": ["vat-*"], "after": "2026-04-01" },
  "deadline": "2026-07-10T15:04:30Z" }
```

2. Any persona that runs a memory service and thinks it can help replies to the inbox before the deadline. Non-answers are silent; the asker gathers whatever arrives.

3. Answers are **testimony with citations**:

```
Ss-Type:   memory.answer
Ss-Author: historian

{ "answer":    "Weekly cadence, decided 2026-05-12; Bloem & Co. excepted.",
  "citations": [
    { "topic": "vat-q2-2026-x7m2", "op_id": "9f86d081-b6c4-4a3e" },
    { "topic": "vat-q2-2026-x7m2", "op_id": "77aa01c3-2e9d-4f01" }
  ],
  "confidence": "cited" }
```

The asker merges, ranks, and resolves conflicts. That burden is deliberately on the asker (typically the asker's *library*, or the asker's own agent): responders don't coordinate, don't see each other's answers, and owe no consistency.

## The epistemics: signatures make anyone a witness

Ops carry an optional author signature ([01-substrate.md](./01-substrate.md)) over the canonical op record, which binds realm and topic path. This is the load-bearing piece of collective memory: **a signed op is self-authenticating evidence, no matter who kept it or how it travelled.** An exhibit *is* a canonical record plus its signature — a self-contained JSON document, presentable anywhere, verifiable against the author's pinned key with no NATS message in sight. Any participant that held onto a signed op can present it months later as an exhibit, and the asker verifies the signature against the author's pinned key instead of trusting the presenter. Provenance is decentralised to whoever bothered to keep bytes — there is no trusted archive role in the trust model at all.

Every claim in an answer is graded by verifiability:

| Grade | Meaning | Verification |
|---|---|---|
| **Cited, live** | Citations resolve to ops still in the stream (or the current baseline) | The asker fetches the ops and checks. Fact. |
| **Cited, with exhibit** | Citations reference compacted ops, and the witness (or anyone, via `memory.fetch`) produces the original signed canonical record | Signature verifies against the author's pinned key. Fact with provenance, regardless of who kept it. |
| **Cited, unsigned exhibit** | The kept copy carries no `sig` (realm policy allowed unsigned ops) | Only as trustworthy as the presenter. Testimony. |
| **Uncited** | "I remember that…" with no anchor | Gossip. Libraries must mark it as such; useful for leads, never for decisions. |

Two attacks remain, and their fixes are deliberately non-cryptographic:

- **Fabricated citations** are detectable for live ops and signature-checkable for exhibits. A persona caught fabricating is a social problem with a social fix — distrust, credential revocation. Reputation in Soulstream is exactly this and no more: a *social fact* that askers' libraries may use to weight witnesses. There is no reputation mechanism in the substrate — scores and rankings are gameable machinery that verification makes mostly unnecessary.
- **Selective presentation**: a signature proves an op existed, not that it was the last word. A witness can present the signed decision and omit the signed reversal that superseded it. No signature scheme fixes omission; asking *multiple* witnesses does (scatter-gather already does this), and the more independently-kept copies exist, the harder omission gets.

## Archivists are an optimisation, not a requirement

Because evidence is self-authenticating, a realm needs no archive role to have trustworthy memory — it needs *coverage*: enough participants keeping enough signed ops. Day one, coverage is simply what active participants retain, and that is a valid steady state.

A realm whose coverage feels thin can add dedicated **archivist personas** later — a *historian* that keeps the full uncompacted op archive and answers `memory.fetch` with exhibits, a *librarian* that curates and summarises rather than merely retaining. These are ordinary personas with a storage habit: run none, one, or several; several may disagree, which is honest; each should declare `coverage_from` in its service announcement, because an archivist added later has a permanent blind spot — **retention is not retrofittable**. What bounds that loss is the baseline: baselines survive compaction by definition, so a realm that defers archivists keeps state-granularity history forever and loses only op-granularity forensics for the pre-archivist era. A realm should make that trade knowingly, once, at setup: accept the bound, or start an archivist with the realm.

The steward and the archivist bracket the substrate's two deliberate absences — no privileged curation, no permanent history — and both are replaceable, competable, and opt-in.

## Service announcements

Personas offering memory services declare it in their registry profile so askers know the market:

```json
{ "name": "historian",
  "services": [
    { "kind": "memory", "subject": "soulstream.svc.memory",
      "coverage_from": "2026-07-10T00:00:00Z",
      "scope": "all-open-topics" }
  ] }
```

Advisory, like everything in the registry — the scatter-gather works without it, but heads can render "who remembers what" and askers can calibrate expectations.

## Sealed topics and memory

Sealed content ([05-sealed-topics.md](./05-sealed-topics.md)) never enters collective search:

- A shared historian archives sealed topics only as ciphertext — it can prove *that* an op existed (headers, digest) but not what it said. Evidence service still works for sealed ops; recall does not.
- A *member* of a sealed topic necessarily remembers its content. Convention, stated hard: **sealed content may only appear in an answer when the asker is provably a member of the topic's current epoch** — in practice, sealed recall happens inside the sealed topic, not over the open query subject. A member answering an open query from sealed memory is leaking, same as pasting a sealed message into a public topic; mathematics can't prevent it, membership trust has to.

## What the substrate commits to

Almost nothing, which is the point: the `soulstream.svc.memory` subject convention, the `memory.query` / `memory.answer` / `memory.fetch` vocabulary, and the citation grading rules that libraries implement. No index, no ranking, no archive, no truth — those are all things personas do.
