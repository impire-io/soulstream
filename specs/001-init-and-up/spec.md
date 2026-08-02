# Feature Specification: soulnode init and up — the server and the identity plane

**Feature Branch**: `001-init-and-up`
**Created**: 2026-08-02
**Status**: Draft
**Input**: User description: "M1.1 — soulnode init and up: the server and the identity plane (design 0001 §3–§6, §9-M1.1). `soulnode init` performs the entire first-boot ceremony into a state directory … `soulnode up` runs the composition: embedded operator-mode server on a loopback listener and the identity plane in-process … Acceptance per design §9-M1.1."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - First boot on a fresh machine (Priority: P1)

A person downloads one binary onto a machine they own and runs `soulnode
init` with nothing prepared. The entire founding ceremony happens for them
— every trust root, account, key, and credential the realm will ever stand
on is generated and persisted into one state directory — and the command
ends by printing the one secret they will ever need to paste into a
client: their first access token. No key tooling, no configuration files
written by hand, no second binary.

**Why this priority**: First boot is the product (constitution V). If this
journey fails or demands a manual step, SoulNode has no reason to exist.

**Independent Test**: On an empty directory, run `init` and verify every
ceremony artifact exists on disk with owner-only permissions, and exactly
one access token was printed.

**Acceptance Scenarios**:

1. **Given** an empty (or absent) state directory, **When** the person
   runs `soulnode init`, **Then** the ceremony completes with zero
   prompts, zero manual steps, and zero external tools, every artifact of
   the design's ceremony inventory is on disk under the state directory
   (private directories `0700`, secret files `0600`), and the command
   prints exactly one access token with a line explaining it is shown
   only once.
2. **Given** a state directory where `init` already ran, **When** the
   person runs `soulnode init` again, **Then** nothing is regenerated, no
   second token is minted, and the command reports what it verified and
   exits successfully.
3. **Given** a state directory with a missing or corrupted artifact,
   **When** the person runs `soulnode init`, **Then** the command refuses
   with a message naming the damaged artifact — it never silently
   regenerates a trust root that existing data may depend on.

---

### User Story 2 - The node comes up and admits its owner (Priority: P1)

