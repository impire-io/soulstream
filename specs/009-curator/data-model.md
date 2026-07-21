# Data Model: The Curator Persona

## Suggestion (a body convention over ordinary comments — no new op types)

| Kind | Body shape | Recognised by |
|---|---|---|
| duplicate flag | `[curator] this looks similar to <older-path> — consider continuing there` | prefix `[curator] this looks similar to ` |
| dormancy proposal | `[curator] no activity for <span> — close it if it's done` | prefix `[curator] no activity for ` |

- Posted via ordinary `AddComment`, anchored to the topic's current frontier op.
- Recognition is author-independent: any persona's matching comment counts (two
  curators respect each other's suggestions; a human writing the marker by hand is
  simply making the same suggestion).
- Exported constants so tests and future tools share one definition.

## Projection (in-memory, per curator)

```
projection {
  entries  map[path] → cachedTopic
  dirty    set[path]                  // marked by the TOPICS.> subscription
}
cachedTopic {
  view         *topic.MaterializedTopic   // cached Materialise result
  entry        topic.DiscoverEntry        // derived identity fields for answers
  searchText   string                     // lowercased: name ⊕ subject ⊕ tags ⊕ bodies ⊕ attachment names
  lastReal     time.Time                  // newest non-suggestion activity (≥ BaselineTs)
  birth        time.Time                  // BaselineTs
}
```

- Seeded from `Board`; a dirty path re-materialises on next use; unknown paths seen
  on INFO are added. Malformed topics are cached as skip-markers (never matched,
  never commented on).

## Judgment rules (pure)

| Rule | Definition |
|---|---|
| similarity(a, b) | Jaccard of lowercased alphanumeric token sets of name + subject matter + tags, excluding the topic-id suffix tokens; ∈ [0,1] |
| likely duplicate | newer topic N and older topic O (birth order, path tiebreak) with similarity ≥ 0.5; flag names the best-scoring O |
| dormant(t, now, window) | lifecycle ∉ {closed, archived}, not malformed, and now − lastReal > window |
| flag needed | likely duplicate ∧ no duplicate-kind suggestion anywhere in N |
| proposal needed | dormant ∧ no dormancy-kind suggestion newer than lastReal |

## MaterializedTopic (additive library change)

| Field | Type | Rules |
|---|---|---|
| BaselineTs | time.Time (`json:"baseline_ts,omitempty"`) | the first record's timestamp; post-rollup, the rollup baseline's |

## Options (curator.Run)

| Field | Default | Meaning |
|---|---|---|
| IdleWindow | 336h (14 d) | quiet period before a dormancy proposal |
| ScanEvery | 1 m | cadence of the duplicate + dormancy passes (answers are event-driven) |
| OnEvent | nil | observability callback (one line per answer/flag/proposal) |

## Invariants

- Zero curators ⇒ nothing anywhere changes (SC-004: the realm has no curator
  concept; this whole model lives client-side in one process).
- All curator writes are ordinary attributed comments; every existing client
  renders them with no changes (SC-005).
- No local persistence: restart ⇒ rebuild from replay; idempotence holds because
  the log is the memory.
