# Feature Specification: Persona Accountability & Stream Hygiene

**Feature Branch**: `014-persona-accountability`
**Created**: 2026-07-23
**Status**: Draft
**Input**: User description: "Persona accountability and stream hygiene. Backwards compatibility is explicitly NOT required — we are the only users; remove things outright rather than deprecating. Part 1 — remove `kind` from persona profiles entirely. Part 2 — reframe identity documentation around principal chains and upgrade `operated_by` from an unverified string claim to a countersigned, verifiable attestation. Part 3 — two stream-hygiene defects: discovery request traffic is accidentally retained forever in the realm's permanent store, and persona mention inboxes grow without bound while inbox reads replay full history."

## Why now

The persona `kind` field (human / agent / service) encodes a distinction the protocol cannot
verify and — by its own rule (006-signing FR-005) — may never act on. It invites exactly the
ambiguity it pretends to resolve: is the persona the human at the keyboard, the tool session,
or an autonomous process? Meanwhile the one distinction that *is* meaningful — who answers for
a persona — exists only as `operated_by`, an unverified string anyone can write into their own
profile. Because we are the sole users today, removing the misleading field and hardening the
honest one costs nothing; every day of delay adds profiles and habits shaped around the wrong
model. The two hygiene defects compound silently with use (every discovery request and every
mention is retained forever in a store that never ages out), so they are cheapest to fix now.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A persona is a voice, not a classification (Priority: P1)

A persona publishes its profile without ever declaring itself "human", "agent", or "service".
The profile carries a name, optional display name, optional free-form description, the signing
key material, and (optionally) who operates it. Nothing in the product — profile publishing,
profile display, documentation — asks for or shows a persona classification. Whoever controls
the persona's signing key *is* the persona; how it is driven (a person typing, a tool session,
an autonomous process) is invisible to the protocol because the protocol could never verify it
anyway.

**Why this priority**: The classification is actively misleading today: two entry points
default the same concept differently (one assumes "human", the other assumes "agent"), and the
enum forces every integrator to answer an unanswerable taxonomy question. Removing it is the
smallest change and unblocks the documentation reframe.

**Independent Test**: Publish a profile through both the command-line and the adapter without
any classification input; read the profile back from another client and confirm no
classification appears anywhere — not in the stored profile, not in any display, not as a
silent default.

**Acceptance Scenarios**:

1. **Given** a persona publishing its profile via any entry point, **When** the profile is
   written to the realm's persona directory, **Then** the stored profile contains no kind /
   classification field and no input for one exists.
2. **Given** a published profile, **When** any client displays it, **Then** name, display
   name, description, operator, and key information are shown and no classification is shown
   or inferred.
3. **Given** a profile document in the persona directory that still carries a classification
   field (published by a pre-change client), **When** a client reads it, **Then** the profile
   is rejected loudly as invalid, naming the offending field — never silently accepted or
   silently stripped.
4. **Given** the project's documentation, **When** a reader looks up what a persona is,
   **Then** they find the voice-with-a-key definition and no human/agent/service taxonomy.

---

### User Story 2 - Verifiable chains of accountability (Priority: P2)

A persona that is operated by another persona — a coding assistant run by its human, a
scheduled process run by a team persona — names its operator in its profile, and the operator
*countersigns* that claim with their own signing key. Anyone reading the operated persona's
profile sees not just "X claims to be operated by Y" but whether Y has actually vouched for
that claim. Following `operated_by` links from any persona leads, in finitely many steps, to a
persona that answers for itself: the principal. The documentation explains this model: every
persona either answers for itself or names an operator; whether a tool session speaks as its
human (the tool is a pen) or as its own operated persona (its own voice and reputation) is the
operator's choice, and the protocol never tries to detect the difference.

**Why this priority**: Accountability is the honest replacement for the removed taxonomy. It
depends on Story 1 conceptually (docs describe one coherent model) but is independently
shippable and testable.

**Independent Test**: Publish persona B with `operated_by: A` and a countersignature produced
by A's key; read B's profile from a fresh client and confirm the claim reports as attested.
Publish persona C with `operated_by: A` and no countersignature; confirm the claim reports as
unverified. Corrupt B's countersignature; confirm the claim reports as failed while the
profile remains readable.

