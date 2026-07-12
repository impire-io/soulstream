# Tasks: Participation — Mentions & Attachments

**Input**: Design documents from `specs/003-participation/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: INCLUDED. **Organization**: by 3 user stories; extends the `topic` package.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Vocabulary + view extensions shared by both mentions and attachments.

- [X] T001 [P] Extend `topic/vocab.go`: add `mentions,omitempty` to `TurnPayload` and
  `CommentPayload`; add `NotifyPayload{Topic,OpID,Author}`, `AttachmentPayload{Name,Object,Digest,
  Size,ContentType,Anchor}`, and constants `TypeMentionNotify = "mention.notify"`,
  `TypeAttachmentAdd = "attachment.add"`. (FR-002/003/008)
- [X] T002 Extend `topic/view.go`: add `Attachment` type and `MaterializedTopic.Attachments`; in
  `apply`, handle `attachment.add` (append to Attachments, count as content op), extend the dangling
  check to attachments. (FR-012/013/010)
- [X] T003 Extend `topic/view_test.go`: pure fold — an `attachment.add` becomes an Attachment,
  activates the topic, and a bad-anchor attachment is flagged dangling. (SC-004)

**Checkpoint**: `go test ./topic/...` green (pure); existing 002 tests still pass.

---

## Phase 3: User Story 1 — Mention a persona and notify (Priority: P1) 🎯 MVP

**Independent Test**: post a turn with `@bookkeeper-agent`; a follower on that inbox receives a
`mention.notify`; the op payload lists the mention.

- [X] T004 [P] [US1] `topic/mention.go`: `ParseMentions(body) []string` (regex `@([a-z0-9]+(-[a-z0-9]+)*)`,
  de-dup, `identity.ValidName`); rewire `PostTurn`/`AddComment` to fill `mentions` and, after
  publishing, call the notify helper. (FR-001/002/003/005/006)
- [X] T005 [P] [US1] `topic/notify.go`: `NotifySubject(persona)`, `Notification` type, an internal
  `publishNotify`, and `FollowInbox(ctx, c, persona, onNotify)` (ordered consumer, DeliverAll,
  parse `mention.notify`, cancellable). (FR-003/004)
- [X] T006 [P] [US1] `topic/mention_test.go`: pure parse matrix — `@Daan`/`@@`/`@ x` yield nothing,
  duplicates collapse, `@bookkeeper-agent!` → `bookkeeper-agent`, self-mention parsed. (SC-002)
- [X] T007 [US1] Integration test `topic/notify_test.go`: post a turn mentioning a second persona;
  a `FollowInbox` on that persona receives the notification (topic, op-id, author); the op payload's
  `mentions` lists the persona. (SC-001)
- [X] T008 [P] [US1] ELI5 doc `docs/mentions.md`: mentions as "tapping someone on the shoulder; the
  ping lands in their pigeonhole even if they're out". (Constitution III)

**Checkpoint**: Mentions parse, record, and notify an inbox.

---

## Phase 4: User Story 2 — Attach a file (Priority: P1)

**Independent Test**: attach a file; bytes land in the object store; `attachment.add` records
name+digest+size; materialise shows it and the topic is active.

- [X] T009 [US2] `topic/attachment.go`: `Attach(ctx, name, contentType, data, anchor)` — reject empty
  name; `os := c.JetStream().ObjectStore(realm.ObjectBucket)`; `PutBytes` under
  `attachments/<path>/<uuid>`; publish `attachment.add` with the returned digest+size. (FR-007/008/009/011)
- [X] T010 [US2] Integration test `topic/attachment_test.go`: attach a small file; assert the object
  exists with matching bytes, the `attachment.add` records name/digest/size/content-type, materialise
  lists the attachment with its anchor, and the topic is `active`; a zero-byte file is allowed; an
  empty name is rejected. (SC-004, FR-011)
- [X] T011 [P] [US2] ELI5 doc `docs/attachments.md`: attachments as "a shared filing cabinet; the
  notebook keeps a little claim ticket (name + fingerprint), the big thing lives in the cabinet".
  (Constitution III)

**Checkpoint**: Files attach and appear on the topic.

---

## Phase 5: User Story 3 — Retrieve an attachment (Priority: P2)

**Independent Test**: fetch the attachment's bytes by reference; they equal the original and match
the digest; a missing object errors cleanly.

- [X] T012 [US3] Extend `topic/attachment.go`: `GetAttachment(ctx, c, object) ([]byte, error)`
  (map `ErrObjectNotFound` to a clear error) and `VerifyDigest(data, digest) bool`
  (`"SHA-256="+base64url(sha256(data))`). (FR-014/015/016)
- [X] T013 [US3] Integration test in `topic/attachment_test.go`: after attaching, `GetAttachment`
  returns the exact bytes, `VerifyDigest` passes for them and fails for other bytes, and fetching a
  non-existent object returns a not-found error (no panic). (SC-003/005)

**Checkpoint**: Attachments retrieve and verify.

---

## Phase 6: Polish

- [X] T014 [P] Update `README.md`: note mentions + attachments on the `topic` row; add the two docs.
- [X] T015 Extend `topic/quickstart_test.go` (or add `participation_test.go`) mirroring the 003
  quickstart: mention → inbox, attach → materialise → get + verify.
- [X] T016 Run `go mod tidy`; `go vet ./...` clean.
- [X] T017 Final gate: `make check` green — all tests pass (none skipped), lint 0. (SC-005)

---

## Dependencies

- **Foundational (T001–T003)** blocks the stories.
- **US1 (T004–T008)**, **US2 (T009–T011)** after Foundational (independent tracks).
- **US3 (T012–T013)** after US2 (retrieval reads what US2 stored).
- **Polish** last.

## Notes

- Stay in the `topic` package; keep `ParseMentions`/`VerifyDigest` pure (no NATS) for server-free tests.
- Do not implement `attachment.remove`, encryption, mention digests/presence (deferred).
- Commit per story (signed); `make check` before each commit.
