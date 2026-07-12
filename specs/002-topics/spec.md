# Feature Specification: Topics — the Op-Log Engine

**Feature Branch**: `002-topics`
**Created**: 2026-07-12
**Status**: Draft
**Input**: User description: "The topics op-log engine on top of the foundation: announce a topic with an initial baseline, hold a conversation (turn.post, comment.add), move it through its lifecycle (proposed→active→closed), organise with sub-topics, materialise a topic by replaying its ops in stream order, and discover the realm's topics from the info board. Deferred: mentions and attachments (next cycle), and rollup/edit/reply/resolve/dormant/archived/scatter-gather (day-2)."

## Overview

The foundation gave us a realm and a well-formed operation record. This feature makes those
operations *mean* something: it turns a stream of records into **topics** — the shared
workbenches personas collaborate on.

A topic is an append-only **op-log** on its own subject. Starting a topic publishes an
announcement (so others can find it) and an initial baseline (the zero-point its operations build
on). Personas then contribute operations — a turn in the conversation, a comment anchored to an
earlier op, a lifecycle transition. Anyone can rebuild the current state of a topic by replaying
its log; anyone can list the realm's topics by replaying the info board. No coordinator, no
database — just deterministic replay of the log the foundation already stores.

The consumers are **library integrators** building clients and agents. The value: a persona can
start a topic, converse in it, move it through its life, nest focused sub-topics, see the whole
thing materialise the same way for everyone, and discover what topics exist — all from the stream
alone.

## Clarifications

### Session 2026-07-12

- Q: In MVP, how is the order of operations within a topic decided? → A: **By JetStream stream
  sequence** — the total order the server already assigns. The DAG (`Soulstream-Parents`) is
  recorded on every op but is **not consulted** for ordering yet; the merge algorithm (eg-walker)
  is day-2. Rare concurrent ops may therefore render in a different order than a future CRDT would
  choose — acceptable for conversation, and stated as a known limitation.
- Q: How does a persona contribute an operation — does the library set parents? → A: The library
  offers a **topic handle** bound to a client + topic; posting an op through it stamps the author,
  fills `Soulstream-Parents` with the current frontier the handle has observed (from its
  materialised view), generates the op-id, and publishes to the topic's OPS subject. The caller
  supplies only intent (type + payload + optional anchor).
- Q: What exactly does "materialise a topic" produce? → A: A **materialised view**: the topic's
  announcement metadata, its baseline state, the lifecycle state (derived), an ordered list of
  contributions (turns and comments with author, timestamp, body, and anchor), and the current
  frontier (leaf op-ids). It is a pure projection of the log; two replicas replaying the same log
  produce the same view.
- Q: How is lifecycle state derived in MVP? → A: **Deterministically from the log.** `proposed` =
  announced with only the initial baseline (no content ops). `active` = at least one content op
  after the baseline. `closed` = a `life.transition` to `closed` is present. `dormant` and
  `archived` are out of scope (no idle automation, no re-baselining) this cycle.
- Q: How do sub-topics work on the wire? → A: A sub-topic is announced with `parent` set to its
  parent's topic-id and lives at `SOULSTREAM.TOPICS.OPS.<parent>.<child>` (and INFO likewise),
  nesting by subject depth with no new machinery. Its topic-path is the dotted chain; it
  materialises independently by replaying its own subject.
- Q: Does the board implement info-update rollup in MVP? → A: **No.** The board projection replays
  `SOULSTREAM.TOPICS.INFO.>` and keeps, per info subject, the **latest** announcement by stream
  sequence. Announcement revisions via `topic.info.update` + `Nats-Rollup: sub` are deferred; MVP
  publishes one `topic.announce` per topic and the projection tolerates (takes the last of)
  duplicates.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start a topic and discover it (Priority: P1)

A persona starts a topic: the library publishes the announcement to the topic's info subject and
an initial baseline to its ops subject. Any persona can then replay the info board and see the new
topic listed with its name, subject matter, tags, and lifecycle state.

