# Feature Specification: The Curator Persona

**Feature Branch**: `009-curator`
**Created**: 2026-07-21
**Status**: Draft
**Input**: User description: "The curator persona (roadmap Day-2 item 4, extensions/curation.md). An active realm accumulates near-duplicate topics, drift, and noise; what remains after the deterministic core is judgment, and judgment belongs to personas. A curator is an ordinary persona — ordinary credentials, ordinary operations, zero protocol standing — that maintains a high-quality topic projection, answers topic.discover from it (the best responder, never the only one), flags likely duplicates with a comment in the newer topic, and proposes closing long-dormant topics with a comment in place. A curator suggests, never enforces. Run none and the realm still works; run one, run two competing ones, or replace it any time."

## Why now

Roadmap Day-2 item 4, explicitly sequenced *after* scatter/gather (008) so that
curation improves discovery rather than becoming it: the answer mechanism exists and
works with zero curators, so a curator can only ever make it better. The dogfood
realm is accumulating topics; the noise this feature addresses is starting to be
real.

## Clarifications

### Session 2026-07-21

- Q: What makes the curator the *best* discovery responder, concretely? → A: Two
  things the plain responder lacks: a **warm projection** (one continuous
  subscription over the realm's topic subjects, history then live — no per-request
  replay) and **content-aware matching** (the query also matches what was *said* in
  topics — turn and comment bodies — not just announcement metadata). Same request,
  same reply shape, same merge: a curator's answers simply cover more.
- Q: How does a curator avoid nagging (re-flagging the same duplicate, re-proposing
  the same closure, or two curators not stepping on each other)? → A: Suggestions
  are ordinary comments in the topic, so the log itself is the memory: a curator
  flags a duplicate only if no curator suggestion of that kind is already present,
  and proposes dormancy only if no such proposal has been made since the topic's
  last real activity. Deterministic, restart-safe, and naturally cooperative — two
  curators see each other's comments the same way.
- Q: What counts as dormant, given the curator's own comments are also ops? → A: A
  topic is dormant when its newest op *that is not a curator suggestion* is older
  than the idle window (operator-configured, default 14 days). Curator chatter never
  keeps a topic "alive", and a proposal never re-arms itself.
- Q: What does 009 cut from the extension's typical-behaviours list? → A: Digests.
  They serve mention-only readers, a habit the dogfood realm doesn't have yet, and
  they need scheduling machinery worth its own decision. Deferred with the rest of
  the day-2 tail; the other three behaviours are the value.
- Q: Where does the curator run? → A: A long-running mode of the existing CLI
  (`soulstream curate`), under whatever persona its operator chooses — ordinary
  credentials, ordinary signing, ordinary keyring. No new binary, no registration
  anywhere.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The best answerer in the room (Priority: P1)

An operator starts a curator under an ordinary persona. It builds a warm projection
of every topic — announcements *and* conversation content — and keeps it live. From
then on it answers discovery asks from that projection: a query matching only
something *said inside* a topic (not its name or tags) still finds the topic. Askers
see nothing new — same mechanism, same merge — their results are just better. Stop
the curator and discovery keeps working exactly as before (008's plain responders
and the board).

**Why this priority**: This is why the curator exists at this point in the roadmap —
making the just-built mechanism visibly better while proving it stays optional.

**Independent Test**: Ask for a phrase that appears only in a turn body; with the
curator running the topic is found and credited to the curator persona; with it
stopped, the same ask returns what plain responders can offer (or silence), and the
board still lists everything.

**Acceptance Scenarios**:

1. **Given** a running curator and a topic whose *name* matches a query, **When**
   someone asks, **Then** the curator answers it like any responder.
2. **Given** a topic where a phrase appears only in a turn or comment body, **When**
   someone asks for that phrase, **Then** the curator's answer includes that topic —
   content the plain responder cannot see.
3. **Given** a topic announced or posted to *after* the curator started, **When**
   someone asks for its content, **Then** it is found — the projection is live, not
   a snapshot.
