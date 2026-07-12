# Contract: `topic` package

The op-log engine, built on `record`, `identity`, and `realm`. A new package `topic` (the only
new NATS-touching package besides `realm`).

## Subjects & vocabulary

```go
package topic

const (
	OpsSubjectPrefix  = "SOULSTREAM.TOPICS.OPS."
	InfoSubjectPrefix = "SOULSTREAM.TOPICS.INFO."
)

// Operation types defined this cycle.
const (
	TypeAnnounce   = "topic.announce"
	TypeBaseline   = "baseline"
	TypeTurnPost   = "turn.post"
	TypeCommentAdd = "comment.add"
	TypeLifeTransition = "life.transition"
)

// Lifecycle states derivable this cycle.
type Lifecycle string
const (
	Proposed Lifecycle = "proposed"
	Active   Lifecycle = "active"
	Closed   Lifecycle = "closed"
)
```

## Starting a topic

```go
// StartTopicInput is the intent to start a topic.
type StartTopicInput struct {
	Name          string   // display name (required)
	SubjectMatter string
	Tags          []string
	Expected      []string // hint only
	Parent        string   // "" for top-level; else the parent's topic-path
	State         json.RawMessage // optional initial baseline state (defaults to {})
}

// StartTopic generates a topic-id (<slug>-<suffix>), publishes topic.announce to the INFO
// subject and the initial baseline to the OPS subject, and returns a handle to the new topic.
func StartTopic(ctx context.Context, c *realm.Client, in StartTopicInput) (*Handle, error)

// Open returns a handle to an existing topic by path (no publish).
func Open(c *realm.Client, path string) *Handle
```

`StartTopic` errors if the initial state exceeds the inline baseline threshold (FR-028), or the
parent is malformed. A generated topic-id always satisfies the slug grammar.

## The handle

```go
// Handle binds a client to one topic-path; posts ops and materialises/follows the topic.
type Handle struct { /* client, path, observed frontier */ }

func (h *Handle) Path() string

// Post builds a record (author, op-id, ts, parents=observed frontier), enforces attribution, and
// publishes to the OPS subject; returns the new op-id. It warns (does not block) if the topic is
// known-closed (FR-022).
func (h *Handle) Post(ctx context.Context, opType string, payload json.RawMessage, anchor string) (opID string, err error)

// Convenience posters.
func (h *Handle) PostTurn(ctx context.Context, body string) (string, error)
func (h *Handle) AddComment(ctx context.Context, body, anchorOpID string) (string, error)
func (h *Handle) Transition(ctx context.Context, to Lifecycle) (string, error) // rejects undefined states (FR-021)

// Materialise drains the topic's ops backlog and returns the current view; it also updates the
// handle's observed frontier.
func (h *Handle) Materialise(ctx context.Context) (*MaterializedTopic, error)

// Follow materialises, then keeps applying live ops via one ordered consumer, calling onOp after
// each applied op with the updated view. It blocks until ctx is cancelled.
func (h *Handle) Follow(ctx context.Context, onOp func(*MaterializedTopic)) error
```

## The materialised view

```go
type Contribution struct {
	OpID      string
	Author    string
	Timestamp time.Time
	Type      string // turn.post | comment.add
	Body      string
	Anchor    string // comment's anchored op-id ("" for turns)
	Dangling  bool   // comment anchor not present
	StreamSeq uint64
}

type MaterializedTopic struct {
	Path          string
	Announcement  *Announcement // nil if not seen
	BaselineState json.RawMessage
	Lifecycle     Lifecycle
	Contributions []Contribution // ordered by StreamSeq
	Frontier      []string       // leaf op-ids
	Malformed     string         // non-empty reason if first op isn't a baseline (FR-015)
}

type Announcement struct {
	TopicID, Name, SubjectMatter, Parent string
	Expected, Tags                       []string
}
```

## Discovery board

```go
type BoardEntry struct {
	Path         string
	Announcement Announcement
	Parent       string
	ParentKnown  bool
	Lifecycle    Lifecycle // where derivable (best-effort)
}

// Board replays SOULSTREAM.TOPICS.INFO.> and returns one entry per topic (latest announcement per
// subject). An empty realm yields an empty board, not an error.
func Board(ctx context.Context, c *realm.Client) ([]BoardEntry, error)
```

## Contract guarantees (map to spec)

- **Handle populates parents from observed frontier** (FR-002/018): Post stamps parents = leaves of
  what the handle has seen.
- **Baseline-first invariant** (FR-005/015): StartTopic publishes baseline as the first op;
  Materialise reports `Malformed` if the first op isn't a baseline.
- **Stream-sequence ordering** (FR-011/012): contributions ordered by `StreamSeq`; parents recorded,
  not consulted.
- **Pure projection** (FR-014): Materialise is a function of the log; identical logs → identical views.
- **Seam-free follow** (FR-017): one ordered consumer delivers history then live.
- **Lifecycle derivation** (FR-019/020): proposed/active/closed from the log; idempotent transitions.
- **Sub-topics by subject depth** (FR-023/024): nested path → nested subject; independent materialise.
- **Board** (FR-025/026/027): one entry per topic, parent flagged when unknown, empty realm → empty.
- **Unknown types ignored with warning** (FR-008); **dangling comment kept & flagged** (FR-016).
