# Topics

*Focused conversations as operation logs.*

---

A **topic** is a focused, multi-party conversation: subject matter, participants, and a flow of operations — turns, comments, angles, attachments, edits. Topics are the only collaboration surface in v1; documents, decisions, and (later) sandbox coordination are all things that happen *inside* topics.

Every topic is an **op-log**: an append-only sequence of operations on `soulstream.ops.<topic-path>` — record in the headers, pure data in the payload, as defined in [01-substrate.md](./01-substrate.md). Modelling everything as operations on one log is the simplest shape that accommodates the variety, and it gives replay, audit, and merge for free.

## Starting a topic

Topics are never silently created. Whoever starts one publishes to `soulstream.announce`:

```
Ss-Type:   topic.announce
Ss-Author: daan

{ "topic_id":       "vat-q2-2026-x7m2",
  "name":           "Q2 VAT filing",
  "subject_matter": "Preparing and checking the Q2 2026 VAT return.",
  "expected":       ["daan", "bookkeeper-agent"],
  "tags":           ["finance", "recurring"],
  "parent":         null }
```

Anyone whose credentials can publish to `soulstream.announce` can start a topic — human, agent, no separate creation right. `expected` is a hint for heads and the steward, **not** a membership gate; posting rights are subject permissions, nothing else. Before announcing, a well-behaved participant checks the steward's topic projection for an existing topic on the same matter — the defence against the "everyone starts their own topic" failure mode.

Alongside the announcement, the creator publishes the topic's first message on its ops subject: the initial **baseline** (see *Baselines* below) — the zero-state all subsequent operations are relative to. From its first instant to its last, a topic's ops subject has one invariant shape: baseline first, operations after.

**Sub-topics** keep focus without fragmenting: something that deserves its own thread *about the same subject matter* is announced with `parent` set and lives at `soulstream.ops.<parent>.<child>`, nesting arbitrarily. A true tangent becomes a new top-level topic instead. The call is editorial; the steward helps when it's misjudged.

**Direct messages** are not a separate mechanism: a DM is a topic with two expected participants. Narrowness lives in the announcement, not the transport.

## Lifecycle

Transitions publish `life.transition` ops on `soulstream.life.<topic-path>`:

| State | Meaning | Triggered |
|---|---|---|
| `proposed` | Announced, nothing posted yet. | By announcement. |
| `active` | First operation posted. | Automatic. |
| `dormant` | Idle past a configured window; resumable. | By the steward (suggested) or any participant. |
| `closed` | Explicitly finished; readable, not writable by convention. | Explicit, by a participant. |
| `archived` | Terminal. Final re-baseline; content pushed to the object store, op tail compacted away. | Explicit; the loud, logged reclamation act mentioned in [01-substrate.md](./01-substrate.md). |

## Operation vocabulary

