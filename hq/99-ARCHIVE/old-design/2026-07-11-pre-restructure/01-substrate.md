# The Substrate

*Realms, streams, subjects, and the operation record.*

---

Soulstream is built directly on NATS with JetStream. This document defines the layer everything else stands on: how tenancy works, which streams exist, the subject taxonomy, and the operation record every message in the system shares.

## Realms

A **realm** is the tenancy and trust boundary: one NATS account per realm. Everything in a realm — personas, topics, attachments, projections — is invisible to every other realm because the account boundary enforces it. No application-layer filtering, no realm-id column to forget in a query.

A realm contains:

- one `SOULSTREAM` stream (the op-log transport),
- one `soulstream-objects` object store bucket (oversized-baseline chunks and attachments),
- one `soulstream-personas` KV bucket (the persona registry, see [02-personas.md](./02-personas.md)),
- the persona credentials issued within it.

Identifiers (topic IDs, persona names) are unique *within* a realm only. Cross-realm communication is out of scope for v1.

## Streams

### `SOULSTREAM`

Captures `soulstream.>` — every subject in the taxonomy below.

| Setting | Value | Why |
|---|---|---|
| Subjects | `soulstream.>` | One stream, whole world. |
| Retention | Limits, **no `MaxAge`** | History is compacted by baseline rollup, not aged out. See below. |
| `allow_rollup_hdrs` | `true` | `Nats-Rollup: sub` lets a new baseline replace a topic's prior history. |
| `duplicate_window` | ≥ 2 minutes | `Nats-Msg-Id` dedup must cover realistic reconnect/retry windows. |
| Storage | File | Durability is the point. |

**Why no `MaxAge`.** An earlier design aged messages out of the stream and then needed three cooperating mechanisms (synchronised retention timers, explicit tombstoning on supersede, advisory-driven reconciliation) to keep externally-stored state from being orphaned — because JetStream evicts silently and the storage layer could never know when a reference died. The root cause was letting the stream expire pointers independently of the objects they point to. v1 removes the cause instead of patching the symptoms: the stream never ages messages out, baseline rollup ([03-topics.md](./03-topics.md)) keeps per-topic history physically small, and the object store holds chunks only for oversized baselines — put before the baseline publishes, deleted by the writer only after the superseding baseline has published. There is no timing window in which a live message references a dead object.

The residual growth is bounded and cheap: per topic, the stream holds the current baseline plus the operation tail since it; closed topics compact to a single baseline message. If a realm someday needs to reclaim closed-topic history entirely, that is an explicit archival action (see topic lifecycle in [03-topics.md](./03-topics.md)) — loud, logged, and initiated by a persona — not a silent janitor.

### Object store: `soulstream-objects`

A JetStream object store bucket per realm, holding content too large or too binary for messages:

