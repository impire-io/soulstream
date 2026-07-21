# Quickstart: 010-work — revise a document, claim a task

Two personas, one realm: `daan` (human, CLI) and `scribe` (AI, MCP — CLI shown
here for brevity; the MCP tools mirror every step).

## 0. Prerequisites

```sh
soulstream provision                      # realm artefacts exist (idempotent)
soulstream start design-notes --subject "design notes for the new gadget"
```

## 1. A document that remembers (stage 1)

```sh
# daan attaches the first draft
soulstream --persona daan attach design-notes ./notes.md
# → prints the object key; the revision's op-id shows up in `artefacts`

# scribe revises it whole-file — anchored to the predecessor automatically
soulstream --persona scribe revise design-notes ./notes-v2.md --of notes.md
# → revised notes.md (op 2222-..., supersedes 1111-...)

# anyone lists artefacts and history
soulstream artefacts design-notes
# → notes.md  root 1111-...  2 revisions  tip by scribe
soulstream artefacts design-notes notes.md
# → 1111-... daan   2026-07-21T10:00Z  notes.md
#   2222-... scribe 2026-07-21T10:05Z  notes.md  ← tip

# fetch the tip, or any old revision — nothing was overwritten
soulstream get design-notes --artefact notes.md -o current.md
soulstream get design-notes --artefact notes.md --revision 1111-... -o first.md
```

Concurrent revisions of the same tip? Both land, both stay in history, and every
reader derives the same winner: the one later in stream order.

## 2. Claiming work without a lock service (stage 2)

```sh
# daan opens a task
soulstream --persona daan work open design-notes "draft the intro section" \
  --body "@scribe want to take this?"
# → opened work item 3333-...   (scribe gets a mention notification)

# both personas try to claim it at nearly the same instant
soulstream --persona scribe work claim design-notes 3333-...   # lands first
# → claimed — you own it
soulstream --persona daan  work claim design-notes 3333-...    # lands second
# → void — owned by scribe (your claim is recorded, changes nothing)

soulstream work list design-notes
# → 3333-...  claimed  scribe  draft the intro section

# evidence is ordinary anchored ops
soulstream --persona scribe attach design-notes ./intro-draft.md --anchor 3333-...
soulstream --persona daan  comment design-notes "looks great" --anchor 3333-...

# scribe finishes
soulstream --persona scribe work done design-notes 3333-...
soulstream work show design-notes 3333-...
# → done — full timeline: open(daan) → claim(scribe) → claim(daan, VOID) → done(scribe)
#   evidence: intro-draft.md, 1 comment
```

## 3. Letting go

```sh
soulstream --persona daan work open design-notes "polish the diagrams"
soulstream --persona scribe work claim design-notes 4444-...
soulstream --persona scribe work abandon design-notes 4444-...   # reopens it
soulstream --persona daan  work claim design-notes 4444-...      # fresh race, daan wins
```

## 4. Nothing forgets across compaction

```sh
soulstream rollup design-notes
soulstream work list design-notes        # same items, same owners
soulstream artefacts design-notes        # same lineages, same tips
```

The baseline baked the items and the attachment lineage; a cold reader sees
exactly what a live follower saw.

## MCP mirror

`soulstream_open_work`, `soulstream_claim_work` (returns the verdict directly),
`soulstream_complete_work`, `soulstream_abandon_work`, `soulstream_revise_text`,
`soulstream_list_artefacts`, `soulstream_read_artefact` — plus the existing
`soulstream_show_topic`, whose output now includes `work_items`.
