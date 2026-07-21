<!-- SPECKIT START -->
Active feature: **009-curator** — the curator persona (Day-2 #4, extensions/curation.md).
New top-level `curator` package built ONLY on public library surfaces (the package boundary
proves "zero protocol standing"). Projection = cache of `Materialise` views seeded from `Board`,
dirty-marked by ONE core-NATS subscription on `SOULSTREAM.TOPICS.>`, lazily re-materialised.
Answering: `topic.RespondDiscoveryWith(ctx, c, answer, onServed)` refactor (008's responder =
board-backed wrapper); curator answers cover contribution bodies + attachment names too.
Suggestions = ordinary AddComment anchored to the frontier, body convention
`[curator] this looks similar to <path> — …` / `[curator] no activity for <span> — …`;
recognition author-independent; idempotence FROM THE LOG (flag once per topic; proposal once
per quiet spell = no dormancy suggestion newer than lastReal). Similarity = token Jaccard ≥ 0.5
over name+subject+tags (topic-id suffix excluded). Dormant = now − lastReal > window (default
336h), lastReal excludes suggestions, ≥ BaselineTs — NEW additive field
`MaterializedTopic.BaselineTs`. Skip closed/archived/malformed. CLI: `curate [--idle] [--scan-every]`.
No MCP changes, no new op types, no storage.

For details read: [specs/009-curator/plan.md](specs/009-curator/plan.md)
(spec: `specs/009-curator/spec.md`, contract: `specs/009-curator/contracts/library.md`).
Done: `001`–`005` (MVP), `006-signing`, `007-rollup`, `008-discover` merged + pushed.

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` import NO NATS; `realm`, `topic`, `registry`, `curator` are NATS-touching.
- Keep pure logic (folds, chain validation, discovery match/merge, curator judgment) separate
  from NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; modern `nats.go/jetstream` API.
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
- Push to origin after merging a completed feature to main.
<!-- SPECKIT END -->
