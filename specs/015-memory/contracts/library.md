# Library Contract: Memory Convention & Exhibits (015)

The public Go surface this feature adds. The external archivist repository builds against
EXACTLY this (plus already-public read surfaces: `realm.Connect`/`NewClient`,
`topic.Materialise`/`Follow`, `registry.All`/`BuildKeyring`, `internal-free` keystore is NOT
public — the archivist keeps its own key/pin files or uses the CLI to provision identity).

## Package `record` (pure — imports no NATS)

```go
// Exhibit is a portable, self-contained, self-authenticating capture of one operation:
// the wire form verbatim plus the two verification inputs (realm, binding).
type Exhibit struct {
    Version  int                 `json:"version"`  // 1
    Realm    string              `json:"realm"`
    Binding  string              `json:"binding"`  // canonical binding the signature covers
    Subject  string              `json:"subject"`  // original subject — display/provenance only
    Headers  map[string][]string `json:"headers"`  // verbatim, incl. Soulstream-Sig and extras
    Payload  []byte              `json:"payload_b64"` // verbatim op payload (std base64 in JSON)
}

// Record reconstructs the operation via the standard wire parser.
func (e Exhibit) Record() (Record, error)

// ParseExhibit strict-decodes an exhibit document (unknown fields rejected, version checked).
func ParseExhibit(data []byte) (Exhibit, error)

// MarshalExhibit produces the stable serialized form (deterministic field order via
// encoding/json struct order; pretty-printing is the caller's choice).
func (e Exhibit) Marshal() ([]byte, error)
```

Guarantees:
- Round-trip stable: `Marshal → ParseExhibit → Marshal` is byte-identical.
- Strict decode: unknown fields, missing required fields, or wrong version fail loudly.
- NO NATS imports; unit-tests server-free.

## Package `topic` (NATS-touching)

### Constants & errors

```go
const ServiceMemory = "MEMORY"                       // canonical binding for all memory traffic
const SvcMemorySubject = SvcSubjectPrefix + "MEMORY" // SOULSTREAM.SVC.MEMORY

const (
    DefaultMemoryTimeout = 3 * time.Second
    MinMemoryTimeout     = 100 * time.Millisecond
    MaxMemoryTimeout     = 30 * time.Second
    MaxMemoryAnswers     = 100                       // safety cap per query
)

// Op types (vocab): "memory.query", "memory.answer", "memory.fetch", "memory.exhibit"

var ErrOpNotLive error // CaptureExhibit: op not found in the topic's live subjects
```

### Grades

```go
type MemoryGrade string
const (
    GradeFact         MemoryGrade = "fact"
    GradeProvenance   MemoryGrade = "fact-with-provenance"
    GradeTestimony    MemoryGrade = "testimony"
    GradeGossip       MemoryGrade = "gossip"
    GradeUnverifiable MemoryGrade = "unverifiable"
)
```

### Asker surface

```go
// MemoryQuery publishes a query, gathers answers until the (clamped) deadline or the
// 100-answer cap, verifies each answer op (binding MEMORY; failed sig ⇒ discarded,
// unsigned ⇒ kept + status), and grades every citation by live resolution
// (one Materialise per distinct cited topic, memoised). Zero witnesses ⇒ (&MemoryResult{}, nil).
func MemoryQuery(ctx context.Context, c *realm.Client, in MemoryQueryInput, kr *identity.Keyring) (*MemoryResult, error)

// CaptureExhibit scans the topic's OPS/INFO subjects for the op and captures it verbatim.
// Returns ErrOpNotLive when the op is not in the stream (compacted or never existed).
func CaptureExhibit(ctx context.Context, c *realm.Client, path, opID string) (record.Exhibit, error)

// FetchExhibit is live-first: CaptureExhibit, else scatter/gather memory.fetch until the
// first VERIFYING exhibit (immediate win), holding an unsigned exhibit as fallback,
// discarding failed ones. Nothing at deadline ⇒ (nil, nil): silence is an answer.
func FetchExhibit(ctx context.Context, c *realm.Client, path, opID string, timeout time.Duration, kr *identity.Keyring) (*ExhibitResult, error)

// VerifyExhibit reconstructs the record and verifies its embedded signature against the
// author's validated chain in kr. Pure check: no realm connectivity.
func VerifyExhibit(e record.Exhibit, kr *identity.Keyring) (SigStatus, error)

// GradeForVerdict maps a fetched exhibit's verdict to the citation grade it supports:
// verified → fact-with-provenance, unsigned → testimony, failed/unknown-key → unverifiable.
func GradeForVerdict(s SigStatus) MemoryGrade
```