Day-one types (the record's `Ss-Type`, payload shape per type; unknown types are ignored with a warning so the vocabulary can grow additively):

- **`turn.post`** — a contribution to the conversation.
- **`comment.add`** — commentary anchored to another op's ID; **`comment.reply`** anchors to a comment; **`comment.resolve`** closes one without deleting it.
- **`angle.introduce`** — opens a new thread of discussion inside the topic, optionally parentless within the DAG.
- **`edit`** — anchors to and supersedes a prior op. Supersession is a projection rule (readers render the latest in the chain); history stays replayable.
- **`attachment.add` / `attachment.remove`** — object-store references, below.
- **`baseline`** — the moving zero-point of the topic, below.

### Comments, anchors, mentions

Comments anchor by **op-ID**, so they stay attached to the right contribution no matter how the topic evolves:

```
Ss-Type:    comment.add
Ss-Author:  daan
Ss-Parents: 3c1e00ab-98d2-47b0

{ "body":     "@architect should this threshold be configurable?",
  "mentions": ["architect"],
  "anchor":   { "kind": "op", "op_id": "9f86d081-b6c4-4a3e" } }
```

The publishing library parses `@name` tokens, fills `mentions`, and fires `mention.notify` at each `soulstream.mention.<name>` (see [02-personas.md](./02-personas.md)). Mentions are a convention on top of the op-log, not a primitive.

### Attachments

Blobs never enter the log. `attachment.add` references the realm's object store:

```
Ss-Type:   attachment.add
Ss-Author: architect

{ "name":         "architecture.png",
  "object":       "attachments/vat-q2-2026-x7m2/8fa3c1",
  "digest":       "SHA-256=...",
  "content_type": "image/png",
  "size_bytes":   142000,
  "anchor":       { "kind": "op", "op_id": "9f86d081-b6c4-4a3e" } }
```

Anchored or unanchored (topic-wide). `attachment.remove` references the add-op's ID; the blob itself is deleted only at topic archival, so replay never dangles within a topic's lifetime.

## Concurrency: eg-walker over JetStream

Sequential turn-taking needs no cleverness. But peers *will* write concurrently — two agents answering at once, a human editing while an agent comments — and the substrate must merge deterministically without coordination. The merge algorithm is **eg-walker**, an event-graph CRDT designed exactly for op-log storage. The fit with JetStream is one-to-one:

| Eg-walker concept | JetStream realisation |
|---|---|
| Event graph | Messages on the topic's ops subject |
| Operation ID | `Nats-Msg-Id` header — identity and idempotent publish are the same value |
| Parent references | `Ss-Parents` header (DAG edges) |
| Live updates | Core subscription on the ops subject |
| Cold open / replay | Consumer from the subject's start |
| Baseline compaction | `Nats-Rollup: sub` |
| Critical version (all caught up) | The steady-state common case |

In steady state only the materialised topic sits in memory; CRDT machinery spins up only when the DAG actually forks. Concurrent ops are ordered by graph position, with `author` as the deterministic tie-break — every replica converges on the same rendering without a coordinator.

## Baselines

The Soulstream is an **ops stream, not a state stream** — it carries the operations executed, and state is always derived. The one sanctioned exception is the **baseline**: the moving zero-point of a topic, the state all subsequent operations are relative to. A baseline is not state interleaved with ops; it is the identity element of the op algebra, and it is always the *first* message on a topic's ops subject:

- **At birth**, the creator publishes an initial baseline (typically near-empty: the topic's starting state, possibly just its declared structure). First operations reference the baseline's op-ID as their parent — no parentless special case.
- **At compaction**, a writer takes the current baseline, applies all operations since, and publishes the result as the new baseline with `Nats-Rollup: sub` — replacing the old baseline and the consumed op tail in one atomic stroke. The topic's shape after rollup is identical to its shape at birth: baseline first, ops after.

`frontier` in the baseline is the set of leaf op-IDs at compaction time; subsequent ops reference frontier members as parents, so the DAG continues cleanly across the boundary.

### The single-message invariant, and what happens when state outgrows it

**A baseline is always exactly one message.** Chunking a baseline across the ops subject cannot be made crash-safe: rollup replaces all prior messages, so chunks published before the commit are destroyed by it, and chunks published after leave a truncated baseline if the writer dies mid-sequence — with the history it replaced already gone. One message, atomic, no exceptions. The payload has two forms:

**Inline** — canonical state ≤ the realm's inline threshold (default 128 KB, comfortably under the 1 MB message ceiling):

```
Ss-Type:     baseline
Ss-Author:   steward
Nats-Rollup: sub

{ "state":    { "...": "materialised topic state, inline" },
  "frontier": ["<op-id>", "..."] }
```

Most topics live and die here: fully self-contained in the stream, no object store involved.

**Manifest** — state too large to inline. The writer first puts chunks to the object store, then publishes a baseline whose payload references them:

```
Ss-Type:     baseline
Ss-Author:   steward
Nats-Rollup: sub

{ "manifest": {
    "chunks":     ["topics/vat-q2-2026-x7m2/baseline/000", "..."],
    "digest":     "SHA-256=...",
    "size_bytes": 4718592 },
  "frontier": ["<op-id>", "..."] }
```

**Write order is the invariant:** put chunks → publish the manifest baseline (the atomic commit point) → delete the superseded chunks. A crash before publish leaves orphaned chunks — harmless garbage, sweepable by any janitor routine — never a broken log. Combined with the no-`MaxAge` stream ([01-substrate.md](./01-substrate.md)), no live baseline ever references a dead object. The digest keeps manifest baselines exhibit-grade verifiable ([06-memory.md](./06-memory.md)) even though their state lives outside the message.

**Replay** for a cold consumer: subscribe from the subject's start; the first message is the baseline; materialise (inline directly, manifest via chunk fetch), then apply the op tail. Warm consumers never refetch anything.

**Triggers:** manual ("save a version"), periodic for active topics (library-scheduled), and lifecycle-driven (`closed` and `archived` always re-baseline).

## The steward

An active realm accumulates stale topics, near-duplicates, and drift. Two layers keep it liveable, per the no-privileged-plumbing principle ([00-vision.md](./00-vision.md)):

**Mechanical lifecycle** — idle-detection, periodic re-baselining — lives in the topic library as time-based routines any participant process can run. No judgment, no inference.

**Curatorial judgment** lives in the **steward**: an agent persona with ordinary credentials that subscribes to `soulstream.announce` and `soulstream.life.>`, maintains a projection of the topic landscape, and publishes it as a regular (system-tagged) topic that heads render as "what's being talked about." It flags likely duplicates with a comment in the newer topic, proposes archival of long-dormant topics with a comment in place, and publishes digests for mention-only readers.

The steward **suggests, never enforces**. Merging, closing, archiving are participants' decisions. And because the steward is just a persona, a realm can replace it, run two competing ones, or run none.

## Other op-log surfaces

The topic library is reusable over any subject namespace: a use-case that wants a session-scoped op-log (a design review tool, a pair-writing surface) runs the same record, merge, and baseline machinery over its own subjects rather than under `soulstream.ops.>`. Placement is the signal: under `soulstream.` it is a world conversation other personas may discover and join; elsewhere it is a private workflow that happens to use the same mechanics. This is also the intended hook for future sandbox coordination — a sandbox session is an op-log of intents and results, with the sandbox's artefacts attached back into the sponsoring topic.
