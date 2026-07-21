# Feature Specification: Op Signing & Key Distribution

**Feature Branch**: `006-signing`
**Created**: 2026-07-21
**Status**: Draft
**Input**: User description: "Op signing and key distribution (roadmap Day-2 item 2, feature 006-signing). Personas can hold an Ed25519 signing keypair and sign every op they publish: Soulstream-Sig carries a signature over the canonical op record. Signing is optional per persona; unsigned ops remain valid forever. Key distribution is the minimal persona-registry slice: a soulstream-personas KV bucket with signing_key, TOFU pinning, rotation signed by the old key. Verification on read paths surfaces per-op status and never drops an op. CLI and MCP adapter gain key setup, automatic signing, key publication, and verification display."

## Why now

Signing is a one-way door ([ROADMAP.md](../../hq/03-IMPLEMENTATION/ROADMAP.md)): every op published
before signing lands is unsigned forever — testimony-grade, never exhibit-grade. It also gates
re-baselining (Day-2 item 1), which must not compact a realm whose history was never signed. Landing
signing now starts the clock as early as possible for the dogfood realm.

## Clarifications

### Session 2026-07-21

- Q: When a fresh client first encounters a persona whose profile already contains rotations, what
  does it pin — the current key or the chain? → A: The validated chain as first seen. TOFU trusts
  the chain root; every rotation inside the profile must carry a valid proof or the profile is
  treated as substitution. Later profile changes must extend the already-pinned chain.
- Q: How does an op signed under a superseded key verify after rotation? → A: A signature is
  verified if it verifies against any key in the author's validated chain. Era-matching by the
  `since` timestamp is never used for verification decisions; `since` stays informational.
- Q: Who creates the persona directory in a realm? → A: Realm provisioning creates it,
  create-or-report, exactly like the op stream and the objects bucket. All read paths tolerate a
  realm without the directory (signed ops degrade to unknown-key), so older realms keep working.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A persona signs what it publishes (Priority: P1)

A persona (human at the CLI, or an AI persona over the MCP adapter) sets up a signing identity once.
From then on, every operation that persona publishes — turns, comments, attachments, announcements,
lifecycle transitions — carries a signature over the full canonical record, so any kept copy of the
op (archived, quoted, exported) remains attributable to that persona without trusting whoever kept it.
The secret half of the identity never leaves the persona's machine.

**Why this priority**: This is the one-way door itself. Ops signed today are exhibit-grade forever;
everything else in this feature (distribution, pinning, display) can arrive later without loss, but
unsigned ops can never be retro-signed.

**Independent Test**: Configure a keypair for a persona, publish ops of every type, and confirm each
op on the stream carries a signature that verifies against the persona's public key held out-of-band
— no registry required.

**Acceptance Scenarios**:

1. **Given** a persona with a configured signing key, **When** it posts a turn, **Then** the published
   op carries a signature that verifies against the persona's public key over the op's canonical record.
2. **Given** a persona with a configured signing key, **When** it publishes any other op type
   (announce, comment, attach, close, mention notification), **Then** that op is signed the same way.
3. **Given** a persona with no signing key configured, **When** it posts, **Then** the op is published
   exactly as before — no signature, no error, no behaviour change.
4. **Given** a signed op whose content is altered in any field (author, topic, payload, timestamp,
   parents), **When** the signature is checked, **Then** verification fails.
5. **Given** a signed op copied into a different topic or realm, **When** the signature is checked
   there, **Then** verification fails (splicing is detectable).

---

### User Story 2 - Published keys with first-use pinning (Priority: P2)

A persona publishes its public signing key to the realm's persona directory so that every other
participant can discover it. Readers pin the first key they see for each persona and refuse silently
substituted keys from then on: if the directory later shows a different key without a proper rotation
announcement, verification for that persona hard-fails and the reader is warned loudly about a
possible substitution attack.

**Why this priority**: Signatures without distributed keys only help parties who exchanged keys
out-of-band. The directory makes verification work realm-wide; pinning makes a directory compromise
detectable by anyone who saw the original key.

**Independent Test**: Publish a persona's profile with its key, read it back from another client,
verify ops with it; then overwrite the key without announcement and confirm the second client
hard-fails and warns.

**Acceptance Scenarios**:

1. **Given** a persona with a signing key, **When** it publishes its profile, **Then** any client can
   read the persona's name, display metadata, and public key from the realm's persona directory.
2. **Given** a client that has verified a persona's ops before, **When** the directory suddenly holds
   a different key with no rotation proof, **Then** the client reports a possible substitution attack
   and treats that persona's signatures as failed rather than silently trusting the new key.
