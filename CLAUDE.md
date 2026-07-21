<!-- SPECKIT START -->
Active feature: **010-work** — work stages 1–2 (Day-2 #5, extensions/work.md).
Stage 1, versioned artefacts: ZERO new op types — a revision is `attachment.add` anchored to a
prior attachment op (anchor→attachment = revision, unconditionally); artefacts DERIVED on demand
(`mt.Artefacts()`, pure over `mt.Attachments`; lineage = anchor connectivity, identity = root
op-id, tip = highest slice index — slice order IS stream order, baked-safe; never compare
StreamSeq, it's 0 on baked). Stage 2, work items: 4 new op types `work.open/claim/done/abandon`
folded into NEW `MaterializedTopic.WorkItems`; the in-order fold IS the arbiter (first claim in
stream order wins); state machine open→claimed→done, claimed→open on abandon, done terminal,
author-agnostic (like life.transition). Malformed (unreadable/missing anchor/empty title) =
warn+skip ≠ void (readable but loses) = WorkEvent{Void:true} on the item timeline; unknown item
ref = warning. Item refs reuse `Anchor{kind:"op"}`. Bake `BakedState.WorkItems` (strip
StreamSeq/Sig recursively, KEEP void flags; seed ids as seen+referenced). Work ops = content ops
(activate proposed); curator lastReal must count them. Claims do NOT use expected-sequence
(lost claim must land in the log). Handle: OpenWork (mentions parsed) / ClaimWork /
CompleteWork / AbandonWork / Revise. CLI: `work open|claim|done|abandon|list|show`,
`artefacts [ref]`, `revise --of`, `get --artefact [--revision]`; claim verdict = publish then
materialise ("claimed" / "void — owned by X"). MCP: 7 new tools (18 total), read_artefact
UTF-8-only. ELI5 docs: artefacts.md + work-items.md, same change.

For details read: [specs/010-work/plan.md](specs/010-work/plan.md)
(spec: `specs/010-work/spec.md`, contract: `specs/010-work/contracts/library.md`,
model: `specs/010-work/data-model.md`).
Done: `001`–`005` (MVP), `006-signing`, `007-rollup`, `008-discover`, `009-curator` merged + pushed.

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` import NO NATS; `realm`, `topic`, `registry`, `curator` are NATS-touching.
- Keep pure logic (folds, chain validation, discovery match/merge, curator judgment) separate
  from NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; modern `nats.go/jetstream` API.
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
- Push to origin after merging a completed feature to main.
<!-- SPECKIT END -->