The person runs `soulnode up`. The node's messaging server and identity
plane start inside the one process, reachable only on the machine's own
loopback address. A client presenting the printed token (alongside the
node's public sentinel credential) is admitted and attributed; anything
else is refused, and the refusal is visible in the node's log.

**Why this priority**: Together with US1 this is the whole M1.1 gate —
`init && up` on a fresh machine reaches a working, admission-guarded
realm substrate. (Same-priority pair: US1 without US2 proves persistence
only; US2 without US1 has nothing to boot.)

**Independent Test**: After `init`, run `up`, then drive the three
admission observations from a separate client process: valid token
admits and is correctly scoped; garbage token refused; revoked token
refused.

**Acceptance Scenarios**:

1. **Given** a state directory from `init`, **When** the person runs
   `soulnode up`, **Then** the identity plane answers its status probe
   over the loopback listener within seconds, and the startup log names
   the listener address and the state directory in use.
2. **Given** the node is up, **When** a client connects with the sentinel
   plus the printed token, **Then** it is admitted, and the server itself
   asserts the client's persona and confines its permissions to that
   persona's own prefix.
3. **Given** the node is up, **When** a client presents a malformed or
   unknown token, **Then** no session forms and the refusal appears in
   the node's audit log.
4. **Given** a token that has been revoked, **When** a client presents
   it, **Then** no session forms.
5. **Given** the node is up, **When** the person interrupts it (Ctrl-C),
   **Then** the planes drain and the process exits cleanly, and a
   subsequent `up` on the same state directory works.

---

### Edge Cases

- The configured loopback port is already taken: `up` refuses to start
  with a message naming the port and the configuration key that changes
  it — never a silent fallback to another port.
- `init` interrupted partway (process killed during the ceremony): a
  subsequent `init` detects the incomplete state and says so; the
  documented recovery for a never-booted directory is deleting it and
  running `init` fresh.
- The state directory lives on a filesystem that cannot express
  owner-only permissions: `init` refuses rather than persisting secrets
  with weaker modes.
- `up` on a directory `init` never touched: refused with a pointer to
  `init`, not an implicit initialization.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `soulnode init` MUST generate and persist the complete
  founding ceremony of design 0001 §4 — trust root, system/authorization/
  realm domains with their admission configuration, service credentials,
  sealing keys — into a single state directory, creating it if absent.
- **FR-002**: All persisted secrets MUST be owner-readable only
  (directories `0700`, secret files `0600`); `init` MUST fail rather than
  persist secrets with weaker permissions.
- **FR-003**: `init` MUST complete with zero interactive prompts, zero
  manual steps, and zero external binaries (constitution V).
- **FR-004**: `init` MUST perform the founding administrative acts —
  storage buckets, sealing-key imports, the public sentinel credential,
  the first access token — through the components' public client
  surfaces over the node's own loopback connection, exactly as any
  operator would (constitution I/II).
- **FR-005**: The first access token MUST be printed exactly once, at
  first `init`, with wording that says it will not be shown again; it
  MUST NOT be persisted in plaintext anywhere.
- **FR-006**: `init` on an already-initialized state directory MUST be a
  verified no-op: nothing regenerated, no token minted, a report of what
  was checked. On a damaged or partial state directory it MUST refuse
  with the damaged artifact named.
- **FR-007**: `soulnode up` MUST run the embedded messaging server
  (operator mode, storage on the state directory) bound to loopback only,
  and the identity plane inside the same process, each plane connected
  over an ordinary loopback connection (constitution III) — never an
  in-process transport.
- **FR-008**: Admission MUST behave exactly as the composition research
  measured: sentinel + valid token admitted with the server-asserted
  persona confined to its own prefix; malformed and revoked tokens
  refused with the refusal recorded in the audit log.
- **FR-009**: `up` MUST fail fast, with the reason named, on: an
  uninitialized state directory, a bind conflict on the configured
  listener, or a sealing key that does not match the persisted vault.
- **FR-010**: On interrupt, `up` MUST drain its planes and exit; the
  state directory MUST remain valid for the next `up`.
- **FR-011**: The listener address MUST be configurable (default
  loopback on the conventional port); every plane MUST reach the server
  through its configured URL so that no plane behaves differently when
  that URL is not loopback (constitution III).

### Key Entities

- **The state directory**: the realm's physical home — configuration,
  trust root and account material, service credentials, sealing keys, the
  sentinel, and the message store. Copying it *is* backing up the realm.
- **The ceremony inventory**: the ordered list of artifacts `init`
  generates (design 0001 §4); `init`'s verification mode checks disk
  against exactly this list.
- **The first token**: the owner's founding credential — plaintext shown
  once, only its digest retained by the system.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On a fresh machine, `soulnode init && soulnode up` reaches
  a running, admission-guarded node in under one minute, with exactly two
  commands and zero manual steps.
- **SC-002**: The three admission observations (valid admits + correctly
  scoped; garbage refused; revoked refused) pass against the running
  node, driven from a separate client process, with refusals visible in
  the audit log.
- **SC-003**: `init` re-run is a verified no-op (nothing regenerated, no
  second token), and interrupt → restart of `up` works on the same state
  directory.
- **SC-004**: Every ceremony-inventory artifact exists on disk with the
  stated permission modes after `init`, verified mechanically by the test
  suite.
- **SC-005**: The full quality gate passes with the feature's end-to-end
  test included, nothing skipped.

## Assumptions

- Scope is exactly design 0001 §9-M1.1: server + identity plane. Realm
  provisioning, the memory plane, and the runtime plane are M1.2/M1.3;
  the front door is Phase 2.
- The state directory default location and the listener default port are
  plan-time decisions recorded in the design's configuration section;
  the spec requires only that both are configurable and loud on conflict.
- Additional tokens beyond the founding one are out of scope for M1.1
  (the identity plane's own client surface already supports them for
  operators who need one sooner).
- BYO-NATS mode (embedded server disabled) is out of scope (design §4's
  named [O]); the plane/URL configuration shape must simply not preclude
  it.
