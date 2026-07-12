# Tasks: Foundation — Realm Provisioning & the Operation Record

**Input**: Design documents from `specs/001-foundation/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: INCLUDED. The spec's success criteria (SC-003…SC-006) are explicit test matrices and the
constitution mandates all tests pass (none skipped), so every story ships tests alongside code.

**Organization**: Grouped by the five user stories from spec.md so each is independently testable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: US1…US5 (maps to spec.md user stories)
- Exact file paths are in each task. Module root is the repository root; module `github.com/impire/soulstream`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: A buildable, lintable, formatted empty Go module.

- [X] T001 Initialise the Go module: `go.mod` at repo root with `module github.com/impire/soulstream` and `go 1.26`.
- [X] T002 Add dependencies to `go.mod`/`go.sum` via `go get`: `github.com/nats-io/nats.go`, `github.com/synadia-io/orbit.go/natscontext`, `github.com/google/uuid`, `github.com/gowebpki/jcs`, and (test) `github.com/nats-io/nats-server/v2`.
- [X] T003 [P] Create `Makefile` at repo root with `fmt` (gofmt -w + goimports), `test` (`go test ./...`), `lint` (`golangci-lint run`), and a default `check: fmt test lint` target.
- [X] T004 [P] Create `.golangci.yml` at repo root enabling a sensible default linter set (govet, staticcheck, errcheck, ineffassign, revive/gofmt) for Go 1.26.
- [X] T005 [P] Create package skeletons with doc comments only: `record/doc.go`, `identity/doc.go`, `realm/doc.go`, `internal/natstest/doc.go`; and `docs/.gitkeep`.

**Checkpoint**: `make fmt && make lint && go build ./...` succeed on an empty module.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The two primitives shared across stories — the slug grammar (used by record, realm, and
identity) and the in-process test server (used by both provisioning stories).

**⚠️ CRITICAL**: No user story work begins until this phase is complete.

- [X] T006 [P] Implement the slug grammar in `identity/name.go`: `ValidName(string) bool` and `CheckName(string) error` returning a `*NameError{Name,Reason}` for the grammar `^[a-z0-9]+(-[a-z0-9]+)*$`, length 1–64 (rejecting empty, >64, uppercase, dot, whitespace, `*`/`>`, leading/trailing/double hyphen). (FR-024)
- [X] T007 [P] Table-driven test `identity/name_test.go`: exhaustive accept/reject matrix incl. `daan`, `invoice-agent`, `vat-q2-2026-x7m2` (accept) and `Daan`, `a.b`, `-x`, `x-`, `a--b`, ``, `>`, `a b`, 65-chars (reject with reason).
- [X] T008 Implement the test server helper in `internal/natstest/server.go`: `StartJetStream(t *testing.T) (url string, cleanup func())` using `server.NewServer(&server.Options{JetStream:true, StoreDir:t.TempDir(), Host:"127.0.0.1", Port:-1, NoLog:true, NoSigs:true})`, `go ns.Start()`, `ns.ReadyForConnections(10*time.Second)`, returning `ns.ClientURL()` and `ns.Shutdown`. (research §5)

**Checkpoint**: `go test ./identity/... ./internal/...` green; the helper starts and tears down a JetStream server.

---

## Phase 3: User Story 1 — Provision a realm from nothing (Priority: P1) 🎯 MVP

**Goal**: Point the library at an empty NATS server (named context) and create the mandated stream +
object store, reporting what was created.

**Independent Test**: Against a fresh in-process server, `ProvisionOn` creates both artefacts; inspect
server config and confirm `SOULSTREAM.>`, Limits, MaxAge 0, AllowRollup, Duplicates ≥2m, File storage,
and the `soulstream-objects` bucket. Report shows both `created`.

### Implementation for User Story 1

- [X] T009 [P] [US1] `realm/spec.go`: the non-configurable `RealmSpec` — stream name `SOULSTREAM`, subjects `["SOULSTREAM.>"]`, `LimitsPolicy`, `MaxAge 0`, `AllowRollup true`, `Duplicates 2*time.Minute`, `FileStorage`; object-store bucket `soulstream-objects`. (FR-003/004/005)
- [X] T010 [P] [US1] `realm/report.go`: `Artefact`, `Outcome` (`created`/`conformant`/`nonconformant`), `ArtefactResult{Artefact,Outcome,Nonconformities}`, `ProvisionReport{Results}` with `Conformant() bool`. (FR-009)
- [X] T011 [US1] `realm/provision.go`: `ProvisionOn(ctx, js jetstream.JetStream) (*ProvisionReport, error)` — for each artefact do a lookup (`js.Stream` / `js.ObjectStore`); on `ErrStreamNotFound`/`ErrBucketNotFound` create it and report `created`; on found (US1 scope) report `conformant`. Never call Update/CreateOrUpdate. (FR-006/007/008)
- [X] T012 [US1] `realm/connect.go`: `Config{ContextName,Realm,Persona}`, `Client`, `Connect(ctx, cfg)` — validate `cfg.Realm` (and `cfg.Persona` if set) via `identity.CheckName`, connect with `natscontext.Connect(cfg.ContextName)`, build `jetstream.New(nc)`, fail fast before any write; `Client.Provision(ctx)` delegates to `ProvisionOn`; `Close`, `Realm()`. (FR-001/002/028)
- [X] T012a [US1] Test `realm/connect_test.go`: invalid `cfg.Realm`/`cfg.Persona` rejected before any server contact; a non-existent context name errors from `natscontext.Connect` without partial mutation. (FR-002, US1 scenarios 2–3)
- [X] T013 [US1] Integration test `realm/provision_test.go` (fresh case): using `natstest`, connect directly, `ProvisionOn` a clean server, assert both `created` and read back the stream config field-by-field against `RealmSpec` and the bucket exists. (SC-001)
- [X] T014 [P] [US1] ELI5 doc `docs/realm.md`: the realm as "a private workshop — one account, one workbench-log, one supply cupboard", plain words + everyday analogy. (Constitution III)
- [X] T015 [P] [US1] ELI5 doc `docs/provisioning.md`: provisioning as "setting up the empty workshop; if it already exists we look, we don't rearrange". (Constitution III)

**Checkpoint**: A fresh realm is provisioned and verified; `docs/realm.md` + `docs/provisioning.md` exist.

---

## Phase 4: User Story 2 — Re-provision an existing realm safely (Priority: P1)

**Goal**: Re-running provisioning is a no-op on a conformant realm, completes a partial one, and reports
drift on a non-conformant one — never modifying an existing artefact in place.

**Independent Test**: Provision twice → second run reports both `conformant`, zero changes. Delete/omit
the bucket → run creates only the bucket, leaves the stream untouched. Pre-create a stream with MaxAge set
→ run reports `nonconformant` with the specific drift and does not mutate it.

### Implementation for User Story 2

- [X] T016 [US2] `realm/conformance.go`: compare an existing stream's `CachedInfo().Config` to `RealmSpec`, returning one nonconformity string per mismatch — incl. `MaxAge != 0`, `!AllowRollup`, `Retention != Limits`, `Storage != File`, `Duplicates < 2m`, subjects != `["SOULSTREAM.>"]`. (FR-008)
- [X] T017 [US2] Extend `realm/provision.go`: on found artefacts call `conformance` and report `conformant` or `nonconformant` (with drift); the run still succeeds when nonconformant (report is informational). (FR-006/008/009)
- [X] T018 [US2] Integration tests in `realm/provision_test.go` (idempotent re-run): provision twice, assert second run reports both `conformant` and made zero changes (compare `CachedInfo` before/after). (SC-002)
- [X] T019 [US2] Integration test (partial): provision, then simulate the bucket missing (fresh server + only the stream created), assert the run creates the bucket and leaves the stream `conformant`/untouched.
- [X] T020 [US2] Integration test (drift): pre-create `SOULSTREAM` with `MaxAge` set and `AllowRollup` false, run provisioning, assert `nonconformant` with both drifts named and the stream config unchanged after the run.
- [X] T021 [P] [US2] Extend `docs/provisioning.md` with the "already-exists: look, don't rearrange" behaviour and the drift example (why we never auto-fix age-expiry).

**Checkpoint**: Idempotent re-run, partial completion, and drift reporting all verified; no in-place mutation.

---

## Phase 5: User Story 3 — Build and read back an operation record losslessly (Priority: P1)

**Goal**: Construct an operation, serialise to wire headers + payload, parse back to an identical record.

**Independent Test**: Round-trip a matrix (0/1/many parents; sig/no-sig; with unknown `Soulstream-*`
headers) with no server, asserting full field equality and the absent-vs-empty-parents rule.

### Implementation for User Story 3

- [X] T022 [P] [US3] `record/id.go`: `NewID() string` → `uuid.NewString()` (lowercase 8-4-4-4-12); unit test `record/id_test.go` asserting format + uniqueness across N calls. (FR-013)
- [X] T023 [P] [US3] `record/errors.go`: `FieldError{Field,Reason}` + sentinels `ErrMissingField`, `ErrBadVersion`, `ErrBadTimestamp`, `ErrBadAuthor`, `ErrBadID` (errors.Is-compatible). (FR-016)
- [X] T024 [US3] `record/record.go`: `Record` struct (ID, Author, Parents, Type, Timestamp, Signature, Payload, Extras); `const Version = 1`; `Validate()`; `Build() (headers, payload, err)` writing `Nats-Msg-Id`, `Soulstream-Version/Author/Parents/Type/Ts/Sig` (+Extras), with the **absent⇔empty parents** rule; `Parse(headers, payload) (Record, error)` enforcing required fields, `Version==1`, RFC3339 ts, well-formed author (via `identity.CheckName`), preserving unknown `Soulstream-*` headers into `Extras`. (FR-010/011/012/014/015/016/017/018)
- [X] T025 [US3] Round-trip test `record/record_test.go`: matrix {0,1,many parents} × {sig, no-sig} × {with/without unknown headers}; assert `Parse(Build(r)) == r`; assert empty parents ⇒ no header and absent header ⇒ empty slice; assert unknown headers survive. (SC-003, FR-015/017)
- [X] T026 [US3] Negative test `record/record_test.go`: missing field, `Version==2`, malformed timestamp, bad author each rejected with the specific sentinel. (SC-005, FR-016)
- [X] T027 [P] [US3] ELI5 doc `docs/operation-record.md`: the record as "a delivery slip — the details are written on the label (headers), the box (payload) holds only the goods; the tracking number is also the anti-duplicate stamp". (Constitution III)
- [X] T027a [US3] Dedup integration test `realm/dedup_test.go`: provision via `natstest`, `Build()` a record, publish its headers+payload to a subject under `SOULSTREAM.>` twice with the same `Nats-Msg-Id` (== record ID) inside the duplicate window, assert exactly one message lands on the stream. Proves op-id doubles as the idempotency key. (FR-012, US3 scenario 3, SC-006)

**Checkpoint**: Record build/parse round-trips across the full matrix with no server; a retried publish is de-duplicated to one message.

---

## Phase 6: User Story 4 — Produce and verify the canonical record (Priority: P2)

**Goal**: A deterministic, field-order-independent canonical serialisation bound to realm + topic.

**Independent Test**: Same content in different field orders → byte-identical output; wire⇆canonical is
lossless; the canonical record carries realm + topic.

### Implementation for User Story 4

- [X] T028 [US4] `record/canonical.go`: `Record.Canonical(realm, topic string) ([]byte, error)` — build the canonical object `{v,realm,topic,id,author,parents,ts,type,data[,sig]}` (payload parsed to a JSON value under `data`; `sig` omitted when empty), `json.Marshal` → `jcs.Transform`. (FR-019/021/022/023)
- [X] T029 [US4] Determinism test `record/canonical_test.go`: construct the same logical record with fields supplied in different orders (and map-based payloads) → assert byte-identical `Canonical` output. (SC-004, FR-020)
- [X] T030 [US4] Losslessness + binding test: assert every wire field maps to exactly one canonical key and back; assert `realm` and `topic` are present and bind (changing either changes the bytes). (FR-021/022)
- [X] T031 [P] [US4] ELI5 doc `docs/canonical-record.md`: canonical form as "re-typing the slip onto a standard government form so any two people who fill it from the same facts get the exact same page — that sameness is what a signature will one day sign". (Constitution III)

**Checkpoint**: Canonical output is deterministic, lossless, and realm/topic-bound.

---

## Phase 7: User Story 5 — Validate persona names and attribution (Priority: P2)

**Goal**: Reject malformed persona names (grammar from Phase 2) and enforce honest attribution — write-side
refusal to speak as another persona, read-side structural check plus optional trusted-resolver cross-check.

**Independent Test**: Feed the name matrix (reuses T007); attempt to publish a record whose author ≠ the
client's persona → refused; with a resolver returning a different identity → mismatch flagged.

### Implementation for User Story 5

- [ ] T032 [US5] `identity/author.go`: `EnforceAuthor(ownPersona, recordAuthor string) error` (→ `ErrForeignAuthor`); `Resolver` interface (`DeliveredBy(headers) (persona, ok)`); `VerifyAuthor(claimedAuthor, headers, r Resolver) error` — always `CheckName(claimedAuthor)`, and when a resolver resolves an identity require equality (→ `ErrAuthorMismatch`). (FR-025)
- [ ] T033 [US5] Wire write-side enforcement into `realm/connect.go`/publish path: when `Config.Persona` is set, a publish helper calls `EnforceAuthor(cfg.Persona, record.Author)` before any send (this feature has no publish op yet, so expose it as a guard method + unit-test it directly). (FR-025)
- [ ] T034 [US5] Test `identity/author_test.go`: `EnforceAuthor` accepts self / rejects foreign; `VerifyAuthor` passes with no resolver (shape-only), passes on match, returns `ErrAuthorMismatch` on mismatch. (SC-005)
- [ ] T035 [P] [US5] ELI5 doc `docs/persona-and-attribution.md`: personas as "everyone signs their own name; the library won't let you sign someone else's, and if a trusted doorman is on duty it double-checks the name at the door". (Constitution III)

**Checkpoint**: Name validation and both sides of attribution are enforced and tested.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T036 [P] Write module `README.md` (top-level) summarising the three packages, the quickstart, and the `make check` gate; link the spec.
- [ ] T037 Run `go mod tidy`; ensure `go vet ./...` is clean.
- [ ] T038 Validate `specs/001-foundation/quickstart.md` end-to-end against the built code (adjust any signature drift in docs to match the code).
- [ ] T039 Final gate: `make fmt && make test && make lint` all green, no skipped tests, `go build ./...` clean. (SC-007)

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (P1)** → no deps.
- **Foundational (P2)** → depends on Setup; **blocks all stories** (grammar + test server).
- **US1 (P1)** → after Foundational. **MVP.**
- **US2 (P1)** → after US1 (extends `realm/provision.go` + adds `conformance.go`).
- **US3 (P1)** → after Foundational (independent of US1/US2). Uses `identity.CheckName`.
- **US4 (P2)** → after US3 (uses `Record`).
- **US5 (P2)** → after Foundational; write-side wiring (T033) touches `realm` so also after US1.
- **Polish** → after all desired stories.

### Within a story

- Models/types (`spec.go`, `report.go`, `errors.go`, `id.go`) before services (`provision.go`, `record.go`, `canonical.go`, `author.go`).
- Implementation before its tests are meaningful, but tests ship in the same story.
- Each story's `docs/` (ELI5) task MUST be done before the story is "done" (Constitution III).

### Parallel opportunities

- Setup: T003, T004, T005 in parallel.
- Foundational: T006/T007 (identity) parallel with T008 (natstest).
- US1 types T009 (spec) ∥ T010 (report) ∥ docs T014/T015.
- US3 T022 (id) ∥ T023 (errors) ∥ doc T027.
- Because US3/US4 (record) and US5 (identity) touch different packages than US1/US2 (realm), the
  record+identity track and the realm track can be built in parallel once Foundational is done.

## Parallel Example

```text
# After Foundational, two independent tracks:
Track A (realm):    US1 (T009→T015) → US2 (T016→T021)
Track B (record):   US3 (T022→T027) → US4 (T028→T031)
Track C (identity): US5 (T032→T035)  # T033 waits on US1's connect.go
```

## Implementation Strategy

1. **Setup + Foundational** → buildable module with the shared grammar and test server.
2. **US1** → provision a realm (MVP; stop and validate against a live/embedded server).
3. **US2** → make it idempotent + drift-aware.
4. **US3 → US4** → the record and its canonical form (pure, no server).
5. **US5** → attribution.
6. **Polish** → README, tidy, quickstart validation, full green gate.

## Notes

- Commit after each task or logical group; every commit signed; `make check` before each commit.
- `record`/`identity` MUST NOT import anything from `nats.go`; only `realm` and `internal/natstest` may.
- Do not start deferred scope (topics, lifecycle, baselines, mentions, discovery, signing, CLI, MCP).
