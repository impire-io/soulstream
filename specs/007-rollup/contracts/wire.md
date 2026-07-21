# Contract: Wire Shapes & Race Guard

## The rollup publish

```
Subject:  SOULSTREAM.TOPICS.OPS.<path>
Headers:  (ordinary record headers: Nats-Msg-Id, Soulstream-*, Soulstream-Sig when keyed)
          Nats-Rollup: sub
Publish:  expected-last-subject-sequence = StreamSeq(last consumed op)
Payload:  baseline payload (below)
```

- The publish is the atomic commit: the server purges every prior message on the
  subject and admits the baseline in one act. Rejected (wrong last sequence) ⇒ the
  log is untouched.
- The baseline record's `Soulstream-Parents` = the consumed frontier (DAG record);
  `payload.frontier` = the same ids (normative parenting rule for what follows).
- Canonical binding for the signature is unchanged (subject suffix = topic path).

## Baseline payload forms

**Birth** (unchanged, written by StartTopic):

```json
{ "state": { "...": "workbench artefact" }, "frontier": [] }
```

**Inline rollup** (state document ≤ 128 KB):

```json
{
  "state":    { "...": "workbench artefact, carried forward" },
  "frontier": ["<leaf-op-id>", "..."],
  "baked": {
    "contributions": [ { "op_id": "...", "author": "...", "ts": "...", "type": "turn.post",
                         "body": "...", "mentions": ["..."], "anchor": "...", "...": "..." } ],
    "attachments":   [ { "op_id": "...", "name": "...", "object": "...", "digest": "...",
                         "size": 123, "...": "..." } ],
    "lifecycle": "active"
  }
}
```

**Manifest rollup** (state document > 128 KB):

```json
{
  "frontier": ["<leaf-op-id>", "..."],
  "manifest": { "chunks": ["baseline/<path>/<baseline-op-id>"],
                "digest": "<digest over the full state document>",
                "size": 262144 }
}
```

The manifest object's bytes are exactly the inline form's `{state, baked}` document.
Write order: put object → publish manifest baseline (commit point, same guard) →
delete the superseded baseline's objects. Failure anywhere leaves either the old log
intact (plus one orphaned object) or the new baseline committed (plus at most an
undeleted superseded object). Orphans are garbage, never corruption.

## Fold rules a reader must honour

1. First message must be a `baseline` (else malformed, as today).
2. Resolve the state document: inline directly; manifest by fetch + digest check —
   any failure marks the topic malformed with a reason; never partial state.
3. Seed the view from `baked` (order-preserving); baked interior ids are
   anchor-resolvable and never frontier; `frontier` ids are the frontier candidates
   (the baseline op-id is the candidate only when `frontier` is empty — birth).
4. Fold the tail exactly as before.
5. Terminality: once lifecycle is `archived`, ignore later transitions (warn).
6. Verification: baked elements carry the baseline op's status; tail ops their own.

## Compatibility

- Unknown payload fields are ignored by pre-007 readers (additive JSON); pre-007
  baselines have no `baked`/`manifest` and fold identically under the new rules
  (empty `frontier` ⇒ baseline-id-as-frontier, as at birth).
- No new op types; `archived` is a new `life.transition` target value.
- Inbox, INFO, and notify subjects are untouched by ops-subject compaction.