**Why this priority**: A topic that can't be started and found doesn't exist. This is the entry
point for every other scenario and the smallest end-to-end slice (announce → discover).

**Independent Test**: Announce a topic against an embedded server; replay the info board and
confirm the topic appears with its metadata and a `proposed` state; confirm the ops subject holds
exactly the baseline as its first message.

**Acceptance Scenarios**:

1. **Given** a provisioned realm, **When** a persona starts a topic with a name, subject matter,
   and tags, **Then** the info subject carries a `topic.announce` and the ops subject's first
   message is the initial `baseline`.
2. **Given** one or more started topics, **When** a persona builds the discovery board, **Then**
   every topic appears once with its current announcement metadata and derived lifecycle state.
3. **Given** a topic-id collision is impossible by construction (random suffix), **When** two
   personas start topics with the same display name, **Then** they are distinct topics with
   distinct ids, both discoverable.
4. **Given** a well-behaved persona, **When** it starts a topic, **Then** the topic-id is a
   readable slug with a short random suffix and the display name lives in the announcement, not the
   id.

### User Story 2 - Hold a conversation and materialise it (Priority: P1)

Personas post turns and comments to a topic. Anyone can replay the topic's ops subject and
materialise the same view: the baseline, then every contribution in a single agreed order, with a
comment shown against the op it anchors to.

**Why this priority**: This is the workbench actually being used. Materialisation is the heart of
the engine — the guarantee that everyone sees the same topic from the same log.

**Independent Test**: Post a baseline, several turns, and a comment anchored to one turn; replay
and materialise; assert the contributions appear in stream order, the comment records its anchor,
and the lifecycle is `active`.

**Acceptance Scenarios**:

1. **Given** a started topic, **When** a persona posts a `turn.post`, **Then** the op is published
   to the topic's ops subject with the author stamped and parents set to the frontier the poster
   had observed.
2. **Given** a topic with several ops, **When** any persona materialises it by replaying from the
   start, **Then** the result is the baseline followed by the contributions in stream-sequence
   order, identical for every replica.
3. **Given** a `comment.add` anchored to an earlier op's id, **When** the topic is materialised,
   **Then** the comment is associated with the anchored op, and it survives even as later ops are
   added.
4. **Given** a topic with only its initial baseline, **When** materialised, **Then** its lifecycle
   is `proposed`; after the first content op, `active`.
5. **Given** a cold consumer that has never seen the topic, **When** it replays from the subject's
   start, **Then** the first message is the baseline and the tail applies cleanly on top.

### User Story 3 - Follow a topic live (Priority: P1)

A persona subscribes to a topic and receives new operations as they are published, updating its
materialised view incrementally without re-replaying from the start.

**Why this priority**: Collaboration is live. A human's client and an agent both need to react to
new ops as they land, not by polling a cold replay. This is what makes a topic a *shared* bench in
real time.

**Independent Test**: Open a live view of a topic; from another connection, post a turn; assert the
follower's view updates to include the new turn without a full re-replay, and its frontier advances.

**Acceptance Scenarios**:

1. **Given** a materialised topic view, **When** a new op is published by anyone, **Then** the live
   follower applies it incrementally and its frontier advances to the new op.
2. **Given** a follower that connects after some ops exist, **When** it starts following, **Then**
   it first materialises the existing log, then receives subsequent ops live, with no gap and no
   duplicate at the seam.
3. **Given** a follower, **When** it posts its own op through its handle, **Then** its parents
   reflect the frontier it has actually observed at post time.

### User Story 4 - Move a topic through its lifecycle (Priority: P2)

A persona transitions a topic from `proposed` to `active` (implicitly, by the first content op) and
explicitly to `closed`. The materialised view reflects the current lifecycle state, and closing is
a recorded operation, not a special mechanism.

**Why this priority**: Topics have a life; "this is finished" needs to be expressible and visible.
It is P2 because conversation (P1) is usable before formal closing exists.

**Independent Test**: Start a topic (`proposed`), post a turn (`active`), post a `life.transition`
to `closed`; materialise after each step and assert the derived state; assert two concurrent
`closed` transitions converge to the same state.