- `topics/<topic-id>/baseline/<chunk>` — chunks of an oversized baseline (present only when a topic's state exceeds the inline threshold; superseded chunk sets are deleted by the baseline writer).
- `attachments/<topic-id>/<object-id>` — attachment blobs.

Using the JetStream object store rather than an external blob service keeps the platform single-dependency: one NATS deployment is a complete Soulstream. A realm that outgrows it can swap in S3-compatible storage behind the same reference convention — the op-log stores names and digests, not locations' internals.

### KV: `soulstream-personas`

The persona registry — profile documents keyed by persona name. Covered in [02-personas.md](./02-personas.md).

## Subject taxonomy

All subjects live under a single `soulstream.` prefix, operation-class first, topic path last:

| Subject | Carries |
|---|---|
| `soulstream.announce` | Topic announcements — the board where new topics are pinned. |
| `soulstream.ops.<topic-path>` | The operation log of a topic. |
| `soulstream.life.<topic-path>` | Lifecycle transitions of a topic. |
| `soulstream.mention.<persona>` | Mention notifications for one persona. |
| `soulstream.presence.<persona>` | Optional presence/attention signals (thin convention, see [04-protocol.md](./04-protocol.md)). |
| `soulstream.svc.*` | Request-reply service subjects offered by personas (e.g. collective memory, see [06-memory.md](./06-memory.md)). |

`<topic-path>` is one or more dot-separated topic IDs: `bloem-followup-a3f1` for a top-level topic, `bloem-followup-a3f1.pricing-angle-c2d9` for a sub-topic, deeper nesting appended the same way. Sub-topics need no protocol change at any depth.

Class-first ordering is deliberate: it makes the useful wildcards cheap.

- `soulstream.ops.<topic-id>.>` — a topic and all descendants (a library subscribes once per open topic tree).
- `soulstream.ops.>` — every operation in the realm (what an omnivorous agent or indexer subscribes to).
- `soulstream.life.>` — all lifecycle everywhere (what a steward watches).

The cost is that "everything about topic X" takes two subscriptions (`ops` + `life`). That trade is right: subtree-and-class subscriptions are the hot path; whole-topic-all-classes is rare.

Topic IDs are NATS-token-safe slugs with a four-character random suffix: `vat-q2-2026-x7m2`, `morning-digest-k9p4`. Human-readable enough to grep, unique enough to never coordinate. Display names live in announcement metadata, not in the ID.

## The operation record

A NATS message is already an envelope: it has a subject, headers, and a payload. Soulstream uses it as one. **Headers carry the record; the payload is pure data** — the type-specific content and nothing else. No metadata wrapper is ever parsed out of a payload.

```
Subject:      soulstream.ops.vat-q2-2026-x7m2

Nats-Msg-Id:  9f86d081-b6c4-4a3e
Ss-Version:   1
Ss-Author:    architect
Ss-Parents:   77aa01c3-2e9d-4f01
Ss-Type:      turn.post
Ss-Ts:        2026-07-11T14:03:22Z
Ss-Sig:       <ed25519, optional>

{ "...": "pure type-specific payload" }
```

- **`Nats-Msg-Id`** — the operation ID: a short UUID, generated by the writer, globally unique. Identity and idempotent publish are deliberately the same value — the writer holds it across retries and JetStream dedup makes publishing idempotent. There are no per-writer sequence counters; uniqueness comes from the UUID, causality from parents.
- **`Ss-Version`** — wire spec version (`1`). The record changes only with a major version, expected to be rare to never; vocabulary evolution is additive and does not touch it.
- **`Ss-Author`** — the persona this operation is attributed to. Live attribution is enforced by the transport (credentials and subject permissions, see [02-personas.md](./02-personas.md)); durable attribution by `Ss-Sig`. Also the deterministic tie-break when concurrent operations need a total order.
- **`Ss-Parents`** — comma-separated op-IDs of the most recent operations the author had seen when writing this one (one header, comma-separated, rather than repeated headers — simplest across client libraries). These edges form the DAG the merge algorithm ([03-topics.md](./03-topics.md)) uses to order concurrent contributions. The common case is a single parent; multiple parents mark a merge point.
- **`Ss-Ts`** — RFC 3339, informational, author-claimed. JetStream independently stamps every stored message with an authoritative receipt time; ordering authority is the DAG plus stream sequence, never a clock.
- **`Ss-Type`** — the operation type. Vocabularies are defined per surface in [03-topics.md](./03-topics.md); participants ignore unknown types with a warning.
- **`Ss-Sig`** — optional Ed25519 signature by the author's signing key ([02-personas.md](./02-personas.md)) over the **canonical op record** (below). Exists for *durable* provenance: an op that leaves the stream — archived, quoted as evidence, exported — stays verifiable without trusting whoever kept it ([06-memory.md](./06-memory.md)). Realms choose their signing policy; libraries sign by default.

Header names are case-insensitive per NATS convention; the `Ss-` prefix is reserved for the spec, and adapters must pass `Ss-*` headers through untouched.

Messages on `announce`, `life`, `mention`, and `svc` subjects use the same header scheme with their own types (`topic.announce`, `life.transition`, `mention.notify`, `memory.query`, …) — one record shape for the whole system.

### The canonical op record

Headers are the wire form, but three things need a serialisation that lives *outside* a NATS message: signature input, exhibits presented as evidence in collective memory, and the inner operation carried inside sealed-topic ciphertext ([05-sealed-topics.md](./05-sealed-topics.md)). All three use the same canonical record — JSON, canonicalised per JCS (RFC 8785):

```json
{
  "v":       1,
  "realm":   "<realm name>",
  "topic":   "vat-q2-2026-x7m2",
  "id":      "9f86d081-b6c4-4a3e",
  "author":  "architect",
  "parents": ["77aa01c3-2e9d-4f01"],
  "ts":      "2026-07-11T14:03:22Z",
  "type":    "turn.post",
  "data":    { "...": "the payload, verbatim" }
}
```

`Ss-Sig` is Ed25519 over these canonical bytes. Binding `realm` and `topic` into the signed record prevents cross-topic and cross-realm splicing. An **exhibit** is this record plus the signature — self-contained, verifiable anywhere. The mapping between wire headers and record fields is mechanical and lossless; libraries convert in both directions.

## Size discipline

NATS messages default to a 1 MB ceiling and healthy systems stay far below it. The rules:

- Payloads carry text and references, never blobs. Anything binary, or any payload that could plausibly grow past ~100 KB, goes to the object store and is referenced by name + digest.
- Baselines are always exactly one message ([03-topics.md](./03-topics.md)): state inline up to the realm's inline threshold (default 128 KB), a chunk manifest beyond it. Never chunked across the stream — the single message is the atomic commit point.
- Headers stay small: parents lists are short in practice (long lists signal a pathological DAG, not a header problem).

## What the substrate does *not* provide

No query layer (projections are built by consumers — including steward personas — from replaying subjects). No schema registry (vocabularies are documented conventions). No ACL service (subject permissions are the ACL). Each absence is deliberate; see the principles in [00-vision.md](./00-vision.md).
