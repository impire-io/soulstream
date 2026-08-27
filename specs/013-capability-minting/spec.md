# Feature Specification: Capability minting in the house

**Feature Branch**: `013-capability-minting`
**Created**: 2026-08-27
**Status**: Draft
**Input**: soul-hq design [`0005-agent-declaration.md`](../../../soul-hq/02-DESIGN/soulstream-workloads/0005-agent-declaration.md) §5; workloads spec `010-capability-minting` (the scope carries the selectors; the scoped lane) and identity spec `004-agent-scope` (the exported agent template). The product's half: the founding grows the **agent capability key** — a scoped signing key on the realm account carrying `client.AgentScope*` — and the runtime plane routes capability-bearing declarations through the scoped lane. A declared agent then reaches exactly its granted tools, refused at the transport, with zero authorization code in the runtime (design 0005 §10 #3).

## The load-bearing finding (recorded openly)

The planned wiring — importing an agent role into the identity vault and minting via D28 `mint.ephemeral` — would have **broken every token-lane sign-in**: identity's binding-resolved lanes (`RoleForAccount`) refuse a multi-role account as ambiguous, by decided design (D5 as amended; measured by identity's M3 gate proof 6). The unlock is identity's already-named "token lane's named-role answer" — not this arc's to build. The product therefore mints capability credentials **locally** through workloads' `ScopedSigningKeyMinter` (the identical D28 claim shape signed by the state-held agent role seed); enforcement stays entirely the server's template expansion. The D28 op lane remains the fleet-era path for seedless nodes.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A declared agent reaches exactly its granted tools (Priority: P1)

An operator declares an agent with `capabilities: {role: "agent", tools: [...]}` and runs it through `soulstream workload start`. The workload's minted credential is scoped by the realm account's agent template plus the declaration's tags: granted tool subjects work, everything else on the tool namespace refuses at the server.

**Independent Test**: The upstream `scope-probe` reference workload, declared with capabilities granting its probe subject, completes (in-scope allowed AND out-of-scope denied → exit 0 → work.done); declared with a *different* tool granted, its in-scope publish is denied (exit 2 → work.abandon) — the narrowing bites.

**Acceptance Scenarios**:

1. **Given** a capability declaration granting `probe-ping`, **When** it runs, **Then** the work item completes `done` (the probe's own verdict: granted allowed, ungranted denied).
2. **Given** the same declaration granting only `other-tool`, **When** it runs, **Then** the work item ends `abandoned` (the probe's in-scope publish was denied — the credential really is narrowed).

---

### User Story 2 - Founding grows the agent key; old realms refuse by name (Priority: P1)

A fresh `soulstream init` founds the realm with the agent capability key (scoped signer, `client.AgentScope*` template) beside the plain workload key. A realm founded before this feature runs everything it ran yesterday; a capability-bearing declaration on it refuses by name with the migration in the refusal (pre-v1 clean break).

**Acceptance Scenarios**:

1. **Given** a fresh founding, **Then** the realm JWT endorses the agent key as a scoped signer and the state dir holds `keys/agent-signing.nk` (0600).
2. **Given** a state dir without the agent key, **When** a capability declaration launches, **Then** the refusal names the missing key and the re-init migration; a capability-less declaration runs unchanged.
3. **Given** a capability role other than the founding's `agent`, **When** it launches, **Then** the refusal names the one declared role.

---

### Edge Cases

- The identity vault gains **no** new role: `RoleForAccount` stays single-role on the realm account and the token lane keeps working (the finding above).
- BYO self-hosted: phase 1 generates the agent key; the kit's nsc lines endorse it scoped with the exported template. Synadia Cloud BYON: **named [O]** — the driver's fourth signing-key group lands with a live-run measurement (the 0136 discipline); until then a BYON realm has no agent key and capability declarations refuse by the same named refusal.
- The persona-scope adoption promised at D47 lands here: the ceremony renders `client.PersonaScopePubAllow/SubAllow` instead of its local copy (byte-identical lists, now one source).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `ceremony.Generate` MUST create the agent capability key and endorse it on the realm account as a scoped signer rendering `client.AgentScopePubAllow("")`/`AgentScopeSubAllow("")` under `client.AgentScopeRole`; the seed persists as `keys/agent-signing.nk` (0600).
- **FR-002**: State load MUST treat the agent seed as load-if-present (both flavours); verify MUST assert scoped endorsement when present; founded realms without it MUST still load.
- **FR-003**: `RunWorkload` MUST route capability-bearing declarations through `minter.ScopedSigningKeyMinter` on the agent seed, and capability-less ones through the plain lane byte-identically.
- **FR-004**: `RunWorkload` MUST refuse, before any op publishes, a capability declaration on a realm without the agent key (naming the migration) and a capability role other than `ceremony.AgentRole`.
- **FR-005**: The self-hosted kit MUST carry the agent key's nsc lines (scoped signer with the exported template); `AgentScopeAllows` exposed beside `PersonaScopeAllows` for the account-half drivers.
- **FR-006**: The identity vault import set MUST NOT grow (no second role on the realm account).
- **FR-007**: The ceremony's persona scope MUST render from `client.PersonaScopePubAllow/SubAllow` (the D47 adoption), lists byte-identical.

## Success Criteria *(mandatory)*

- **SC-001**: TestM14CapabilityAgent — the granted probe completes `done`; the narrowed probe ends `abandoned`; both through the full authority chain (founding → scoped signer → tagged local mint → server expansion) [measured in `make test`].
- **SC-002**: The legacy-realm and wrong-role refusals are named errors before any op publishes [measured].
- **SC-003**: All existing gates green — `make check`; the token lane untouched (no vault change to break it).

## Assumptions

- go.mod pins: workloads/identity changes ride sibling working trees via an untracked `go.work` until their mains are pushed; the pin bump (workloads > v0.8.0-rc.1, identity > 2026-08-27 pseudo-version) is the standing exception, resolved with `go mod tidy` after push — the episode 0089 precedent.
- `workloadCredTTL` (1h) stays the TTL for both lanes.
