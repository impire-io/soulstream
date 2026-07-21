# Research & Decisions: 011-vocab

## D1 — Resolve and remove are marks, not entries

**Decision**: `comment.resolve` and `attachment.remove` fold as in-place
annotations (`Resolved`/`ResolvedBy` on the contribution, `Removed`/`RemovedBy`
on the attachment). The ops themselves are not view entries and their op-ids are
not preserved across compaction.

**Rationale**: The design's own precedent is `life.transition`: an op whose
*effect* is state, whose identity nothing anchors to. No scenario replies to a
resolve or comments on a remove. Baking the marks (they ride the carriers that
already bake) satisfies round-trip equality with zero new baked collections.

**Alternatives**: fold them as timeline-style entries (010's `WorkEvent`
pattern) — rejected: work events exist because claim *races* need a visible
trail; resolve/remove have no race semantics worth trailing (duplicates are
no-ops), and the entry would be dead weight in every serialised view.

## D2 — Edit chains: stamps on the target

**Decision**: An applied edit appends `EditStamp{OpID, Author, Ts}` (+ volatile
Sig/StreamSeq) to the target's `Edits`; the fold keeps `editTarget[opID] →
contribution index` covering the target and every stamp, and re-seeds it from
baked stamps. Rendered `Body`/`Mentions` are overwritten by each applied edit
(stream order ⇒ last applied wins — "render the latest in the chain" for free).

**Rationale**: The one hard requirement is FR-003: an edit anchoring a chain
member that a rollup already compacted must still resolve. 010 hit the same
class of problem with work-event op-ids and solved it by baking them; stamps are
that solution sized for edits. Overwriting `Body` in place keeps every existing
consumer (curator search, CLI render, MCP JSON) correct with no changes.

**Alternatives**: (a) derive chains like artefacts — impossible once edits
compact (they are not standalone baked elements the way attachments are);
(b) keep original body + `EditedBody` — two sources of truth for every reader,
and the "original" dies at compaction anyway; (c) edits as separate
contributions with a render rule — pushes chain logic into every client.

## D3 — Same-author rule for edit

**Decision**: An edit takes effect only when `edit.author == target.author`;
otherwise a warning. Author-agnostic everywhere else stands (resolve, remove,
work ops, lifecycle).

**Rationale**: Rendered words carry the original author's name and (when signed)
their signature's authority; letting another persona swap the words under that
name is a misattribution engine — the exact thing 006 exists to prevent. The
rule is pure projection (the log carries both authors); no authorization
machinery appears. Disagreement has a word already: `comment.reply`.

**Alternatives**: anyone-edits (artefact symmetry) — rejected: a document
revision is *new work attributed to the reviser*; an edit renders under the
*original* author's name. The asymmetry is the point.

## D4 — Empty-body edits are malformed

**Decision**: `edit` with an empty body = malformed (warn + skip), like an
empty-titled `work.open`. Retraction of prose is out of scope.

**Rationale**: "Render the latest in the chain" with an empty latest would
silently blank testimony — a delete in edit's clothing. The design has no
delete-for-prose, deliberately.

## D5 — Dormant in the fold

**Decision**: `Dormant` joins the lifecycle constants; fold accepts
proposed/active/dormant→dormant (idempotent), ignores it with a warning from
closed/archived, and flips dormant→active inside the loop whenever a content op
folds (turn, comment, reply, edit-applied, attachment add/remove-applied, work
op). The end-of-fold `Proposed && contentOps>0 → Active` rule is untouched.
Baked dormant + tail content op → active by the same in-loop rule.

**Rationale**: Core's table verbatim: "posting a content op makes it active
again." In-loop handling is required because reactivation is order-sensitive
(content *after* the transition wakes it; content before it does not).

## D6 — Eligibility clocks are pure functions

**Decision**: `DormantEligible(mt, window, now)`: newest op timestamp of *any*
kind (contributions, attachments incl. removed marks' carriers, work items +
events, edit stamps, BaselineTs floor) older than `window`. `StaleClaims(mt,
window, now)`: for each claimed item, newest of {winning claim ts, later
timeline events, evidence anchored to the item} older than `window`. Both take
`now` explicitly; nothing inside the fold reads a clock.

**Rationale**: Determinism lives in the log; clocks live in the caller. Tests
pin `now`. The dormancy clock counts curator chatter (core's "last op" rule —
simplest, and self-limiting: a suggestion defers eligibility one window at
most); the curator's own *proposal* clock (009, suggestion-excluding) is a
different question and stays untouched.

## D7 — Sweeps: curator flags, off by default; manual command for the rule

**Decision**: `curator.Options.MarkDormant bool` and `ReclaimAfter
time.Duration` (zero = off) run two passes on the existing scan tick, posting
`life.transition{dormant}` / `work.abandon` as the curator persona. CLI:
`curate --mark-dormant --reclaim <dur>`, plus standalone `mark-dormant <path>
[--idle <dur>]` for curator-less realms.

**Rationale**: The curator already owns the projection, tick, and window —
adding a process would duplicate all three. Off-by-default honours 009's
"comments are its entire vocabulary" for unmodified deployments; the spec
amends that contract only behind explicit flags. Races converge by
construction: duplicate dormant marks are idempotent, the second abandon folds
void.

**Alternatives**: always-on (surprises 009 operators); a separate `sweep`
binary (new process, no new capability); read-time derived dormancy (projection
would depend on wall clock + config — non-deterministic across replicas,
rejected outright).

## D8 — Blob GC at archival

**Decision**: After `Archive`'s final compaction succeeds, delete the objects of
attachments marked removed in the final view — best-effort, logged as warnings
on failure, exactly like superseded-manifest cleanup (007). If compaction lost
its retries, GC waits for the finishing re-archive (the code path that already
exists for half-done archivals).

**Rationale**: The design line is precise: "the blob itself is deleted only at
topic archival, so replay never dangles within a topic's lifetime." GC before
the final baseline is durable would risk a readable log referencing deleted
bytes; after it, the removed marks are baked into the terminal baseline and the
delete is safe garbage collection, not state.

## D9 — MCP scope

**Decision**: Three tools — reply, resolve, edit. No remove, no mark-dormant, no
sweeps over MCP.

**Rationale**: Reply/resolve/edit are conversation verbs an agent needs daily.
Removal and dormancy are housekeeping with reclamation consequences —
operator-surface by the same reasoning that keeps archive off MCP (a stance the
docs already explain). Agents can still *see* removed/resolved/edited state via
`show_topic`.
