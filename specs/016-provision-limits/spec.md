# Feature Specification: Provisioning Byte Limits

**Feature Branch**: `016-provision-limits`
**Created**: 2026-07-27
**Status**: Draft
**Input**: User description: "Provisioning byte limits: realm.ProvisionOn (and the CLI provision command) must be able to set byte limits (MaxBytes) on the artefacts it creates — the op-log stream, the notify stream (already has a mandated 64MiB), the registry KV, and the object store — so that provisioning succeeds on limit-enforced NATS accounts (NGS R1 tier requires MaxBytes on every stream; today provision fails with err 10113 and the operator must pre-create everything by hand). Create-or-report semantics must hold: existing artefacts are never mutated, limits on existing artefacts are reported, absent limits on a limit-requiring account produce the same clear failure as today. Defaults should make an NGS R1 account work out of the box while unlimited self-hosted realms stay exactly as they are."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Provision a realm on a limit-enforced account (Priority: P1)

An operator points Soulstream at a hosted NATS account whose tier requires an
explicit storage budget on everything it creates (NGS R1 is the known case).
They ask for a limit-honouring provision and the realm comes up in one
command — op-log, notify stream, persona directory, and attachment store all
created with sensible storage budgets — with no manual pre-creation of any
artefact.

**Why this priority**: This is the reason the feature exists. Today this
operator gets error 10113 and has to hand-create three artefacts with the
right shapes before provisioning will report a realm conformant — the
documented workaround since 2026-07-21.

**Independent Test**: Against a server configured to reject unlimited
streams (or any account tier that does), one provisioning run with the
limits option creates every artefact successfully; the same run without the
option still fails with the clear per-artefact error.

**Acceptance Scenarios**:

1. **Given** an empty limit-enforced account, **When** the operator
   provisions with storage budgets enabled, **Then** every realm artefact is
   created with an explicit budget and the report shows a conformant realm.
2. **Given** an empty limit-enforced account, **When** the operator
   provisions without any budgets (today's behavior), **Then** provisioning
   fails with an error naming the artefact that was refused, exactly as
   today.

---

### User Story 2 - Choose the budgets explicitly (Priority: P2)

An operator with opinions (or a bigger paid tier) sets each artefact's
budget: the op-log gets what history deserves, the attachment store gets
what files deserve, the persona directory stays small. Only the budgets they
name deviate from the defaults.

**Why this priority**: Defaults fit the known free tier; real deployments
have real quotas. Without per-artefact control the feature would just move
the hand-tuning from creation time to a different workaround.

**Independent Test**: Provision with one named budget and verify that
artefact carries it while the others carry defaults.

**Acceptance Scenarios**:

1. **Given** budgets enabled with an explicit attachment-store budget,
   **When** the realm is provisioned, **Then** the attachment store carries
   the explicit budget and the remaining artefacts carry the documented
   defaults.

---

### User Story 3 - Re-provision an existing realm honestly (Priority: P3)

An operator re-runs provisioning against a realm that already exists —
including one whose artefacts were hand-created during the workaround era.
Nothing is mutated; the report states each artefact's current budget so the
operator can see how the realm actually stands.

**Why this priority**: Create-or-report is a standing promise of
provisioning; budgets must not weaken it. The hand-created NGS realm from
the workaround era is a live instance that must keep reporting conformant.

**Independent Test**: Provision twice with different budget choices; the
second run changes nothing and reports the budgets from the first.

**Acceptance Scenarios**:

1. **Given** a realm provisioned with budgets, **When** provisioning runs
   again with different (or no) budgets, **Then** no artefact is modified
   and the report shows the budgets the artefacts actually carry.
2. **Given** the realm whose artefacts were hand-created with the
   workaround's shapes, **When** provisioning runs with budgets enabled,
   **Then** the realm reports conformant and nothing is mutated.

---

### Edge Cases

- A budget of zero or a negative number is rejected up front with a clear
  message — it never reaches the server (zero means "unlimited" in the
  underlying system, which would silently contradict the operator's intent
  on a limit-enforced account).
- The notify stream already carries a mandated budget; enabling budgets
  must not change it, and an explicit notify budget overrides the mandate
  only for creation of a missing notify stream (an existing one is reported,
  never resized).
- Budgets on an account that does not require them: honoured exactly as
  given — a self-hosted realm may want budgets too.
- Existing artefact whose budget differs from the requested one: reported
  with both values visible in the outcome; never treated as an error, never
  mutated.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provisioning MUST accept an optional storage budget (in
  bytes) for each realm artefact it can create: the op-log stream, the
  notify stream, the persona directory, and the attachment store.
- **FR-002**: Provisioning MUST offer a single switch that applies
  documented default budgets fitting the known limit-enforced free tier
  (the shapes proven by the manual workaround: 1 GiB op-log, 64 MiB notify,
  64 MiB persona directory, 512 MiB attachment store), so that tier works
  out of the box with no per-artefact tuning.
- **FR-003**: With no budget options, provisioning MUST behave exactly as
  today: no budgets are applied, unlimited accounts provision unchanged,
  and limit-enforcing accounts fail with the same clear per-artefact error
  as today.
- **FR-004**: Provisioning MUST never modify an existing artefact,
  regardless of budget options; existing artefacts' budgets MUST be
  reported as found.
- **FR-005**: Explicit budgets MUST be validated before any server contact:
  zero and negative values are rejected with a message naming the artefact
  and the reason.
- **FR-006**: The CLI provision command MUST expose the same choices —
  defaults switch and per-artefact budgets — and its report MUST show each
  artefact's budget (or "unlimited") so an operator can read the realm's
  standing at a glance.
- **FR-007**: The plain-words documentation MUST explain budgets: why
  limit-enforced tiers need them, what the defaults are, and that existing
  realms are never resized by provisioning.

### Key Entities

- **Storage budget**: a per-artefact byte ceiling chosen at creation time;
  absent means unlimited (today's behavior).
- **Provision outcome (extended)**: the existing per-artefact
  created/existed report, now also carrying the budget each artefact has.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On a fresh limit-enforced account, a realm provisions in one
  command with zero manually pre-created artefacts (today: three).
- **SC-002**: On an unlimited account with no budget options, provisioning
  behavior is indistinguishable from the current release for every
  artefact (created shapes and report identical).
- **SC-003**: Re-running provisioning against the existing hand-created NGS
  realm mutates nothing and reports conformant, with each artefact's budget
  visible in the output.
- **SC-004**: An operator following only the documentation provisions a
  limit-enforced account successfully on the first attempt.

## Assumptions

- Default budgets are the empirically proven workaround shapes
  (1 GiB / 64 MiB / 64 MiB / 512 MiB); they exist to fit the known free
  tier, not to model anyone's real capacity needs.
- Budgets apply at creation only. Changing an existing realm's budgets is
  an operator act with operator tools, out of scope here — provisioning
  reports, it does not resize.
- Budget options live on the provisioning surfaces (library option, CLI
  flags). The identity config file keeps carrying identity only; realm
  shape does not belong there.
- Per-message size caps, message-count caps, and age-based retention are
  out of scope — this feature is about the byte ceilings that
  limit-enforced tiers refuse to create without.
