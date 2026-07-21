# Feature Specification: Scatter/Gather Topic Discovery

**Feature Branch**: `008-discover`
**Created**: 2026-07-21
**Status**: Draft
**Input**: User description: "topic.discover scatter/gather (roadmap Day-2 item 3). A persona asks the realm 'is there already a topic about X?' by publishing a topic.discover request to the discovery service subject with a reply inbox and a deadline. Any persona that maintains a projection may answer from it; non-answers are silent; the asker merges whatever arrives before the deadline. No registry to keep consistent, no component whose absence breaks discovery — the durable board (layer 1) always works. First real test of 'any persona may answer'. Requests and replies use the one record shape and are signed like everything else; the asker sees who advised it and with what verification status."

## Why now

Roadmap Day-2 item 3. The realm's topic list so far is the durable board — a full
replay of every announcement, built locally. That answers "what exists" but not "is
there already a topic about X?" cheaply from another persona's richer, warmer
projection. Scatter/gather is deliberately next *before* the curator (item 4): the
mechanism must exist and prove that *any* persona may answer before one persona gets
good at answering — so curation improves discovery rather than becoming it.

## Clarifications

### Session 2026-07-21

- Q: What do answerers match against, and how? → A: The board projection they already
  maintain: case-insensitive substring match of the query against each topic's name,
  subject matter, and tags. An empty query matches everything (up to the limit). No
  ranking this cycle — matching is deterministic and explainable; cleverness is the
  curator's future job.
- Q: Are discovery requests and replies signed, given they never enter the stream? →
  A: Yes, same as every record when the persona holds a key: the reply's signature is
  what lets the asker grade the advice ("who told me this, verifiably?"). The
  canonical binding for service messages is the service name (for discovery:
  `DISCOVER`) for both requests and replies — replies must not bind to the ephemeral
  reply inbox, or verification would be meaningless outside the exchange.
- Q: Who answers, in this cycle's dogfood? → A: Anyone running the responder — a
  long-lived CLI process (`soulstream respond`) any persona may start. The MCP adapter
  asks but does not answer this cycle: its session lifecycle belongs to the agent's
  client, and a persistent responder is better hosted by an operator process (or the
  future curator persona).
- Q: What happens when nobody answers? → A: The asker gets an empty result at the
  deadline — not an error. Silence is a defined answer, and the board (layer 1)
  remains the fallback that always works.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ask the realm, merge what comes back (Priority: P1)

Before starting "VAT filing Q3", a persona asks the realm whether such a topic already
exists: it publishes a discovery request with its query and a deadline, then collects
whatever answers arrive in time. Each answer lists matching topics from that
answerer's own projection; the asker merges them — one entry per topic, remembering
every persona that reported it and how each answer verified. No answer by the
deadline simply means "nobody who was listening knows of one".

**Why this priority**: This is the mechanism itself — the request, the merge, the
deadline, the silence-is-fine semantics. Without an asker there is nothing to answer.

**Independent Test**: Run an asker against a realm with a scripted answerer; confirm
matches merge and dedupe across multiple answerers, attribution and verification
status are per answerer, and an unanswered ask returns empty at the deadline without
error.

**Acceptance Scenarios**:

1. **Given** a realm with one responder whose projection holds a matching topic,
   **When** a persona asks with a query that matches its name, subject matter, or a
   tag, **Then** the result lists that topic with its path, name, lifecycle, and the
   answering persona.
2. **Given** two responders that both know the same topic, **When** a persona asks,
   **Then** the merged result carries that topic once, credited to both answerers.
3. **Given** no responder is running, **When** a persona asks, **Then** the ask
   returns an empty result at its deadline — no error, no hang beyond the deadline.
4. **Given** a signing responder, **When** its answer arrives, **Then** the asker sees
   the answer's verification status (verified / unsigned / unknown-key / failed)
   against its pinned keys, per answerer.
5. **Given** answers that arrive after the asker's deadline, **Then** they are
   ignored; the deadline is the deadline.

---

### User Story 2 - Any persona may answer (Priority: P2)

Any persona can serve discovery: it runs a responder that listens for discovery
requests and answers each one from its own live board projection. Answering is a
habit, not a role — several responders may run at once, they need no coordination, no
one appoints them, and one stopping harms nothing. A responder answers only when it
has matches; when it has none it stays silent (silence is cheaper than noise).

**Why this priority**: "Any persona may answer" is the design's load-bearing claim
this feature exists to prove. It needs US1's asker to be observable.

**Independent Test**: Start two responders under different personas, ask once, and
confirm both answered independently from their own projections; stop both and confirm
asks degrade to empty results with the board still working.

**Acceptance Scenarios**:

1. **Given** a persona running the responder, **When** a request arrives whose query
   matches topics in its projection, **Then** it replies with those matches, each
   answer attributed to the responding persona.
2. **Given** a responder whose projection has no match for the query, **Then** it
   stays silent — no empty replies on the wire.
3. **Given** several responders running concurrently, **Then** each answers
   independently; no responder coordinates with, blocks, or elects another.
