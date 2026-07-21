<!-- SPECKIT START -->
Active feature: **011-vocab** — remaining vocabulary (Day-2 #7, core/03-topics.md).
Four new op types + one lifecycle state: `comment.reply` (folds like comment.add, own type),
`comment.resolve` (MARK on target: Resolved/ResolvedBy — resolve op is NOT a list entry,
vanishes at compaction like transitions; duplicate = silent no-op), `edit` (SAME-AUTHOR-ONLY
projection rule — foreign edit = warning, no effect; rewrites target Body/Mentions in place,
appends EditStamp{op-id,author,ts} to Contribution.Edits; editTarget map covers target + stamp
op-ids, baked stamps RE-SEED it so post-rollup edits of compacted chain members still resolve;
empty body = malformed), `attachment.remove` (mark Removed/RemovedBy, author-agnostic, bytes
fetchable until archival; Artefacts() tip = newest NON-removed, fully-removed lineage leaves the
list; Archive deletes removed blobs AFTER final compaction, best-effort), `Dormant` lifecycle
(Transition stops rejecting it; fold: proposed/active→dormant, closed/archived ignore+warn; ANY
content op while dormant → Active IN-LOOP — order-sensitive). Payloads: reply/edit reuse
CommentPayload; resolve/remove use new RefPayload{Anchor}. Pure rules in topic/upkeep.go:
DormantEligible(mt,window,now) (newest op of ANY kind, incl. curator chatter; only from
proposed/active) + StaleClaims(mt,window,now) (claim/timeline/anchored-evidence clock). Reclaim
= rule + ordinary author-agnostic work.abandon (ZERO fold changes). Curator: Options.MarkDormant
+ Options.ReclaimAfter (both OFF by default — 009 contract intact), passes on existing scan
tick; cachedTopic.lastAny beside lastReal. Handle: Reply/Edit/Resolve/RemoveAttachment/
MarkDormant. CLI: reply, edit, resolve, detach, mark-dormant, curate --mark-dormant --reclaim.
MCP: +3 tools (reply/resolve/edit → 21). ELI5: docs/editing.md new; lifecycle/attachments/
work-items/curator docs updated.

For details read: [specs/011-vocab/plan.md](specs/011-vocab/plan.md)
(spec: `specs/011-vocab/spec.md`, contract: `specs/011-vocab/contracts/library.md`,
model: `specs/011-vocab/data-model.md`).
Done: `001`–`005` (MVP), `006-signing`, `007-rollup`, `008-discover`, `009-curator`,
`010-work` merged + pushed.

Project conventions:
- Go 1.26; module `github.com/impire-io/soulstream`.
- `record` and `identity` import NO NATS; `realm`, `topic`, `registry`, `curator` are NATS-touching.
- Keep pure logic (folds, chain validation, discovery match/merge, curator judgment) separate
  from NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; modern `nats.go/jetstream` API.
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
- Push to origin after merging a completed feature to main.
<!-- SPECKIT END -->
