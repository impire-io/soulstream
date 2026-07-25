# Data Model: Memory Convention & Exhibits (015)

## Wire vocabulary (new op types, all on `SOULSTREAM.SVC.MEMORY`, binding `MEMORY`)

All four ride ordinary `record.Record` wire form (headers + JSON payload), signed at the
`buildOpMsg` choke point when the sender has a signer, exactly like discovery traffic.

### `memory.query` (asker → all witnesses)

```json
{
  "query":    "what did we decide about the Q2 VAT reminder cadence?",
  "scope":    { "topics": ["vat-*"], "after": "2026-04-01T00:00:00Z" },
  "deadline": "2026-07-25T15:04:30Z"
}
```

- `query` (string, required, non-empty): free text.
- `scope.topics` ([]string, optional): topic-name patterns — a relevance *hint*, witness-interpreted (case-insensitive substring/glob, witness's choice); never enforced by the library.
- `scope.after` (RFC3339, optional): interest horizon hint.
- `deadline` (RFC3339, required): witnesses seeing a past deadline stay silent.
- Reply subject: the asker's ephemeral inbox (carried in the NATS `Reply` field, never signed over).

### `memory.answer` (witness → asker's inbox)

```json
{
  "answer":        "Weekly cadence, decided 2026-05-12; Bloem & Co. excepted.",
  "citations":     [ { "topic": "vat-q2-2026-x7m2", "op_id": "9f86d081-…" } ],
  "coverage_from": "2026-05-01T00:00:00Z"
}
```

- `answer` (string, required, non-empty): the witness's prose testimony.
- `citations` ([]Citation, optional): each `{topic (path string), op_id (string)}`. Never inline exhibits.
- `coverage_from` (RFC3339, optional, `omitzero`): start of the witness's op-granularity memory.
- Multiple answers per witness allowed; asker caps total gathered at 100.

### `memory.fetch` (asker → all witnesses)

```json
{ "topic": "vat-q2-2026-x7m2", "op_id": "9f86d081-…", "deadline": "2026-07-25T15:04:33Z" }
```

- Exactly one op per fetch. Same deadline/staleness rules as query.

### `memory.exhibit` (witness → asker's inbox)

```json
{ "exhibit": { …exhibit document, see below… } }
```

## Exhibit document (`record.Exhibit`, pure — NO NATS)

Verbatim wire capture plus verification inputs. Strict decode (`DisallowUnknownFields`).

| Field | Type | Meaning |
|---|---|---|
| `version` | int (1) | Exhibit format version. |
| `realm` | string | Realm name the op was published in — canonicalisation input. |
| `binding` | string | Canonical binding the signature was computed over (topic ops: subject suffix like `OPS.<path>`; the *verification* input). |
| `subject` | string | Full original subject — human provenance display only. |
| `headers` | map[string][]string | The op's headers verbatim (`Nats-Msg-Id`, `Soulstream-*` incl. `Soulstream-Sig`, unknown extras preserved). |
| `payload_b64` | string (std base64) | The op's payload bytes verbatim. |

- `Exhibit.Record() (record.Record, error)` — `record.Parse(headers, payload)`, the existing parser.
- Invariant: verbatim bytes + stored binding ⇒ recomputed canonical bytes match the signing input; any alteration (content, attribution, binding, signature) flips the verdict to failed.
- Serialization: plain JSON file (pretty-printed by CLI), stable and re-verifiable anywhere.

## Verdict (reused, not new)

`topic.SigStatus` is the exhibit verdict vocabulary — no new enum:

| Verdict | Meaning |
|---|---|
| `verified` | Signature valid against a key in the author's validated chain (per the verifier's keyring). |
| `failed` | Signature present but invalid — includes any tampering; also: author distrusted. |
| `unsigned` | Op never carried a signature — testimony-grade content. |
| `unknown-key` | Signed, but verifier knows no key of the author's chain. |

## Citation grades (asker-computed, per citation)

