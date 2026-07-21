# Data Model: Re-baselining (Rollup), Manifest Baselines & Archived

## BaselinePayload (extended, additive)

| Field | Type | Rules |
|---|---|---|
| state | raw JSON | the opaque workbench artefact; unchanged meaning. Present in inline form; absent in manifest form. |
| frontier | []op-id | leaf op-ids at baseline time. Empty at birth (the baseline op-id is then the frontier); non-empty after any rollup. |
| baked | *BakedState | the conversation folded in at rollup. Absent at birth and in manifest form (it lives inside the manifest object). |
| manifest | *ManifestRef | set instead of state/baked when the state document exceeds the inline threshold. Exactly one of {state(+baked), manifest} is present. |

## BakedState

| Field | Type | Rules |
|---|---|---|
| contributions | []Contribution | in original stream order; op-ids preserved; `stream_seq` omitted (meaningless post-compaction) |
| attachments | []Attachment | same rules |
| lifecycle | Lifecycle | the folded lifecycle at compaction (proposed/active/closed/archived) |

Derived at read time, never stored: dangling flags, sig statuses (baked elements
inherit the baseline op's status), active-vs-proposed from content presence.

## ManifestRef

| Field | Type | Rules |
|---|---|---|
| chunks | []string | object-store names, fetch-and-concatenate order. This cycle always length 1: `baseline/<topic-path>/<baseline-op-id>` |
| digest | string | over the full state document (existing attachment digest format) |
| size | uint64 | total bytes of the state document |

The manifest object's content is the JSON state document `{state, baked}` — the same
bytes that would have been inline.

## State document forms

```
inline:   payload = { state, frontier, baked }          (≤ 128 KB state document)
manifest: payload = { frontier, manifest }              (state document in the store)
birth:    payload = { state, frontier: [] }             (exactly as today)
```

## Lifecycle (extended)

| State | New? | Rules |
|---|---|---|
| proposed / active / closed | no | unchanged, including closed's warn-but-allow |
| archived | **yes** | terminal: fold ignores any transition after archived; every Handle write path (Post, PostTurn, AddComment, Attach, Transition, Rollup) refuses when last-observed lifecycle is archived |

Transitions: `Close()` = transition(closed) + one rollup attempt (lost race
tolerated). `Archive()` = transition(archived) + rollup with bounded retry (3); on
exhaustion the transition stands, the error is loud, the topic remains readable.

## Fold (apply) changes

```
seed:
  payload.baked → view contributions/attachments/lifecycle
  baked interior op-ids → seen ∪ referenced      (anchor-resolvable, never frontier)
  payload.frontier ids  → seen                   (frontier candidates)
  payload.frontier empty → baseline op-id is the sole frontier candidate (as today)
fold tail: unchanged
terminal rule: lifecycle == archived ⇒ later transitions ignored (warning)
frontier: seen − referenced (unchanged formula)
```

**Round-trip invariant** (the heart of SC-001): for any log L,
`apply(rollup(L) + tail)` ≡ `apply(L + tail)` in every field except `stream_seq` and
per-op `sig`.

## Rollup attempt

| Property | Value |
|---|---|
| Guard | expected-last-subject-sequence = StreamSeq of last consumed op |
| Commit | the baseline publish itself (rollup header purges predecessors atomically) |
| Lost race | `ErrRollupLost` — nothing changed, retryable |
| Empty tail | `ErrNothingToCompact` — log is already just a baseline |
| Malformed topic | refused with the malformation reason |
| Cleanup | after commit: delete the *superseded* baseline's manifest objects, if any |

## Errors (new, topic package)

- `ErrRollupLost` — a concurrent write invalidated the attempt; retry if you still care.
- `ErrNothingToCompact` — no tail; publishing nothing.
- `ErrTopicArchived` — write refused: archived is terminal.
