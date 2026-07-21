# Contract: 010-work — library, CLI, and MCP surfaces

The exported additions. Everything existing is unchanged (FR-018).

## Go library (`topic` package)

### Vocabulary (vocab.go)

```go
const (
    TypeWorkOpen    = "work.open"
    TypeWorkClaim   = "work.claim"
    TypeWorkDone    = "work.done"
    TypeWorkAbandon = "work.abandon"
)
```

### Write side (work.go, artefact.go)

```go
// OpenWork opens a work item. Parses @mentions in body, fires mention.notify.
// Returns the new item's ID (the op-ID). Title must be non-empty.
func (h *Handle) OpenWork(ctx context.Context, title, body string) (string, error)

// ClaimWork publishes a claim on the item. Winning is decided by the log, not
// the return: materialise afterwards (or use the CLI/MCP verdict flows).
func (h *Handle) ClaimWork(ctx context.Context, itemID string) (string, error)

// CompleteWork publishes work.done for the item.
func (h *Handle) CompleteWork(ctx context.Context, itemID string) (string, error)

// AbandonWork publishes work.abandon for the item.
func (h *Handle) AbandonWork(ctx context.Context, itemID string) (string, error)

// Revise attaches data as a new whole-file revision superseding predecessor
// (an attachment op-ID in this topic). Thin wrapper over Attach; predecessor
// must be non-empty.
func (h *Handle) Revise(ctx context.Context, name, contentType string, data []byte, predecessor string) (string, error)
```

All return the published op-ID. All refuse archived topics via the existing
`Post` path (`ErrTopicArchived`); all sign via the existing choke point.

### Read side (view.go, artefact.go)

```go
type WorkStatus string            // "open" | "claimed" | "done"
type WorkItem struct { ... }      // see data-model.md
type WorkEvent struct { ... }
// MaterializedTopic.WorkItems []WorkItem `json:"work_items,omitempty"`

type Artefact struct {            // derived, never persisted
    Root      string
    Name      string
    Tip       Attachment
    Revisions []Attachment
}

// Artefacts derives the topic's lineages from its attachments. Pure.
func (mt *MaterializedTopic) Artefacts() []Artefact

// FindArtefact resolves ref (root op-ID, member op-ID, or display name).
// A name matching several lineages returns ErrAmbiguousArtefact.
func FindArtefact(mt *MaterializedTopic, ref string) (Artefact, error)

var ErrAmbiguousArtefact = errors.New(...) // message lists candidate roots
```

Fetching bytes reuses `GetAttachment` + `VerifyDigest` on `Tip` or any member of
`Revisions` — no new fetch API.

### Baking (rollup.go)

`BakedState.WorkItems []WorkItem` (json `work_items,omitempty`). Rollup bakes
folded items with volatile fields stripped; the fold seeds them back. Baselines
without the field behave exactly as today.

## CLI (`soulstream`)

```text
work open <topic> <title> [--body <text>]        → prints item id
work claim <topic> <item-id>                     → publishes, materialises, prints
                                                   "claimed" or "void — owned by <p>"
work done <topic> <item-id>                      → marks done
work abandon <topic> <item-id>                   → reopens
work list <topic>                                → id, status, owner, title per item
work show <topic> <item-id>                      → item + timeline (incl. void) +
                                                   anchored evidence
artefacts <topic>                                → root, name, #revisions, tip info
artefacts <topic> <ref>                          → one artefact's revision history
revise <topic> <file> --of <ref> [--content-type <ct>]
                                                 → new revision superseding the tip
                                                   of artefact <ref>; prints op-id
get <topic> --artefact <ref> [--revision <op-id>] [-o <out>]
                                                 → fetch tip (or chosen revision)
                                                   bytes, digest-verified
```

`<ref>` = root op-ID, any revision op-ID, or display name (error on ambiguity).
Existing `get <topic> <object>` positional form unchanged. Exit codes follow the
house pattern (0 ok, 2 usage, 1 runtime).

## MCP (`soulstream-mcp`) — 7 new tools, total 18

| Tool | Input | Output |
|---|---|---|
| `soulstream_open_work` | topic, title, body? | item id |
| `soulstream_claim_work` | topic, item_id | verdict: claimed / void + current owner |
| `soulstream_complete_work` | topic, item_id | op id |
| `soulstream_abandon_work` | topic, item_id | op id |
| `soulstream_revise_text` | topic, ref, text, name?, content_type? | revision op id |
| `soulstream_list_artefacts` | topic | artefacts JSON (roots, names, revisions) |
| `soulstream_read_artefact` | topic, ref, revision? | UTF-8 content; error for binary (points at CLI) |

Work items (with timelines) already appear in `soulstream_show_topic` output via
the view's new `work_items` field — no separate list tool.

## Compatibility contract

- Pre-010 baselines parse unchanged (missing `work_items` ⇒ no items).
- Pre-010 readers of post-010 logs: `work.*` ops warn-and-skip (existing unknown-
  type rule); baked `work_items` is an ignored JSON field.
- A log with no work ops and no attachment-anchored attachments produces a view
  JSON-identical to 009's.
- No new subjects, buckets, headers, op-envelope fields, or server features.