**Acceptance Scenarios**:

1. **Given** operator A with a published signing key, **When** persona B publishes a profile
   naming A as operator together with A's countersignature over the claim, **Then** any client
   reading B's profile reports the operator claim as **attested**.
2. **Given** persona C naming operator A without a countersignature, **When** any client reads
   C's profile, **Then** the claim is shown as **unverified** — visible, never hidden, never
   presented as vouched-for.
3. **Given** an operator countersignature that does not verify against any key in the
   operator's validated key chain, **When** the profile is read, **Then** the claim reports as
   **failed** and the reader is warned loudly, while the rest of the profile stays visible.
4. **Given** an operator who has rotated keys since countersigning, **When** the attestation
   is verified, **Then** it verifies against any key in the operator's validated chain (same
   rule as op signatures).
5. **Given** a profile whose `operated_by` names the persona itself, or a set of profiles
   whose operator links form a cycle, **When** a client resolves the chain, **Then** the chain
   is reported as invalid rather than looping, and each individual profile stays readable.
6. **Given** the documentation, **When** a reader asks "who is the persona when a human uses a
   tool session?", **Then** the docs answer with the principal-chain model: it is the
   operator's choice which voice speaks, and both configurations are legitimate.

---

### User Story 3 - Discovery traffic leaves no permanent residue (Priority: P3)

Participants use realm-wide topic discovery as often as they like. Discovery requests are
transient ask-and-answer traffic: once answered, they are gone. The realm's permanent store —
which never ages anything out, by design — retains topic history, announcements, and mention
notifications, and nothing else. Re-provisioning an existing realm brings it to this shape.

**Why this priority**: A silent defect, not a behaviour change: request traffic was only ever
*accidentally* retained. Left alone it grows the permanent store forever with data nobody can
read back through any product surface.

**Independent Test**: On a freshly provisioned realm, run a number of discovery round-trips
and confirm the realm's permanent store gained no messages from them; on a realm provisioned
before this change, re-provision and confirm the same holds for subsequent discovery traffic.

**Acceptance Scenarios**:

1. **Given** a provisioned realm, **When** any number of discovery requests and replies are
   exchanged, **Then** the realm's permanent store retains none of that traffic.
2. **Given** a realm provisioned before this change, **When** provisioning is run again,
   **Then** it converges the realm to the new retention shape (create-or-update, no data loss
   for topic history, announcements, or inboxes).
3. **Given** the new retention shape, **When** topics, announcements, and mention
   notifications are published, **Then** they are all still retained exactly as before.

---

### User Story 4 - Inboxes stay fast forever (Priority: P4)

A persona that has been mentioned thousands of times over months of participation checks its
inbox and gets an answer just as fast as a persona mentioned twice. The inbox retains a
bounded window of the most recent mention notifications per persona; checking it reads an
amount of data proportional to that bound, never to the persona's lifetime mention count.
Older notifications fall away — which loses nothing durable, because a notification is only a
pointer: the mentioning turn itself remains in its topic's permanent history.

**Why this priority**: A scaling defect that only hurts once usage accumulates. Fixing it now
is cheap; fixing it after inboxes are large means the fix has to migrate data.

**Independent Test**: Generate mentions for one persona well past the retention bound; confirm
the inbox check returns the most recent notifications, that its cost does not grow with total
lifetime mentions, and that the realm's stored notification count for that persona respects
the bound.

**Acceptance Scenarios**:

1. **Given** a persona with more lifetime mentions than the retention bound, **When** it
   checks its inbox, **Then** it receives the most recent notifications (newest first, up to
   the existing per-check cap) and the check reads no more than the bounded window.
2. **Given** the retention bound is reached for a persona, **When** a new mention notification
   arrives, **Then** it is retained and the oldest notification for that persona falls away.
3. **Given** a notification that has fallen out of the inbox window, **When** the mentioned
   turn is read via its topic, **Then** the turn and its mention are fully intact in topic
   history — only the pointer expired.
4. **Given** any other retained category (topic history, announcements), **When** the inbox
   bound is applied, **Then** their retention is unaffected — the bound applies to mention
   notifications only.

