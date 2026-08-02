# Feature Specification: the realm joins — provisioning and the memory plane

**Feature Branch**: `002-realm-joins`
**Created**: 2026-08-02
**Status**: Implemented (landed 2026-08-02 — journey 0004)
**Input**: User description: "M1.2 — the realm joins (design 0001 §6, §9-M1.2): init additionally provisions the realm's stream substrate on the embedded server and the node gains the memory plane — the archivist keeps every operation and answers the memory convention, attributed to its own persona with a vault-held key."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - What happens in the realm is kept, and memory answers (Priority: P1)

The owner's node is no longer just admission — it is a realm. `init`
founds the realm's record substrate too, and `up` runs the archivist
inside the node: every operation posted to any topic is captured verbatim
while it is live, and anyone in the realm can ask the memory convention
and get cited answers back. The archivist is an ordinary persona — its
answers are attributed and its key lives in the identity plane's vault,
never in a file it carries around.

**Why this priority**: This is the record half of the product — the
reason the ecosystem exists. Without it the node runs but remembers
nothing.

**Independent Test**: Found a fresh realm, run the node, post a turn *as
the owner admitted through the token lane*, then ask memory for it: the
answer cites the turn, and the kept exhibit's author is the owner.

**Acceptance Scenarios**:

1. **Given** a fresh `init`, **Then** the realm's record substrate
   (stream, notify, personas, objects) exists on the embedded server —
   provisioning is part of the founding, not a later act.
2. **Given** the node is up, **When** the owner (admitted with sentinel +
   token, signing through the identity plane) posts a turn to a topic,
   **Then** the archivist keeps it, and a memory query over the realm's
   own transport returns an answer citing it.
3. **Given** the node is up, **Then** the archivist's served answers are
   attributed to its persona, whose signing key materialized in the
   identity-plane vault on first touch — no key file exists for it in
   the state directory.
4. **Given** the node restarts, **Then** the archivist resumes from
   where it left off — nothing kept twice, nothing lost.

---

### Edge Cases

- Re-`init` on a founded realm: provisioning is create-or-verify — never
  a duplicate substrate, never an error on the second run.
- Memory asked before anything was posted: an honest empty answer with
  the archivist's declared coverage window, not an error.
- The memory plane disabled in configuration: the node runs without it
  (admission and the realm substrate unaffected); the config block is
  the design §2 plane block, present from this feature on.
- The archive directory is damaged/unwritable at `up`: the node refuses
  to start with the plane and path named (fail loud — constitution III's
  no-silent-plane-failure rule).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `init` MUST provision the realm's record substrate on the
  embedded server as part of the founding acts, under a configurable
  realm name (default recorded in the design; written into
  `config.json`); re-runs MUST be create-or-verify.
- **FR-002**: The ceremony MUST gain the memory plane's own service
  credential (bypass lane, persisted like its peers) and the archive
  store MUST live under the state directory; both join the verified
  inventory.
- **FR-003**: `up` MUST run the memory plane in-process when enabled
  (default enabled): capture every operation subject verbatim, resuming
  across restarts; serve the memory convention on the realm's transport.
- **FR-004**: The archivist MUST participate as an ordinary persona: its
  answers attributed to it, its signing key held by the identity plane
  (materialized on first touch), no persona key material in the state
  directory.
- **FR-005**: `config.json` MUST gain the per-plane configuration block
  of design 0001 §2 for the memory plane (`enabled`; URL/creds
  defaulting to the node's own loopback and state dir), and the node
  MUST honor `enabled: false`.
- **FR-006**: A memory-plane startup failure MUST prevent `up` from
  reporting healthy, with the plane and reason named — never a silently
  missing archivist.
- **FR-007**: The owner's end-to-end path MUST work through admission:
  a token-lane connection can post to a topic and query memory using
  only public surfaces plus the identity plane's signing oracle.

### Key Entities

- **The realm substrate**: the record's streams and buckets, provisioned
  at founding under the realm name.
- **The memory plane**: the in-process archivist — keeper (capture),
  witness (answers), archive store (`<state>/archive`).
- **The plane block**: the first instance of design §2's per-plane
  configuration in `config.json`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On a fresh state dir: `init && up`, one token-lane post,
  one memory query returning a citation to it — end to end in the test
  suite, no external processes.
- **SC-002**: Restart continuity: post N ops, restart the node, post
  more; the archive holds each exactly once.
- **SC-003**: The archivist's persona key exists in the vault (readable
  via the identity plane's directory op) and nowhere in the state
  directory's files.
- **SC-004**: With the memory plane disabled in config, `up` serves
  admission exactly as M1.1 (its e2e still passes) and no archive
  directory is created.
- **SC-005**: Full quality gate green, nothing skipped.

## Assumptions

- The realm name default and the plane-block shape are plan-time
  decisions propagated into design 0001 at landing.
- Discovery, curation, and any other realm services stay out of scope —
  this feature is exactly provisioning + memory.
- The runtime plane (workloads) is M1.3.
