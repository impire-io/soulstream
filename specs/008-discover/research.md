# Research: Scatter/Gather Topic Discovery

## R1. Transport shape

- **Decision**: plain core-NATS request-reply. Asker: `NewInbox()` +
  `SubscribeSync(inbox)` + `PublishMsg{Subject: SOULSTREAM.SVC.DISCOVER, Reply:
  inbox}` + `NextMsg(remaining)` loop until the deadline. Responder: plain
  `Subscribe(SOULSTREAM.SVC.DISCOVER)` (NO queue group), reply via
  `PublishMsg{Subject: msg.Reply}`. Nothing touches JetStream.
- **Rationale**: the design says NATS-micro style with a reply inbox and deadline;
  scatter/gather requires *every* responder to hear every request (a queue group
  would load-balance to one, defeating the merge). `nc.Request` is one-reply-only, so
  the manual inbox loop is the correct primitive for many replies.
- **Alternatives considered**: the `micro` package (adds service registration/stats
  machinery for a mechanism that must stay component-free); JetStream-backed requests
  (persistence for traffic that is ephemeral by design).

## R2. Vocabulary & payloads

- **Decision**: `topic.discover` request `{query, limit, deadline}`;
  `topic.discover.reply` reply `{matches: [{path, name, subject_matter, tags,
  lifecycle}]}`. One reply per responder per request, and only when matches exist.
- **Rationale**: additive vocabulary, one record shape for the whole system
  (design doc); deadline rides in the payload so answerers can skip stale work,
  while enforcement stays the asker's (it just stops listening).
- **Alternatives considered**: ack-style empty replies ("I heard you, nothing
  matches") — rejected: silence is cheaper than noise and the spec makes silence a
  defined answer.

## R3. Canonical binding for service messages

- **Decision**: service records bind to the **service name**: `canonicalBinding`
  gains the `SOULSTREAM.SVC.` prefix case (suffix = `DISCOVER`), and *replies* sign
  over the same binding even though they travel on an `_INBOX.*` subject. The record
  build in `wire.go` is factored so a caller can supply an explicit binding when the
  publish subject is not the binding source (exactly one caller: the discovery
  reply).
- **Rationale**: binding to an ephemeral inbox would make reply signatures
  meaningless outside the exchange; binding to the service name says exactly what
  the signer meant: "this is my discovery answer in this realm". The asker verifies
  with `VerifyRecord(rec, realm, "DISCOVER", keyring)` — same machinery as every
  read path.
- **Alternatives considered**: unsigned SVC traffic (throws away the 006 investment
  precisely where advice-grading starts to matter); binding to the request's op-id
  (verifier-side statefulness for no added protection at this trust level).

## R4. Matching & merging (pure)

- **Decision**: `matchEntries(entries []BoardEntry, query string, limit int)` —
  case-insensitive substring against name, subject matter, and each tag; empty query
  matches all; deterministic board order; capped at limit (default 10, capped at 50).
  `mergeReplies` folds replies into `map[path] → DiscoverResult`, appending one
  `{persona, sig}` credit per answerer (deduping repeat replies from the same
  persona), preserving first-seen entry fields.
- **Rationale**: deterministic and explainable beats clever (cleverness is the
  curator's job, Day-2 #4); pure functions keep the serverless-test convention.
- **Alternatives considered**: token/fuzzy scoring (unneeded at dogfood scale, and
  ranking disagreements across answerers would demand a merge policy this cycle
  doesn't want).

## R5. Responder projection

- **Decision**: rebuild via `Board(ctx, c)` per request.
- **Rationale**: always answers from fresh truth (a topic announced seconds ago is
  findable), zero cache-invalidation machinery; the board replay is trivially cheap
  at dogfood scale. The curator persona is the future home of warm/smart projections.
- **Alternatives considered**: a watched, cached projection (state and staleness
  logic for no current need — constitution II).

## R6. Client surface

- **Decision**: CLI `discover <query> [--timeout 2s] [--limit 10] [--json]` renders
  merged results with per-answerer sig glyphs and reuses the reader keyring; CLI
  `respond` runs the responder until Ctrl-C (persona-bound, signs when keyed). MCP
  `soulstream_discover` (11th tool): `{query, limit?}` with the default deadline;
  results include per-answer sig status and reuse the session keyring. No MCP
  responder this cycle (session lifecycle belongs to the agent's client).
- **Alternatives considered**: auto-falling back to a board scan inside `discover`
  when no replies arrive — rejected: conflating the layers hides which mechanism
  answered; `board` already exists one command away.