---

### Edge Cases

- A pre-change profile containing the removed classification field is read → rejected loudly
  as invalid, naming the field; the operator republishes the profile (we are the only users;
  no migration shim).
- `operated_by` names a persona with no published signing key → the claim can only ever be
  **unverified**; it is shown as such, never as attested.
- The *operated* persona has no signing key but its operator does → the operator can still
  attest; the attestation binds to the operated persona's name (unique in the directory)
  rather than a key, and later addition of a key invalidates nothing but MAY warrant
  re-attestation.
- `operated_by` names a persona that does not exist in the directory → claim shown as
  unverified; chain resolution reports the link as dangling.
- Operator link cycles or self-reference → chain reported invalid; no client may loop or
  crash; individual profiles remain readable.
- An operator wishes to disavow a persona it once attested → out of scope for this feature
  (see Assumptions); the attestation is a timestamped historical fact.
- Discovery is used on a realm re-provisioned mid-flight → transient traffic exchanged during
  re-provisioning may or may not be retained for that instant; steady state after provisioning
  must retain none.
- A persona is mentioned in a burst that overshoots the inbox bound between two inbox checks →
  the persona sees the most recent bounded window; earlier burst mentions are discoverable
  through the topics themselves.

## Requirements *(mandatory)*

### Functional Requirements

**Persona classification removal**

- **FR-001**: Persona profiles MUST NOT contain a kind / classification field; the profile
  schema consists of name, optional display name, optional description, optional operator
  claim (with optional attestation), creation time, and signing-key material (current key and
  rotation chain).
- **FR-002**: No entry point (command-line, adapter/tool surface) may accept, default, or
  display a persona classification; all inputs, flags, tool parameters, and outputs referring
  to it MUST be removed rather than deprecated.
- **FR-003**: A profile document containing any unknown field — including the removed
  classification field — MUST be rejected loudly as invalid, naming the field and the persona,
  and MUST NOT be silently accepted, stripped, or repaired.
- **FR-004**: All project documentation and specifications that present the human / agent /
  service taxonomy as current behaviour MUST be updated to the voice-with-a-key model;
  historical feature specs remain as archived records of their time.

**Principal chains & operator attestation**

- **FR-005**: A persona's profile MAY name exactly one operator persona (`operated_by`); the
  named operator MUST be a valid persona name and MUST NOT be the persona itself.
- **FR-006**: An operator claim MAY carry an attestation: a countersignature by the operator
  over a canonical representation of the claim that binds, at minimum, the operator's name,
  the operated persona's name, and the operated persona's current signing key when one exists
  (so an attestation cannot be replayed onto a different persona or key; for an unkeyed
  operated persona the binding is to the name alone, which the directory keeps unique).
- **FR-007**: Verification of an attestation MUST succeed if and only if the countersignature
  verifies against any key in the operator's validated key chain (the same chain rule as op
  signature verification in 006-signing).
- **FR-008**: Every surface that displays a profile MUST report the operator claim's status as
  exactly one of: **attested** (valid countersignature), **unverified** (no countersignature,
  or operator has no published key), or **failed** (countersignature present but invalid).
  A failed or unverified claim MUST never hide the profile or the claim itself.
- **FR-009**: Clients resolving a chain of operator links MUST terminate: a persona with no
  operator claim is a principal; dangling links, self-reference, and cycles MUST be reported
  as such without looping or failing the read.
- **FR-010**: Creating an attestation MUST be possible with the same key material the operator
  already uses for op signing — no new key types or side channels — and the operator's secret
  key MUST never leave the operator's side in the process.
- **FR-011**: Documentation MUST describe the accountability model: every persona either
  answers for itself (a principal) or names an operator; chains terminate at a principal; the
  protocol never defines or detects "human" versus "agent"; whether a tool session speaks as
  its human or as its own operated persona is the operator's choice.

**Stream hygiene**

- **FR-012**: The realm's permanent store MUST NOT retain transient service (request/reply)
  traffic, including discovery requests; retained categories are exactly: topic operation
  history, topic announcements, and mention notifications (the latter bounded per FR-014).
