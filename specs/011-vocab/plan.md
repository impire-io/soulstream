# Implementation Plan: Remaining Vocabulary — Edit, Replies, Resolve, Removal, Dormant

**Branch**: `011-vocab` | **Date**: 2026-07-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/011-vocab/spec.md`

## Summary

Five additive vocabulary words over the existing op-log, plus two opt-in sweeps:

1. **`comment.reply`** — folds exactly like `comment.add` under its own type
   string (one new switch case; threading is anchor + type, no new structure).
2. **`comment.resolve`** — annotates the anchored comment/reply in place:
   `Resolved`, `ResolvedBy` on `Contribution`. The op is not a list entry; its
   effect bakes (like lifecycle), duplicates are silent no-ops.
3. **`edit`** — same-author supersession for turns/comments/replies. The fold
   rewrites the target's rendered `Body`/`Mentions` and appends an **edit stamp**
   (op-id, author, ts + volatile sig/seq) to `Contribution.Edits`. A chain map
   (edit-op-id → target index) lets later edits anchor any prior chain member;
   stamps bake and re-seed that map, so chains survive rollup (the 010 lesson
   about compacted op-ids, applied on day one). Non-author/unknown/empty edits →
   `Warnings`, no effect.
4. **`attachment.remove`** — annotates the attachment in place: `Removed`,
   `RemovedBy`. `Artefacts()` picks the newest *non-removed* member as tip and
   drops fully-removed lineages; `Archive` deletes removed blobs after the final
   compaction lands (best-effort, beside the existing superseded-manifest
   cleanup).
5. **`dormant`** — new `Lifecycle` constant; `Transition` stops rejecting it; the
   fold accepts proposed/active→dormant (warns from closed/archived) and flips
   dormant→active on any content op *in the loop* (order matters, the end-of-fold
   proposed→active counter is unchanged). Pure rules `DormantEligible(mt, window,
   now)` and `StaleClaims(mt, window, now)`; CLI `mark-dormant`; curator opt-in
   passes `--mark-dormant` and `--reclaim <window>` (posting ordinary transitions
   / `work.abandon` — zero new fold rules for reclaim, 010's author-agnostic
   abandon is the whole mechanism).

Surfaces: `Handle.Reply/Resolve/Edit/RemoveAttachment/MarkDormant`; CLI `reply`,
`resolve`, `edit`, `detach`, `mark-dormant`, `curate --mark-dormant --reclaim`;
MCP +3 tools (reply/resolve/edit → 21). No new subjects, deps, or server features.

## Technical Context

**Language/Version**: Go 1.26, module `github.com/impire-io/soulstream` (unchanged)
**Primary Dependencies**: existing only — nats.go v1.52 + jetstream, orbit
natscontext, jcs, uuid, go-sdk v1.6.1
**Storage**: JetStream only; no new buckets/subjects; blob GC touches the
existing `soulstream-objects` bucket at archival
**Testing**: server-free fold/rule tests + embedded `natstest` integration; gate
`make check`
**Performance/Scale**: fold stays O(ops); sweeps are O(topics) per scan tick —
dogfood scale
**Constraints**: additive-only (FR-016); pure rules take `now` as a parameter —
no clock reads inside the fold; 009 curator behaviour unchanged unless flags set
**Minimum NATS server version**: nothing new (2.12+ per constitution)

## Constitution Check

- **I. NATS-Native First** — PASS. Vocabulary over the existing stream;
  idempotent transitions + void-by-projection replace any coordination; blob GC
  uses the existing object store API. Nothing new beside NATS.
- **II. Smallest Viable Implementation** — PASS. Reply is one switch case;
  resolve/remove are two booleans + attribution; edit's only real machinery (the
  stamp list) exists because compaction-safe chains are an acceptance scenario,
  not speculation. No un-resolve, no un-remove, no retraction, no edit-of-work
  items — none demanded by a scenario. Sweeps are opt-in flags on the existing
  curator scan, not a new process. Claim timeout adds zero fold rules.
- **III. ELI5 Documentation** — PASS (planned): new `docs/editing.md` (pencil
  edits & margin notes: crossed-out, never torn out); updates to `lifecycle.md`
  (dormant = napping), `attachments.md` (withdrawing a file), `work-items.md`
  (lapsed magnets), `curator.md` (the two opt-in sweeps), `cli.md`, `mcp.md`,
  `docs/README.md`, root `README.md`.

*Post-design re-check: PASS — the only persisted additions are fields on existing
baked carriers (`Resolved`, `ResolvedBy`, `Edits`, `Removed`, `RemovedBy`), each
demanded by FR-013's round-trip requirement.*

## Project Structure

### Documentation (this feature)

```text
specs/011-vocab/
├── spec.md              # done (clarifications encoded)
├── plan.md              # this file
├── research.md          # decisions + rejected alternatives
├── data-model.md        # payloads, fold rules, baked fields, pure rules
├── quickstart.md        # upkeep walkthrough (edit/reply/resolve/detach/dormant)
├── contracts/library.md # exported surface: Go, CLI, MCP
├── checklists/requirements.md
└── tasks.md             # (/speckit-tasks)
```

### Source Code (repository root)

```text
topic/
├── vocab.go             # + TypeCommentReply/TypeCommentResolve/TypeEdit/
│                        #   TypeAttachmentRemove; Dormant const; payload structs
│                        #   (EditPayload = CommentPayload shape; RefPayload{Anchor})
├── view.go              # Contribution.{Resolved,ResolvedBy,Edits};
│                        #   Attachment.{Removed,RemovedBy}; EditStamp; fold cases;
│                        #   dormant transition + content-op reactivation;
│                        #   edit-chain map incl. baked-stamp seeding
├── lifecycle.go         # definedLifecycle += Dormant; Handle.MarkDormant;
│                        #   Archive: delete removed blobs post-compaction
├── upkeep.go            # NEW: Handle.Reply/Resolve/Edit/RemoveAttachment;
│                        #   DormantEligible, StaleClaims (pure)
├── artefact.go          # tip skips removed; fully-removed lineage dropped
├── rollup.go            # cleanBaked*: strip volatile inside Edits; keep marks
├── verify.go            # annotate edit stamps' sig statuses
├── upkeep_fold_test.go  # NEW: server-free fold tests (edit/reply/resolve/
│                        #   remove/dormant + round-trips via fullLog growth)
├── upkeep_rules_test.go # NEW: DormantEligible/StaleClaims pure tests
├── upkeep_integration_test.go  # NEW: end-to-end incl. archival GC, signed marks
└── (rollup_fold_test.go, artefact_test.go extended)

