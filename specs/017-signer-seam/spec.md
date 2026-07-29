# Feature Specification: Signer Seam

**Feature Branch**: `017-signer-seam`
**Created**: 2026-07-29
**Status**: Draft
**Input**: User description: "Signer interface seam (017): decouple record signing from the concrete local Ed25519 key so signing can be delegated to an external custodian (SoulIdentity's NATS sign/record service) without soulstream depending on it. Introduce a small Signer abstraction in the identity package (public key + fallible signing over canonical bytes); the existing local SigningKey satisfies it. realm.Client's Signer config becomes the abstraction; the topic publish chokepoint must treat a signing failure as a publish failure (never silently publish unsigned when a signer is configured). Registry statement-signing surfaces (attestation token, rotation proof) accept the abstraction where signing-only capability is needed. The keystore and key-generation surfaces stay concrete — they custody seeds, which a delegated signer never possesses. identity package continues to import no NATS. Gate: any Signer implementation (proven with a test double simulating remote delegation, including failure injection) produces records that verify identically to local-key signing, and a failing signer aborts the publish loudly."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sign through a custodian, verify like any other record (Priority: P1)

A persona's signing key does not live on the machine running Soulstream: it is
guarded by an external custodian (an identity service that signs on the
persona's behalf and never releases the key). A consumer of the Soulstream
library supplies its own signing implementation — one that asks the custodian
for each signature — and everything the persona publishes (turns, comments,
work items, profile activity) carries a signature that every existing reader
verifies exactly as if the key had been local. Readers, archivists, and
offline verifiers cannot tell — and must not need to care — where the
signature was produced.

**Why this priority**: This is the reason the feature exists. It is the
half of SoulIdentity's "consumers wire in" milestone that lives in this
repository, and the prerequisite for the remote multi-user node, where
holding every user's seed locally would be unacceptable custody.

**Independent Test**: Configure a client with a substitute signing
implementation (a test double standing in for the remote custodian, holding
the key behind the same narrow contract) and publish operations; confirm
unmodified readers verify them and that the wire form is indistinguishable
from local-key output.

**Acceptance Scenarios**:

1. **Given** a client configured with a delegated signing implementation for
   persona P whose public key is published in the persona directory, **When**
   P posts a turn, **Then** the stored operation carries P's signature and
   every reader reports it verified — identical verdict and wire shape to the
   same operation signed by a locally held key.
2. **Given** the same canonical bytes and the same underlying key, **When**
   the operation is signed locally and via the delegated implementation,
   **Then** the resulting signatures are identical — delegation adds nothing
   and loses nothing.
3. **Given** a client configured with no signer at all, **When** it publishes,
   **Then** the operation goes out unsigned exactly as before this feature
   existed — absence of a signer keeps its current meaning.

---

### User Story 2 - A failing custodian never produces an unsigned record (Priority: P2)

The custodian is a remote service: it can be down, refuse the request (the
caller may not act as that persona), or time out. When a client that is
configured to sign cannot obtain a signature, the operation must not be
published at all — not unsigned, not partially — and the caller must receive
an error that names the underlying cause so the operator can act on it.

**Why this priority**: Signing exists for accountability. A silent fallback
to unsigned would convert a custodian outage into a quiet integrity downgrade
that readers have no way to distinguish from a persona that never signs —
the exact substitution scenario the signing design defends against.

**Independent Test**: Configure a client with a signing implementation that
fails on demand (error injection) and attempt to publish; confirm nothing
landed on the log and the returned error carries the injected cause.

**Acceptance Scenarios**:

1. **Given** a client whose configured signer returns an error, **When** it
   attempts any publishing operation, **Then** the operation fails, nothing
   is appended to the topic log, and the error presented to the caller names
   the signing failure.
2. **Given** a client whose configured signer returns an empty signature,
   **When** it attempts to publish, **Then** the attempt is treated exactly
   like a signing error — an empty result must never travel as "unsigned".

---

### User Story 3 - Operator statements through the same seam (Priority: P3)

Beyond operation records, personas sign standalone statements: the operator
attestation ("I operate this persona") and the key-rotation proof (the old
key endorsing the new). An operator whose key lives with a custodian must be
able to produce these statements through the same delegated contract, where
only the ability to sign — never the seed — is required.

**Why this priority**: Completes the seam so no signing surface silently
requires local key material; without it, a custodian-held operator key could
publish records but never attest or rotate. It is lower priority because the
day-one consumer (the remote node) needs record signing first.

**Independent Test**: Produce an attestation token and a rotation proof via
a delegated signing double; confirm existing verification paths accept both.

**Acceptance Scenarios**:

1. **Given** an operator whose key is behind a delegated signer, **When**
   they produce an attestation token for a persona they operate, **Then**
   the token verifies through the existing attestation verification exactly
   as a locally signed one.
2. **Given** a rotation where the old key signs through a delegated signer,
   **When** the rotation is published, **Then** the rotation chain validates
   exactly as with local keys.

---

### User Story 4 - Existing local-key users notice nothing (Priority: P1)

Every current user signs with a key loaded from a local seed file (CLI and
MCP flows). After this feature, those flows behave identically: same
commands, same config, same key files, same signatures, same errors. The
local key is simply the first implementation of the new contract.

**Why this priority**: This seam refactors the most security-sensitive path
in the library; regressing the shipped signing behavior for every existing
user would be worse than not shipping the seam. It shares P1 because it is a
constraint on the P1 work, not a follow-up.

**Acceptance Scenarios**:

1. **Given** an existing user with a seed file and published profile,
   **When** they run any signing workflow (post, comment, attest, rotate)
   after the upgrade, **Then** behavior, wire output, and verification
   results are unchanged.
2. **Given** the existing test suites for signing, rotation, attestation,
   and verification, **When** run against the seamed code, **Then** all
   pass with unchanged expectations about user-visible behavior.

---

### Edge Cases

- A delegated signer that hangs: the seam itself imposes no timeout;
  bounding the wait is the implementation's responsibility (documented in
  Assumptions). A hang therefore blocks that publish call, not the log's
  integrity — no partial or unsigned record can result.
