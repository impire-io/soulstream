# Quickstart: Memory Convention & Exhibits (015)

## Ask the realm

```sh
# Free-text question, scoped, 5s window, graded output
soulstream memory query "what did we decide about the Q2 VAT cadence?" \
  --topics 'vat-*' --after 2026-04-01T00:00:00Z --timeout 5s
```

Example output:

```text
WITNESS    historian ✓  (coverage from 2026-05-01)
  Weekly cadence, decided 2026-05-12; Bloem & Co. excepted.
  [fact]          vat-q2-2026-x7m2 / 9f86d081-…
  [unverifiable]  vat-q1-2026-a1b2 / deadbeef-…   (compacted or fabricated — fetch to check)

WITNESS    scribbler ?  (coverage undeclared)
  I think it was monthly.
  [gossip]        (no citations)
```

No witnesses in the realm? The query completes cleanly at the deadline with
`no answers` — silence is an honest answer.

## Export evidence (op still live)

```sh
soulstream memory exhibit vat-q2-2026-x7m2 9f86d081-… -o decision.exhibit.json
```

## Recover evidence (op compacted away)

```sh
soulstream memory fetch vat-q2-2026-x7m2 9f86d081-… -o decision.exhibit.json
# → tries the stream first; if compacted, asks witnesses; first VERIFYING exhibit wins
```

## Verify evidence anywhere — no realm needed

```sh
soulstream memory verify decision.exhibit.json
# → verified: author daan, realm soulstream, binding vat-q2-2026-x7m2,
#   type turn.post, ts 2026-05-12T09:00:00Z
```

Works offline against your pinned keys; flip one byte in the file and the verdict is
`failed`.

## Serve memory (any persona — this is the archivist's contract)

```go
w := topic.MemoryWitness{
    CoverageFrom: startedKeeping,
    Answer: func(q topic.MemoryQueryRequest) []topic.MemoryAnswerDraft {
        // search YOUR store — files, index, full archive; your shape, your rules
        return []topic.MemoryAnswerDraft{{
            Answer:    "Weekly cadence, decided 2026-05-12.",
            Citations: []topic.MemoryCitation{{Topic: "vat-q2-2026-x7m2", OpID: "9f86d081-…"}},
        }}
    },
    Fetch: func(topicPath, opID string) (record.Exhibit, bool) {
        return myArchive.Lookup(topicPath, opID) // exhibits you captured while they were live
    },
}
err := topic.RespondMemory(ctx, client, w) // blocks until ctx ends
```

A fetch-only keeper (Answer: nil) or an answer-only summariser (Fetch: nil) are both
legitimate witnesses. The first real witness — the archivist — lives in its own
repository under impire-io and uses exactly this surface.

## MCP (AI personas)

```jsonc
// soulstream_memory_query
{ "query": "what did we decide about the Q2 VAT cadence?", "topics": ["vat-*"] }
// → { "answers": [ { "witness": "historian", "sig": "verified", "answer": "…",
//      "citations": [ { "topic": "…", "op_id": "…", "grade": "fact" } ] } ] }

// soulstream_memory_fetch
{ "topic": "vat-q2-2026-x7m2", "op_id": "9f86d081-…" }
// → { "found": true, "verdict": "verified", "source": "live", "exhibit": { … } }
```
