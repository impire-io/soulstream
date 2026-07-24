# Feature Specification: Work Stages 1–2 — Versioned Artefacts & Work Items

**Feature Branch**: `010-work`
**Created**: 2026-07-21
**Status**: Shipped (v0.1.0)
**Input**: User description: "Work stages 1–2 from hq/02-DESIGN/extensions/work.md (Day-2 #5). Stage 1 — versioned artefacts: a document is a topic-anchored attachment revised whole-file; each revision is an attachment.add anchored to its predecessor revision's op-ID; an artefact is the resulting lineage — a named chain of revisions with full authorship history and a current tip determined by projection rule (same supersession mechanic as edit). No new machinery — existing ops only. Stage 2 — work items: a work-tracking vocabulary over ordinary ops — work.open, work.claim, work.done, work.abandon — tasks are conversations with status, attached evidence, and an owner. Any persona may publish work.claim; when two race, the first claim in stream order wins and later claims are void by projection — deterministic, no lock service, no arbiter. An abandoned claim reopens the item. Both stages are additive vocabulary over the existing op-log: no wire-format changes, no new subjects, no privileged services. Expose through the library (topic package projection + Handle methods), CLI, and MCP as appropriate."

## Why now

Roadmap Day-2 item 5. The dogfood scenario — a project run entirely in topics — needs
two things a topic does not yet name: a document that can be revised without losing
its history, and a task that can be claimed without a coordinator. Stage 1 costs
almost nothing (existing operations plus a projection rule); stage 2 is the first
vocabulary whose interesting property — first claim in stream order wins — exercises
the house coordination rule on something personas race for daily. Both are
prerequisites for stage 4 (executable workloads) later; neither needs stage 3 (live
co-editing), which stays gated on stage 1 demonstrably chafing in real use.

## Clarifications

### Session 2026-07-21

- Q: Does anchoring an attachment to a prior attachment *always* mean "revision",
  given anchors today mean "relates to"? → A: Yes — one deterministic rule, no
  flags: an attachment anchored to an attachment operation is a revision of it. A
  file that merely *relates to* an attachment is anchored to the surrounding
  conversation (the turn or comment discussing it) or left unanchored; the design
  doc's mechanism ("anchored to its predecessor's op-ID") is the whole mechanism.
- Q: If a revision anchors to a mid-chain member instead of the tip, is that a new
  artefact? → A: No — lineage identity is connectivity: anchoring to *any* member
  extends that member's lineage (same root). The tip stays "the lineage member
  latest in stream order", whichever member each revision anchored to. Anchoring
  mid-chain is just what a concurrent revision looks like after the race.
- Q: Are work operations and revisions "content" in the existing senses — do they
  activate a proposed topic, and do they count as real activity for dormancy
  observers? → A: Yes to both. They are ordinary contributions to the workbench;
  a proposed topic whose first op is a work.open becomes active, and curators
  already count any non-suggestion op as real activity — nothing special-cased.
- Q: How do claim/done/abandon reference their item, and what separates *malformed*
  from *void*? → A: They reference the opening operation's ID using the existing
  anchor convention (same shape as comments). Structurally broken ops (missing or
  empty item reference, unparseable payload) are **malformed** — skipped with a
  warning, like today. Structurally sound ops whose transition the state machine
  rejects (unknown item, claim on a claimed item, duplicate done…) are **void** —
  folded into the item's timeline with no state effect. Malformed = can't read it;
  void = read it, and it lost.
- Q: What does topic lifecycle do to work items? → A: Nothing. Closing a topic
  neither completes nor abandons its items (the close op warns writers, as today);
  archiving refuses all writes (existing rule) so item state freezes with the
  topic. No coupling in either direction — an open item never blocks a close.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A document that remembers (Priority: P1)

A persona attaches a draft to a topic. Later — maybe them, maybe someone else —
a revised version of the whole file is attached as a *revision of* the first. And
again. The topic now holds an **artefact**: one named document whose current content
is the newest revision and whose past is every revision that led there — who, when,
which bytes. Any reader materialising the topic sees the same current tip; nobody's
history disagrees. Fetching the artefact yields the tip; fetching any older revision
still works, because nothing was overwritten — only superseded.

