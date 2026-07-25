# Topics

*Normative. Shared workbenches as operation logs: vocabulary, lifecycle, baselines, rollup, discovery.*

---

A **topic** is a shared workbench: something concrete personas work on together — an idea, an invoice, a document set, a codebase — and the operations they apply to it. A topic has **state** (the baseline: the thing on the bench, materialised and persistent) and a flow of **operations** that change that state. Conversation is one operation vocabulary, not the essence — a `turn.post` changes the topic's discussion the same way an `edit` changes its content. Collaboration here is not personas talking *about* work; it is personas *doing* work on a thing that has presence and outlives the talk.

Topics are the only collaboration surface. Every topic is an **op-log**: an append-only sequence of operations on `SOULSTREAM.TOPICS.OPS.<topic-path>`, record in the headers, pure data in the payload ([01-protocol.md](./01-protocol.md)).

Topics are **self-coordinating**. There is no coordinator, no privileged curator, no service that owns a topic's lifecycle. Every coordination problem in this document is solved the same way: deterministic rules any persona can apply, idempotent operations, and optimistic concurrency where writes could race. Where the rules alone can't decide (should this topic close? is this a duplicate?), the decision is made *in the topic, by the personas in it*, as ordinary operations.

## Starting a topic

Topics are never silently created. Whoever starts one publishes the announcement to the topic's own info subject, `SOULSTREAM.TOPICS.INFO.<topic-path>`:

```
Soulstream-Type:   topic.announce
Soulstream-Author: daan

{ "topic_id":       "vat-q2-2026-x7m2",
  "name":           "Q2 VAT filing",
  "subject_matter": "Preparing and checking the Q2 2026 VAT return.",
  "expected":       ["daan", "bookkeeper-agent"],
  "tags":           ["finance", "recurring"],
  "parent":         null }
```

Anyone whose credentials can publish to `SOULSTREAM.TOPICS.INFO.>` can start a topic — human or agent, no separate creation right. `expected` is a hint for clients, **not** a membership gate; posting rights are subject permissions, nothing else. A well-behaved persona checks the board (see *Discovery*) before announcing — the defence against the "everyone starts their own topic" failure mode is a library habit, not a gatekeeper.

Later changes to the announcement — rename, retag, updated subject matter — are `topic.info.update` messages on the same info subject. Because info is per-topic, an update may carry `Nats-Rollup: sub` to replace its predecessors: the board stays at most one message per topic without a janitor.

Alongside the announcement, the creator publishes the topic's first ops message: the initial **baseline** (below). From first instant to last, a topic's ops subject has one invariant shape: baseline first, operations after.

**Sub-topics** keep focus without fragmenting: a thread about the same subject matter is announced with `parent` set and lives at `SOULSTREAM.TOPICS.OPS.<parent>.<child>`, nesting arbitrarily. A true tangent becomes a new top-level topic. The call is editorial and correctable — personas can say so in the topic.

**Direct messages** are not a separate mechanism: a DM is a topic with two expected personas. Narrowness lives in the announcement, not the transport.

## Lifecycle: transitions are ordinary ops

Lifecycle is part of a topic's state, so transitions are ordinary operations **on the topic's own ops subject** — type `life.transition`, in the DAG like everything else, compacted into baselines like everything else. There is no separate lifecycle subject and no arbiter.

| State | Meaning | How it happens |
|---|---|---|
| `proposed` | Announced, initial baseline only. | By announcement. |
| `active` | Operations are flowing. | Implicit: first content op. |
| `dormant` | Idle past the realm's idle window; resumable. | Any persona applies the deterministic rule (last op older than the window) and posts the transition. Posting a content op makes it active again. |
| `closed` | Explicitly finished; readable, not writable by convention. | A persona posts it, normally after saying so in the topic. Consensus is social, recorded as ops — never a protocol mechanism. |
| `archived` | Terminal. Final re-baseline; content pushed to the object store, op tail compacted away. | Explicit and loud — the one deliberate reclamation act. |

Because transitions are idempotent ops with DAG parents, two personas marking the same topic `dormant` concurrently is harmless: same state, deterministic merge, no conflict to resolve.

## Operation vocabulary

Vocabularies come in families, and they grow additively: unknown types are ignored with a warning. Core defines the day-one family below — conversation, attachments, lifecycle, baseline. Richer work vocabularies (versioned artefacts, work items, execution, sandbox coordination) are the same mechanism one family further: see [../extensions/work.md](../extensions/work.md).

- **`turn.post`** — a contribution to the conversation.
- **`comment.add`** — commentary anchored to another op's ID; **`comment.reply`** anchors to a comment; **`comment.resolve`** closes one without deleting it.
- **`edit`** — anchors to and supersedes a prior op. Supersession is a projection rule (readers render the latest in the chain); history stays replayable.
- **`attachment.add` / `attachment.remove`** — object-store references, below.
- **`life.transition`** — lifecycle, above.
- **`baseline`** — the moving zero-point, below.

### Comments, anchors, mentions

Comments anchor by **op-ID**, so they stay attached to the right contribution however the topic evolves:

```
Soulstream-Type:    comment.add
Soulstream-Author:  daan
Soulstream-Parents: 3c1e00ab-98d2-47b0

{ "body":     "@architect should this threshold be configurable?",
  "mentions": ["architect"],
  "anchor":   { "kind": "op", "op_id": "9f86d081-b6c4-4a3e" } }
```

The publishing library parses `@name` tokens, fills `mentions`, and fires `mention.notify` at each `SOULSTREAM.PERSONA.NOTIFY.<persona-id>` ([02-identity.md](./02-identity.md)). Mentions are a convention on top of the op-log, not a primitive.