3. **Given** a fresh client that has never seen a persona, **When** it first reads that persona's
   profile, **Then** it pins that key and uses it for all subsequent verification.
4. **Given** profile metadata such as the persona kind (human / agent / service), **When** any client
   or library processes it, **Then** nothing but presentation may differ — no permission, capability,
   or protocol behaviour branches on profile fields.

---

### User Story 3 - Verification status wherever ops are read (Priority: P3)

Anyone reading a topic — materialising it, watching it live, or fetching their inbox — sees for each
op whether it is unsigned, verified, failed, or signed by an unknown key. A failed signature never
hides the op: the content stays visible, flagged, because losing testimony is worse than reading it
with a warning.

**Why this priority**: Verification is what makes the first two stories observable. It can ship after
signing and distribution because signed history verifies retroactively the moment display lands.

**Independent Test**: Materialise a topic containing an unsigned op, a validly signed op, a tampered
op, and an op from a persona with no published key; confirm the four statuses are reported and all
four ops remain visible.

**Acceptance Scenarios**:

1. **Given** a topic with signed and unsigned ops, **When** a participant reads it via CLI or MCP,
   **Then** each op's verification status is visible: unsigned, verified, failed, or unknown-key.
2. **Given** an op whose signature does not verify, **When** the topic is read, **Then** the op is
   still shown with its content, clearly flagged as failed.
3. **Given** a realm with no persona directory at all, **When** a topic with signed ops is read,
   **Then** reading succeeds and signed ops report unknown-key.