**Why this priority**: Immediately useful in the dogfood realm (specs, designs, and
notes are already attachments); zero new machinery, so it is the cheapest real value
in the feature.

**Independent Test**: Attach a file, revise it twice from two different personas,
materialise the topic cold: one artefact, three revisions in order with correct
authors, tip = the third; fetching the artefact returns the third file's bytes;
fetching revision one still returns the original bytes.

**Acceptance Scenarios**:

1. **Given** a topic with an attachment, **When** a persona attaches a new whole
   file as a revision of it, **Then** readers see one artefact whose tip is the new
   file and whose history lists both revisions with their authors and times.
2. **Given** an artefact with several revisions, **When** any persona fetches it,
   **Then** they receive the tip's bytes; **When** they fetch a specific revision,
   **Then** they receive exactly that revision's bytes, integrity-checkable.
3. **Given** two personas who each revise the same tip concurrently, **When** the
   topic is materialised anywhere, **Then** every reader agrees on the same single
   tip (the revision later in stream order) and both revisions remain in history —
   nothing lost, no coordinator consulted.
4. **Given** an attachment added the ordinary way (no revision reference), **Then**
   it behaves exactly as before this feature — a standalone attachment, which is
   simply an artefact with one revision.
5. **Given** a topic compacted after several revisions, **When** a cold reader
   materialises it, **Then** the artefact's lineage — revision order, authors, tip —
   is identical to what a reader saw before compaction.

---

### User Story 2 - Claiming work without a lock service (Priority: P2)

A persona opens a work item in a topic: a task with a title and description, visible
to everyone materialising the topic. Any persona — human or AI — may claim it. When
two personas claim in the same instant, the substrate does not arbitrate: whichever
claim landed first in the stream owns the item, and the later claim is void — still
visible in history, changing nothing. The loser sees the item already owned and moves
on. When the owner finishes, they mark the item done, with evidence (results,
comments, attachments) added the ordinary ways. Status, owner, and the full trail are
derivable by anyone from the log alone.

**Why this priority**: This is the coordination pattern every later work stage builds
on (stage-4 runners claim execution items the same way); proving first-claim-wins
with zero new machinery is the heart of stage 2.

**Independent Test**: Open an item, have two personas claim it back-to-back,
materialise: the item is owned by the first claimant in stream order and the second
claim is void; the owner marks it done; the item shows done with the full timeline.

**Acceptance Scenarios**:

1. **Given** a topic, **When** a persona opens a work item, **Then** every reader
   sees it: open, unowned, titled, attributed to its opener.
2. **Given** an open item, **When** a persona claims it, **Then** readers see the
   item claimed and owned by that persona.
3. **Given** an open item and two racing claims, **When** the topic is materialised
   anywhere, **Then** the first claim in stream order owns the item and the later
   claim is void by projection — the same verdict on every replica, no arbiter.
4. **Given** a claimed item, **When** it is marked done, **Then** the item is done
   (terminally) and no further claims take effect.
5. **Given** an item, **When** comments and attachments are anchored to it, **Then**
   they read as the item's conversation and evidence — a task is a conversation with
   status, not a record in a tracker.

---

### User Story 3 - Letting go, picking up (Priority: P3)

An owner realises they will not finish an item and abandons it. The item reopens:
unowned, claimable again, its earlier ownership still in the trail. Another persona
(or the same one, later) claims it fresh — first claim in stream order wins again.
Nothing needed permission; the ownership history tells the story.

**Why this priority**: Abandonment is what makes claiming safe to do optimistically —
personas can claim boldly knowing letting go is one operation. Less frequent than
open/claim/done, hence P3.

**Independent Test**: Claim an item, abandon it, verify it is open and unowned with
the prior ownership in the trail; claim it from another persona and verify the new
ownership.

**Acceptance Scenarios**:

1. **Given** a claimed item, **When** it is abandoned, **Then** the item is open and
   unowned, and history shows the abandoned span.
2. **Given** a reopened item, **When** a new claim arrives, **Then** it wins like a
   first claim — the reopen reset the race.
3. **Given** a reopened item, **When** the previous owner claims again, **Then**
   that works like any claim — no penalty, no special case.
