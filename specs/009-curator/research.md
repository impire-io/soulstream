# Research: The Curator Persona

## R1. Projection shape

- **Decision**: cache of `topic.Materialise` views keyed by path, seeded from
  `Board`, invalidated by one **core** NATS subscription on `SOULSTREAM.TOPICS.>`
  (any message on a topic subject marks that path dirty; INFO messages on unknown
  paths add them). Dirty topics re-materialise lazily on next use (search or scan).
- **Rationale**: warm (no per-request replay — 008's plain responder rebuilds the
  board every ask; the curator answers from cache), live (a topic announced or
  posted to after start is findable), and built purely on public surfaces — no
  duplicate fold, no JetStream consumer bookkeeping, and a missed signal can only
  delay freshness, never corrupt (the next scan tick refreshes dirty paths anyway).
- **Alternatives considered**: one ordered JetStream consumer folding incrementally
  (duplicates the fold logic or demands exporting it; more machinery for the same
  answers at this scale); per-request Board+Materialise (not warm — indistinguishable
  from 008's responder).

## R2. The curator's answering edge

- **Decision**: `topic.RespondDiscoveryWith(ctx, c, answer func(query string, limit
  int) []DiscoverEntry, onServed)` — extracted from 008's responder, which becomes
  the board-backed wrapper. The curator's `answer` searches cached views:
  case-insensitive substring over name, subject matter, tags, **and contribution
  bodies + attachment names**.
- **Rationale**: same request, same reply shape, same merge (FR-003) — the curator
  is a better *answerer*, not a different *mechanism*. Substring keeps matching
  deterministic and explainable, consistent with 008.
- **Alternatives considered**: a parallel curator-specific service subject (would
  fork the mechanism — exactly what the steward post-mortem forbids); token/fuzzy
  ranking (cleverness without need; future curator improvement).

## R3. Duplicate likeness

- **Decision**: token-set Jaccard over lowercased alphanumeric tokens of name +
  subject matter + tags, threshold 0.5, compared newer-vs-every-older (birth order
  by `BaselineTs`, path as tiebreak). The best-scoring older topic above threshold
  is named in the flag. The topic-id's random suffix tokens are excluded (they would
  poison the overlap).
- **Rationale**: deterministic, explainable in one sentence, and computable from
  announcement metadata alone (FR-004). 0.5 means "half their words in common" —
  conservative enough that unrelated topics stay unflagged (US2 scenario 4).
- **Alternatives considered**: edit distance on names (misses tag/subject overlap);
  content similarity (heavier, and the spec deliberately scopes likeness to
  announcement metadata this cycle).

## R4. Suggestion convention & idempotence

- **Decision**: suggestions are ordinary comments whose body starts with a stable
  marker: `[curator] this looks similar to <path> — …` (duplicate kind) and
  `[curator] no activity for <span> — …` (dormancy kind). Recognition is by prefix,
  author-independent. Idempotence reads the log: flag only if no duplicate-kind
  suggestion exists in the topic; propose only if no dormancy-kind suggestion exists
  **newer than the last real activity**. Anchored to the topic's current frontier op.
- **Rationale**: the log as memory is restart-safe and multi-curator-cooperative for
  free (FR-006); author-independent recognition means two curators respect each
  other's flags; a body convention needs no vocabulary change (FR-008 without new op
  types — constitution II).
- **Alternatives considered**: a `curation.flag` op type (vocabulary growth for what
  a comment already expresses; clients would need updates — FR/SC-005 wants none);
  curator-local state files (breaks across restarts and across curators — exactly
  the failure the log avoids).

## R5. Dormancy rule

- **Decision**: last real activity = the newest of `BaselineTs` and every
  contribution/attachment timestamp whose body is **not** a recognised suggestion.
  Dormant when `now − lastReal > idle window` (default 336h/14d). Proposed at most
  once per quiet spell: a dormancy suggestion newer than lastReal suppresses the
  next one. Closed, archived, and malformed topics are skipped.
- **Rationale**: matches the clarified spec exactly; curator chatter can neither
  keep a topic alive nor re-arm a proposal. Timestamps are author-claimed and that
  is fine — this is judgment producing a *suggestion*, not protocol ordering.
- **Alternatives considered**: stream-sequence-based recency (sequences carry no
  wall-clock meaning for an idle *duration*); posting `life.transition(dormant)`
  (the dormant state is deferred vocabulary, and the curator must suggest, not act).

## R6. BaselineTs

- **Decision**: additive field `MaterializedTopic.BaselineTs` — the fold records
  `recs[0].Record.Timestamp`. After a rollup it is the rollup baseline's time.
- **Rationale**: announcement-only topics need a birth time for the idle window
  (spec edge case); the fold already holds the record. Post-rollup semantics are
  honest: compaction is real activity by a persona.
- **Alternatives considered**: curator-side raw stream reads to find the first op
  (works, but duplicates replay plumbing for one timestamp the fold can simply
  surface).

## R7. Run loop & CLI

- **Decision**: `curator.Run(ctx, c, Options{IdleWindow, ScanEvery, OnEvent})`:
  build projection → start the dirty-marking subscription → serve discovery via
  `RespondDiscoveryWith(projection.Search)` → on each scan tick (default 1m):
  refresh dirty views, then duplicate pass and dormancy pass, posting suggestions
  via `topic.Open(...).AddComment` (signed when the client is keyed, like any
  persona). CLI: `soulstream curate [--idle 336h] [--scan-every 1m]`, long-running,
  event lines via OnEvent.
- **Rationale**: one loop, one cadence, event-driven answering; comments through the
  ordinary Handle path means attribution, signing, mention parsing, and archived
  refusal all come for free (FR-007's "comments are its entire vocabulary").
- **Alternatives considered**: separate binary (`cmd/soulstream-curator`) — deferred;
  the CLI already carries connection/key/pin plumbing and `respond` set the
  long-running-mode precedent. Split-out becomes worthwhile with digest scheduling,
  not before.