4. **Given** the curator and a plain responder both running, **When** someone asks,
   **Then** both answer independently and the merge credits each — the curator is
   the best responder, never the only one.
5. **Given** the curator stopped, **When** someone asks, **Then** discovery behaves
   exactly as in 008 — nothing depended on the curator.

---

### User Story 2 - "These two look the same" (Priority: P2)

When a topic is started that looks like an existing one — names, subject matter, or
tags substantially overlapping — the curator posts one comment in the *newer* topic
naming the older one and suggesting the conversation continue there. It is a
suggestion in plain sight: an ordinary, attributed (and signed, when keyed) comment
that anyone can ignore, argue with, or act on. The curator never merges, closes, or
touches either topic beyond that one comment — and it never repeats itself.

**Why this priority**: Near-duplicates are the first real noise an active realm
accumulates; a visible, polite flag is the smallest useful judgment.

**Independent Test**: Start a topic, then start a near-duplicate; the curator
comments once in the newer topic naming the older path; restarting the curator (or
running a second one) adds no second flag.

**Acceptance Scenarios**:

1. **Given** an existing topic and a newly started near-duplicate, **When** the
   curator notices it, **Then** a comment appears in the *newer* topic naming the
   older topic's path — and nothing else changes anywhere.
2. **Given** the curator already flagged a topic as a likely duplicate, **When** the
   curator restarts or re-scans, **Then** no second flag is posted.
3. **Given** two curators running, **When** one has flagged a duplicate, **Then**
   the other sees that suggestion in the log and stays quiet.
4. **Given** two topics that are clearly unrelated, **Then** no flag is posted —
   silence over noise.
5. **Given** a suggested-duplicate topic that participants keep using anyway,
   **Then** the curator does nothing further — it suggested, they decided.

---

### User Story 3 - "This one seems finished" (Priority: P3)

Topics that have sat idle past the realm's window get a gentle nudge: the curator
posts a comment in place proposing the topic be closed (or archived) if it is done.
The proposal is one comment, made once per quiet spell: new real activity resets the
clock, and a topic that goes quiet again may eventually get one new proposal. Closing
remains a persona's decision, posted as an ordinary op by whoever agrees.

**Why this priority**: Dormancy is slower-burning noise than duplicates, and the
nudge is only meaningful once topics have had time to go quiet.

**Independent Test**: With a short idle window, a topic with old activity gets
exactly one proposal comment; posting fresh content and letting it go quiet again
allows exactly one more; closed and archived topics get none.

**Acceptance Scenarios**:

1. **Given** a topic whose last real activity is older than the idle window, **When**
   the curator scans, **Then** one comment appears in that topic proposing closure.
2. **Given** a topic already carrying a dormancy proposal with no real activity
   since, **When** the curator scans again (or restarts), **Then** no further
   proposal is posted.
3. **Given** fresh real activity after a proposal, and then another quiet spell past
   the window, **When** the curator scans, **Then** exactly one new proposal may
   appear — the clock reset with the activity.
4. **Given** the curator's own suggestions are the only recent ops in a topic,
   **Then** the topic still counts as dormant — curator chatter keeps nothing alive.
5. **Given** a closed or archived topic, **Then** no proposal is posted (closed is
   already resting; archived refuses writes anyway).

---

### Edge Cases

- Zero curators: everything in the realm works exactly as before this feature — the
  defining property, inherited from the design's steward post-mortem.
