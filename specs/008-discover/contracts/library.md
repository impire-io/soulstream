# Contract: Library API (new & changed)

## realm

```go
// Conn returns the client's raw NATS connection, for core request-reply surfaces
// (discovery). Same exposure grounds as JetStream().
func (c *Client) Conn() *nats.Conn
```

## topic

```go
// subjects.go
const (
    SvcSubjectPrefix   = "SOULSTREAM.SVC."
    SvcDiscoverSubject = "SOULSTREAM.SVC.DISCOVER"
)
// canonicalBinding: SVC subjects bind to the service name (suffix after the
// prefix); replies to discovery sign over "DISCOVER" explicitly.

// vocab.go
const (
    TypeDiscover      = "topic.discover"
    TypeDiscoverReply = "topic.discover.reply"
)
type DiscoverPayload struct {
    Query    string    `json:"query"`
    Limit    int       `json:"limit,omitempty"`
    Deadline time.Time `json:"deadline,omitempty"`
}
type DiscoverEntry struct {
    Path          string    `json:"path"`
    Name          string    `json:"name"`
    SubjectMatter string    `json:"subject_matter,omitempty"`
    Tags          []string  `json:"tags,omitempty"`
    Lifecycle     Lifecycle `json:"lifecycle,omitempty"`
}
type DiscoverReplyPayload struct {
    Matches []DiscoverEntry `json:"matches"`
}

// discover.go
type DiscoverAnswer struct {
    Persona string    `json:"persona"`
    Sig     SigStatus `json:"sig,omitempty"`
}
type DiscoverResult struct {
    DiscoverEntry
    Answers []DiscoverAnswer `json:"answers"`
}
type DiscoverInput struct {
    Query   string
    Limit   int           // default 10
    Timeout time.Duration // default 2s — the ask's deadline
}

// Discover publishes the request and gathers replies until the deadline, merging
// one result per topic path with per-answerer verification status against kr.
// Zero replies ⇒ (nil, nil): silence is an answer.
func Discover(ctx context.Context, c *realm.Client, in DiscoverInput, kr *identity.Keyring) ([]DiscoverResult, error)

// RespondDiscovery serves discovery as c's persona until ctx is cancelled: for each
// request it rebuilds the board projection, matches, and replies only when there
// are matches (signed when keyed). onServed, if non-nil, is called after each
// request with the number of matches sent (-1 for silently skipped requests) —
// observability for the CLI, nothing more.
func RespondDiscovery(ctx context.Context, c *realm.Client, onServed func(query string, sent int)) error

// pure, serverless-tested:
//   matchEntries(entries []BoardEntry, query string, limit int) []DiscoverEntry
//   mergeReplies(...) — fold of (answerer, sig, entries) into []DiscoverResult
```

## internal/cli

| Command | Behaviour |
|---|---|
| `discover <query> [--timeout d] [--limit n] [--json]` | ask + merged render: one line per topic (lifecycle, path, name) plus `answered by: persona ✓, other ?`; empty result prints "no answers before the deadline (the board still works: soulstream board)" and exits 0 |
| `respond` | long-running responder (persona required; Ctrl-C to stop); logs one line per served request |

## internal/mcpserver

| Tool | Change |
|---|---|
| `soulstream_discover` | NEW (11th): input `{query, limit?}`; default deadline; JSON result of merged DiscoverResults (per-answer `sig`); empty list on silence |
| (no responder) | the adapter asks only this cycle |
