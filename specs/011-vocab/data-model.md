# Data Model: 011-vocab

## New op types (topic/vocab.go)

| Constant | Wire value | Payload | Fold effect |
|---|---|---|---|
| `TypeCommentReply` | `comment.reply` | `CommentPayload` (reused) | Contribution entry, threaded by anchor |
| `TypeCommentResolve` | `comment.resolve` | `RefPayload` | mark on target: Resolved/ResolvedBy |
| `TypeEdit` | `edit` | `CommentPayload` (reused: body+anchor+mentions) | rewrites target body/mentions + EditStamp |
| `TypeAttachmentRemove` | `attachment.remove` | `RefPayload` | mark on target: Removed/RemovedBy |

`RefPayload{ Anchor *Anchor \`json:"anchor"\` }` — the 010 `WorkRefPayload`
shape generalised (missing/empty anchor ⇒ malformed). All four are ordinary
signed ops on the topic's ops subject.

New lifecycle constant: `Dormant Lifecycle = "dormant"` — `definedLifecycle`
accepts it; `TransitionPayload` unchanged.

## View field additions (topic/view.go)

```go
// EditStamp records one applied edit — the chain that survives compaction.
type EditStamp struct {
    OpID      string    `json:"op_id"`
    Author    string    `json:"author"` // always the contribution's author (rule)
    Ts        time.Time `json:"ts"`
    Sig       SigStatus `json:"sig,omitempty"`        // volatile
    StreamSeq uint64    `json:"stream_seq,omitempty"` // volatile; 0 baked
}

Contribution += Resolved   bool        `json:"resolved,omitempty"`
                ResolvedBy string      `json:"resolved_by,omitempty"`
                Edits      []EditStamp `json:"edits,omitempty"`
Attachment   += Removed    bool        `json:"removed,omitempty"`
                RemovedBy  string      `json:"removed_by,omitempty"`
```

Rendered `Body`/`Mentions` of an edited contribution are the newest applied
edit's. `Type` stays the original (`turn.post` / `comment.add` /
`comment.reply`).

## Fold rules (topic/view.go `apply`)

Maintain `editTarget map[opID]int` (contribution index) covering every
turn/comment/reply op-id **and every edit-stamp op-id** (baked stamps seed it).

| Op | Valid when | Effect | Otherwise |
|---|---|---|---|
| `comment.reply` | payload readable | Contribution entry (content op); dangling flag as comments | malformed warning |
| `edit` | readable, body non-empty, anchor resolves via `editTarget`, author == target author | overwrite Body/Mentions, append stamp, register stamp op-id, content op | warning (`foreign edit`, `unknown target`, `malformed edit`), no effect |
| `comment.resolve` | readable, anchor resolves to a comment/reply contribution | set Resolved + ResolvedBy (first resolver wins the attribution; later resolves no-op silently), content op | warning (non-comment / unknown), no effect |
| `attachment.remove` | readable, anchor resolves to an attachment | set Removed + RemovedBy (first remover; later removes silent no-op), content op | warning, no effect |
| `life.transition{dormant}` | lifecycle ∉ {closed, archived} | Lifecycle = Dormant (idempotent) | warning, ignored |
| any content op while Dormant | — | Lifecycle = Active (in-loop, order-sensitive) | — |

Content ops for reactivation/activation counting: turn, comment, reply,
*applied* edit/resolve/remove, attachment.add, all readable work ops. Malformed
and no-effect ops count nothing.

## Baking (topic/rollup.go)

- No new baked collections: `Resolved`/`ResolvedBy`/`Edits` ride baked
  contributions; `Removed`/`RemovedBy` ride baked attachments; `Dormant` rides
  `BakedState.Lifecycle`.
- `cleanBakedContributions` additionally zeroes `Sig`/`StreamSeq` inside each
  `EditStamp` (deep copy of the slice); marks are kept — they are history.
- Baseline seeding registers baked edit-stamp op-ids into `editTarget` and
  seen/referenced (chain continuity + frontier hygiene).

## Pure rules (topic/upkeep.go)

```go
// DormantEligible: newest op of ANY kind older than window.
// Newest = max(BaselineTs, contribution ts + edit stamps, attachment ts,
//              work item ts + timeline events).
func DormantEligible(mt *MaterializedTopic, window time.Duration, now time.Time) bool
// Eligible only when Lifecycle ∈ {Proposed, Active} (dormant/closed/archived: false).

// StaleClaims: ids of claimed items whose newest related activity — the winning
// claim, any later timeline event, any contribution/attachment anchored to the
// item — is older than window.
func StaleClaims(mt *MaterializedTopic, window time.Duration, now time.Time) []string
```

## Write side

```go
func (h *Handle) Reply(ctx, body, anchorOpID string) (string, error)   // mentions parsed+notified
func (h *Handle) Edit(ctx, targetOpID, newBody string) (string, error) // mentions parsed+notified; empty body refused
func (h *Handle) Resolve(ctx, targetOpID string) (string, error)
func (h *Handle) RemoveAttachment(ctx, addOpID string) (string, error)
func (h *Handle) MarkDormant(ctx) (string, error)                      // = Transition(Dormant)
```

`Archive` addition: after the final compaction succeeds, best-effort delete of
every `Removed` attachment's object (from the final folded view), mirroring
superseded-manifest cleanup.

## Curator (curator/)

- `cachedTopic.lastAny time.Time` — newest op of any kind (built beside
  `lastReal`).
- `Options.MarkDormant bool`, `Options.ReclaimAfter time.Duration` (0 = off).
- `dormantMarkPass`: for eligible topics (rule over `lastAny` via
  `DormantEligible` semantics), open + `MarkDormant`; event line per mark.
- `reclaimPass`: `StaleClaims` per topic view; `AbandonWork` per stale item;
  event line per reclaim.

## Invariants

1. Same log ⇒ same rendered bodies, marks, lifecycle, warnings — every replica,
   warm/cold, before/after compaction (SC-001..003).
2. `apply(rollup(L)+tail) ≡ apply(L+tail)` extended: tails containing edits of
   compacted chain members, resolves/removes of baked targets, dormant
   transitions and reactivations.
3. A view with none of the new vocabulary serialises byte-identically to 010
   (`omitempty` on every new field) (FR-016/SC-004).
4. `Removed` ⇒ bytes fetchable until archival; after archival, exactly the
   non-removed blobs remain (SC-002).
5. Owner semantics from 010 unchanged; a sweep's abandon is indistinguishable in
   fold terms from a manual one.