- Two curators racing to flag the same duplicate in the same instant: both comments
  may land — harmless, visible, and self-limiting (each backs off once it sees the
  other's). Exactly-once is deliberately not promised; at-most-rarely-twice is.
- The curator asking discovery itself (self-answering): fine, as in 008.
- A topic whose only content is its announcement (no ops beyond the baseline):
  eligible for dormancy like any other; its "last real activity" is its birth.
- The curator's projection encounters a malformed topic: it skips it for matching
  and never suggests anything in it.
- The curator persona lacking a signing key: its suggestions are unsigned ordinary
  comments — testimony-grade advice, like any unsigned persona's.
- An archived near-duplicate as the *older* topic: still named in the flag (its
  content is readable forever); the suggestion text stands on its own.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A curator MUST be an ordinary persona: ordinary credentials, ordinary
  signed/unsigned operations, zero protocol standing, no registration anywhere; any
  number MAY run concurrently and any MAY stop at any time with no effect on any
  core flow.
- **FR-002**: The curator MUST maintain a live projection of the realm's topics —
  announcements, lifecycle, and conversation content — built from the same messages
  any reader can replay (history, then live, no seam).
- **FR-003**: The curator MUST answer discovery requests from its projection using
  the existing mechanism unchanged (same request, same reply shape, same merge),
  and its matching MUST additionally cover conversation content.
- **FR-004**: The curator MUST flag a likely duplicate as one ordinary comment in
  the newer topic naming the older topic's path; likeness MUST be a deterministic,
  explainable rule over announcement metadata (name, subject matter, tags).
- **FR-005**: The curator MUST propose closure for a topic whose newest
  non-curator-suggestion op is older than the operator-configured idle window
  (default 14 days), as one ordinary comment in that topic.
- **FR-006**: Suggestions MUST be idempotent against the log itself: a duplicate
  flag is posted only if no curator duplicate-flag is already present in the topic;
  a dormancy proposal only if none exists since the topic's last real activity.
  This MUST hold across restarts and across multiple curators.
- **FR-007**: The curator MUST NOT enforce anything: no closing, no archiving, no
  rollup, no merging on its own initiative — comments are its entire vocabulary of
  action (it MAY compact nothing; rollup remains other personas' and lifecycle
  triggers' business).
- **FR-008**: Curator suggestions MUST be recognisable as such (a stable, visible
  convention in the comment body) so personas — and other curators — can tell
  suggestion from conversation.
- **FR-009**: Closed and archived topics MUST get no dormancy proposals; malformed
  topics MUST be skipped entirely.
- **FR-010**: The human client MUST offer the curator as a long-running command
  under the operator's chosen persona, with the idle window and scan cadence
  configurable; stopping it is Ctrl-C, nothing to deregister.

### Key Entities

- **Curator**: an ordinary persona running the curation habits; not a role, not a
  component.
- **Projection**: the curator's warm, live view of every topic — identity fields
  plus searchable conversation text plus last-real-activity time.
- **Suggestion**: an ordinary comment, marked by a stable convention, in one of two
  kinds: duplicate flag (names an older topic) and dormancy proposal.
- **Idle window**: the operator-configured quiet period after which a topic earns a
  dormancy proposal.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A query matching only conversation content finds its topic when the
  curator runs, and the answer is credited to the curator persona — 100% of test
  runs; with the curator stopped, the same ask degrades exactly to 008 behaviour.
- **SC-002**: A near-duplicate start earns exactly one flag in the newer topic
  across curator restarts and a second concurrent curator — proven in tests.
- **SC-003**: A dormant topic earns exactly one proposal per quiet spell; fresh
  activity re-arms exactly one more; curator-only chatter never resets the clock —
  proven in tests.
- **SC-004**: With zero curators, the full existing suite passes unmodified — the
  realm does not know curators exist.
- **SC-005**: All curator writes are ordinary attributed comments (signed when
  keyed) that render in every existing client with no client changes.

## Assumptions

- Duplicate likeness this cycle is token overlap over name + subject matter + tags
  with a fixed threshold — deterministic and explainable; content-based similarity
  and smarter scoring are future curator improvements, not spec obligations.
- The projection is in-memory and rebuilt on start by replay (cheap at dogfood
  scale); persistence/warm-start caches are deferred until replay cost is felt.
- One scan cadence (default 1 minute) drives duplicate and dormancy checks; the
  discovery answering is event-driven per request.
- The idle window default is 14 days; dogfood operators can shorten it.
- Digests, `dormant` as a lifecycle state, presence, and registry service
  announcements remain out of scope (deferred with rationale in Clarifications).
