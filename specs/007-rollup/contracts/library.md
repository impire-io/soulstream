# Contract: Library API (new & changed)

## topic

```go
// rollup.go (NEW)
var (
    ErrRollupLost       error // concurrent write beat the attempt; nothing changed
    ErrNothingToCompact error // the log is already just a baseline
    ErrTopicArchived    error // write refused: archived is terminal
)

// Rollup compacts the topic: materialise baseline+tail, publish the result as the
// new baseline (rollup header + expected-last-subject-sequence guard), replacing
// baseline and tail atomically. Signs like any op when the client has a key.
// Returns the new baseline op-id. Errors: ErrNothingToCompact, ErrRollupLost,
// ErrTopicArchived, or the topic's malformation reason. On success the handle's
// frontier becomes the payload frontier.
func (h *Handle) Rollup(ctx context.Context) (string, error)

// lifecycle.go
const Archived Lifecycle = "archived" // now defined; Transition accepts it —
// but personas use Archive(); a bare transition(archived) leaves an unarchived-
// looking (uncompacted) terminal topic, valid but unusual.

// Close posts life.transition(closed) then makes ONE best-effort rollup attempt;
// a lost race is not an error (a closed uncompacted topic is valid).
func (h *Handle) Close(ctx context.Context) (string, error) // returns transition op-id

// Archive posts life.transition(archived) then rolls up with bounded retry (3,
// re-materialising between attempts). Exhausted retries return an error; the
// transition stands and the topic remains readable.
func (h *Handle) Archive(ctx context.Context) (string, error) // returns final baseline op-id

// post.go — behaviour change: every write path (Post, PostTurn, AddComment, Attach,
// Transition, Rollup) returns ErrTopicArchived when the handle last observed the
// topic archived. Closed keeps the existing warn-but-allow.

// vocab.go
type BaselinePayload struct {
    State    json.RawMessage `json:"state,omitempty"`
    Frontier []string        `json:"frontier"`
    Baked    *BakedState     `json:"baked,omitempty"`
    Manifest *ManifestRef    `json:"manifest,omitempty"`
}
type BakedState struct {
    Contributions []Contribution `json:"contributions,omitempty"`
    Attachments   []Attachment   `json:"attachments,omitempty"`
    Lifecycle     Lifecycle      `json:"lifecycle,omitempty"`
}
type ManifestRef struct {
    Chunks []string `json:"chunks"`
    Digest string   `json:"digest"`
    Size   uint64   `json:"size"`
}

// view.go — view structs gain explicit lowercase JSON tags (op_id, author, ts, body,
// mentions, anchor, dangling, sig, stream_seq omitempty, …). This changes MCP tool
// result key casing; internal tests updated in the same change.
```

Compatibility:

- Pre-007 baselines (no `baked`, empty `frontier` semantics) fold exactly as before.
- `StartTopic` unchanged: births still write `{state, frontier: []}`.
- `apply` remains a pure function; manifest fetch happens before the fold
  (materialise resolves the baseline's state document, then folds).
- Materialise/Follow/Board signatures unchanged.

## internal/cli

| Command | Behaviour |
|---|---|
| `rollup <path>` | compact; prints ops-folded count or "nothing to compact"; a lost race prints a retryable message and exits non-zero |
| `archive <path>` | loud confirmation line; refuses if already archived |
| `close <path>` | now also compacts (best effort; silent on lost race) |
| any write to an archived topic | clear "archived is terminal" error |

## internal/mcpserver

| Tool | Change |
|---|---|
| `soulstream_rollup_topic` | NEW (10th): input `{path}`; returns the new baseline op-id or "nothing to compact"; lost race is a retryable error |
| `soulstream_close_topic` | now compacts best-effort after the transition |
| all write tools | surface the archived refusal verbatim |
| (no archive tool) | archival is an operator act |
