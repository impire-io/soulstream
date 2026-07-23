# Tasks: Persona Accountability & Stream Hygiene

**Input**: Design documents from `/specs/014-persona-accountability/`
**Prerequisites**: plan.md, spec.md, research.md (D1–D6), data-model.md, contracts/library.md

**Tests**: included — the project's quality gate mandates them (all green, none skipped).

**Organization**: grouped by user story. US3 and US4 share the realm reshape: the
notify stream MUST exist in the same provision pass that narrows the main stream
(otherwise new notifications land in no stream), so US3 carries the shared realm
tasks and US4 builds on them.

## Phase 1: Setup

*(none — existing codebase, no new dependencies, gate already green)*

## Phase 2: Foundational

*(none — no prerequisite blocks any story; US1 is the natural first increment)*

---

## Phase 3: User Story 1 — A persona is a voice, not a classification (P1)

**Goal**: no `kind` anywhere; profiles strict-decoded, failures loud and scoped.
**Independent test**: publish via CLI + MCP with no classification input; read back —
no classification stored/shown; a stored profile carrying `kind` rejects loudly on
direct read and is skipped-with-warning on bulk reads.

- [x] T001 [US1] Remove `Kind` field, `KindHuman/KindAgent/KindService` constants, and the kind validation branch from `registry/profile.go`; add strict `decodeProfile([]byte) (Profile, error)` (json.Decoder + DisallowUnknownFields, error names persona + field) and use it in `Publish`/`Rotate`/`Lookup` in `registry/kv.go`; change `All` to return `([]Profile, []ProfileWarning, error)` with `ProfileWarning{Persona, Err}` for skipped entries; update `registry/profile_test.go` + `registry/kv_test.go` (drop kind cases; add: unknown-field rejection naming field, `kind`-carrying legacy document rejected on Lookup, skipped-but-warned on All)
- [x] T002 [US1] Update every `registry.All` caller to the new signature and surface warnings loudly: `internal/cli` (stderr warning per skipped persona wherever keyrings are built or profiles listed) and any other caller found via grep; adjust affected tests
- [x] T003 [P] [US1] Remove `--kind` flag and `kind:` display line from `internal/cli/profile.go`; update `internal/cli/profile_test.go`
- [x] T004 [P] [US1] Remove `Kind` from `publishProfileInput` (and its agent-default) in `internal/mcpserver/tools.go`; update `internal/mcpserver/profile_test.go`
- [x] T005 [P] [US1] Docs (ELI5): rewrite `docs/persona-and-attribution.md` and `docs/persona-directory.md` around voice-with-a-key (no human/agent/service taxonomy; description is free-form self-presentation; strict decode = "the librarian refuses a card with scribbles in the margin"); update publish sections of `docs/cli.md` + `docs/mcp.md`

**Checkpoint**: `make check` green; grep for `Kind` in non-historical code returns nothing.

---

## Phase 4: User Story 2 — Verifiable chains of accountability (P2)

**Goal**: countersigned `operated_by`, three claim statuses, principal-chain walk.
**Independent test**: quickstart.md flow — attest → publish → show reports
`attested`; no token → `unverified`; corrupted sig → `FAILED` with profile readable.

- [x] T006 [P] [US2] Add `AttestationBytes(operator, operated, operatedKeyB64) []byte` (domain-separated, mirrors `RotationProofBytes`) to `identity/sign.go`; tests in `identity/sign_test.go` (distinct statements for distinct inputs, empty-key form)
- [x] T007 [US2] Add `OperatorAttestation{OperatedKey, Sig}` + `Profile.OperatorAttestation` to `registry/profile.go`; extend `Validate` (operated_by ≠ name; attestation requires operated_by; b64 shape checks); add `NewAttestationToken`/`ParseAttestationToken` (base64 JSON `{operator, operated, operated_key, sig}`) and pure `AttestationStatus(p, operatorChain, operatorDistrusted, operatedChain) string` ("attested"/"unverified"/"failed"/"" per data-model.md) and `OperatorChain(profiles map[string]Profile, start) ([]string, ChainTerminal)` (visited-set walk; terminals principal/dangling/cycle) in `registry/attest.go`; tests in `registry/attest_test.go` (attested happy path; operator rotation still attested; sig-vs-wrong-chain failed; bound key ∉ operated chain failed; unkeyed operator unverified; unkeyed operated "" binding attested; self-reference/cycle/dangling terminals; distrusted operator failed)
- [x] T008 [US2] CLI in `internal/cli/profile.go`: new `profile attest <operated>` subcommand (requires signer; Lookup operated persona for current key, absent → ""; print token + hint); `profile publish --attestation <token>` (parse, match operator==--operated-by and operated==persona, store); `profile show` prints `operated by: X [status]` + `principal:` line from the chain walk (or dangling/cycle report); register in `internal/cli/commands.go` if needed; tests in `internal/cli/profile_test.go` covering the three statuses end-to-end over the embedded server
- [x] T009 [P] [US2] MCP: add `Attestation string` (token) to `publishProfileInput` in `internal/mcpserver/tools.go`, parse/validate/store like the CLI path; test in `internal/mcpserver/profile_test.go`
- [x] T010 [P] [US2] Docs (ELI5): NEW `docs/operators.md` — principal chains + attestation ("a parent co-signs the permission slip; anyone can check the signature"), the three statuses, the token hand-off, revocation-is-out-of-scope honesty; link from `docs/persona-directory.md` + `docs/signing.md`; update `docs/cli.md` (attest, --attestation, show output) + `docs/mcp.md` (publish_profile schema)