`topic.MemoryGrade` (string enum):

| Grade | Constant value | When |
|---|---|---|
| Fact | `fact` | Citation resolves in the cited topic's current state (live op or baked element) — checked by `ContainsOp`, never trusted. |
| Fact with provenance | `fact-with-provenance` | A follow-up fetch produced an exhibit that verifies. Never assigned during query. |
| Testimony | `testimony` | A fetch produced only an unsigned exhibit. |
| Gossip | `gossip` | Answer-level standing when an answer carries zero citations. |
| Unverifiable | `unverifiable` | Cited but resolves nowhere live (compacted or fabricated — indistinguishable); presented with caution, never as fact. |

State transitions: `unverifiable` —(explicit fetch, exhibit verifies)→ `fact-with-provenance`;
`unverifiable` —(fetch yields only unsigned)→ `testimony`. `fact` is terminal within a query.
No other transitions; grading is deterministic for fixed inputs (FR-011).

## Asker-side result types (`topic`, NATS-touching)

```go
type MemoryQueryInput struct {
    Query   string        // required
    Topics  []string      // optional scope hint
    After   time.Time     // optional scope hint (omitzero on wire)
    Timeout time.Duration // 0 ⇒ 3s default; clamped [100ms, 30s]
}

type MemoryCitation struct { Topic, OpID string }             // wire + result form
type GradedCitation struct { MemoryCitation; Grade MemoryGrade }

type MemoryAnswer struct {
    Witness      string          // Soulstream-Author of the answer op
    Sig          SigStatus       // answer-op signature status (failed ⇒ discarded earlier)
    Answer       string
    CoverageFrom time.Time       // zero ⇒ undeclared
    Citations    []GradedCitation
}

type MemoryResult struct { Answers []MemoryAnswer }           // merged, attributed; ≤100

type ExhibitResult struct {
    Exhibit record.Exhibit
    Verdict SigStatus       // exhibit's own embedded-signature verdict
    Source  string          // "live" | witness persona name
}
```

## Witness surface (`topic`, NATS-touching)

```go
type MemoryQueryRequest struct {           // what a witness's Answer func sees
    Query  string
    Topics []string
    After  time.Time
}

type MemoryAnswerDraft struct { Answer string; Citations []MemoryCitation }

type MemoryWitness struct {
    CoverageFrom time.Time
    Answer   func(q MemoryQueryRequest) []MemoryAnswerDraft      // nil ⇒ queries unserved
    Fetch    func(topic, opID string) (record.Exhibit, bool)     // nil ⇒ fetches unserved
    OnServed func(kind string, n int)                            // optional; kind "query"|"fetch"; n = items sent, -1 = stale skip
}
```

- Both capabilities independently optional (Clarification #4); nil handler or empty return ⇒ silence.
- The library owns signing, payload assembly, deadline checks, and reply publishing.

## Resolver (pure)

`(mt *MaterializedTopic) ContainsOp(opID string) bool` — scans announcement op id,
contributions + edit stamps, attachments, work items + timeline events, current baseline op
id, and frontier. Live + baked already unified in the view. Compaction-vanished ops (marks,
transitions, superseded chain members not baked as ids) honestly return false — the exhibit
path exists for those.

## Relationships

```text
MemoryQueryInput ──publish──▶ memory.query ──▶ witnesses (RespondMemory.Answer)
                                                   │ drafts
MemoryResult ◀──gather+verify+grade── memory.answer ┘
     │ per citation: ContainsOp ⇒ fact | unverifiable
     └ explicit follow-up: FetchExhibit
FetchExhibit ──1: CaptureExhibit (live) ──▶ ExhibitResult{Source:"live"}
             └─2: memory.fetch ──▶ witnesses (RespondMemory.Fetch) ──▶ memory.exhibit
                   first verifying wins; unsigned held as fallback; failed discarded
record.Exhibit ──VerifyExhibit(kr)──▶ SigStatus  (offline: kr from pins alone)
```
