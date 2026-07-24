# Feature Specification: Re-baselining (Rollup), Manifest Baselines & Archived

**Feature Branch**: `007-rollup`
**Created**: 2026-07-21
**Status**: Shipped (v0.1.0)
**Input**: User description: "Re-baselining (rollup), manifest baselines, and the archived lifecycle (roadmap Day-2 item 1). Any persona can compact a topic: take the current baseline, apply all operations since, and publish the result as the new baseline in one atomic stroke that replaces the old baseline and the consumed tail. Leaderless, race-safe by optimistic concurrency (first writer wins, losers discard). Baselines stay exactly one message: inline under the realm threshold, manifest via the object store above it. Triggers: manual plus lifecycle-driven (closed and archived always re-baseline). Archived is terminal: final re-baseline, readable, refuses writes. CLI gains rollup and archive; MCP gains a rollup tool."

## Why now

Roadmap Day-2 item 1, gated on signing — which landed in 006, so the gate is open. Long
dogfood topics currently replay their entire history on every cold read; rollup makes
replay cost proportional to what happened *since the last compaction*, not since birth.
The one-way door is respected: compaction destroys the op tail, so it never happens
without an explicit act — a persona's deliberate command, or the two lifecycle moments
the protocol mandates (closing and archiving).

## Clarifications

### Session 2026-07-21

- Q: What exactly is the baseline state, so replay round-trips? → A: The materialised
  view a reader would have built from the log — contributions, attachments, and
  lifecycle, with their op-ids, authors, timestamps, mentions, and anchors — plus the
  frontier. Anything derived (dangling flags, active-vs-proposed) is recomputed, not
  stored.
- Q: What verification status do baked elements carry after their signed ops are
  compacted away? → A: The baseline op's own status. The compacted tail's per-op
  signatures are destroyed with it; the roll-upper's signature over the baseline
  becomes the state's provenance. An unsigned persona may still compact (rollup is an
  optimisation any persona may run); its baseline simply attests nothing.
- Q: What does "closed and archived always re-baseline" mean when the rollup race is
  lost? → A: The transition always stands; the rollup is an *attempt* with a small
  bounded retry. A closed-but-uncompacted topic is valid (rollup is optional for
  correctness); archival retries until the final baseline lands or reports failure
  loudly, leaving a closed-equivalent readable topic.
- Q: Is archival exposed to AI personas over MCP? → A: No. Archival is the one
  deliberate reclamation act — an operator decision, like key rotation. MCP gets
  rollup only.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Compact a topic and nothing changes (Priority: P1)

A persona working in a long-running topic runs "save a version". The topic's history —
possibly hundreds of operations — is folded into a fresh zero-point: one baseline
message carrying the materialised state, followed by nothing. Everyone who reads the
topic afterwards, cold or live, sees exactly the conversation they saw before:
same contributions, same attachments, same anchors, same lifecycle. New posts continue
seamlessly, parenting onto the frontier recorded in the baseline. If someone else
posts (or compacts) at the same moment, exactly one writer wins and the loser's
attempt evaporates without a trace.

**Why this priority**: This is the capability itself — compaction with replay
equivalence and race safety. Everything else (manifests, lifecycle triggers) refines
when and how it fires.

**Independent Test**: Build a topic with every element kind (turns, comments with
anchors, attachments, mentions, a transition), roll it up, and compare the
materialised view before and after — identical except per-op provenance; then race a
concurrent post against a rollup and confirm the post survives and the rollup attempt
reports "lost".

**Acceptance Scenarios**:

1. **Given** a topic with a baseline and a tail of operations, **When** a persona
   triggers rollup, **Then** the topic's log becomes exactly one baseline message and
   a cold reader materialises the identical view (contributions, attachments,
   mentions, anchors, lifecycle) the full log produced.
2. **Given** a freshly compacted topic, **When** a persona posts, **Then** the new op
   parents onto the baseline's frontier and the topic continues as if never compacted.
3. **Given** two writers, one posting and one compacting concurrently, **When** the
   post lands first, **Then** the rollup attempt is rejected, the topic is unchanged
   (post included), and the compactor can simply try again later.
4. **Given** two personas compacting concurrently, **When** one baseline lands first,
   **Then** the other attempt is rejected and discards cleanly — no duplicate
   baseline, no lost operations.