curator/
├── curator.go           # Options.{MarkDormant bool, ReclaimAfter time.Duration};
│                        #   dormantPass + reclaimPass behind the flags
├── projection.go        # cachedTopic.lastAny (newest op of any kind)
└── curator_test.go / projection_test.go extensions

internal/cli/
├── cli.go               # dispatch + usage: reply/resolve/edit/detach/mark-dormant
├── upkeep_commands.go   # NEW: the five commands
├── curate_cmd.go        # --mark-dormant / --reclaim flags
└── upkeep_commands_test.go  # NEW

internal/mcpserver/
├── server.go            # register 3 tools (21 total)
├── upkeep_tools.go      # NEW: reply_comment / resolve_comment / edit
└── upkeep_tools_test.go # NEW

docs/  editing.md (NEW) + lifecycle.md, attachments.md, work-items.md,
       curator.md, cli.md, mcp.md, README.md updates; root README.md
```

**Structure Decision**: all in existing packages; conversation-upkeep write
helpers and the two pure rules share `topic/upkeep.go` (one home for "the
housekeeping words"), fold stays in `view.go` beside the switch it extends.

## Key design decisions (detail in research.md)

1. **Marks over entries**: resolve/remove annotate their targets; the ops vanish
   at compaction, their effects bake — the lifecycle-transition precedent, chosen
   over list entries because nothing anchors to a resolve/remove in any scenario.
2. **Edit stamps bake; nothing else about edits does**: rendered body lives in
   the existing `Body` field (no original-body archaeology in the view — raw log
   history until compaction, same as everything), and stamps exist solely so
   post-rollup edits can join compacted chains + display "edited by".
3. **Same-author edit is a projection rule, not authorization**: decided from log
   data (`author` field), deterministic on every replica; rejected alternative
   (anyone-edits) breaks attribution that signing exists to protect.
4. **Sweeps live in the curator, off by default**: it already has the
   projection, the scan tick, and the idle window; a flag keeps 009's
   "comments-only" contract intact for existing operators. Manual `mark-dormant`
   covers curator-less realms; both sweeps are races-welcome idempotent.
5. **Reclaim = rule + ordinary abandon**: 010 already made author-agnostic
   abandon reopen items and fold rival paths void; the sweep is a client of that
   machinery, not new machinery.

## Complexity Tracking

No constitution violations to justify.
