# Feature Specification: Tenants in the house — the accounts surface reaches the hand

**Feature Branch**: `012-tenants-in-the-house`
**Created**: 2026-08-27
**Status**: Implemented 2026-08-27 (the gate measured end to end —
`node/tenancy_test.go`)
**Input**: hq episode 0133 and its graduated design —
`soul-hq/02-DESIGN/soulstream-identity/platform-topology.md` (D46–D49),
plus episode 0134 (D47 landed upstream: tenants born admissible). This
feature is the product's half the design named: the built `accounts.*`
op family was dark in the house — no `SystemConn`, no client surface,
and a resolver that forgot every runtime tenant at shutdown.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - An isolated account is created from the hand (Priority: P1)

The operator of a running realm creates an isolated account for a
second tenant — a client, a team, a separated project — with one
command. People and agents admitted into that account see only it: its
own record, its own personas, its own streams. No restart, no config
edit, no external tooling.

**Acceptance Scenarios**:

1. **Given** a founded realm with `soulstream up` running, **When** the
   operator runs `soulstream account create acme`, **Then** the account
   exists (name → key resolution answering), a token minted for it
   admits a person through the sentinel lane, and that person is
   USABLE — subscribing and publishing inside the persona scope,
   refused outside it (upstream D47's born-admissible, reached from the
   hand).
2. **Given** a created account, **When** the operator runs `soulstream
   account suspend acme`, **Then** the account's next admission is
   refused and its data is untouched; `resume` restores admission.

### User Story 2 - Tenants survive a restart (Priority: P1)

Accounts created at runtime are part of the realm, not of the process:
stopping and starting the node keeps every tenant — its account, its
admission, AUTH's knowledge of it, its vault binding.

**Acceptance Scenarios**:

1. **Given** a realm with a created account, **When** the node stops
   and starts again on the same state directory, **Then** the tenant's
   token still admits a usable person and the account still resolves.

### Edge Cases

- A BYO realm holds no operator or SYS material (design 0003): the
  tenancy ops stay off and the service answers `accounts.*` with its
  own refusal — the command shows it, honestly, rather than inventing
  a second answer.
- A realm founded before this feature gains the operator key in its
  vault on the next `up` (the ensure posture, F1) and its resolver
  directory on the next start — no migration act, no new secret on
  disk.
- The resolver seed never overwrites: a restart re-seeds only absent
  accounts, so AUTH's runtime-amended `allowed_accounts` (each created
  tenant) survives — overwriting would silently unlearn every tenant.

## Requirements *(mandatory)*

- **FR-001**: The embedded server's account resolver MUST persist
  runtime-created accounts across restarts (`<state>/resolver`, a dir
  resolver seeded create-if-absent from the founding JWTs).
- **FR-002**: The node MUST wire the identity plane's `SystemConn` on
  the embedded flavour, with a SYS user minted in memory from the
  ceremony's SYS account key — no new artifact in the state directory.
- **FR-003**: The node MUST ensure the operator key is in the vault
  (`operator/root`) whenever the tenancy ops are enabled — an ensure at
  start, so pre-existing realms gain it without a founding re-run.
- **FR-004**: `soulstream account create|list|show|suspend|resume`
  MUST drive the sealed `accounts.*` ops over the operator's own creds
  against the running node.
- **FR-005**: On BYO flavours the tenancy ops MUST stay disabled and
  every failure mode MUST surface as a named refusal, never a hang.

## Success Criteria *(mandatory)*

- **SC-001**: `account create` → usable token-lane admission in under
  5s on loopback (measured: 8.8ms).
- **SC-002**: A created tenant admits after a node restart on the same
  state directory (measured: the gate's second-run prove).
- **SC-003**: `make check` fully green; the M1.1 gate unchanged and
  passing on the dir resolver.
