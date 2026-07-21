# Contract: Library API (new & changed)

## topic (additive)

```go
// view.go
type MaterializedTopic struct {
    // ...existing fields...
    BaselineTs time.Time `json:"baseline_ts,omitempty"` // NEW: the baseline op's timestamp
}

// discover.go
// RespondDiscoveryWith serves discovery with a caller-supplied answerer: answer
// returns the entries to send for a query (nil/empty ⇒ silence). RespondDiscovery
// becomes the board-backed wrapper; wire behaviour is unchanged either way.
func RespondDiscoveryWith(ctx context.Context, c *realm.Client,
    answer func(query string, limit int) []DiscoverEntry,
    onServed func(query string, sent int)) error
```

## curator (NEW package — public-library surfaces only)

```go
// suggest.go (pure)
const (
    SuggestionDuplicatePrefix = "[curator] this looks similar to "
    SuggestionDormantPrefix   = "[curator] no activity for "
)
func DuplicateSuggestion(olderPath string) string
func DormantSuggestion(idle time.Duration) string
func IsSuggestion(body string) bool          // either kind, author-independent
func IsDuplicateSuggestion(body string) bool
func IsDormantSuggestion(body string) bool

// judge.go (pure)
func Similarity(a, b topic.DiscoverEntry) float64   // token Jaccard, id-suffix excluded
const DuplicateThreshold = 0.5

// curator.go
type Options struct {
    IdleWindow time.Duration               // default 336h
    ScanEvery  time.Duration               // default 1m
    OnEvent    func(event string)          // optional observability
}
// Run curates as c's persona until ctx is cancelled: warm projection, discovery
// answering (content-aware), duplicate flags, dormancy proposals. Ordinary ops
// only; stopping needs nothing but the cancel.
func Run(ctx context.Context, c *realm.Client, opts Options) error
```

Internal (unexported, but pure and unit-tested): projection cache with
dirty-tracking; `search(query, limit)`; per-topic `lastReal`; flag/proposal
neededness per data-model rules.

## internal/cli

| Command | Behaviour |
|---|---|
| `curate [--idle 336h] [--scan-every 1m]` | long-running curator under the session persona (required); prints one line per event (answered/flagged/proposed); Ctrl-C to stop; signs when keyed like every command |

## Explicitly unchanged

- No MCP changes (no new tools; curator suggestions arrive as ordinary comments in
  `show_topic` results).
- No wire changes: no op types, subjects, headers, or storage. `BaselineTs` is a
  view field derived from data already on the wire.
- 008's `RespondDiscovery` keeps its exact signature and behaviour.