4. **Given** an op signed by a key that is not the author's pinned key, **When** the topic is read,
   **Then** that op reports failed (a signature is only good if it is the *author's*).

---

### User Story 4 - Key rotation without losing history (Priority: P4)

A persona replaces its signing key by announcing the new key signed with the old one. Verifiers
accept the rotation, update their pin, and keep verifying: ops signed with the earlier key still
verify as that persona's, and new ops verify against the new key.

**Why this priority**: Rotation matters only once keys have been in use long enough to need
replacing. It must exist before any key is compromised or retired, but it is not needed on day one.

**Independent Test**: Sign ops with key A, rotate to key B with a proof signed by A, sign more ops
with B; confirm a verifier accepts both eras and a rotation *without* proof is rejected as
substitution.

**Acceptance Scenarios**:

1. **Given** a persona rotating its key, **When** it publishes the new key with a proof signed by the
   old key, **Then** verifiers accept the new key and update their pin without warnings.
2. **Given** a completed rotation, **When** a topic containing ops from both key eras is read,
   **Then** ops from each era verify against the key that was current when they were signed.
3. **Given** a key change published without a valid proof, **When** a pinning client sees it,
   **Then** it is treated exactly like a substitution attack (US2, scenario 2).

---

### Edge Cases

- An op carries a signature but its author has no published key: status is unknown-key, never an
  error; if the key appears later, subsequent reads verify retroactively.
- The persona directory is unreachable or absent: reads never fail; all signed ops degrade to
  unknown-key, unsigned ops are unaffected.
- A second client sets up the *same* persona with a *different* key while the first key is published:
  the setup must refuse to silently replace a published key (that path is rotation or nothing).
- A signature header is present but malformed (not decodable): status is failed, op stays visible.
- A persona's profile exists but contains no signing key while its ops carry signatures: unknown-key.
- The `since` timestamp on a key is author-claimed and informational, like `Soulstream-Ts`; it never
  decides verification outcomes (only rotation-proof chains do).
- Pre-feature history (all existing dogfood ops): renders exactly as before, status unsigned.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A persona MUST be able to create a signing identity (one keypair per persona); the
  secret key MUST be stored only on the persona's side and MUST never be published, transmitted, or
  logged.
- **FR-002**: When a persona has a signing identity configured, every op it publishes — on topic,
  info, and notify subjects alike — MUST carry a signature over that op's complete canonical record,
  which binds realm and topic so signed ops cannot be spliced across topics or realms.
- **FR-003**: Signing MUST be optional per persona: personas without keys publish exactly as today,
  and unsigned ops remain valid and readable forever.
- **FR-004**: The realm MUST offer a persona directory in which each persona can publish a profile:
  name, display name, kind (human / agent / service), description, operator attribution, and public
  signing key with its start time. Realm provisioning creates the directory create-or-report, like
  the op stream and objects bucket. Publishing and updating one's own profile MUST be possible from
  both clients (CLI and MCP).
- **FR-005**: No permission, capability, or protocol behaviour may branch on profile fields; `kind`
  and all other profile data are presentation and audit metadata only.
- **FR-006**: Readers MUST pin, per persona, the validated key chain as first observed (chain root
  trusted on first use; internal rotation proofs must all verify, else the profile is treated as a
  substitution attack), and the pin MUST persist per client across sessions. Later profile changes
  MUST extend the pinned chain with valid proofs to be accepted.
- **FR-007**: A key change without a valid rotation proof MUST cause verification for that persona to
  hard-fail with a loud, explicit substitution-attack warning; the reader MUST NOT silently adopt the
  new key.
- **FR-008**: Key rotation MUST be announced by publishing the new key together with a proof signed
  by the previous key; verifiers MUST accept a valid proof, extend their pinned chain, and keep both
  eras verifiable: an op verifies if its signature matches any key in the author's validated chain,
  and the `since` timestamp never participates in the decision.
- **FR-009**: Every read path (materialise, live follow, inbox fetch) MUST report a per-op
  verification status of exactly one of: unsigned, verified, failed, unknown-key.
- **FR-010**: Verification MUST be non-destructive: a failed or unknown signature never drops,
  hides, or truncates an op — it flags it.
- **FR-011**: A signature MUST only count as verified if it verifies against the *claimed author's*
  pinned key; a valid signature from any other key is failed.
- **FR-012**: The CLI MUST let the human persona generate its keypair, publish/update its profile,
  rotate its key, and see per-op verification status when showing, watching, or reading inbox.
- **FR-013**: The MCP adapter MUST sign automatically when its session persona has a key configured,
  MUST expose profile publication for its persona, and MUST surface per-op verification status in
  every tool result that returns ops.
- **FR-014**: Signature verification of a kept op MUST be possible offline: given only the op's
  canonical record, the signature, and the public key, a third party can verify without access to
  the realm.

### Key Entities

- **Signing identity**: a persona's Ed25519 keypair; public half is published, secret half stays
  client-side. Exactly one active signing key per persona at any time.
- **Persona profile**: the directory entry for a persona — identity metadata plus the public signing
  key and its lineage (rotation proofs). Advisory except for key material.
- **Persona directory**: the realm-wide, watchable store of profiles keyed by persona name, with
  history.
- **Pin**: a client's durable record of the validated key chain as first seen for a persona,
  extended only by valid rotation proofs.
- **Verification status**: per-op outcome of checking the signature against the author's pinned key:
  unsigned / verified / failed / unknown-key.
- **Rotation proof**: a statement of the new key signed by the previous key, forming a verifiable
  chain from the pinned key to the current one.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A signed op exported from the realm verifies with only the op, its signature, and the
  persona's public key — no server, no directory, no trust in the keeper of the copy.
- **SC-002**: 100% of ops published by a key-configured persona are signed; a persona with no key
  publishes byte-identical to pre-feature behaviour.
- **SC-003**: Any single-field tampering of a signed op (author, topic, realm, payload, parents,
  timestamp, type, id) is detected as failed on the next read — 100% detection in tests.
- **SC-004**: An unannounced key substitution is surfaced as an explicit attack warning on the next
  read that touches the persona; zero silent pin updates in tests.
- **SC-005**: All pre-feature history remains readable with status unsigned; no existing test or
  behaviour regresses.
- **SC-006**: After a valid rotation, ops from both key eras verify; after an invalid rotation,
  verification hard-fails — both proven by tests.
- **SC-007**: A realm without a persona directory still reads every topic successfully (signed ops
  degrade to unknown-key), proving the directory is an extension, not a dependency.

## Assumptions

- One active signing key per persona realm-wide; multiple credentials (devices) for the same persona
  share the one key — key material distribution between a persona's own devices is the persona's
  concern, out of scope.
- The pin store lives client-side alongside the persona's own key material; "loud warning" means an
  unmissable, machine-distinguishable error surface in each client, not merely a log line.
- The persona directory is the minimal slice of the registry extension needed for key distribution
  (profiles + history + watchability); service announcements, sealing keys, avatars-as-objects, and
  registry tooling stay out of scope.
- Rotation proofs are carried in the profile itself (the directory's history preserves the chain);
  no separate announcement op type is introduced in this feature.
- Verification is offered on read paths of this codebase's library and clients; third-party offline
  verification is enabled by the canonical record format already in the wire spec.
- Existing dogfood realms adopt signing by personas publishing profiles at their own pace; no
  migration step, no flag day.
- Out of scope: sealed topics and sealing keys (X25519), re-baselining/rollup, scatter/gather
  discovery, auth callout, external PKI, out-of-band fingerprint verification UX, digests/presence.
