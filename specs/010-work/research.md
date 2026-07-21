# Research & Decisions: 010-work

Phase 0 output. No NEEDS CLARIFICATION markers remained after `/speckit-clarify`;
this file records the design decisions and the alternatives they beat.

## D1 — Artefacts: derived projection vs. fold state

**Decision**: Artefacts are computed on demand by a pure function over
`MaterializedTopic.Attachments` (`mt.Artefacts()`); nothing is added to the fold,
the view's persisted fields, or the baked payload for stage 1.

**Rationale**: The attachments slice already carries everything a lineage needs
(op-ID, anchor, author, ts, name, digest, size, content type), in stream order —
and baked attachments keep their op-IDs and order across compaction (007), so
derivation survives rollup with no new code paths. Storing artefacts as a second
view field would duplicate every attachment in every serialised view and create a
consistency invariant to maintain. Smallest viable implementation wins.

**Alternatives considered**:
- *`Artefacts` field folded in `apply` and baked*: rejected — duplicated state,
  larger baselines, a new invariant (attachments ↔ artefacts agreement), zero
  additional capability.
- *A new `artefact.revise` op type*: rejected — the design doc is explicit that
  stage 1 is "no new machinery; existing ops".

## D2 — Tip rule: stream order, via slice position

**Decision**: The tip is the lineage member latest in stream order, implemented as
"highest index in `mt.Attachments`". Revisions may anchor to any lineage member;
connectivity determines membership, position determines the tip.

**Rationale**: `apply` appends attachments in traversal order, and the baked
prefix precedes the tail in original order — slice order *is* stream order, even
though baked entries have `StreamSeq == 0`. Comparing StreamSeq directly would
break precisely for baked entries; index comparison is correct everywhere and
free. This is the same total order that decides claims — one rule, both stages
(spec assumption).

**Alternatives considered**:
- *Compare `StreamSeq`*: rejected — zeroed on baked entries by design (007).
- *Tip = deepest chain member (graph depth)*: rejected — concurrent revisions of
  the same predecessor would tie and need a second rule; stream order is already
  total and already the house rule for races.

## D3 — Anchor-to-attachment means revision, unconditionally

**Decision**: An `attachment.add` whose anchor resolves to another attachment op in
the topic extends that attachment's lineage. No flag, no second meaning. An anchor
that points at a non-attachment op (or dangles) leaves the attachment a standalone
root, exactly as today.

**Rationale**: Encoded in the spec Clarifications. The design doc's sentence
("anchored to its predecessor's op-ID") is the entire mechanism; a "relates to an
attachment" nuance would demand a payload marker — new machinery stage 1 forbids.
Contextual relation is expressed by anchoring to the conversation instead.

## D4 — Work-item state machine: the fold is the arbiter

**Decision**: Fold cases process `work.*` ops in stream order against a
`map[opID]*WorkItem` (plus ordered slice for stable output). Transitions:
open→claimed (claim), claimed→done (done), open→done (done), claimed→open
(abandon). Everything else that is *readable* is a void `WorkEvent` appended to
the item timeline; a readable op referencing an unknown item becomes a `Warnings`
entry (there is no timeline to attach it to). Unreadable payloads / missing anchor
/ empty title on open → `Warnings` + skip (malformed).

**Rationale**: First-claim-wins needs no dedicated code: in-order traversal
guarantees the first valid claim flips the item before any later claim is
examined. Author-agnostic validity mirrors `life.transition` (no authorization
machinery — attribution + social correction, per spec assumption). Void events are
kept *on the item* so "the full trail is derivable from the log" survives baking.

**Alternatives considered**:
- *Owner-only done/abandon*: rejected — introduces authorization semantics the
  substrate deliberately lacks; a rogue done is attributable, like a rogue close.
- *Dropping void ops entirely*: rejected — FR-011 requires the ownership trail
  including voids; and two racing claimants must be able to *see* the race's
  outcome in the item's timeline.
- *Claim timeout*: deferred to `dormant` automation (roadmap item 7), recorded in
  spec Assumptions.

## D5 — Baking work items

**Decision**: `BakedState` gains `WorkItems []WorkItem` (json `work_items`,
omitempty). `cleanBakedWorkItems` strips StreamSeq/Sig on items *and* their
timeline events (Dangling has no meaning here). Seeding marks item IDs and
timeline event op-IDs as seen + referenced, so (a) frontier never resurrects them,
(b) later `work.*` ops and comment anchors resolve against baked items, and
(c) evidence comments anchored to a baked item are not flagged dangling.

**Rationale**: Identical treatment to baked contributions/attachments (007
conventions); FR-014. Old baselines simply lack the field → nil slice → no items,
no error — backward compatible by construction.

## D6 — Item references reuse the anchor convention

**Decision**: `work.claim/done/abandon` payloads carry
`{"anchor": {"kind": "op", "op_id": "<work.open op-id>"}}` — the same `Anchor`
struct comments and attachments use.

**Rationale**: Readers already know the shape; per the Clarifications this also
gives the malformed/void boundary a crisp definition (missing anchor = malformed;
resolvable-but-losing = void). A bespoke `item` field would be a second way to
spell the same thing.

## D7 — Mentions in work.open

**Decision**: `Handle.OpenWork` parses `@name` tokens in the body (same helper as
turns/comments), fills `mentions`, and fires `mention.notify` per mention.

**Rationale**: FR-013 — "tasks are conversations"; reusing the existing helper is
one call. Title is not scanned (titles are labels; bodies are prose — matches how
turn bodies are the scanned surface today).

## D8 — Curator and lifecycle interplay

**Decision**: Work ops count as content ops in the fold (a proposed topic with a
`work.open` becomes active) — one `case` addition to the existing content counter.
For the curator, verify `lastReal` derives from a surface that includes work ops;
if it scans typed view fields (contributions/attachments) rather than raw ops, add
work items/events to that scan, with a test. Lifecycle transitions never touch
item state (spec FR-019); nothing to implement for that half — absence of code is
the feature, guarded by a test.

## D9 — Client surface shape

**Decision**:
- CLI: `work open|claim|done|abandon|list|show` (subcommand pattern like `key`/
  `profile`), `artefacts <topic> [ref]` (list / history), `revise <topic> <file>
  --of <ref>`, and `get <topic> --artefact <ref> [--revision <op-id>]`. Artefact
  refs accept a root op-ID or a display name; ambiguous names error with the
  candidate roots listed (spec: names are labels, not keys).
- MCP: 7 new tools — `soulstream_open_work`, `soulstream_claim_work` (returns the
  post-claim verdict), `soulstream_complete_work`, `soulstream_abandon_work`,
  `soulstream_revise_text`, `soulstream_list_artefacts`,
  `soulstream_read_artefact` (UTF-8 only; binary → error naming the CLI). Work
  items and their timelines ride along in `soulstream_show_topic` for free once
  the view carries them.

**Rationale**: Follows 004/005 conventions exactly; the claim-verdict UX (publish,
materialise, report) is the smallest honest answer to "did I get it?" without any
new mechanism.

## D10 — No new NATS features

**Decision**: Nothing beyond what 001–009 already use. Claims deliberately do
*not* use `Nats-Expected-Last-Subject-Sequence`: a claim must land in the log even
when it loses (void-by-projection is the spec), whereas the expected-sequence
guard would reject the publish outright and erase the trail.

**Rationale**: The one place optimistic concurrency was considered, the spec
explicitly wants the lost race *recorded*, not prevented. Rollup keeps its guard;
claims keep the log.
