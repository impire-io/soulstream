# Data Model: 010-work

## New op types (topic/vocab.go)

| Constant | Wire value | Payload | Parents | Notes |
|---|---|---|---|---|
| `TypeWorkOpen` | `work.open` | `WorkOpenPayload` | frontier | creates an item; op-ID = item ID |
| `TypeWorkClaim` | `work.claim` | `WorkRefPayload` | frontier | first-in-stream-order wins |
| `TypeWorkDone` | `work.done` | `WorkRefPayload` | frontier | terminal |
| `TypeWorkAbandon` | `work.abandon` | `WorkRefPayload` | frontier | claimed → open |

All four are ordinary ops on `SOULSTREAM.TOPICS.OPS.<path>`: signed at
`publishOp`, verified with the topic-path binding, carried in headers exactly like
existing vocabulary. No wire-format change.

## Payloads (topic/work.go)

```go
// WorkOpenPayload is the body of a work.open op.
type WorkOpenPayload struct {
    Title    string   `json:"title"`              // required; empty ⇒ malformed
    Body     string   `json:"body,omitempty"`     // optional prose; @mentions parsed
    Mentions []string `json:"mentions,omitempty"` // filled by the library, like turns
}

// WorkRefPayload is the body of work.claim / work.done / work.abandon.
type WorkRefPayload struct {
    Anchor *Anchor `json:"anchor"` // {kind:"op", op_id:<work.open op-id>}; nil/empty ⇒ malformed
}
```

`Anchor` is the existing struct (kind `"op"` + op-ID) used by comments and
attachments — reused, not redefined.

## View additions (topic/view.go)

```go
// WorkStatus is a work item's derived state.
type WorkStatus string

const (
    WorkOpen    WorkStatus = "open"
    WorkClaimed WorkStatus = "claimed"
    WorkDone    WorkStatus = "done"
)

// WorkEvent is one op that touched an item — including ops that lost.
type WorkEvent struct {
    OpID      string    `json:"op_id"`
    Kind      string    `json:"kind"` // "claim" | "done" | "abandon"
    Author    string    `json:"author"`
    Ts        time.Time `json:"ts"`
    Void      bool      `json:"void,omitempty"` // true = state machine rejected it
    StreamSeq uint64    // volatile; 0 on baked — MIRROR Attachment's exact tag
    Sig       SigStatus // volatile; recomputed at read — MIRROR Attachment's tag
}

// WorkItem is a task derived from the log.
type WorkItem struct {
    ID        string      `json:"id"`     // work.open op-ID
    Title     string      `json:"title"`
    Body      string      `json:"body,omitempty"`
    Mentions  []string    `json:"mentions,omitempty"`
    Author    string      `json:"author"` // opener
    Ts        time.Time   `json:"ts"`     // opened at
    Status    WorkStatus  `json:"status"`
    Owner     string      `json:"owner,omitempty"` // author of the winning claim
    Timeline  []WorkEvent `json:"timeline,omitempty"`
    StreamSeq uint64      // volatile — MIRROR Attachment's exact tag
    Sig       SigStatus   // volatile — MIRROR Attachment's tag
}

*Field-tag note (analysis I1)*: the volatile fields (`StreamSeq`, `Sig`) MUST
carry exactly the same json tags as `Attachment`/`Contribution` use today, so
`cleanBaked*` zeroing and round-trip equality work identically across all baked
element kinds. The tags shown elsewhere in this file are indicative.
```

`MaterializedTopic` gains `WorkItems []WorkItem` (json `work_items,omitempty`),
appended in opening order. Items and events get sig annotation like contributions
(baked entries — `StreamSeq == 0` — inherit the baseline's status).

## State machine (fold cases in `apply`)

Processed in stream order; the traversal order is the arbiter.

| Current state | Op | Result | Timeline entry |
|---|---|---|---|
| — | `work.open` (title non-empty) | new item, `open` | — |
| — | `work.open` (title empty / unreadable) | skipped | warning |
| open | `work.claim` | `claimed`, owner = author | claim |
| open | `work.done` | `done` (no owner) | done |
| open | `work.abandon` | unchanged | abandon, **void** |
| claimed | `work.claim` | unchanged | claim, **void** |
| claimed | `work.done` | `done` | done |
| claimed | `work.abandon` | `open`, owner cleared | abandon |
| done | any `work.*` ref | unchanged | kind, **void** |
| any | ref with unreadable payload / missing anchor | skipped | warning (malformed) |
| any | ref to unknown item ID | skipped | warning ("void — unknown item") |

Additional fold effects:
- Every readable `work.*` op counts as a content op (proposed → active).
- Work op IDs participate in seen/referenced bookkeeping (frontier correctness);
  comments/attachments anchored to an item ID resolve as non-dangling.
- Lifecycle transitions never modify items (no code path exists; test guards it).

## Baking (topic/rollup.go)

`BakedState` gains:

```go
WorkItems []WorkItem `json:"work_items,omitempty"`
```

- `cleanBakedWorkItems`: copy items, zero `StreamSeq`, clear `Sig` — on items and
  every timeline event (void flags are *kept*: they are history, not volatility).
- Seeding (`apply` baseline case): baked item IDs and their timeline event op-IDs
  → `seen` + `referenced`; items appended to the view before tail ops fold.
- Old baselines: absent field → nil → no items seeded. Pre-010 readers of new
  baselines: unknown JSON field ignored. Both directions backward compatible.

## Artefact derivation (topic/artefact.go — no persisted state)

```go
// Artefact is a lineage of whole-file revisions, derived from the view.
type Artefact struct {
    Root      string       `json:"root"` // root revision op-ID (identity)
    Name      string       `json:"name"` // tip's display name
    Tip       Attachment   `json:"tip"`
    Revisions []Attachment `json:"revisions"` // stream order, root first
}
```

Derivation (`mt.Artefacts()`), pure over `mt.Attachments`:
1. Index attachments by op-ID.
2. Parent(a) = a.Anchor if it resolves to another attachment's op-ID, else none.
3. Lineage = connected chains to a root (no parent). Identity = root op-ID.
4. Revisions sorted by slice position (= stream order, baked-safe); tip = last.
5. Artefacts returned ordered by root position; Name = tip.Name.

Resolver `FindArtefact(mt, ref)`: ref matches a root op-ID, any member op-ID, or
a display name; a name matching more than one lineage returns
`ErrAmbiguousArtefact` listing candidate roots. `GetRevision` = existing
`GetAttachment` + `VerifyDigest` on a chosen member.

## Entity relationships

```text
Topic 1—n WorkItem (by opening op)      Topic 1—n Artefact (derived)
WorkItem 1—n WorkEvent (timeline)       Artefact 1—n Revision (= Attachment)
WorkItem 1—0..1 Owner (persona slug)    Revision 0..1—1 predecessor (anchor)
WorkItem ←anchor— Comment/Attachment (evidence, existing mechanics)
```

## Invariants

1. Same log ⇒ same items, owners, void sets, artefacts, tips — on every replica,
   warm or cold, before or after compaction (SC-001/002/003).
2. `apply(rollup(L)) ≡ apply(L)` and `apply(rollup(L)+tail) ≡ apply(L+tail)`
   extended to work items and artefacts (existing round-trip tests grow cases).
3. A view containing no `work.*` ops and no attachment-anchored attachments is
   byte-identical to a pre-010 view of the same log (FR-018; `omitempty` fields).
4. Done is terminal; owner is non-empty iff status is claimed.
5. Artefact tips: every lineage has exactly one tip; every attachment belongs to
   exactly one lineage.