**Checkpoint**: `make check` green; quickstart attest flow works against an embedded server test.

---

## Phase 5: User Story 3 — Discovery traffic leaves no permanent residue (P3)

**Goal**: SVC.> captured by nothing; realm = op-log stream + notify stream; legacy
realms converge. (Carries the realm reshape shared with US4.)
**Independent test**: fresh realm → discovery round-trips add zero stored messages;
legacy-shaped realm → provision reports `updated`, converges, history intact.

- [x] T011 [US3] Reshape `realm/spec.go`: `StreamSubject` → `"SOULSTREAM.TOPICS.>"`; add `NotifyStreamName "SOULSTREAM_NOTIFY"`, `NotifyStreamSubject "SOULSTREAM.PERSONA.NOTIFY.>"`, `InboxWindow 100`, `NotifyMaxBytes 64<<20`, `notifyStreamConfig()` (limits, MaxAge 0, MaxMsgsPerSubject InboxWindow, DiscardOld, MaxBytes, file, 2m dupes); split `realm/conformance.go` into per-stream checks (`streamNonconformities` for the op-log, `notifyNonconformities` for the notify stream)
- [x] T012 [US3] Converge in `realm/provision.go` + `realm/report.go`: provision creates BOTH streams when missing; when the existing op-log stream's subjects are exactly the legacy `["SOULSTREAM.>"]` → update ONLY Subjects (preserve MaxBytes etc.), create notify stream, migrate newest ≤`InboxWindow` notify messages per persona verbatim (headers+data, same subjects) from the op-log stream, purge `SOULSTREAM.PERSONA.>` + `SOULSTREAM.SVC.>` from it, report new `OutcomeUpdated`; any other drift stays report-only; tests in `realm/provision_test.go` (fresh = two streams created + SVC not captured by either; legacy = converge + migrate ≤100 + purge + topic history untouched; alien shape = report-only)
- [x] T013 [US3] Integration test: discovery round-trip stores nothing — count both streams' messages before/after `topic.Discover`+`RespondDiscovery` on a provisioned realm (extend `topic/discover_test.go` or `realm/provision_test.go`); assert signed notify migrated in the legacy path still verifies (`SigStatus` verified after convergence)
- [x] T014 [P] [US3] Docs (ELI5): update `docs/realm.md` + `docs/provisioning.md` (two shelves: the eternal logbook and the small message tray; provision converges old realms), `docs/discovery.md` (asking around leaves no paper trail; pub-ack oddity gone — keep the malformed-reply rule as belt-and-braces)

**Checkpoint**: `make check` green; US3 acceptance scenarios pass.

---

## Phase 6: User Story 4 — Inboxes stay fast forever (P4)

**Goal**: inbox reads hit the bounded notify stream only.
**Independent test**: >100 mentions for one persona → stored ≤100, fetch returns 50
newest fast, older mentions still intact inside topic history.

- [x] T015 [US4] Point `FetchInbox` + `FollowInbox` at `realm.NotifyStreamName` in `topic/notify.go` (empty-guard `GetLastMsgForSubject` moves with it); a missing notify stream (realm not re-provisioned) must fail with a clear "re-provision this realm" error, not a bare stream-not-found
- [x] T016 [US4] Tests in `topic/notify_test.go` (or the existing inbox test home): publish 120 mentions → notify stream holds exactly 100 for that persona, `FetchInbox` returns 50 newest-first, empty inbox still returns immediately, `FollowInbox` receives live notifies, the mentioning turns remain readable in the topic, and other subjects/topic history are unaffected by the bound
- [x] T017 [P] [US4] Docs (ELI5): update `docs/mentions.md` — the fridge door holds the newest 100 notes; the letters themselves stay in the mailbox (topic) forever; checking the fridge is instant no matter how popular you are

**Checkpoint**: `make check` green; SC-005 flatness argument demonstrated by the bound test.

---

## Phase 7: Polish & Delivery

- [x] T018 [P] Sweep every remaining current-behaviour `kind` reference: `plugins/soulstream/README.md`, `plugins/soulstream/skills/setup/SKILL.md` (if present), `docs/README.md` index, root `README.md`; historical `specs/00x` stay untouched (archived records)
- [x] T019 [P] Bump `plugins/soulstream/.claude-plugin/plugin.json` + `.claude-plugin/marketplace.json` to 0.3.0
- [x] T020 Full gate `make check` (fmt+tidy+build+test+lint — all green, none skipped); fix everything it raises
- [x] T021 Merge `014-persona-accountability` → `main` with `--no-ff`, push, tag `v0.3.0` (signed) and push the tag — the release workflow publishes the binaries the 0.3.0 plugin wrapper will download

## Dependencies & Execution Order

- US1 (Phase 3) first — smallest, unblocks the coherent-model docs.
- US2 (Phase 4) depends on US1's Profile shape (kind gone, strict decode in place).
- US3 (Phase 5) is independent of US1/US2; carries the realm reshape.
- US4 (Phase 6) depends on US3 (notify stream must exist).
- Phase 7 last; T021 strictly after T020 is green.

Parallel opportunities: within US1 — T003, T004, T005 after T001/T002; within US2 —
T006 with T007's test scaffolding, T009/T010 after T007/T008; T014/T017/T018/T019
are docs/metadata and parallel to anything in their phase.

## Implementation Strategy

Deliver in story order; run `make check` at every checkpoint, commit per phase
(signed). MVP = US1 alone is already shippable (taxonomy gone); US2 completes the
accountability model; US3+US4 land together operationally via one provision pass.
