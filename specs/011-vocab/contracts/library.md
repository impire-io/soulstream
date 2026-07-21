# Contract: 011-vocab — library, CLI, and MCP surfaces

## Go library (`topic`)

```go
const (
    TypeCommentReply     = "comment.reply"
    TypeCommentResolve   = "comment.resolve"
    TypeEdit             = "edit"
    TypeAttachmentRemove = "attachment.remove"
)
const Dormant Lifecycle = "dormant" // Transition/definedLifecycle accept it

type RefPayload struct{ Anchor *Anchor `json:"anchor"` }
type EditStamp struct{ OpID, Author string; Ts time.Time; Sig SigStatus; StreamSeq uint64 }
// Contribution += Resolved, ResolvedBy, Edits []EditStamp
// Attachment  += Removed, RemovedBy

func (h *Handle) Reply(ctx context.Context, body, anchorOpID string) (string, error)
func (h *Handle) Edit(ctx context.Context, targetOpID, newBody string) (string, error)
func (h *Handle) Resolve(ctx context.Context, targetOpID string) (string, error)
func (h *Handle) RemoveAttachment(ctx context.Context, addOpID string) (string, error)
func (h *Handle) MarkDormant(ctx context.Context) (string, error)

func DormantEligible(mt *MaterializedTopic, window time.Duration, now time.Time) bool
func StaleClaims(mt *MaterializedTopic, window time.Duration, now time.Time) []string
```

Behavioural contract: same-author edits only (others warn, no effect); resolve/
remove idempotent, author-agnostic, attributed; artefact tips skip removed
revisions and fully-removed lineages leave `Artefacts()`; archival deletes
removed blobs after the final compaction; every new op signs/verifies at the
existing choke points.

## Curator (`curator`)

```go
// Options += MarkDormant bool          // opt-in dormant sweep (uses IdleWindow)
//            ReclaimAfter time.Duration // opt-in stale-claim sweep; 0 = off
```

Both passes run on the existing scan tick, post ordinary ops as the curator's
persona, and are log-idempotent (dormant marks idempotent; second abandon void).

## CLI (`soulstream`)

```text
reply <path> <op-id> <body>          reply under a comment (mentions ping)
edit <path> <op-id> <body>           correct your own turn/comment/reply
resolve <path> <op-id>               mark a comment settled (stays readable)
detach <path> <attachment-op-id>     withdraw a file (bytes reclaimed at archival)
mark-dormant <path> [--idle 336h]    apply the idle rule; posts only when eligible
curate … [--mark-dormant] [--reclaim <dur>]   the two opt-in sweeps
```

`work show` renders resolved/edited marks on evidence; `show` renders
`(edited by …)`, `resolved`, and `removed by …` markers; `artefacts` skips
removed tips per the library rule.

## MCP (`soulstream-mcp`) — 3 new tools, total 21

| Tool | Input | Output |
|---|---|---|
| `soulstream_reply_comment` | topic, anchor_op_id, body | op id |
| `soulstream_resolve_comment` | topic, op_id | op id |
| `soulstream_edit` | topic, op_id, body | op id (error text explains the same-author rule when the edit will be void — best-effort pre-check from the view) |

Removal, mark-dormant, and sweeps stay operator surfaces (CLI), like archive.

## Compatibility contract

- Pre-011 baselines parse unchanged; absent fields ⇒ zero values.
- Pre-011 readers of post-011 logs warn-and-skip the four new op types; dormant
  transitions read as unknown lifecycle target and are ignored by old folds
  (their switch has no case) — views stay readable.
- A topic using none of the new words serialises identically to 010.
- No new subjects, buckets, headers, or server features.
