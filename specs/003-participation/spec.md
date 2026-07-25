# Feature Specification: Participation — Mentions & Attachments

**Feature Branch**: `003-participation`
**Created**: 2026-07-12
**Status**: Shipped (v0.1.0)
**Input**: User description: "The collaborative layer on top of topics: mentions (@name → a notification in the mentioned persona's inbox) and attachments (put a file in the realm's object store and reference it from the topic, then retrieve it). Deferred: attachment removal, encryption, mention digests/presence."

## Overview

Topics let personas converse and work; **participation** makes that work reach people and carry
real files. Two additions, both pure conventions over the existing op-log and the realm's object
store:

- **Mentions** — writing `@name` in a turn or comment tells that persona "you're wanted here." The
  library parses the mention, records it on the operation, and drops a notification in the
  mentioned persona's inbox. Humans get pinged; agents wake up. Same mechanism for both.
- **Attachments** — a file too big or too binary for a message goes into the realm's object store,
  and the topic carries a small reference (name, digest, size, type). Anyone can fetch it back by
  that reference, and it shows up in the materialised topic.

The consumers are library integrators; the value is that a topic becomes a place where people are
*summoned* and real artefacts are *exchanged*, not just a text log — while nothing new is added
beside NATS (the object store is JetStream's own).

## Clarifications

### Session 2026-07-12

- Q: What exactly is an `@mention` in body text? → A: An `@` immediately followed by a valid persona
  slug (`@[a-z0-9]+(-[a-z0-9]+)*`), matched maximally; the surrounding punctuation is not part of the
  name (the slug grammar can't include it). Mentions are de-duplicated and only kept if they are
  valid persona names. `@@`, `@ x`, and `@Bad` (uppercase) yield no mention.
- Q: What does a notification carry, and where does it go? → A: A `mention.notify` record is
  published to `SOULSTREAM.PERSONA.NOTIFY.<persona-id>` with payload `{topic, op_id, author}` — the
  mentioned persona learns which topic, which operation, and who mentioned them, enough to go read
  the anchoring op and react.
- Q: When are mentions parsed and notifications fired? → A: In the posting path for turns and
  comments: the library parses the body, fills the op payload's `mentions`, publishes the op, then
  publishes one `mention.notify` per mentioned persona. A persona mentioning itself still notifies
  itself (simple and predictable); clients may choose to ignore self-notifications.
- Q: How is an attachment stored and referenced? → A: The blob is put into the realm's
  `soulstream-objects` store under the key `attachments/<topic-path>/<object-id>` (object-id a fresh
  UUID). The object store computes a content digest. An `attachment.add` op then carries `{name,
  object, digest, size, content_type, anchor?}` — `name` is the human filename, `object` is the store
  key, `digest`/`size` come from the store, `anchor` optionally ties it to another op.
- Q: How does an attachment appear when a topic is materialised? → A: As its own list on the view
  (`Attachments`), each entry carrying the op-id, author, timestamp, name, object key, digest, size,
  content type, and anchor. An `attachment.add` is a content op, so it also moves a topic from
  `proposed` to `active`.
- Q: What is deferred? → A: `attachment.remove` (blobs are deleted only at topic archival, which is
  day-2), encryption/sealed attachments, object lifecycle cleanup, and mention digests /
  presence-aware deferral. This cycle is add + retrieve for attachments, and parse + notify for
  mentions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Mention a persona and notify their inbox (Priority: P1)

A persona writes a turn or comment containing `@name`. The library records the mention on the
operation and delivers a notification to each mentioned persona's inbox. A persona watching its
inbox receives the notification with enough information to jump to the anchoring operation.

**Why this priority**: Attention is the scarce resource. Without mentions, agents can't be summoned
and humans can't be pinged; the topic is a place you must actively watch rather than one that
reaches you. It's the minimum attention primitive.

**Independent Test**: Post a turn containing `@bookkeeper-agent`; from a consumer subscribed to that
persona's notify subject, receive a `mention.notify` naming the topic, op-id, and author; confirm
the op payload's `mentions` lists the persona.

**Acceptance Scenarios**:

1. **Given** a topic, **When** a persona posts a turn with `@bookkeeper-agent` in the body, **Then**
   the op's payload records `bookkeeper-agent` in `mentions`, and a `mention.notify` is published to
   that persona's notify subject carrying the topic, op-id, and author.
2. **Given** a persona following its own notify subject, **When** it is mentioned, **Then** it
   receives the notification live and can read the referenced operation.
3. **Given** a body with several mentions, some repeated and some invalid (`@Daan`, `@@`, `@ x`),
   **When** it is posted, **Then** only the distinct, valid persona names become mentions and get
   notified.
4. **Given** a comment (not just a turn) with a mention, **When** it is posted, **Then** the same
   parse-and-notify behaviour applies.

### User Story 2 - Attach a file to a topic (Priority: P1)

A persona attaches a file: the library stores the bytes in the realm's object store and publishes
an `attachment.add` referencing it by name and digest. Materialising the topic shows the attachment.

**Why this priority**: Collaboration is *around concrete things*. A topic that can't carry a file is
conversation-only — exactly what the design refuses. This is the artefact half of the workbench.

**Independent Test**: Attach a small file to a topic; confirm the bytes are in the object store and
an `attachment.add` op references them with a correct digest and size; materialise the topic and
find the attachment listed.

**Acceptance Scenarios**:

1. **Given** a topic, **When** a persona attaches a file with a name and content type, **Then** the
   bytes are stored in `soulstream-objects` under `attachments/<topic>/<object-id>`, and an
   `attachment.add` op records the name, object key, digest, size, and content type.
2. **Given** an attachment optionally anchored to an operation, **When** the topic is materialised,
   **Then** the attachment appears on the view with its metadata and anchor, and the topic is
   `active`.
3. **Given** a blob and its stored digest, **When** the digest is recomputed from the reference,
   **Then** it matches — the reference is verifiable.

### User Story 3 - Retrieve an attachment (Priority: P2)

A persona (or agent) that sees an attachment in a materialised topic fetches its bytes back from the
object store by the reference and can verify them against the recorded digest.

**Why this priority**: Storing without retrieving is useless, but retrieval is a thin read on top of
US2, so it is its own smaller slice at P2.

**Independent Test**: After US2, fetch the attachment's bytes by its object key and confirm they
equal the original and match the recorded digest.

**Acceptance Scenarios**:

1. **Given** an `attachment.add` in a materialised topic, **When** a persona fetches the referenced
   object, **Then** it receives the exact original bytes.
2. **Given** the fetched bytes, **When** the digest is verified against the reference, **Then** it
   matches; a mismatch is reported.
3. **Given** a reference to an object that does not exist, **When** a fetch is attempted, **Then** a
   clear not-found error is returned.

### Edge Cases

- **Self-mention**: a persona mentions itself → it is notified (predictable); ignoring it is the
  client's choice.
- **Mention of an unknown persona**: `@ghost` where no such persona exists → the mention is still
  recorded and a notify is published to that (currently empty) inbox; the substrate does not verify
  persona existence (no registry in core), and nobody is listening, so it is harmlessly delivered.
- **Duplicate mentions in one body**: collapsed to one mention and one notify per persona.
- **Mention token adjacent to punctuation** (`@daan.`, `@daan,`): the name is `daan`; the trailing
  punctuation is excluded.
- **Empty / whitespace-only attachment name**: rejected with a clear error.
- **Oversized attachment**: the object store handles large blobs by design (chunked); no inline size
  limit as with baselines — but a zero-byte attachment is allowed (a valid, if unusual, file).
- **Attachment anchored to an unknown op**: recorded with its anchor; flagged dangling like a
  comment, never dropped.
- **Retrieving after the object was never written** (only the op exists): not-found error surfaced,
  not a panic.
- **Unknown persona-id in notify subject**: the notify subject accepts any valid slug; delivery to
  an inbox nobody reads is a no-op, not an error.

## Requirements *(mandatory)*

### Functional Requirements

**Mentions**

- **FR-001**: The library MUST parse `@name` tokens from a turn or comment body, matching `@`
  followed maximally by a valid persona slug; only distinct, valid persona names become mentions.
- **FR-002**: A posted turn or comment MUST record its parsed mentions in the operation payload's
  `mentions` field.
- **FR-003**: After publishing a turn or comment with mentions, the library MUST publish one
  `mention.notify` record to each mentioned persona's `SOULSTREAM.PERSONA.NOTIFY.<persona-id>`
  subject, carrying the topic-path, the op-id, and the author.
- **FR-004**: The library MUST let a persona follow its own notify subject and receive
  `mention.notify` records live, each sufficient to locate and read the anchoring operation.
- **FR-005**: A self-mention MUST notify the persona (no special-casing); ignoring it is a client
  choice.
- **FR-006**: Mention parsing MUST NOT verify persona existence (there is no registry in core); an
  unknown but well-formed `@name` is a valid mention.

**Attachments — add**

- **FR-007**: Attaching a file MUST store its bytes in the realm's `soulstream-objects` object store
  under the key `attachments/<topic-path>/<object-id>`, where `<object-id>` is a fresh UUID.
- **FR-008**: The library MUST publish an `attachment.add` operation to the topic recording: the
  human name, the object key, the content digest and size (as reported by the object store), the
  content type, and an optional anchor op-id.
- **FR-009**: The object-store digest MUST be recorded so the reference is independently verifiable.
- **FR-010**: An attachment MAY be anchored to another operation; an anchor to an absent op MUST be
  flagged dangling on materialisation, never dropped.
- **FR-011**: An empty or whitespace-only attachment name MUST be rejected with a clear error; a
  zero-byte file MUST be allowed.

**Attachments — materialise & retrieve**

- **FR-012**: Materialising a topic MUST surface attachments as their own list, each entry carrying
  the op-id, author, timestamp, name, object key, digest, size, content type, and anchor.
- **FR-013**: An `attachment.add` MUST count as a content operation, moving a topic from `proposed`
  to `active`.
- **FR-014**: The library MUST retrieve an attachment's bytes from the object store by its object
  key, returning the exact stored bytes.
- **FR-015**: The library MUST support verifying fetched bytes against the recorded digest, reporting
  a mismatch.
- **FR-016**: Fetching a reference to a non-existent object MUST return a clear not-found error, not
  a panic.

**Discipline & non-goals**

- **FR-017**: This cycle MUST NOT implement `attachment.remove`, encrypted/sealed attachments,
  object lifecycle cleanup, mention digests, or presence-aware notification deferral.
- **FR-018**: Mentions and attachments MUST be additive over the existing op-log and object store —
  no new infrastructure, no change to the wire record or the realm's artefacts.

### Key Entities *(include if feature involves data)*

- **Mention**: a validated persona name parsed from a body; recorded on the op and used to address a
  notification.
- **Notification (`mention.notify`)**: a record on a persona's notify subject — `{topic, op_id,
  author}` — the substrate's attention primitive.
- **Notify inbox**: a persona's `SOULSTREAM.PERSONA.NOTIFY.<persona-id>` subject; followed like a
  topic to receive notifications live.
- **Attachment reference (`attachment.add`)**: `{name, object, digest, size, content_type, anchor?}`
  — a small, verifiable pointer into the object store.
- **Attachment (materialised)**: the reference plus op-id, author, timestamp, and dangling flag, as
  it appears on the topic view.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Posting a body with `@name` results in exactly one notification per distinct valid
  mentioned persona, delivered to their notify subject and received by a follower — verified end to
  end against a running server.
- **SC-002**: 100% of invalid or duplicate mention tokens in a defined set (`@Daan`, `@@`, `@ x`,
  repeated `@daan @daan`) produce the correct mention set (no invalids, no duplicates).
- **SC-003**: An attached file's bytes are retrievable byte-for-byte and match the recorded digest in
  100% of cases; a tampered/other object fails digest verification.
- **SC-004**: A materialised topic lists every attachment with correct metadata and anchor, and is
  `active` after an attachment — verified against a running server.
- **SC-005**: A not-found fetch and an empty attachment name are handled with clear errors (no
  panic), and the whole feature verifies green — all tests pass (none skipped), formatting applied,
  linting clean.

## Assumptions

- **Topics in place**: `002-topics` is merged — the `topic` package (Handle, Post, Materialise,
  Follow, Board) and the realm's provisioned object store are available and extended here.
- **Object store is JetStream's**: the realm's `soulstream-objects` bucket (provisioned in 001) is
  the storage; no external blob service.
- **No registry**: persona existence is not verified (mentions to unknown personas are valid); a
  richer registry is a separate extension.
- **JSON payloads**: turn/comment payloads gain a `mentions` array; `attachment.add` is a small JSON
  reference — consistent with the canonical record's data field.
- **Notify is general**: the notify subject is deliberately general (mentions are its first use);
  future notification types reuse it.

## Dependencies

- `002-topics` (merged): `topic` package + the pure `apply` fold (extended here for
  `attachment.add`), and `realm` (its object store + `JetStream()` accessor).
- A reachable/embedded NATS server with JetStream (stream + object store) for the notify and
  attachment scenarios.