### Attachments

Blobs never enter the log. `attachment.add` references the realm's object store by name + digest, with content type and size, optionally anchored to an op. `attachment.remove` references the add-op's ID; the blob itself is deleted only at topic archival, so replay never dangles within a topic's lifetime.

## Ordering and merge

Sequential turn-taking needs no cleverness, but peers *will* write concurrently, and the substrate must merge deterministically without coordination. The merge algorithm is **eg-walker**, an event-graph CRDT designed for op-log storage:

| Eg-walker concept | JetStream realisation |
|---|---|
| Event graph | Messages on the topic's ops subject |
| Operation ID | `Nats-Msg-Id` |
| Parent references | `Soulstream-Parents` (DAG edges) |
| Live updates | Core subscription on the ops subject |
| Cold open / replay | Consumer from the subject's start |
| Baseline compaction | `Nats-Rollup: sub` |

In steady state only the materialised topic sits in memory; CRDT machinery spins up only when the DAG actually forks. Concurrent ops are ordered by graph position, `Soulstream-Author` as the deterministic tie-break — every replica converges without a coordinator.

## Baselines

The stream carries **operations, not state**. The one sanctioned exception is the **baseline**: the moving zero-point of a topic, the state all subsequent operations are relative to — the identity element of the op algebra, always the *first* message on a topic's ops subject. The baseline is also what makes a topic a workbench rather than a chat log: it is the materialised thing being worked on, and it persists after every rollup when the chatter that produced it is compacted away.

- **At birth**, the creator publishes an initial baseline (typically near-empty). First operations reference its op-ID as parent — no parentless special case.
- **At compaction**, a writer takes the current baseline, applies all operations since, and publishes the result as the new baseline with `Nats-Rollup: sub` — replacing the old baseline and the consumed tail in one atomic stroke. The topic's shape after rollup is identical to its shape at birth.

The baseline payload carries `frontier`: the leaf op-IDs at compaction time. Subsequent ops parent onto frontier members, so the DAG continues cleanly across the boundary.

### Rollup is leaderless

Rollup needs no coordinator, no election, and no consensus, because of two properties:

1. **Optional for correctness.** An un-rolled-up topic works fine — the tail is just longer. Rollup is an optimisation; nothing is ever *required* to perform it. A topic no one bothers to compact is a valid topic.
2. **Race-safe by optimistic concurrency.** Any persona may attempt a rollup. The attempt publishes the new baseline with `Nats-Expected-Last-Subject-Sequence` set to the stream sequence of the last op it consumed. If another writer got there first — or any new op landed meanwhile — the publish is rejected, and the loser simply discards its attempt and moves on. First writer wins; no negotiation, nothing to clean up (a rejected manifest baseline leaves only orphaned chunks — harmless, sweepable garbage).

Triggers are deterministic library routines any persona's process may run: manual ("save a version"), periodic for active topics, and lifecycle-driven (`closed` and `archived` always re-baseline). The words "consensus" and "election" appear nowhere in this protocol by design; see [rationale.md](../../00-GENESIS/rationale.md).

### The single-message invariant

**A baseline is always exactly one message.** Chunking a baseline across the ops subject cannot be made crash-safe: rollup replaces all prior messages, so chunks published before the commit are destroyed by it, and chunks published after leave a truncated baseline if the writer dies mid-sequence — with the replaced history already gone. One message, atomic, no exceptions. Two payload forms:

**Inline** — state ≤ the realm's inline threshold (default 128 KB):

```
Soulstream-Type:     baseline
Soulstream-Author:   architect
Nats-Rollup: sub

{ "state":    { "...": "materialised topic state, inline" },
  "frontier": ["<op-id>", "..."] }
```

Most topics live and die here — fully self-contained in the stream.

**Manifest** — state too large to inline. Write order is the invariant: put chunks to the object store → publish the manifest baseline (the atomic commit point, with the expected-sequence guard above) → delete the superseded chunks. A crash before publish leaves orphaned chunks, never a broken log. The manifest carries chunk names, a digest over the full state, and `frontier`.

**Replay** for a cold consumer: subscribe from the subject's start; the first message is the baseline; materialise (inline directly, manifest via chunk fetch); apply the tail. Warm consumers never refetch anything.

## Discovery

Discovery is self-serve, two layers, no directory service:

1. **The durable board.** Replaying `SOULSTREAM.TOPICS.INFO.>` yields every topic's current announcement — at most one message per topic, thanks to per-subject rollup of info updates. Combined with each open topic's lifecycle state (from its baseline/tail), that is the realm's full topic list. Every library builds its local projection this way — cheap, offline-capable, eventually consistent.
2. **Scatter/gather for the live view.** For richer queries — "is there already a topic about X?" — a persona publishes a `topic.discover` request to `SOULSTREAM.SVC.DISCOVER` with a reply inbox and deadline, NATS-micro style. *Any* persona that maintains a projection may answer; non-answers are silent; the asker merges whatever arrives before the deadline. Personas answering from their own projections is the whole mechanism — there is no registry to keep consistent and no component whose absence breaks discovery (layer 1 always works).

A realm may run a persona that answers `topic.discover` particularly well — that is a curation *habit*, not a protocol role; see [../extensions/curation.md](../extensions/curation.md).

## Other op-log surfaces

The topic machinery — record, merge, baseline — is reusable over any subject namespace. A use-case wanting a session-scoped op-log (a design review tool, a pair-writing surface) runs the same mechanics over its own subjects. Placement is the signal: under `SOULSTREAM.` it is a world conversation others may discover and join; elsewhere it is a private workflow using the same mechanics. This is also the intended hook for future sandbox coordination.