**Acceptance Scenarios**:

1. **Given** an active topic, **When** a persona posts a `life.transition` to `closed`, **Then**
   materialising the topic yields lifecycle `closed`.
2. **Given** a topic, **When** two personas independently post a `closed` transition, **Then** the
   materialised state is `closed` either way — idempotent, no conflict to resolve.
3. **Given** a `closed` topic, **When** the library is asked to post a content op to it, **Then** it
   warns that the topic is closed (closed is not-writable *by convention*; the library surfaces the
   state rather than hard-blocking, since enforcement is social).
4. **Given** an invalid transition request (e.g., a state the MVP does not define), **When** it is
   attempted, **Then** the library rejects it with a clear error naming the allowed transitions.

### User Story 5 - Organise with sub-topics (Priority: P2)

A persona announces a sub-topic under an existing topic. It lives at a nested subject, materialises
independently, and appears on the board under its parent.

**Why this priority**: Focused threads without fragmentation is a real need, but a realm is usable
with flat topics first, so this is P2.

**Independent Test**: Announce a parent topic, then a sub-topic with `parent` set; confirm the
sub-topic's ops/info subjects are nested under the parent's id; materialise the sub-topic
independently; confirm the board shows the parent/child relationship.

**Acceptance Scenarios**:

1. **Given** an existing topic, **When** a persona announces a sub-topic with `parent` set to the
   parent's id, **Then** the sub-topic's ops and info subjects are `…OPS.<parent>.<child>` and
   `…INFO.<parent>.<child>`.
2. **Given** a sub-topic, **When** it is materialised, **Then** it replays only its own subject and
   yields its own independent view.
3. **Given** a parent with sub-topics, **When** the board is built, **Then** each sub-topic is
   discoverable and its parent relationship is visible.
4. **Given** arbitrarily deep nesting, **When** sub-topics are announced under sub-topics, **Then**
   no protocol change is needed — depth is just subject depth.

### Edge Cases

- **Ops before baseline**: a topic's ops subject must begin with the baseline; if a replay finds a
  non-baseline first message, materialisation reports a malformed topic rather than guessing.
- **Unknown op type**: an op whose type this cycle does not define is **ignored with a warning**
  during materialisation, never fatal — vocabularies grow additively.
- **Comment anchored to an unknown op-id**: the comment still materialises (its anchor is recorded)
  but is flagged as dangling; it is not dropped.
- **Concurrent turns**: two turns posted concurrently (same parent frontier) are ordered by stream
  sequence; both appear; the render order is deterministic though not necessarily causal (known
  MVP limitation).
- **Announcing a sub-topic under a non-existent parent**: allowed on the wire (subjects are free),
  but the board flags the parent as unknown rather than failing.
- **Posting to a closed topic**: surfaced as a warning, not blocked (US4 scenario 3).
- **Empty board**: replaying an info board with no topics yields an empty projection, not an error.
- **Very large baseline**: out of scope this cycle — MVP baselines are inline and small; manifest
  (chunked) baselines are day-2. A baseline exceeding the inline threshold is rejected with a clear
  error pointing to the deferred capability.

## Requirements *(mandatory)*

### Functional Requirements

**Publishing operations**

- **FR-001**: The library MUST provide a topic handle bound to a client and a topic-path, through
  which a persona posts operations to the topic's ops subject `SOULSTREAM.TOPICS.OPS.<topic-path>`.
