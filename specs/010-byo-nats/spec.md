# Feature Specification: BYO NATS — founding on a server soulstream does not run

**Feature Branch**: `010-byo-nats`
**Created**: 2026-08-16
**Status**: Implemented 2026-08-16 (self-hosted flavour measured end to end
against a config-file operator-mode rig; synadia-cloud flavour stub-proven,
live run = the quickstart runbook)
**Input**: Design 0003 (`soul-hq/02-DESIGN/soulstream/0003-byo-nats.md`), resolving
composition 0001 §4's BYO [O]. Two flavours — a self-hosted operator-mode
server whose operator speaks `nsc`, and Synadia Cloud BYON driven through the
control-plane API. Operator mode required; conf-auth and NGS shared refused by
name. No operator or account master key ever travels; self-hosted, no seed
crosses the boundary in either direction.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Self-hosted founding through the kit (Priority: P1)

An operator who runs their own operator-mode NATS server founds a soulstream
realm on it: `soulstream init --byo self-hosted --url nats://…` emits the kit
(exact `nsc` commands and config fragments); the operator applies it and hands
back the two account public keys; `init` re-run verifies the substrate
behaviourally and performs the wire half (buckets, vault imports, first token,
sentinel). soulstream never authors an account, never pushes a JWT, never
edits the server's configuration.

**Acceptance Scenarios**:

1. **Given** an empty state dir and a reachable operator-mode server, **When**
   `soulstream init --byo self-hosted --url … --realm home` runs, **Then** it
   persists local material (three signing-key seeds, issuer-user seed, curve
   seeds — no operator, SYS, or account master seed), writes and prints the
   kit with exact values (no placeholders the operator must compute), and
   exits 0 telling the operator what to apply and how to hand back.
2. **Given** the kit applied verbatim (accounts authored from the kit's
   public keys, pushed to the server's resolver), **When** `init` re-runs with
   `--auth-account` and `--realm-account`, **Then** it mints the bypass-lane
   user creds (signing-key-signed, `IssuerAccount` set), verifies the
   substrate, boots the planes against the external URL, performs the wire
   half, runs one callout smoke round (sentinel + token admits the founding
   persona scoped to its own prefix; a garbage token refuses), writes the
   sentinel last, and prints the one token.
3. **Given** a founded BYO realm, **When** `soulstream up` runs, **Then** the
   node serves with the embedded server off, every plane on the external URL,
   and `init` re-run is a verified no-op naming the artifact count.
4. **Given** phase 1 done but the kit not yet applied, **When** `init` re-runs
   with no account keys, **Then** it re-emits the kit unchanged and exits 0 —
   idempotent, never an error, never a regeneration of existing seeds.

### User Story 2 - Refusals are named (Priority: P1)

Substrates that cannot carry the admission model are refused by name with the
migration spelled out — an honest break, never a degraded lane.

**Acceptance Scenarios**:

1. **Given** a target server not in operator mode (an anonymous connect
   succeeds), **When** phase 2 verification runs, **Then** the refusal names
   operator mode as the requirement and points at the kit's conversion
   fragments.
2. **Given** a partially applied kit (e.g. the realm account lacks the plain
   workload signing key), **When** phase 2 verification runs, **Then** the
   refusal names the specific kit item that was not applied. Verification
   never repairs the substrate.
3. **Given** a state dir founded with the embedded server, **When** `init`
   re-runs with `--byo`, **Then** it refuses: the substrate is fixed at
   founding.

### User Story 3 - Synadia Cloud BYON founding (Priority: P2)

The same end state driven through the control-plane API: accounts, signing-key
groups (programmatic — seeds returned exactly once, straight into local
state), the issuer user under an on-demand group, callout configuration. One
command, no kit document. The Cloud API token arrives via
`SOULSTREAM_SYNADIA_TOKEN` and is never persisted.

**Acceptance Scenarios**:

1. **Given** a Synadia Cloud system and a PAT in the environment, **When**
   `soulstream init --byo synadia-cloud --url … --synadia-system NAME` runs,
   **Then** the driver ensures the two accounts, three signing-key groups
   (realm scoped + realm plain + AUTH), callout enabled with the AUTH account
   as control account, the issuer user under an on-demand group with its
   creds downloaded — each step idempotent by name — and phase 2 proceeds as
   in Story 1. *(Automated coverage: a control-plane API stub asserting the
   call sequence, idempotence, and seed capture. The live Cloud run is the
   manual runbook in `quickstart.md` — the Entra-lane precedent.)*
2. **Given** the platform custodies the callout xkey and exposes no seed,
   **Then** the callout runs unsealed (the identity plane's `CalloutKey`
   stays empty) and the founding output says so — recorded honestly, never
   silently.

## Success Criteria

- **SC-001**: Against a stock operator-mode nats-server the test rig stands up
  (accounts authored from the kit's declared publics, memory resolver),
  BYO founding completes end to end and meets M1.1 semantics: status answers,
  sentinel + printed token admits the persona scoped to its own prefix,
  garbage refuses with an audited refusal, `init` re-run is a verified no-op.
- **SC-002**: The custody audit passes: after founding, the state dir contains
  no operator seed, no account master seed, no Cloud API token; on
  self-hosted, only public keys crossed outward and only two account public
  keys crossed back.
- **SC-003**: A conf-auth target and a partially applied kit each draw their
  named refusal (User Story 2); the wire carries no specific reason (D20's
  generic-refusal rule holds — reasons live in the local output/audit only).
- **SC-004**: The kit's `nsc` command sequence is validated against a real
  `nsc` (documented in `quickstart.md`); the scope template it declares is
  byte-identical to the embedded ceremony's.
- **SC-005**: `make fmt && make test && make lint` all green; no skipped
  tests; the embedded path's behavior is byte-for-byte unchanged (existing
  gates keep passing untouched).