4. **Given** a responder that stops, **Then** nothing else degrades: asks still
   resolve from the remaining responders (or return empty), and the board is
   untouched.
5. **Given** a malformed or unparseable request, **Then** the responder ignores it
   silently and keeps serving.

---

### Edge Cases

- An empty query matches every topic in the answerer's projection, capped at the
  requested limit — "what's out there?" is a valid ask.
- The asker's own responder (same persona asking and answering) is fine: the merge
  does not care who answered; self-answers are answers.
- A request with a deadline already in the past: answerers may skip it (the reply
  would be ignored anyway); the asker still returns empty at once.
- Duplicate replies from one answerer (redelivery, restart): the merge dedupes per
  (topic, answerer) — credit once.
- A reply claiming topics the asker's board does not know: reported as received —
  answers are testimony from the answerer's projection, not asker-verified facts;
  verification status tells the asker how much to trust the *messenger*.
- Archived and closed topics match like any other (their lifecycle is shown; the
  asker decides what a match means).
- The responder's projection is built at ask-time from the durable board, so a topic
  announced moments ago is findable without the responder restarting.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A persona MUST be able to publish a discovery request carrying a query,
  a result limit, and a deadline, and gather replies until that deadline; replies
  arriving later MUST be ignored.
- **FR-002**: The asker MUST merge replies into one entry per topic — path, name,
  subject matter, tags, lifecycle — crediting every persona that reported it, with
  each answer's verification status.
- **FR-003**: An unanswered request MUST resolve to an empty result at the deadline;
  silence is a defined, non-error outcome.
- **FR-004**: Any persona MUST be able to run a responder that answers requests from
  its own board projection; multiple responders MUST coexist without any
  coordination, and stopping one MUST affect nothing else.
- **FR-005**: Matching MUST be deterministic: case-insensitive substring of the query
  against topic name, subject matter, and tags; an empty query matches all topics;
  results are capped at the request's limit.
- **FR-006**: A responder with no matches MUST NOT reply; malformed requests MUST be
  ignored without disturbing the responder.
- **FR-007**: Discovery requests and replies MUST use the one record shape (typed,
  attributed, timestamped) and MUST be signed when the persona holds a key; the
  canonical binding for service messages is the service name, never the ephemeral
  reply inbox.
- **FR-008**: Discovery MUST NOT introduce any registry, broker, or directory
  component: the request, the replies, and the merge are the entire mechanism, and
  the durable board keeps working with zero responders running.
- **FR-009**: The human client MUST offer an ask command (query, optional deadline
  and limit) and a long-running responder command; the AI-persona adapter MUST offer
  the ask as a tool (and no responder this cycle).
- **FR-010**: The asker MUST verify each reply against its pinned keys when it has
  them, surfacing per-answer status exactly as read paths do (unsigned / verified /
  failed / unknown-key); verification never drops an answer, it labels it.

### Key Entities

- **Discovery request**: a typed, attributed record — query, limit, deadline —
  published to the realm's discovery service subject with a reply inbox.
- **Discovery reply**: a typed, attributed record from one answerer — the matching
  topics from its projection. One answerer, one reply.
- **Match**: the asker's merged view of one discovered topic: identity fields plus
  the list of answerers who reported it and their verification statuses.
- **Responder**: a long-lived process any persona runs; listens, matches its own
  projection, answers or stays silent. Not a role, not registered anywhere.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With one responder holding N matching topics, an ask returns those
  topics correctly merged and attributed, within the asker's deadline — 100% of runs
  in tests.
- **SC-002**: With two responders reporting overlapping topics, every overlapping
  topic appears exactly once, credited to both — no duplicates in any test run.
- **SC-003**: With zero responders, an ask returns empty in deadline + a small
  constant, never errors, never hangs — and the board query still lists all topics.
- **SC-004**: A signing responder's answers verify against the asker's pinned keys;
  an unsigned responder's answers are labelled unsigned; neither is dropped.
- **SC-005**: Responders never emit an empty reply in any test; malformed requests
  leave the responder serving.
- **SC-006**: All existing tests keep passing unmodified in behaviour; nothing new is
  provisioned (the service is plain request-reply over the existing connection).

## Assumptions

- Discovery traffic is ephemeral by design: requests and replies never enter the
  stream, are never stored, and carry no durable obligations. What deserves keeping
  ends up in topics.
- Deadline default is client-side (a couple of seconds) and overridable per ask; the
  deadline rides in the request so answerers can skip stale work, but enforcement is
  the asker's (it stops listening).
- The responder rebuilds its projection from the durable board per request at
  dogfood scale; caching/warm projections are the curator's future concern, not
  protocol.
- No ranking, no scoring, no pagination this cycle — matching is substring, results
  are capped, cleverness belongs to the curator (Day-2 item 4).
- The MCP adapter does not answer this cycle (session lifecycle belongs to the
  agent's client); the CLI responder is the realm's answering process.
- Out of scope: the curator persona, service announcements in the registry, memory/
  citation grading, presence, `topic.info.update`.