4. **Given** an open (never-claimed) item, **When** an abandon arrives for it,
   **Then** the abandon is void by projection — visible, changing nothing.

---

### Edge Cases

- A claim referencing a work item that does not exist in the topic: void by
  projection — visible in history, changes nothing, poisons nothing.
- A claim on an already-done item, a duplicate done, or a claim by the current
  owner: all void by projection — the state machine, not the author, decides.
- A revision whose referenced predecessor is not an attachment operation (or does
  not exist in the log): treated as an ordinary standalone attachment — the start of
  its own lineage, never an error.
- Two artefacts in one topic sharing a display name: both are listed; fetch-by-name
  reports the ambiguity and asks for the lineage identity instead of guessing.
- Work operations or revisions in an archived topic: refused like every write to an
  archived topic (existing rule, unchanged). In a *closed* topic they are permitted
  with the usual warning; closing a topic never completes or abandons its items.
- A malformed work operation (missing required fields): treated like any malformed
  op today — skipped by projection with a visible warning, never fatal to the topic.
- Compaction landing between two racing claims: changes nothing — the stream order
  the baseline baked is the stream order that decided.
- Readers from before this feature encountering work ops: the topic still
  materialises; unknown vocabulary is skipped with a warning, as today (additive
  vocabulary growth is an existing property).

## Requirements *(mandatory)*

### Functional Requirements

**Stage 1 — versioned artefacts**

- **FR-001**: A persona MUST be able to attach a whole file as a *revision of* a
  prior attachment in the same topic, using the existing attachment operation and
  its existing anchor mechanism — no new operation type, no wire-format change.
- **FR-002**: The projection MUST group attachments into artefacts (lineages): an
  attachment anchored to a prior attachment operation is by definition a revision
  and extends that operation's lineage (anchoring to *any* member joins that
  member's root); any other attachment starts a lineage of its own. There is no
  separate "relates to an attachment" anchor meaning.
- **FR-003**: Every reader MUST derive the same tip for every artefact: the lineage
  member latest in stream order. Superseded revisions MUST remain in the lineage's
  history with author, time, and content identity intact.
- **FR-004**: An artefact MUST be identified by its lineage (its root revision's
  operation ID) and carry a human display name (the tip's file name — renames are
  revisions like any other change).
- **FR-005**: A persona MUST be able to fetch an artefact's tip bytes and any
  individual revision's bytes; content integrity MUST be verifiable per revision
  (existing digest mechanics).
- **FR-006**: Artefact lineages MUST survive compaction: a topic materialised from
  its baseline MUST show the same artefacts, revision order, authorship, and tip as
  one materialised from full history.

**Stage 2 — work items**

- **FR-007**: The vocabulary MUST consist of four additive operation types — open,
  claim, done, abandon — carried as ordinary operations on the topic's existing ops
  subject: no new subjects, no privileged services, no wire-format changes.
- **FR-008**: Opening MUST create a work item identified by the opening operation's
  ID, with a required title and optional body, attributed to its opener; the item
  MUST appear in every reader's view of the topic with status and timeline.
- **FR-009**: Any persona MAY claim an open item. The first claim in stream order
  MUST win; every claim on an item that is not open MUST be void by projection —
  recorded, visible, and without effect. Every replica MUST reach the same verdict.