- A delegated signer returning malformed signature material (wrong length,
  not decodable): the record travels with it and readers report signature
  failure, exactly as they already do for corrupt signatures today; the
  publish side is not a verifier. Only the *empty* result is a publish-side
  failure (US2), because emptiness silently changes the record's meaning
  from "signed" to "unsigned".
- The signer's key differs from the persona's published key: readers report
  a failed signature (key mismatch) exactly as they do today for a wrong
  key — delegation introduces no new trust; the persona directory remains
  the authority on whose signature counts.
- A client with a persona but no signer, or a signer but no persona:
  unchanged from today — unsigned publishing remains legal, and a persona
  is still required to post at all.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The library MUST accept, everywhere a client is configured to
  sign, any signing implementation that satisfies a minimal contract:
  reveal the public identity it signs as, and produce a signature over given
  canonical bytes — an operation that may fail.
- **FR-002**: The existing locally held signing key MUST satisfy that
  contract unchanged in capability, and remain the default way local users
  sign.
- **FR-003**: Records signed through any conforming implementation MUST be
  indistinguishable on the wire from locally signed records, and MUST verify
  identically through every existing read-side path (materialise, follow,
  inbox, exhibits, offline verify).
- **FR-004**: When a client is configured with a signer and signing fails,
  the publishing operation MUST fail without anything being published, and
  the returned error MUST name the signing failure as its cause. There MUST
  be no fallback to unsigned publishing.
- **FR-005**: An empty signature result from a configured signer MUST be
  treated as a signing failure (FR-004), never as an unsigned record.
- **FR-006**: A client configured with no signer MUST continue to publish
  unsigned, byte-for-byte as today.
- **FR-007**: The statement-signing surfaces — operator attestation tokens
  and key-rotation proofs — MUST accept the same contract wherever only
  signing capability (not seed custody) is required.
- **FR-008**: Seed custody surfaces (key generation, seed file save / load /
  replace) MUST remain exclusive to locally held keys; the signing contract
  MUST NOT expose, require, or imply access to secret key material.
- **FR-009**: The identity abstraction MUST remain free of messaging-system
  dependencies (the `record`/`identity` no-NATS rule stands); the library
  MUST NOT gain a dependency on any external custodian to provide the seam.
- **FR-010**: Existing user-facing workflows (CLI commands, MCP tools,
  configuration files, key file formats) MUST be unchanged in behavior and
  interface by this feature.

### Key Entities

- **Signer (the contract)**: the narrow capability "identify the public key
  you sign as; sign these canonical bytes, possibly failing". The unit this
  feature introduces; everything that signs depends on it, nothing that
  signs depends on where the key lives.
- **Local signing key**: the existing Ed25519 seed-backed key; the first and
  default implementation of the contract; the only thing seed-custody
  surfaces ever touch.
- **Delegated signer**: any implementation that obtains signatures from
  elsewhere (the external custodian service being the motivating case). Lives
  outside this codebase; represented here only by test doubles.
- **Operation record signature**: the existing signature over a record's
  canonical bytes; its wire form, verification semantics, and absence
  semantics are invariants this feature must not move.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operation published through a delegated signing
  implementation is verified by every existing read-side path with the same
  verdict, and its signature is byte-identical to local-key output for the
  same key and bytes — demonstrated across the library's read surfaces
  (live follow, materialise, inbox, exhibit verification, offline verify).
- **SC-002**: Under injected signing failure, 100% of attempted publishes
  abort with an error naming the cause and 0 operations reach the log —
  demonstrated for every operation-publishing surface and for the empty-
  signature case.
- **SC-003**: Attestation tokens and rotation proofs produced through a
  delegated implementation pass the existing verification paths unchanged.
- **SC-004**: All pre-existing behavioral expectations hold: the full test
  suite passes with local keys and unsigned clients behaving exactly as
  before, and the library's dependency set is unchanged (no new external
  dependencies to provide the seam).

## Assumptions

- The remote custodian implementation itself (the adapter that calls
  SoulIdentity's sign service) ships outside this repository — SoulIdentity's
  M2 places it with the consumers; this feature proves the seam with test
  doubles, including failure injection. The live cross-service "signed
  through the service, verified in the realm" measurement belongs to that
  milestone, not this feature.
- Timeout and retry policy for remote signing belongs to the signer
  implementation, not the seam: the seam requires only that failure surfaces
  as an error. If real consumers later prove they need caller-side deadline
  propagation through the seam, that is an additive follow-up.
- Changing the shape of the library's signing surfaces (a fallible signing
  call where today's is infallible) is acceptable at this module's maturity,
  provided user-visible CLI/MCP behavior is untouched (FR-010); compile-time
  breakage for library importers is a documented cost, caught at compile
  time, with the local key remaining a drop-in.
- No configuration surface (files, environment, flags) learns to express
  "delegated signer" in this feature — wiring a remote signer is code-level,
  arriving with the consumer that needs it (the remote node). Out of scope
  here.
