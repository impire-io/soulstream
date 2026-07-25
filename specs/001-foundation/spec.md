# Feature Specification: Foundation — Realm Provisioning & the Operation Record

**Feature Branch**: `001-foundation`
**Created**: 2026-07-12
**Status**: Shipped (v0.1.0)
**Input**: User description: "Soulstream foundation library (Go): the wire layer — realm provisioning and the operation record on NATS JetStream, plus identity basics. Deferred: topics/lifecycle, baselines, materialisation, mentions, discovery, CLI, MCP, and Ed25519 signing implementation."

## Overview

This feature delivers the **wire layer** of Soulstream: the smallest working slice on which every later capability is built. It gives a realm a home (a provisioned stream and object store) and gives every future operation a well-defined shape (the operation record and its canonical form). It carries no collaboration behaviour yet — no topics, no conversation, no lifecycle — only the substrate those things will stand on.

The consumers of this feature are **library integrators** (people building clients, adapters, and agents on top of Soulstream) and the **realm operator** (whoever administers the NATS account). The value is that both can rely on one provably-correct realm setup and one provably-lossless record format, so that no later feature has to re-litigate "how is a realm shaped?" or "what does an operation look like on the wire?".

## Clarifications

### Session 2026-07-12

- Q: What exact format is an operation identity? → A: A standard **UUIDv4 rendered lowercase, hyphenated `8-4-4-4-12`** (e.g., `9f86d081-b6c4-4a3e-9e0c-1f2a3b4c5d6e`). It is unique without coordination, transport-token-safe, and collision-safe within a realm's duplicate window. The abbreviated identities shown in the reference design are illustrative for readability, not a truncation rule to reproduce.
- Q: What is the exact persona-name grammar? → A: **Lowercase alphanumerics in hyphen-separated groups** — `^[a-z0-9]+(-[a-z0-9]+)*$` — length **1–64** characters. No uppercase, no leading/trailing/consecutive hyphens, no dots (dots are subject separators), no whitespace, and none of the transport wildcards `*`/`>`. The same grammar validates realm names and topic-id slugs.
- Q: When an existing realm artefact drifts from mandated settings, does provisioning fix it? → A: **No. Provisioning is create-or-report and never modifies an existing stream or object store in place.** It creates only missing artefacts and reports the conformance of existing ones (conformant vs. the specific nonconformity). Any reconfiguration of an existing artefact is an explicit, separate operator action outside this feature — mixing "safe auto-fix" with the history-destroying cases (age expiry added, rollup disabled) is exactly the ambiguity the design refuses.
- Q: What does "verify the author on read" mean in the transport-scoped default, where a plain stream message carries no trusted publisher identity? → A: **Write-side enforcement is mandatory; read-side verification is structural plus optional cross-check.** A persona-bound client stamps only its own configured persona name and refuses to publish an operation attributed to a different author. On read, the library validates that the author is present and a well-formed persona name, and — when the caller supplies a trusted-identity resolver — cross-checks the claimed author against it and flags mismatches. The library does not claim to detect spoofing it cannot observe in the plain transport-scoped model; that is the operator's auth-callout or the deferred signature's job.
- Q: Where does the realm name (bound into every canonical record) come from? → A: **From explicit, required client configuration.** A JetStream client cannot reliably read its own NATS account name, and the realm name must be stable for future signing, so it is supplied when the client is constructed, validated as a lowercase slug, and bound verbatim into every canonical record.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Provision a realm from nothing (Priority: P1)

An operator points the library at a running NATS server (via a named NATS context) and asks it to prepare a realm. The library creates the one stream and the one object store the protocol requires, with exactly the settings the protocol mandates, and reports what it did.

**Why this priority**: Nothing else in Soulstream can exist without a correctly-configured stream and object store. This is the true floor of the system; it is also the highest-risk step to get wrong because the stream's retention and rollup settings are one-way doors that corrupt history if set incorrectly.

