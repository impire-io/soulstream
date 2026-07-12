# Data Model: Topics — the Op-Log Engine

**Feature**: 002-topics | **Date**: 2026-07-12 | **Source**: [spec.md](./spec.md)

Everything here is a projection of the op-log or a small value type. No database. The wire records
are the foundation's `record.Record`; this feature adds the *meaning* on top.

## Subjects

| Subject | Carries |
|---|---|
| `SOULSTREAM.TOPICS.INFO.<topic-path>` | The topic's announcement (`topic.announce`). |
| `SOULSTREAM.TOPICS.OPS.<topic-path>` | The topic's op-log: baseline first, then ops. |

`<topic-path>` is a dotted chain of topic-ids: `vat-q2-2026-x7m2` or
`vat-q2-2026-x7m2.pricing-angle-c2d9` for a sub-topic. A topic materialises from its **exact** ops
subject (not a wildcard) — a parent never absorbs a child's ops.

## Operation vocabulary (this cycle)

Each is a `record.Record` with a `Soulstream-Type` and a JSON payload.

| Type | Subject | Payload | Meaning |
|---|---|---|---|
| `topic.announce` | INFO | `{topic_id, name, subject_matter, expected[], tags[], parent}` | Starts/labels a topic. |
| `baseline` | OPS (first) | `{state: <json>, frontier: [op-id…]}` | The zero-point; inline state + frontier. |
| `turn.post` | OPS | `{body}` | A contribution to the conversation. |
| `comment.add` | OPS | `{body, anchor: {kind:"op", op_id}}` | Commentary anchored to an op. |
| `life.transition` | OPS | `{to, from?}` | A lifecycle change. |

Unknown types are ignored with a warning during materialisation.

## Entity: TopicID / topic-path

- **TopicID**: `<slug>-<suffix>` — display-name slug + 4 random `[a-z0-9]`. Satisfies the foundation
  slug grammar. Generated without coordination.
- **topic-path**: dot-joined chain of TopicIDs from the root to this topic.

## Entity: Announcement

Fields: `TopicID`, `Name`, `SubjectMatter`, `Expected []string` (hint only), `Tags []string`,
`Parent string` (empty for top-level). Serialised as the `topic.announce` payload; carried on the
INFO subject.

## Entity: TopicHandle

A live binding of a `realm.Client` to one topic-path. Responsibilities:

- **Post(type, payload, anchor?)** → builds a `record.Record` (author = client persona, fresh
  op-id, timestamp now, `Parents` = current frontier), enforces write-side attribution, publishes
  to the OPS subject; returns the op-id.
- Convenience posters: `PostTurn(body)`, `AddComment(body, anchorOpID)`, `Transition(to)`.
- Tracks the **observed frontier** from its materialised view (see below).
- **Materialise(ctx)** → a MaterializedTopic by draining the ops backlog.
- **Follow(ctx, onOp)** → materialise then keep applying live ops (one ordered consumer).

## Entity: Contribution

A materialised turn or comment:

| Field | Meaning |
|---|---|
| OpID | the op's id |
| Author | persona |
| Timestamp | author-claimed |
| Type | `turn.post` / `comment.add` |
| Body | extracted from payload |
| Anchor | for comments: the anchored op-id (empty for turns) |
| Dangling | true if a comment's anchor op-id is not present in the topic |
| StreamSeq | JetStream sequence (the ordering key) |

## Entity: MaterializedTopic (the view)

A pure projection of a topic's log (FR-013/014):

| Field | Meaning |
|---|---|
| Path | topic-path |
| Announcement | metadata (may be nil if only OPS seen) |
| BaselineState | opaque JSON from the baseline |
| Lifecycle | `proposed` / `active` / `closed` (derived, below) |
| Contributions | ordered by StreamSeq |
| Frontier | leaf op-ids = observed − referenced-as-parent |
| Malformed | set with a reason if the first op is not a baseline (FR-015) |

**Lifecycle derivation** (FR-019): `proposed` = only the baseline seen; `active` = ≥1 content op
after the baseline; `closed` = a `life.transition{to:"closed"}` present. `dormant`/`archived` out
of scope.

**Frontier** (FR-018, clarified): the set of observed op-ids minus every op-id that appears in some
observed op's `Parents`. At birth = `[baseline-op-id]`.

## Entity: BoardEntry / DiscoveryBoard

- **BoardEntry**: `Path`, `Announcement`, `Parent`, `ParentKnown bool`, and `Lifecycle` where
  derivable.
- **DiscoveryBoard**: built by replaying `SOULSTREAM.TOPICS.INFO.>` and keeping the latest
  announcement per info subject (by stream sequence). One entry per topic; empty realm → empty
  board (FR-027). A sub-topic whose parent path is absent is flagged `ParentKnown=false`, not
  dropped (FR-026).

## Lifecycle state machine (derived, not stored)

```
proposed --(first content op)--> active --(life.transition closed)--> closed
   |________________(life.transition closed)_______________________________^
```

Transitions are derivations over the log, not commands; concurrent identical transitions converge
(idempotent). A transition to an undefined state is rejected by the poster (FR-021).