5. **Given** a signing persona compacts a topic containing verified ops, **When** the
   topic is read afterwards, **Then** baked elements report the baseline op's
   verification status (the roll-upper's attestation), and the baseline itself
   verifies like any signed op.
6. **Given** a comment anchored to a turn that was baked into the baseline, **When**
   a new comment anchors to that same (now baked) op-id, **Then** the anchor resolves
   — baked elements keep their op-ids and are not dangling.

---

### User Story 2 - Oversized state still compacts to one message (Priority: P2)

A topic's materialised state has outgrown the realm's inline threshold. Compaction
still produces exactly one message on the log: the state is stored as chunks in the
realm's object store, and the baseline message is a manifest naming the chunks with a
digest over the whole state. Readers reconstruct the state transparently. The write
order makes crashes harmless: chunks first, then the manifest (the atomic commit
point, with the same race guard), then cleanup of superseded chunks — a crash at any
point leaves either the old topic intact or the new one committed, never a broken log.

**Why this priority**: Without it, rollup silently stops working the day a topic gets
big — exactly when compaction matters most. It builds directly on US1's commit path.

**Independent Test**: Grow a topic's state past the threshold, roll it up, confirm the
log holds one manifest message and a cold reader reconstructs the full state
(digest-verified); simulate a crash between chunk write and manifest publish and
confirm the topic replays exactly as before with only orphaned chunks left behind.

**Acceptance Scenarios**:

1. **Given** a topic whose materialised state exceeds the inline threshold, **When**
   it is compacted, **Then** the log holds exactly one manifest baseline and a cold
   reader materialises the complete state, verified against the manifest's digest.
2. **Given** a manifest rollup that loses the race after its chunks were written,
   **Then** the log is untouched and the orphaned chunks harm nothing.
3. **Given** a manifest baseline whose chunks are missing or corrupted, **When** the
   topic is read, **Then** the reader reports the topic malformed with a clear reason
   — it never crashes and never silently shows partial state.
4. **Given** a successful manifest rollup, **When** it completes, **Then** the chunks
   superseded by it are gone from the object store (no unbounded growth on the happy
   path).

---

### User Story 3 - Closing tidies up; archiving is final (Priority: P3)

Closing a topic leaves it compact: the close is recorded and the topic is re-baselined
so it rests as a single tidy message. Archiving goes further — it is the realm's one
deliberate reclamation act, explicit and loud: a final re-baseline bakes everything
(including the archived state itself) into one terminal baseline. An archived topic
stays fully readable forever, but all writes to it are refused outright — unlike
closed, which merely warns.

**Why this priority**: These are the two lifecycle moments the protocol says *always*
re-baseline. They ride on US1's machinery and complete the topic lifecycle story.

**Independent Test**: Close a topic and confirm it ends compacted with lifecycle
closed; archive a topic and confirm exactly one terminal message remains, reads work,
and every write path (post, comment, attach, transition — CLI, MCP, library) refuses.

**Acceptance Scenarios**:

1. **Given** an active topic, **When** a persona closes it, **Then** the close is
   recorded and the topic is compacted — a cold reader sees lifecycle closed from a
   single baseline message (when no concurrent writer interfered; a lost race leaves
   a valid uncompacted closed topic).
2. **Given** a closed (or active) topic, **When** its persona archives it, **Then**
   the topic ends as exactly one baseline message with lifecycle archived, and the
   archival is reported loudly as the deliberate act it is.
3. **Given** an archived topic, **When** anyone reads it, **Then** the full
   materialised state is there — contributions, attachments, the lot.
4. **Given** an archived topic, **When** any persona attempts any write (turn,
   comment, attachment, transition, another rollup), **Then** the write is refused
   with a clear "archived is terminal" error — in the library, the CLI, and the MCP
   adapter alike.
5. **Given** an archived topic, **When** someone attempts to archive it again,
   **Then** the attempt reports it is already archived and changes nothing.

---

### Edge Cases

- Rollup of a topic that is only a baseline (no tail): nothing to compact — the
  attempt reports "nothing to do" and publishes nothing.
- Rollup of a malformed topic (first message is not a baseline): refused with the
  malformation reason; compaction never launders a broken log into a clean one.
- Rollup of a sub-topic compacts only that sub-topic's log; parent and sibling topics
  are untouched (per-subject compaction).
- A mention inside a baked turn does not re-fire its notification on rollup — the
  notification already went out when the turn was posted; inbox subjects are not
  touched by topic compaction.
- The board after rollup: the announcement lives on its own subject and is unaffected;
  lifecycle shown on the board is derived from the compacted log (baseline included).
- An unsigned persona compacts a topic full of verified ops: allowed — the baked
  elements' status degrades to the unsigned baseline's (the one-way consequence of
  compaction; realms that care run signed personas and land the memory/archivist
  extension before heavy rollup).
- Rollup while a follower is live: the follower's next delivered message is the new
  baseline; its view stays consistent (same materialised state, new zero-point).
- Archive raced by a concurrent post: bounded retry re-materialises and re-attempts;
  if retries exhaust, the archived transition stands, the failure is reported loudly,
  and the topic remains readable (closed-equivalent) — never half-compacted.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Any persona MUST be able to compact a topic: fold the current baseline
  plus all operations since into a new baseline that replaces both in one atomic act,
  leaving the topic's log in its born shape (baseline first, tail after).
- **FR-002**: Compaction MUST preserve the materialised view exactly: contributions,
  attachments, mentions, anchor relationships, op-ids, authors, timestamps, and
  lifecycle read identically before and after; baked op-ids MUST remain valid anchor
  targets.
- **FR-003**: Compaction MUST be race-safe without any coordinator: an attempt commits
  only if no operation landed after the tail it consumed; otherwise it is rejected,
  changes nothing, and the outcome is distinguishable as a lost race (retryable).
- **FR-004**: Compaction MUST never run implicitly except at the two mandated
  lifecycle moments (closing, archiving); there is no periodic or load-triggered
  compaction this cycle.
- **FR-005**: A baseline MUST always be exactly one message. State at or under the
  realm's inline threshold is carried inline; above it, the state is chunked into the
  realm's object store and the baseline is a manifest carrying chunk names, a digest
  over the full state, and the frontier.
- **FR-006**: Manifest writes MUST follow the crash-safe order: chunks first, then the
  manifest as the atomic commit point (with the same race guard), then deletion of
  superseded chunks; a failure at any point leaves either the old log intact or the
  new baseline committed — orphaned chunks are the worst possible debris.
- **FR-007**: Every baseline MUST carry the frontier (leaf op-ids at compaction);
  operations posted after compaction MUST parent onto it so the graph continues
  across the boundary.
- **FR-008**: The compaction baseline is an ordinary operation: signed when the
  persona holds a key, verified like any op on read; view elements baked into a
  baseline MUST report the baseline op's verification status as their own.
- **FR-009**: Closing a topic MUST trigger a compaction attempt; a lost race leaves a
  valid, uncompacted closed topic. Archiving MUST perform the final compaction with a
  bounded retry, ending the topic as a single terminal baseline (or reporting failure
  loudly with the archived transition standing).
- **FR-010**: Archived MUST be terminal: the topic stays fully readable; every write —
  turn, comment, attachment, transition, further compaction — MUST be refused with a
  clear error across library, CLI, and MCP (closed keeps its existing warn-but-allow
  behaviour).
- **FR-011**: Reading a manifest baseline MUST verify the reconstructed state against
  the manifest's digest; missing or corrupt chunks make the topic report malformed
  with a reason — never a crash, never silent partial state.
- **FR-012**: The human client MUST offer explicit compact ("save a version") and
  archive commands; the AI-persona adapter MUST offer compaction but not archival.
- **FR-013**: No election, lock service, or coordinator may appear anywhere in the
  design; racing writers are resolved by the log's own ordering alone.

### Key Entities

- **Baseline**: a topic's zero-point — one message holding the materialised state
  (inline) or naming it (manifest), plus the frontier. Always the first message of a
  topic's log.
- **Frontier**: the leaf op-ids at compaction time; the parents for whatever comes
  next.
- **Rollup attempt**: a materialise-and-publish act that either commits (becoming the
  new baseline, consuming the tail) or is rejected wholesale by the race guard.
- **Chunk**: one object-store piece of an oversized state; meaningful only through the
  manifest that names it. Orphaned chunks are garbage, never corruption.
- **Manifest**: the baseline form that names chunks and carries the digest sealing the
  full state.
- **Archived**: the terminal lifecycle state — final baseline, read-only forever.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For a topic containing every element kind, the materialised view after
  compaction is identical to the view before it (per-op provenance aside) — proven
  field-by-field in tests, 100% match.
- **SC-002**: After compaction, a topic that held N messages holds exactly 1; a
  100-op dogfood topic cold-reads from a single message.
- **SC-003**: In every raced scenario tested (post vs rollup, rollup vs rollup), the
  first writer wins, the loser changes nothing, and zero operations are lost.
- **SC-004**: A state larger than the inline threshold compacts to one manifest
  message whose reconstructed state is digest-identical; a simulated crash before the
  commit point leaves the original log readable and only orphaned chunks behind.
- **SC-005**: 100% of write attempts against an archived topic are refused with a
  clear error, across all three surfaces; 100% of reads still succeed.
- **SC-006**: All 134 existing tests keep passing unmodified in behaviour — an
  un-compacted topic works exactly as today.

## Assumptions

- The dogfood realm is declared throwaway (roadmap): enabling compaction in a realm
  whose history matters is an operator decision made outside the library. The library
  compacts only when told to (or at the two mandated lifecycle moments).
- Destroying the tail destroys its per-op signatures; preserving exhibits before
  compaction is the (deferred) memory/archivist extension's job, not this feature's.
- The inline threshold is the existing realm default (128 KB), not configurable this
  cycle.
- Archival does not delete attachment blobs this cycle (attachment.remove and blob
  lifecycle are deferred vocabulary); an archived topic's attachments stay fetchable.
- Closed keeps its current social semantics (readable, writable-with-warning);
  archived introduces the first hard write refusal.
- The dormant state and periodic compaction remain out of scope (roadmap Day-2
  item 7).
- Out of scope: topic.info.update, eg-walker merge, memory/archivist, scatter/gather
  discovery, blob garbage sweeping beyond the happy-path chunk cleanup.