**Independent Test**: Point the library at an empty NATS server and run provisioning; then inspect the server and confirm the `SOULSTREAM` stream exists capturing `SOULSTREAM.>` with the mandated settings, and the `soulstream-objects` object store exists. Delivers value: a realm ready to receive operations.

**Acceptance Scenarios**:

1. **Given** a running NATS server with JetStream and no Soulstream artefacts, **When** the operator runs provisioning against a named context, **Then** a `SOULSTREAM` stream is created that captures subject `SOULSTREAM.>`, uses limits-based retention with no age-based expiry, permits subject rollup, has a duplicate-tracking window of at least two minutes, and stores to disk; and a `soulstream-objects` object store is created.
2. **Given** a NATS server the library cannot reach or one without JetStream enabled, **When** provisioning runs, **Then** it fails with a clear, actionable error that names the cause and changes nothing.
3. **Given** the operator names a NATS context that does not exist, **When** provisioning runs, **Then** it fails before contacting any server, naming the missing context.

### User Story 2 - Re-provision an existing realm safely (Priority: P1)

An operator runs provisioning again — on redeploy, in a startup script, or just to check state. The library treats an already-correct realm as a no-op and reports the current state rather than erroring or mutating anything.

**Why this priority**: Provisioning that is not idempotent cannot be put in a startup path or a deploy script, which is exactly where it belongs. Idempotence is what makes "setup is a documented script" (the MVP's stated deployment model) safe to run repeatedly.

**Independent Test**: Run provisioning twice against the same server; confirm the second run makes no changes, does not error, and reports that the stream and object store already exist as required.

**Acceptance Scenarios**:

1. **Given** a realm already provisioned correctly, **When** provisioning runs again, **Then** it makes no changes, succeeds, and reports each artefact as already present and conformant.
2. **Given** a `SOULSTREAM` stream that exists but drifts from the mandated settings (for example, an age-based expiry was added, or rollup was disabled), **When** provisioning runs, **Then** the drift is reported clearly as a conformance problem, and no change is made silently that could destroy history.
3. **Given** a realm where the stream exists but the object store is missing, **When** provisioning runs, **Then** the missing object store is created and the existing stream is left untouched.

### User Story 3 - Build and read back an operation record losslessly (Priority: P1)

A library integrator constructs an operation — an author, a type, a set of parent operations, a timestamp, and a pure-data payload — and hands it to the library, which produces the exact set of message headers and payload that go on the wire. Reading that message back reproduces the original operation with nothing added, dropped, or reinterpreted.

**Why this priority**: Every operation in every later feature is one of these records. If construction and parsing are not exactly inverse, every downstream projection, signature, and merge is built on sand. It is also independently valuable: it can be tested with no server at all.

**Independent Test**: Construct a record with representative values (single parent, multiple parents, empty parents; with and without an optional signature), serialise to wire form, parse back, and assert equality. Delivers value: a trustworthy operation format usable by every later feature.

**Acceptance Scenarios**:

1. **Given** an operation with an author, type, timestamp, one or more parents, and a payload, **When** it is put into wire form and parsed back, **Then** the parsed operation equals the original in every field, including an empty-parents case and a many-parents case.
2. **Given** an operation with no parents, **When** it is serialised, **Then** the parents header is absent (not an empty string), and parsing an absent parents header yields an empty parent list.
3. **Given** a message whose operation-identity value is reused on a retry, **When** it is published, **Then** the server treats the second publish as a duplicate of the first rather than a new operation.
4. **Given** a message missing a required record field, or carrying an unsupported record version, **When** it is parsed, **Then** parsing fails with an error that names the specific violation.

### User Story 4 - Produce and verify the canonical record (Priority: P2)

A library integrator needs a stable, byte-for-byte serialisation of an operation that lives outside the NATS message — for a future signature, a future exported exhibit, or a future sealed-topic inner operation. The library produces one canonical form from a record and reproduces it identically regardless of field ordering in the source.

**Why this priority**: The canonical form is what signing and portable evidence will consume. It must exist and be correct now (it is a one-way door — anything signed later must match bytes produced now), even though signing itself is deferred. It is lower than P1 because no live operation depends on it yet.

**Independent Test**: Produce the canonical form of the same logical record presented with differently-ordered fields and confirm the output bytes are identical; confirm the canonical form round-trips to the same record; confirm it binds the realm and topic identifiers.

**Acceptance Scenarios**:

1. **Given** two records identical in content but differing in the order their fields were supplied, **When** each is canonicalised, **Then** the resulting bytes are identical.
2. **Given** a wire message and its canonical record, **When** the mapping is applied in both directions, **Then** it is lossless — every wire field maps to exactly one canonical field and back.
3. **Given** a canonical record, **When** it is produced, **Then** it carries the realm and topic identifiers so that the same operation cannot be presented as belonging to a different realm or topic.

### User Story 5 - Validate persona names and attribution (Priority: P2)

A library integrator relies on the library to reject malformed persona names when constructing operations, and to reject, on read, any operation whose claimed author does not match the identity that actually delivered it.

**Why this priority**: Honest attribution inside a high-trust realm is a convention the library enforces at the edges. Getting name validity and read-time attribution checks right here means every later feature inherits them for free. It is P2 because the transport (credentials and subject permissions) is the primary enforcement; this is the library-side backstop.

**Independent Test**: Feed valid and invalid persona names to the validator and confirm the exact accept/reject set; simulate an operation whose claimed author differs from the delivering identity and confirm the library rejects it on read.

**Acceptance Scenarios**:

1. **Given** a candidate persona name, **When** it is validated, **Then** lowercase token-safe slugs are accepted and names with uppercase, spaces, dots, or transport-reserved characters are rejected with a reason.
2. **Given** an operation read from the stream whose claimed author does not match the identity that published it, **When** the library surfaces the operation, **Then** it is rejected or flagged as an attribution mismatch rather than trusted silently.

### Edge Cases

- **Stream partially created**: provisioning was interrupted, leaving the stream but not the object store (covered by US2 scenario 3) or vice versa — provisioning completes the missing part without disturbing the present part.
- **Setting drift on a history-bearing realm**: the stream exists with unsafe drift (age-based expiry present, or rollup disabled) — the library never silently "fixes" a setting whose change could delete history; it reports and leaves the decision to the operator.
- **Operation-identity collision within the duplicate window**: two genuinely different operations are given the same identity value by a buggy caller — the second is silently absorbed as a duplicate; the library's contract is that identity values are unique per operation and it is the caller's job not to reuse them for different content.
- **Oversized or binary payload**: a caller tries to place a payload that is binary or approaches the message size ceiling — the record format's contract is text-and-references-only; oversized/binary content belongs in the object store (that mechanism is defined here as a bucket but its attachment vocabulary is a later feature).
- **Unknown `Soulstream-*` headers**: a message carries reserved-prefix headers the current version does not define — they are preserved and passed through untouched, never stripped.
- **Absent vs empty parents**: distinguished — no parents means the header is absent; the two must never be conflated.
- **Malformed timestamp or version**: a record with a non-RFC-3339 timestamp or a non-integer/unsupported version is rejected on parse with a specific error.

## Requirements *(mandatory)*

### Functional Requirements

**Connection**

- **FR-001**: The library MUST establish its NATS connection from a named NATS context (the standard Synadia context mechanism), so credentials and server addresses are supplied by configuration, never hard-coded.
- **FR-002**: The library MUST fail fast with a named, actionable error when the requested context is missing, the server is unreachable, or JetStream is unavailable — without partially mutating a realm.

**Realm provisioning**

- **FR-003**: The library MUST provision a realm's `SOULSTREAM` stream capturing subject `SOULSTREAM.>`.
- **FR-004**: The provisioned stream MUST use limits-based retention with **no age-based expiry**, permit subject rollup headers, use a duplicate-tracking window of **at least two minutes**, and use disk-backed storage.
- **FR-005**: The library MUST provision a realm's `soulstream-objects` object store.
- **FR-006**: Provisioning MUST be idempotent: running it against an already-conformant realm changes nothing, succeeds, and reports each artefact's current state.
- **FR-007**: Provisioning MUST create only the parts that are missing, leaving present-and-conformant parts untouched.
- **FR-008**: Provisioning MUST NOT modify an existing stream or object store in place. It is **create-or-report**: it creates only missing artefacts and, for existing ones, reports conformance without mutating any setting — including safe-looking drift, because the history-destroying cases (age-based expiry present, rollup disabled) make in-place reconfiguration a one-way risk. Reconfiguring an existing artefact is an explicit operator action outside this feature.
- **FR-009**: Provisioning MUST report a structured result naming, per artefact, whether it was created, already present and conformant, or present but non-conformant (with the specific nonconformity).

**The operation record**

- **FR-010**: The library MUST represent an operation as a record carried entirely in message headers with a pure-data payload; no protocol metadata is ever embedded in the payload.
- **FR-011**: The record MUST carry: an operation identity, a wire-format version, an author, an ordered set of parent operation identities, an operation type, an author-claimed timestamp, and an optional signature slot.
- **FR-012**: The operation identity MUST double as the message's duplicate-detection identity, so that a retried publish of the same identity is de-duplicated by the server rather than creating a second operation.
- **FR-013**: The library MUST generate operation identities that are unique per operation without coordination, each a UUIDv4 rendered lowercase and hyphenated (`8-4-4-4-12`).
- **FR-014**: The library MUST serialise a record to wire form and parse wire form back to a record such that the two directions are exact inverses for every field.
- **FR-015**: An empty set of parents MUST serialise to an **absent** parents header (not an empty value), and an absent parents header MUST parse to an empty set — the two states are equivalent and must never be confused with each other.
- **FR-016**: Parsing MUST reject a message that is missing a required field, carries an unsupported version, or carries a malformed timestamp — each with an error naming the specific violation.
- **FR-017**: The library MUST preserve unknown headers under the reserved protocol prefix, passing them through untouched rather than stripping them.
- **FR-018**: The timestamp MUST be treated as informational and author-claimed; the library MUST NOT use it as an ordering authority.

**The canonical record**

- **FR-019**: The library MUST produce a canonical serialisation of a record (a deterministic, field-order-independent byte sequence) suitable for use as future signing input, exported evidence, and sealed-topic inner operations.
- **FR-020**: The canonical serialisation MUST be identical for two records that are equal in content regardless of the order their fields were supplied.
- **FR-021**: The mapping between wire form and canonical form MUST be lossless in both directions.
- **FR-022**: The canonical record MUST bind the realm identifier and the topic identifier, so an operation cannot be re-presented as belonging to another realm or topic.
- **FR-023**: The library MUST accept an optional signature slot in both wire and canonical forms and carry it through untouched; producing or verifying signatures is **out of scope** for this feature.

**Identity basics**

- **FR-024**: The library MUST validate persona names against the grammar `^[a-z0-9]+(-[a-z0-9]+)*$`, length 1–64, accepting valid names and rejecting invalid ones with a reason. The same grammar validates realm names and topic-id slugs.
- **FR-025**: The library MUST enforce honest attribution on the **write** side: a persona-bound client stamps only its own configured persona name and MUST refuse to publish an operation attributed to a different author. On **read**, the library MUST validate that the author is present and a well-formed persona name, and — when the caller supplies a trusted-identity resolver — MUST cross-check the claimed author against it and flag mismatches. The library MUST NOT claim to detect spoofing unobservable in the plain transport-scoped model (that is the operator's auth-callout or the deferred signature's role).
- **FR-028**: The realm name MUST be supplied as required client configuration, validated as a lowercase slug (FR-024 grammar), and bound verbatim into every canonical record (FR-022); the library MUST NOT attempt to infer it from the NATS account.

**Discipline & non-goals**

- **FR-026**: The record's payload contract MUST be text-and-references-only; the format MUST NOT be used to carry binary blobs or content approaching the message-size ceiling.
- **FR-027**: This feature MUST NOT implement any topic vocabulary, lifecycle, baseline/rollup, materialisation, mentions, discovery, client, or adapter behaviour; those are later features and MUST NOT be started here.

### Key Entities *(include if feature involves data)*

- **Realm**: the tenancy and trust boundary — one NATS account, containing exactly one operation-log stream and one object store. Identified by a realm name that later binds into every canonical record.
- **Stream (`SOULSTREAM`)**: the realm's single append-only operation log, capturing every subject under the protocol root, retained by limits with no age-based expiry and rollup permitted.
- **Object store (`soulstream-objects`)**: the realm's single store for content too large or too binary for messages. Created here; its usage vocabulary is a later feature.
- **Operation record**: the unit of everything — identity, version, author, parents, type, timestamp, optional signature, and a pure-data payload. Has a wire form (headers + payload) and a canonical form (deterministic bytes).
- **Persona name**: a lowercase, transport-token-safe slug identifying who an operation is attributed to; validated here, bound to credentials by the operator outside the library.
- **Provisioning report**: the structured result of a provisioning run — per artefact, its created / conformant / non-conformant state and any specific nonconformity.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Starting from an empty NATS server, an operator can bring a realm to a ready state with a single provisioning action, and independent inspection confirms every mandated stream and object-store setting.
- **SC-002**: Running provisioning a second time makes zero changes to the realm and reports every artefact as already conformant — verifiable by comparing server state before and after.
- **SC-003**: 100% of constructed operation records round-trip (construct → wire → parse) with full field equality across a representative matrix of cases (no parents, one parent, many parents; with and without a signature slot).
- **SC-004**: 100% of records that are equal in content produce byte-identical canonical serialisations regardless of source field ordering.
- **SC-005**: Every malformed input in a defined negative-test set (missing field, bad version, bad timestamp, invalid persona name, attribution mismatch) is rejected with an error that names the specific violation — no malformed input is silently accepted.
- **SC-006**: A retried publish reusing an operation identity results in exactly one operation on the stream, confirming duplicate suppression works over the configured window.
- **SC-007**: The whole record-format surface (construction, parsing, canonicalisation, validation) is testable with no running server, and the provisioning surface is testable against a locally-run server — the full feature verifies green with tests passing, formatting applied, and linting clean.

## Assumptions

- **Trust model**: the target is a high-trust realm where honest attribution is convention enforced at the library edges; hard-scoped credentials and edge auth-callout rejection of mismatched authorship are an operator concern and out of scope here.
- **Server capability**: the NATS server is a modern release (2.12+) with JetStream enabled, so limits retention without age expiry, subject rollup, duplicate windows, and disk storage are all available.
- **Single realm per account**: exactly one `SOULSTREAM` stream and one `soulstream-objects` store per account; multi-realm-per-account is not a supported shape.
- **Object store backend**: the JetStream object store is used, keeping the deployment single-dependency; swapping in S3-compatible storage behind the same name+digest convention is a later concern.
- **Signature deferral**: the optional signature slot exists in both wire and canonical forms and is carried through, but no signatures are produced or verified in this feature; the canonical bytes produced here are the contract any later signing must match.
- **Identity provisioning**: issuing NATS credentials and binding persona names to them is the operator's job (outside the library); the library validates name shape and checks attribution on read.
- **Reference design**: the normative source is `hq/02-DESIGN/core/01-protocol.md` and `02-identity.md`; the project constitution (`.specify/memory/constitution.md`) governs — NATS-native first, smallest viable implementation, documentation as a first-class citizen.

## Dependencies

- A reachable NATS server (2.12+) with JetStream enabled, and a named NATS context that addresses it, for the provisioning scenarios.
- No other Soulstream feature depends on this being complete before it can be specified, but every other feature depends on it before it can be implemented.