- **FR-013**: Realm provisioning MUST converge both fresh and previously provisioned realms to
  the retention shape of FR-012/FR-014 (create-or-update), without loss of topic history,
  announcements, or in-window notifications.
- **FR-014**: Mention notifications MUST be retained per persona as a bounded window of the
  most recent notifications; the bound MUST be at least the per-check return cap (currently
  50) and MUST apply only to mention notifications, leaving all other retained categories
  unbounded as before.
- **FR-015**: Checking an inbox MUST read an amount of data bounded by the retention window,
  independent of the persona's lifetime mention count.
- **FR-016**: All existing guarantees over retained records MUST survive the retention
  changes: signature verification of retained ops, mention notifications remaining pointers
  (topic, op, author) whose referenced turns stay permanently in topic history, and empty
  inboxes answering immediately.

### Key Entities

- **Persona profile**: A persona's directory entry — name, optional display name, optional
  free-form description, optional operator claim with optional attestation, creation time,
  current signing key, and rotation chain. No classification field exists.
- **Operator claim**: The statement, inside a persona's profile, that another named persona
  operates this one. Carries at most one attestation. Purely an accountability fact — grants
  and implies no permissions (unchanged from 006-signing).
- **Operator attestation**: The operator's countersignature over the canonical operator claim,
  binding operator name, operated persona name, and the operated persona's current key.
  Verified against the operator's validated key chain; reported as attested / unverified /
  failed wherever profiles are shown.
- **Principal**: A persona whose profile names no operator — the terminus of every valid
  chain of operator links.
- **Realm retention shape**: The definition of what a realm's permanent store keeps: topic
  operation history and announcements (forever), mention notifications (bounded most-recent
  window per persona), and no transient service traffic.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: No user-visible surface — profile publishing, profile display, stored profiles,
  documentation — accepts or shows a persona classification; a full-text search of the product
  and its docs for the retired taxonomy as current behaviour returns nothing.
- **SC-002**: For 100% of profiles that name an operator, every profile-displaying surface
  shows one of exactly three claim statuses (attested / unverified / failed), and a corrupted
  attestation is always reported as failed, never as attested or hidden.
- **SC-003**: Any reader can determine the principal behind any persona (or learn that the
  chain is dangling or invalid) by following operator links in the profile directory alone,
  in a number of steps equal to the chain length.
- **SC-004**: After provisioning, a realm's permanent store grows by zero retained messages
  per discovery round-trip, verified by comparing store contents before and after a batch of
  discovery requests.
- **SC-005**: Inbox check cost is flat with respect to history: a persona with 100× more
  lifetime mentions than the retention bound gets inbox results in the same order of time and
  reads no more stored data than a persona with a near-empty inbox.
- **SC-006**: A realm provisioned before this change, once re-provisioned, satisfies SC-004
  and SC-005 with all pre-existing topic history and announcements intact.

## Assumptions

- **No backwards compatibility**: stated by the user. Pre-change profiles are invalid until
  republished (FR-003); pre-change realms are converged by re-provisioning (FR-013). No
  deprecation period, no dual-reading, no migration tooling.
- **Uncountersigned operator claims are allowed but reported unverified** (rather than
  disallowed). Rationale: signing itself is optional per persona (006-signing FR-003) — an
  operator without a key could never countersign, and refusing the claim would punish honesty.
  This mirrors the existing unsigned / verified / failed philosophy: surface status, never
  drop testimony.
- **Attestation revocation is out of scope.** An attestation is a timestamped historical fact,
  like a signature. The operated persona controls its own profile and can drop the claim; an
  operator who wants to disavow can say so on the record in a topic. A dedicated revocation
  mechanism can be layered on later without changing this feature's shape.
- **The inbox retention bound is a realm-level constant, not per-persona configuration.** A
  single sensible bound (at least the per-check cap of 50) keeps provisioning simple; nothing
  in this feature prevents making it tunable later.
- **Previously retained discovery-request residue may simply be left in place or purged at
  re-provisioning time** — nothing reads it through any product surface; the requirement is
  that no *new* residue accumulates (FR-012).
- **`operated_by` retains its 006-signing semantics** as an audit fact, not a permission link;
  attestation strengthens its verifiability, not its authority.