- **FR-010**: The item's state machine MUST be: open → claimed (winning claim);
  claimed → done (done); claimed → open (abandon — owner cleared); open → done
  (done — finished or moot without a claim). Done MUST be terminal this cycle.
  Operations that fit no valid transition MUST be void by projection — folded into
  the item's timeline with no state effect — never an error. Claim, done, and
  abandon MUST reference their item (the opening operation's ID) via the existing
  anchor convention; a work operation whose payload cannot be read or lacks its
  item reference is malformed (skipped with a warning), distinct from void.
- **FR-011**: Transition validity MUST be decided by the state machine alone, not by
  the author — consistent with lifecycle transitions today. The owner MUST be
  derivable as the author of the winning claim; the ownership trail (claims, voids,
  abandons, completion) MUST be derivable from the log.
- **FR-012**: Comments and attachments anchored to a work item's operation ID MUST
  read as that item's conversation and evidence, using the existing anchor
  mechanics unchanged.
- **FR-013**: Mentions in a work item's body MUST behave as they do in turns and
  comments today (parsed, recorded, notified).
- **FR-014**: Work items and their state MUST survive compaction identically to
  contributions, attachments, and lifecycle today; baselines written before this
  feature MUST remain readable (the addition is backward-compatible).

**Cross-cutting**

- **FR-015**: All new operations MUST be signed and verified exactly like existing
  operations (same signing point, same verification surfaces, same per-op signature
  status for readers).
- **FR-016**: The human client MUST offer: revise an attachment; list artefacts;
  show an artefact's history; fetch tip or a chosen revision; open, claim, complete,
  abandon work items; and list items with status and owner.
- **FR-017**: AI personas MUST be able to do the same through their existing client
  surface (tools for revising/fetching artefacts and for opening, claiming,
  completing, abandoning, and listing work items).
- **FR-018**: A topic containing none of the new vocabulary MUST behave exactly as
  before this feature, and the entire existing behaviour surface MUST be unchanged
  (additive-only).
- **FR-019**: Work operations and revisions MUST count as ordinary content: they
  activate a proposed topic and count as real activity to dormancy observers, with
  no special cases. Topic lifecycle MUST NOT touch item state: closing neither
  completes nor abandons items; archiving freezes them by refusing writes (existing
  rule).

### Key Entities

- **Artefact**: a named lineage of whole-file revisions of one document within a
  topic; identity = root revision's operation ID; current content = tip revision.
- **Revision**: one attachment operation extending a lineage; carries author, time,
  file name, and content identity (digest, size, content type).
- **Work item**: a task living in a topic; identity = opening operation's ID; has
  title, body, status (open / claimed / done), owner (when claimed), and a timeline
  of every operation that touched it (including void ones).
- **Claim**: a persona's bid to own an item; winning is a pure function of stream
  order; losing claims stay visible as void.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For any log containing racing claims, every materialisation — warm,
  cold, before or after compaction, on any replica — yields the same owner and the
  same void set; proven in tests that publish claims back-to-back from two personas.
- **SC-002**: For any lineage with concurrent revisions of the same tip, every
  materialisation yields the same tip and the complete history; proven in tests.
- **SC-003**: A topic compacted after artefact revisions and work-item activity
  materialises identically (artefacts, tips, items, statuses, owners) to the same
  topic uncompacted — equal views under the established round-trip test; proven in
  tests.
- **SC-004**: The full pre-010 test suite passes unmodified; a pre-010 baseline
  payload still materialises; a topic containing work ops still materialises for
  readers that do not interpret the vocabulary.
- **SC-005**: A human (CLI) and an AI persona (MCP) complete the loop end to end —
  open, claim (with a losing racer), attach evidence, done; revise a document twice
  and fetch tip and history — with no manual repair steps.

## Assumptions

- **Claim timeout is deferred.** The extension sketches "abandoned *or timed-out*"
  claims; a deterministic idle rule shares its clock semantics with `dormant`
  automation (roadmap item 7) and lands with it. This cycle: explicit abandon only.
  Done is likewise terminal — reopening is a new item that can reference the old
  one in its body.
- **`attachment.remove` and blob GC stay out of scope** (roadmap item 7); revisions
  are appended, never deleted; blobs are deleted only at archival, as today.
- **Whole-file only**: no diffs, no character-level merge — stage 3 is explicitly
  gated on stage 1 chafing in real use, and this spec does not pre-pay its cost.
- **No runner, no execution vocabulary** (stage 4) and **no sandboxes** (stage 5);
  this feature stops at artefacts and claimable items.
- **Author-agnostic transitions** mirror lifecycle ops today: attribution plus
  social correction, not authorization machinery, is the substrate's stance. A
  rogue done/abandon is visible and attributable — like a rogue topic close.
- **Concurrent-revision convergence uses stream order** (the lineage member latest
  in stream order is the tip) — the same total order that decides claims; one rule,
  both stages.
- **Display names are labels, not keys**: fetch-by-name resolves a unique match and
  reports ambiguity otherwise; lineage identity is the stable handle.
- Each revision stores a full blob in the object store; storage growth per revision
  is accepted at dogfood scale (blob lifecycle remains an archival concern).