- **FR-002**: When posting an op, the library MUST stamp the author (the client's persona),
  generate the op-id, set the timestamp, and populate `Soulstream-Parents` with the frontier the
  handle has observed; the caller supplies only the op type, payload, and optional anchor.
- **FR-003**: The library MUST enforce write-side attribution (a persona-bound client posts only as
  itself) before publishing, reusing the foundation's guard.

**Starting a topic**

- **FR-004**: Starting a topic MUST publish a `topic.announce` to the topic's info subject
  `SOULSTREAM.TOPICS.INFO.<topic-path>` carrying at least: topic-id, display name, subject matter,
  tags, expected personas (a hint only), and parent (null for a top-level topic).
- **FR-005**: Starting a topic MUST publish an initial `baseline` as the **first** message on the
  topic's ops subject; from birth the ops subject's shape is baseline-first, ops-after.
- **FR-006**: The library MUST generate topic-ids as readable slugs with a short random suffix,
  unique without coordination; the display name lives in the announcement, not the id.
- **FR-007**: `expected` personas MUST be treated as a hint for clients, never a posting gate;
  posting rights are subject permissions only.

**Vocabulary**

- **FR-008**: The library MUST support these operation types this cycle: `topic.announce` (info),
  `baseline`, `turn.post`, `comment.add`, `life.transition`. Types not in this set MUST be ignored
  with a warning on materialisation (additive growth).
- **FR-009**: A `comment.add` MUST carry an anchor to another op by its op-id, so the comment stays
  attached to the right contribution as the topic evolves.
- **FR-010**: A `baseline` MUST carry the materialised state and a `frontier` (the leaf op-ids at
  baseline time); at birth the frontier is empty and first ops parent onto the baseline.

**Materialisation**

- **FR-011**: The library MUST materialise a topic by replaying its ops subject from the start:
  the first message is the baseline; the tail is applied in **stream-sequence order**.
- **FR-012**: Ordering MUST be by JetStream stream sequence this cycle; the DAG (`Soulstream-Parents`)
  MUST be recorded on every op but MUST NOT be consulted for ordering (eg-walker is deferred).
- **FR-013**: The materialised view MUST include: the announcement metadata, the baseline state,
  the derived lifecycle state, an ordered list of contributions (turns and comments with author,
  timestamp, body, and anchor), and the current frontier (leaf op-ids).
- **FR-014**: Materialisation MUST be a pure function of the log: two replicas replaying the same
  log MUST produce the same view.
- **FR-015**: If the first message on an ops subject is not a baseline, materialisation MUST report
  a malformed topic rather than guess.
- **FR-016**: A `comment.add` whose anchor op-id is not present MUST still materialise (anchor
  recorded) but be flagged as dangling, never dropped.

**Live following**

- **FR-017**: The library MUST support following a topic: after an initial materialisation, it MUST
  apply subsequently-published ops incrementally, advancing the frontier, with no gap or duplicate
  at the seam between replay and live.
- **FR-018**: A handle's posted parents MUST reflect the frontier it has actually observed at post
  time (from its current view).

**Lifecycle**

- **FR-019**: Lifecycle MUST be derived from the log: `proposed` (baseline only), `active` (≥1
  content op after baseline), `closed` (a `life.transition` to `closed` present). `dormant` and
  `archived` are out of scope this cycle.
- **FR-020**: A `life.transition` MUST be an ordinary op on the topic's ops subject; two concurrent
  identical transitions MUST converge to the same state (idempotent), with no arbiter.
- **FR-021**: The library MUST reject a `life.transition` to a state the MVP does not define, naming
  the allowed transitions.
- **FR-022**: Posting a content op to a `closed` topic MUST be surfaced as a warning (closed is
  not-writable by convention), not hard-blocked — enforcement is social, recorded as ops.

**Sub-topics**

- **FR-023**: A sub-topic MUST be announced with `parent` set to its parent's topic-id and live at
  `SOULSTREAM.TOPICS.OPS.<parent>.<child>` (and INFO likewise), nesting arbitrarily by subject
  depth with no new machinery.
- **FR-024**: A sub-topic MUST materialise independently by replaying only its own subject.

**Discovery**

- **FR-025**: The library MUST build a discovery board by replaying `SOULSTREAM.TOPICS.INFO.>` and
  keeping, per info subject, the latest announcement by stream sequence; the board lists every
  topic once with its metadata and (where derivable) lifecycle state.
- **FR-026**: The board MUST expose the parent relationship of sub-topics; a sub-topic whose parent
  is unknown MUST be flagged, not dropped.
- **FR-027**: Building the board on a realm with no topics MUST yield an empty projection, not an
  error.

**Discipline & non-goals**

- **FR-028**: A baseline exceeding the realm's inline threshold MUST be rejected with a clear error
  pointing to the deferred manifest-baseline capability; MVP baselines are inline.
- **FR-029**: This cycle MUST NOT implement: mentions/notify, attachments, `edit`,
  `comment.reply`/`comment.resolve`, rollup/re-baselining, manifest baselines, `dormant` automation,
  `archived`, scatter/gather discovery, or eg-walker merge. Those are later cycles/day-2.

### Key Entities *(include if feature involves data)*

- **Topic**: a shared workbench identified by a topic-path (dotted chain of topic-ids). Has an info
  subject (its announcement) and an ops subject (its op-log). State = baseline + ordered ops.
- **Topic handle**: a live binding of a client to one topic-path; the thing a persona posts through
  and follows. Tracks the observed frontier.
- **Announcement**: the topic's info record — id, display name, subject matter, tags, expected
  personas, parent.
- **Baseline**: the first op on a topic's ops subject; carries materialised state + frontier; the
  zero-point ops build on. Inline this cycle.
- **Contribution**: a materialised turn or comment — author, timestamp, body, op-id, and (for
  comments) an anchor op-id, possibly flagged dangling.
- **Materialised view**: the pure projection of a topic's log — announcement + baseline state +
  lifecycle + ordered contributions + frontier.
- **Discovery board**: the projection of `TOPICS.INFO.>` — one entry per topic with metadata,
  parent relationship, and lifecycle where derivable.
- **Lifecycle state**: `proposed` | `active` | `closed` (this cycle), derived from the log.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A persona can start a topic and, from a separate consumer, discover it on the board
  with correct metadata — verified end to end against a running server.
- **SC-002**: 100% of materialisations of the same log produce an identical view (byte/field-equal
  contributions in the same order), across repeated and independent replays.
- **SC-003**: A comment anchored to an op is shown against that op in the materialised view in 100%
  of cases, and a dangling-anchored comment is flagged (never dropped).
- **SC-004**: A live follower reflects a newly-posted op without re-replaying the whole log, with no
  gap or duplicate at the replay/live seam — verified by posting from a second connection.
- **SC-005**: Lifecycle state is correctly derived (`proposed`/`active`/`closed`) for every defined
  case, and two concurrent `closed` transitions converge to `closed`.
- **SC-006**: A sub-topic announced with a parent lives at the nested subject and materialises
  independently; the board shows its parent relationship — verified against a running server.
- **SC-007**: Unknown op types and malformed topics are handled per spec (ignored-with-warning;
  reported-not-guessed) with no panic, and the whole engine verifies green — all tests pass (none
  skipped), formatting applied, linting clean.

## Assumptions

- **Foundation in place**: `001-foundation` is merged — realm provisioning, the operation record
  (wire + canonical), and identity/attribution are available and used here.
- **Short topics**: MVP topics are short-lived and small; full logs are replayed (no rollup), and
  baselines are inline. This is explicitly a throwaway-realm assumption from the roadmap.
- **Single order authority**: stream sequence is the ordering authority this cycle; causal/CRDT
  merge is deferred, and the resulting rare reordering of concurrent ops is acceptable for
  conversation.
- **Convention over enforcement**: closed-topic write protection and `expected` membership are
  conventions the library surfaces, not gates it enforces; posting rights are NATS subject
  permissions.
- **Payloads are JSON**: op payloads are small JSON documents (consistent with the canonical
  record's data field).
- **Deferred participation**: mentions and attachments — though part of the overall MVP — are the
  next cycle (`003-participation`); this cycle deliberately stops at conversation + structure so
  each cycle stays small and independently testable.

## Dependencies

- `001-foundation` (merged): `record` (Build/Parse/Canonical, op-ids), `identity` (names,
  attribution), `realm` (connect + provisioned stream/object store), and `internal/natstest`.
- A reachable/embedded NATS server with JetStream for the replay, live-follow, and discovery
  scenarios.
