# Tasks: Op Signing & Key Distribution

**Input**: Design documents from `/specs/006-signing/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: included — the spec's success criteria (SC-002…SC-007) are explicitly
proven by tests, and the constitution's quality gate requires everything green.
Convention: pure logic tests are serverless; NATS-touching tests use
`internal/natstest.StartJetStream(t)`.

**Organization**: grouped by user story; stories land in priority order and each
checkpoint leaves `make check` green.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

*No setup tasks: existing Go module, zero new dependencies (`crypto/ed25519` is
stdlib), no scaffolding beyond files created by their own tasks.*

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: the NATS-free crypto primitives every story consumes.

- [X] T001 [P] Implement `identity.SigningKey` in identity/sign.go — `GenerateSigningKey`, `SigningKeyFromSeed` (32-byte check), `Seed`, `PublicKey` (std base64), `Sign` (base64 64-byte sig), package-level `VerifySignature` (malformed key/sig → false, never panic), `RotationProofBytes` (`"soulstream-key-rotation\n"+persona+"\n"+newPubB64`); tests in identity/sign_test.go (round-trip, wrong-key, malformed inputs, proof-bytes shape)
- [X] T002 [P] Implement `identity.Keyring` in identity/keyring.go — `Keys map[string][]string`, `Distrusted map[string]bool`, nil-keyring semantics documented on the type; test in identity/keyring_test.go

**Checkpoint**: `identity` still imports no NATS; `make check` green.

---

## Phase 3: User Story 1 — A persona signs what it publishes (Priority: P1) 🎯 MVP

**Goal**: every op published by a key-configured persona carries `Soulstream-Sig` over
the canonical record; personas without keys publish byte-identical to today. This is
the one-way door — landing it starts the signed-history clock.

**Independent Test**: configure a keypair, publish every op family, verify each sig
out-of-band against the public key (no registry anywhere).

### Implementation for User Story 1

- [X] T003 [US1] Add `Signer *identity.SigningKey` to `realm.Config` and `(*Client).Signer()` in realm/connect.go; extend a connect test in realm/connect_test.go
- [X] T004 [US1] Add the canonical-binding rule in topic/subjects.go — `canonicalBinding(subject)`: OPS/INFO → topic path, NOTIFY → persona (contracts/wire-and-kv.md table); table-test in topic/subjects_test.go
- [X] T005 [US1] Sign in the publish choke point topic/wire.go — when `c.Signer() != nil`: build record, `Canonical(c.Realm(), canonicalBinding(subject))` with empty Signature, set `rec.Signature`, rebuild headers; no signer ⇒ identical pre-feature path
- [X] T006 [US1] End-to-end signing test in topic/sign_test.go — embedded server; signed persona runs StartTopic (announce + baseline), PostTurn, AddComment, Attach, Close, and a mention (notify); read raw stream/notify messages and verify every sig with only `record.Parse` + `Canonical` + `identity.VerifySignature` (SC-001/SC-002, US1 scenarios 1–2); tamper matrix over author/topic/realm/payload/parents/ts/type/id fails verification (SC-003, scenarios 4–5); no-signer persona publishes with no `Soulstream-Sig` header (scenario 3)
- [X] T007 [P] [US1] Create internal/keystore/keystore.go — seed file save/load (base64 single line, 0600, refuse overwrite), default paths under `os.UserConfigDir()/soulstream/` (`keys/<realm>-<persona>.ed25519`, `pins/<realm>.json`), env/flag override resolution (`SOULSTREAM_KEY_FILE`, `SOULSTREAM_PINS_FILE`); tests in internal/keystore/keystore_test.go
- [X] T008 [US1] CLI key setup + signing in internal/cli — `key init` (refuses if exists, prints public key only), `key show`; `--key-file` global flag + env in connect.go Config resolution; connector passes Signer into `realm.Config`; tests in internal/cli/cli_test.go (init/show, signed post carries header, secret never printed)
- [X] T009 [US1] MCP adapter signs — key file env resolution in cmd/soulstream-mcp/main.go, signer wiring in internal/mcpserver/server.go; test in internal/mcpserver/server_test.go that a tool-published op carries a verifying sig
- [X] T010 [US1] Write docs/signing.md (ELI5 — wax-seal analogy: anyone can copy the letter, only you can press your seal; covers why-sign, testimony vs exhibit, signing is optional) and add it to the docs/README.md index

**Checkpoint**: signed dogfood publishing works with keys distributed by hand;
`make check` green.

---

## Phase 4: User Story 2 — Published keys with first-use pinning (Priority: P2)

**Goal**: the persona directory distributes public keys realm-wide; readers pin the
validated chain on first sight and hard-distrust unannounced changes.

**Independent Test**: publish a profile, read it from a second client and verify ops;
overwrite the key without proof and watch the second client distrust + warn.

### Implementation for User Story 2

- [X] T011 [P] [US2] Create registry/profile.go + registry/doc.go — `Profile`, `SigningKeyInfo`, `Rotation` structs with exact JSON tags/omitempty per contracts/wire-and-kv.md; `Validate` (name slug via identity.CheckName, kind enum, key/rotation base64 shapes); tests in registry/profile_test.go
- [X] T012 [P] [US2] Create registry/chain.go — `Chain(p)` (root/contiguity/proof/current-key rules per data-model.md) and `BuildKeyring(profiles, pinned)` (pin-prefix rule: no pin → TOFU pin; prefix → accept + extend; else distrust, keep pin); pure tests in registry/chain_test.go (valid single-key, valid rotated, diverged, shortened, invalid proof, profile without key, persona-bound proof rejected for other persona)
- [X] T013 [US2] Provision the third artefact — `soulstream-personas` KV (history 10) constants in realm/spec.go, create-or-report in realm/provision.go + conformance/report plumbing; tests in realm/provision_test.go (created when missing, reported when present, never modified)
- [X] T014 [US2] Create registry/kv.go — `Publish` (create-or-metadata-update per contracts/library.md: Create when absent; `Update(rev)` preserving stored key material when present; different incoming key ⇒ `ErrKeyConflict`), `Lookup` (absent bucket/key ⇒ `(Profile{}, false, nil)`), `All` (absent bucket ⇒ empty); embedded-server tests in registry/kv_test.go incl. metadata update preserving the key and the two-clients-different-key race (second `Publish` gets `ErrKeyConflict`, never overwrites — spec edge case, FR-004)
- [X] T015 [US2] Add pins to internal/keystore — pins JSON load/save with atomic rewrite (realm match check); tests
- [X] T016 [US2] CLI directory surface in internal/cli — `profile publish` (create-or-metadata-update; flags --display-name/--kind/--description/--operated-by; includes key when present; key conflict message points at `key rotate`), `profile show <persona>` (profile, chain, pin state), provision output gains the personas artefact line; keyring lifecycle helper in connect.go (load pins → registry.All → BuildKeyring → persist extended pins → nil keyring when bucket absent); tests
- [X] T017 [US2] MCP `publish_profile` tool (9th, create-or-metadata-update) in internal/mcpserver/tools.go per contracts/clients.md; test in internal/mcpserver/server_test.go
- [X] T018 [US2] Write docs/persona-directory.md (ELI5 — phone book you check once and remember: pinning = remembering a friend's handwriting; substitution = a stranger's handwriting under a friend's name) and update docs/provisioning.md (third artefact) + docs/README.md index

**Checkpoint**: keys flow through the realm; pins persist; substitution is detected
at keyring-build time; `make check` green.

---

## Phase 5: User Story 3 — Verification status wherever ops are read (Priority: P3)

**Goal**: materialise/follow/inbox annotate every op with
unsigned/verified/failed/unknown-key; nothing is ever hidden.

**Independent Test**: one topic containing all four statuses reads back with all four
ops visible and correctly labelled, in library, CLI, and MCP.

### Implementation for User Story 3

- [ ] T019 [US3] Create topic/verify.go — `SigStatus` constants, `VerifyRecord(rec, realmName, binding, kr)` (clear Signature → Canonical → verify against any chain key; author's chain only per FR-011; distrusted ⇒ failed), `annotate(recs, …) map[opID]SigStatus`; serverless tests in topic/verify_test.go (four statuses, wrong-author's-key ⇒ failed, malformed sig ⇒ failed, nil keyring ⇒ unknown-key, distrusted ⇒ failed)
- [ ] T020 [US3] Thread statuses through views — `Sig SigStatus` on `Contribution`/`Attachment`/`Announcement`/`Notification` in topic/view.go; `Handle.UseKeyring(kr)` in topic/handle.go; annotate in materialise (topic/materialise.go), live follow (topic/follow.go), and inbox (topic/notify.go — `FetchInbox`/`FollowInbox` gain a keyring parameter, update all callers; note: replace the `Notification(np)` struct conversion with explicit field assignment once `Sig` is added); tests: mixed four-status materialise (US3 independent test), follow surfaces status live, inbox statuses, absent-directory degradation to unknown-key (SC-007), statuses never affect inclusion/ordering/lifecycle (FR-010)
- [ ] T021 [US3] CLI status rendering in internal/cli/render.go + commands — glyphs `✓`/`✗`/`?`/none on show/watch/inbox lines; distrust banner `!! possible key substitution for <persona> …` as first line and mirrored to stderr (FR-007); tests asserting glyphs and banner
- [ ] T022 [US3] MCP status surfacing — `sig` field on every op-returning tool result + top-level `distrusted_personas`; per-session keyring lifecycle (pins load/reconcile/persist) in internal/mcpserver/server.go; tests
- [ ] T023 [US3] Update docs — docs/canonical-record.md (the `sig` key is now produced and checked; wax-seal cross-link), docs/cli.md (key/profile commands, glyphs, banner), docs/mcp.md (publish_profile, sig fields)

**Checkpoint**: the quickstart's show output (steps 4–5, 7) is real; `make check` green.

---

## Phase 6: User Story 4 — Key rotation without losing history (Priority: P4)

**Goal**: old-key-signs-new-key rotation; both eras verify; proof-less changes are
substitution.

**Independent Test**: sign under key A, rotate to B with proof, sign under B — both
eras `verified`; a proof-less key change distrusts the persona.

### Implementation for User Story 4

- [ ] T024 [US4] Implement `registry.Rotate` in registry/kv.go — read own profile, build rotation entry (proof = old key over `RotationProofBytes`), set new `signing_key`, KV `Update(rev)`; tests in registry/kv_test.go + registry/chain_test.go: post-rotation chain validates, era-A and era-B ops both `verified` end-to-end (SC-006, US4 scenarios 1–2), proof-less key change ⇒ distrust (scenario 3), lost `Update` race ⇒ error not blind write
- [ ] T025 [US4] CLI `key rotate` in internal/cli — rotate in the directory first, then swap the local seed file keeping `<file>.prev`; refuses without an existing key + published profile; tests (rotation flow, refusal paths, `.prev` retained)
- [ ] T026 [US4] Extend docs/signing.md and docs/persona-directory.md with rotation (new seal endorsed by pressing the old seal on it) and the substitution warning story

**Checkpoint**: all four stories independently demonstrable; `make check` green.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T027 Validate specs/006-signing/quickstart.md end-to-end against the real CLI/MCP behaviour and fix any drift (command names, output shapes)
- [ ] T028 FR-005 guard sweep — grep clients/library for `kind ==` and profile-field branching; assert every hit is presentation-only; root README.md feature list gains signing; final `make fmt && make test && make lint` all green, none skipped

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 2)**: blocks everything.
- **US1 (Phase 3)**: needs T001–T002 only.
- **US2 (Phase 4)**: needs T001–T002; independent of US1 (verifies with out-of-band
  keys in its own tests until US3 wires statuses).
- **US3 (Phase 5)**: needs US1 (signed ops to verify) and US2 (keyring source);
  degrades gracefully without a directory by design.
- **US4 (Phase 6)**: needs US2 (profiles) + US3 (status assertions in its tests).
- **Polish (Phase 7)**: last.

### Within-story ordering

- T003/T004 → T005 → T006; T007 → T008/T009.
- T011/T012 [P] → T013 → T014 → T015 → T016/T017.
- T019 → T020 → T021/T022.
- T024 → T025.
- Docs tasks (T010, T018, T023, T026) close their story — a story is not done
  without them (Constitution III).

### Parallel Opportunities

- T001 ∥ T002; T007 ∥ T003–T006; T011 ∥ T012; T008 ∥ T009 after T007; T016 ∥ T017
  after T015; T021 ∥ T022 after T020.

---

## Implementation Strategy

US1 first and alone is a meaningful stop point (the one-way door closes: dogfood ops
become exhibit-grade even before the directory exists — keys verified out-of-band).
Then US2 → US3 → US4 in order, each ending at a green checkpoint, committing per task
group; merge `006-signing` to main with `--no-ff` after Phase 7.
