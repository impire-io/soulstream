# Feature Specification: an agent runs — the runtime plane

**Feature Branch**: `003-an-agent-runs`
**Created**: 2026-08-02
**Status**: Draft
**Input**: User description: "M1.3 — an agent runs (design 0001 §6, §9-M1.3): a declared agent workload launches through the runtime plane (native backend), posts a turn attributed to its persona, and its lifecycle appears as work ops — soulrealm's own first proof, re-run inside SoulNode."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Declare a workload; the realm runs it and remembers it (Priority: P1)

With the node up, the owner writes a small declaration file (who the
workload is, which topic it serves, which program to run) and runs
`soulnode workload start <file>`. The runtime launches the program as a
child process with a freshly minted, tightly scoped credential — the
workload sees only its own identity, never the node's keys — and
supervises it until it ends. What the workload does is realm activity
like anyone else's: its turn is attributed to its persona, its lifecycle
(opened, claimed, done) is operations on the topic, and the archivist
keeps all of it.

**Why this priority**: This is the run half of the product and the last
Phase 1 milestone — soulstream records, soulrealm runs, and M1.3 proves
both inside one binary's composition.

**Independent Test**: Start a topic, declare an echo agent against it,
run the workload command, and read back from the realm: a turn authored
by the workload's persona, work ops opened/claimed/completed, everything
kept by the archivist.

**Acceptance Scenarios**:

1. **Given** a running node and a valid declaration, **When** the owner
   runs `soulnode workload start`, **Then** the program runs as a child
   process with a minted credential scoped to its persona and topic, and
   the command supervises it to completion.
2. **Given** the workload posted a turn, **Then** the turn's author is
   the workload's persona and the lifecycle ops (open, claim, done)
   appear on the topic, attributed to the runtime's own persona.
3. **Given** the node is not up, **When** the owner runs the workload
   command, **Then** it refuses with a pointer at `soulnode up` — never
   a hang.
4. **Given** a declaration naming an unknown role/lifecycle or a missing
   artifact, **Then** the command refuses with the field named.

---

### Edge Cases

- The workload's credential must be minted by a key that carries
  per-workload permissions: the founding ceremony gains a second,
  *plain* realm signing key for the minter — scoped keys reject carried
  permissions (measured upstream, soulrealm's fleet research). The
  admission scoped key and the workload minting key are distinct
  artifacts with distinct jobs.
- The runtime speaks as its own persona (`runner`) for lifecycle ops,
  with its transport credential from the ceremony and its signing key
  vault-held like every persona's.
- The workload never sees the state directory: its credential is written
  only into its own scratch directory by the backend, environment
  cleaned (upstream's native-backend contract).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The ceremony MUST gain the workload minting key (a plain
  realm signing key, persisted like its peers, endorsed by the realm
  account) and the runtime's transport credential (`runner`); both join
  the verified inventory.
- **FR-002**: `soulnode workload start <declaration> [--state DIR]` MUST
  parse and validate the declaration (upstream's strict contract),
  connect to the running node over its configured listener, mint the
  workload's scoped credential with the workload minting key, launch via
  the native backend with scratch under the state directory, and
  supervise to terminal work ops — all through public upstream surfaces.
- **FR-003**: The runtime MUST participate as persona `runner`: lifecycle
  ops attributed to it, signing through the identity plane, vault-held
  key.
- **FR-004**: An unreachable node, an invalid declaration, or a missing
  artifact MUST each refuse fast with the cause named.
- **FR-005**: The workload's environment MUST contain only its own
  scoped identity and coordinates (upstream native-backend contract);
  node keys and state MUST NOT be reachable from the child.

### Key Entities

- **The declaration**: upstream soulrealm's contract, unchanged.
- **The workload minting key**: the ceremony's new plain signing key —
  mints per-workload users whose permissions ride in their own JWTs.
- **The runner persona**: the runtime plane's own identity.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: End to end in the test suite: topic started → declared
  echo agent run → a turn authored by the workload persona on the topic
  → work open/claim/done ops present → all kept by the archivist.
- **SC-002**: The workload's minted credential admits it and confines
  it (the upstream permission set); the node's own keys are absent from
  the child's environment and scratch.
- **SC-003**: The three refusal paths (node down, bad declaration,
  missing artifact) each fail fast with the cause named.
- **SC-004**: Full quality gate green; Phase 1 exit criteria of design
  0001 §9 all met.

## Assumptions

- The runtime plane is invocation-scoped (the `workload start` command
  supervises one workload), exactly like upstream's own command — the
  long-running claim-race node supervisor is upstream's unbuilt Fleet
  milestone and arrives there first (constitution I). Design 0001 §6's
  "runtime plane" wording is propagated to say so at landing.
- Native backend only in M1.3; msb/k8s are upstream options SoulNode can
  surface later.
- soulrealm joins the tracked pseudo-version pin class (it has no tags).