### Witness surface

```go
// RespondMemory subscribes (plain, no queue group, Flush before live) on the memory
// channel and serves until ctx ends. Nil capabilities stay silent for their kind.
// Stale requests (deadline past) are skipped (OnServed(kind, -1)). Replies are signed
// with binding MEMORY when the client has a signer; coverage_from included when set.
func RespondMemory(ctx context.Context, c *realm.Client, w MemoryWitness) error
```

(Struct shapes for `MemoryQueryInput`, `MemoryResult`, `MemoryAnswer`, `GradedCitation`,
`MemoryCitation`, `ExhibitResult`, `MemoryWitness`, `MemoryQueryRequest`,
`MemoryAnswerDraft` — see [data-model.md](../data-model.md).)

### Resolver

```go
// ContainsOp reports whether opID resolves in the topic's current state: live ops or
// baked elements (announcement, contributions + edit stamps, attachments, work items +
// timeline, current baseline op, frontier). Pure.
func (mt *MaterializedTopic) ContainsOp(opID string) bool
```

## CLI contract (`soulstream memory …`)

| Command | Behaviour | Exit |
|---|---|---|
| `memory query "q" [--topics a,b] [--after RFC3339] [--timeout d] [--json]` | Publish query, print graded attributed answers (witness, sig marker, coverage, per-citation grade; citation-less answers tagged gossip). Empty result prints "no answers" cleanly. | 0 on gather success (even empty); 2 usage |
| `memory fetch <topic> <op-id> [--timeout d] [-o file] [--force] [--json]` | Live-first, then witnesses. Prints verdict + author + realm/binding + source; `-o` writes exhibit JSON (overwrite-guard). Nothing found ⇒ message + exit 1. | 0 found; 1 not found; 2 usage |
| `memory exhibit <topic> <op-id> [-o file] [--force] [--json]` | Live-only export; on ErrOpNotLive: error pointing at `memory fetch`. | 0; 1 not live; 2 usage |
| `memory verify <file>` | OFFLINE: never connects; keyring from pins file alone; prints verdict, author, realm, binding, type, timestamp. Exit mirrors verdict: verified/unsigned ⇒ 0 with clear wording; failed ⇒ 1; unknown-key ⇒ 0 with warning. Works with broken/absent realm config (013 lesson: diagnostics survive). | see behaviour |

## MCP contract

| Tool | Input | Result (JSON) |
|---|---|---|
| `soulstream_memory_query` | `{query, topics?, after?, timeout_ms?}` | `{answers: [{witness, sig, answer, coverage_from?, citations: [{topic, op_id, grade}]}]}` — nil-normalised to `[]` |
| `soulstream_memory_fetch` | `{topic, op_id, timeout_ms?}` | `{found: bool, verdict?, source?, exhibit?}` — exhibit is the document verbatim |

Both use the per-call keyring; casing follows Go `json` tags (snake_case), the 007
convention. Total tools: 23.

## Compatibility

- No changes to existing types, wire headers, subjects, streams, or provisioning.
- New op types are additive vocabulary; old clients ignore unknown SVC traffic (they
  subscribe to none of it).
- The exhibit document is version-stamped (`version: 1`) for future evolution.
