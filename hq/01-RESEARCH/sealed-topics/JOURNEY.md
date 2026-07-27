# sealed-topics — investigation journey

Started 2026-07-27.

## 2026-07-27 — Bar 1 run: the envelope, as drawn, does not sign

Throwaway Go prototype (session scratchpad, public `record`/`identity` API
only; random bytes standing in for XChaCha20-Poly1305 ciphertext — the cipher
is out of scope, the envelope is what's tested). Verdict per the
pre-registered wording: **Bar 1 FAILS for the design's literal shape, and the
smallest passing amendment is recorded.**

- **Shape 1 — raw binary ciphertext payload (the design doc's literal
  sketch): FAIL [measured].** The canonical form refuses non-JSON payloads
  outright (`record: payload is not valid JSON`) — there is no signing input,
  so a raw-payload `sealed.op` can never carry a verifying author signature;
  a forcibly attached signature grades `failed`.
- **Shape 2 — JSON-wrapped payload `{"ct":"<base64>"}`: PASS on every
  sub-check [measured].** Signs at the live path, verifies with zero access
  to epoch keys, and a captured exhibit gets a `verified` verdict OFFLINE
  (pins-only keyring). `Soulstream-Epoch`/`Soulstream-Nonce` ride as extra
  headers and are captured verbatim in the exhibit. This is the smallest
  encoding amendment: the design doc's "payload is raw ciphertext, binary"
  line must become a one-field JSON wrapper.
- **The epoch and nonce headers are NOT covered by the author signature
  [measured].** Rewriting `Soulstream-Epoch` 4→9 (or the nonce) on the wire
  leaves the signature verifying. Controls prove the harness honest:
  tampering the timestamp, the ciphertext, or splicing to another topic all
  flip to `failed`. Consequence for the design: epoch/nonce integrity
  currently rests ONLY on the AEAD's associated data — members detect
  tampering at decryption, but non-members (graders, archivists, curators)
  cannot. If epoch integrity should be publicly checkable, epoch and nonce
  belong INSIDE the signed record (payload fields beside `ct`), not in
  headers [judgment — design-amendment input for graduation].

## 2026-07-27 — Bar 2 run: every distribution need lands on a shipped surface

Written row-by-row mapping (this entry is the protocol artifact) against
[registry.md](../../02-DESIGN/extensions/registry.md) and the `registry`
package public API (`profile.go`, `kv.go`, `chain.go`, `attest.go`). Verdict
per the pre-registered wording: **Bar 2 PASSES — zero rows require a new
server component or stream; every gap is an additive profile field or
additive vocabulary.**

| # | Distribution need | Lands on | Gap |
|---|---|---|---|
| 1 | Sealing-key publication (X25519 beside the Ed25519 signing key) | `registry.Profile` + `Publish`'s create-or-update path (`registry/kv.go`), where stored key material is already authoritative | **Additive profile field.** `sealing_key` is named in registry.md but absent from the shipped `Profile` (`registry/profile.go:31`); zero X25519 anywhere in the module [measured]. Needs the field, a `Validate()` key-shape check, and `Publish`'s key-conflict guard extended to it. |
| 2 | Sealing-key rotation | The `Rotations`/`Chain`/`BuildKeyring` pattern (`registry/chain.go`) | **Additive profile field + design amendment.** `Rotate` (`registry/kv.go:127`) is Ed25519-specific, and X25519 keys cannot sign, so registry.md's "new key signed by the old one" cannot apply literally — see the endorsement finding below. |
| 3 | Epoch wrap targets (current members' sealing keys at wrap time) | `registry.Lookup`/`All` + `BuildKeyring` for trust; the `sealed.epoch` op rides the existing op-log — `record.Record.Type` is "non-empty, not enumerated here" (`record/record.go:36`), and no publish path enforces a type allowlist [measured] | None beyond row 1. |
| 4 | Prior-epoch handoff to joiners | The same op-log: an additive vocabulary op (exact shape left to speckit), its payload wrapped to the joiner's sealing key resolved via `Lookup` | **Additive vocabulary only.** |
| 5 | Epoch-1 material at announcement | `topic.announce` payload fields (additive), wrap targets resolved as row 3 | None beyond row 1. |
| 6 | Sealed mention-notification bodies | `Lookup` for the target's sealing key; delivery over the existing bounded notify stream (`realm.NotifyStreamName`, 014) | None — body encryption is client-side. |
| 7 | Pinning + out-of-band fingerprint verification | `BuildKeyring`'s pin map (persisted via `internal/keystore`); fingerprint display is client presentation | None — and under the endorsement amendment below, the sealing key inherits the signing chain's trust, so no separate pin lineage is needed. |

Findings feeding graduation:

- **Strict decode imposes a rollout order [measured].** `decodeProfile` uses
  `DisallowUnknownFields` (`registry/kv.go:22`): a profile carrying
  `sealing_key` published before readers upgrade makes `Lookup` hard-error
  and `All` warn-and-skip for that persona. The field must ship in the
  library — and reach deployed clients — before any persona publishes one.
  A deployment-ordering constraint, not a bar failure: the criterion
  concerns server components and streams, and this requires neither.
- **Endorse sealing keys with the signing chain [judgment —
  design-amendment input].** Because X25519 cannot sign, each sealing key
  (initial and every rotation) should carry an endorsement by the persona's
  Ed25519 signing chain over a domain-separated statement, in the shape of
  `identity.RotationProofBytes`/`AttestationBytes` (`identity/sign.go:80`).
  Consequence: publishing a sealing key requires a published signing key
  first — coherent, since sealed topics presuppose signing (Bar 1).
- **Two loud-failure edges are library decisions, not surface gaps
  [judgment].** A member with no published sealing key cannot be wrapped
  to — epoch publication must fail loudly, per registry conventions. A
  mentioned persona with none can only receive an existence-only
  notification — already the design's stated leak posture. Whether a
  handoff op survives rollup is Bar 3's territory, noted here only as the
  boundary.

## 2026-07-27 — Bar 3 run: sealed rollup stays leaderless

Written trace (this entry is the protocol artifact) through `topic`'s rollup
path and the archivist flow, ciphertext baseline state assumed throughout.
Verdict per the pre-registered wording: **Bar 3 PASSES — no step requires
plaintext outside a member, and no step requires a distinguished leader.**
The prototype was not run: no trace step was contested — every claim below
reads off shipped code or inherits Bar 1's measurements.

The trace, step by step:

1. **Drain.** `drainOps` reads the topic's ops subject in stream order
   (`topic/replay.go:17`); `record.Parse` validates headers only — payload
   bytes pass through untouched (`record/record.go:115`; payload JSON
   validity is checked only at signing, `record/canonical.go:53`).
   Ciphertext drains identically to plaintext [measured].
2. **Fold.** `apply` is a pure function of the log (`topic/view.go:107`),
   and its DAG bookkeeping — `seen`/`referenced`, hence the frontier — runs
   *before* the type switch, on cleartext headers alone: a sealed log's
   frontier computes with zero decryption [measured]. Folding *content* is
   the one plaintext step in the entire path: the sealed variant feeds
   decrypted inner records to the same pure fold and encrypts the resulting
   state — client-side, inside a member, exactly where the design assigns
   it. To a non-member the same ops hit the warn-and-ignore arm
   (`topic/view.go:381`) — unreadable, never "malformed".
3. **Publish + race guard.** The baseline publishes under
   `WithExpectLastSequencePerSubject` (`topic/rollup.go:132`) — the server
   compares sequence numbers only, content-blind. Any concurrent write
   rejects the attempt wholesale (`ErrRollupLost`), log untouched, first
   writer wins (`TestRollupRaces`, `topic/rollup_test.go:148`). No leader
   anywhere: any member may attempt, simultaneous attempts arbitrate by the
   guard [measured].
4. **Purge.** The `MsgRollup` header purges predecessors per-subject,
   server-side, content-blind (`topic/rollup.go:131`). The manifest form
   writes the state document to the object store *before* the guarded
   publish (`topic/rollup.go:155`) — the crash-safe order is preserved with
   ciphertext chunks, and the digest covers ciphertext, exactly as sealed
   attachments already do [measured].
5. **Recovery.** Keeper side: `keeper.Run` captures every ops/info message
   verbatim — headers and payload bytes copied, payload never interpreted
   (archivist `internal/keeper/keeper.go:97`, `store.Put`); type- and
   content-blind, so sealed ops are kept identically [measured]. Asker
   side: `FetchExhibit` goes live-first (`CaptureExhibit` is verbatim,
   `topic/exhibit.go:76`), and on `ErrOpNotLive` — the post-purge state —
   scatters `memory.fetch`; every returned exhibit is checked
   op-id/realm/binding and signature-verified against pins, cleartext
   headers only (`topic/memory.go:294`). Bar 1 measured a `sealed.op`
   exhibit grading `verified` offline with a pins-only keyring;
   `GradeForVerdict(SigVerified)` is `fact-with-provenance`. The
   keep → rollup → verified-recovery story is the archivist's own e2e test
   for open ops; sealed differs only in payload bytes, which no step of
   this chain inspects [measured + Bar 1].

The criterion's three sub-claims: **any member can compact** — steps 1–4;
the only member-only input is the epoch key for the fold, which is what
membership *is*. **Non-members cannot produce a valid sealed baseline** —
validity is the AEAD's: a valid sealed baseline decrypts under the current
epoch key, which non-members lack. Stated honestly: a non-member with
publish rights *can* publish garbage-with-rollup-header and destroy the
live tail — exactly as in open topics since 007, because leaderless means
anyone may compact; the mitigations are unchanged (signature attribution,
keeper recovery), and detection is *stronger* sealed — AEAD failure is
unambiguous where an open topic's wrong baseline needs semantic review
[judgment]. Subject-permission defense in depth remains available per the
design. **A destroyed interior sealed op returns as a verifying exhibit** —
step 5.

Findings feeding graduation:

- **Rollup destroys `sealed.epoch` ops [measured — design-amendment
  input].** The purge is per-subject and `sealed.epoch` rides the ops
  subject: every epoch op before the baseline dies with the compacted
  tail — and so does any interior handoff op (closing Bar 2's boundary
  note). Epoch-1 material survives only because it travels in
  `topic.announce` on the INFO subject, which rollup never touches
  (`publishBaseline` publishes solely to `OpsSubject`). Amendment: the
  sealed baseline must re-carry the current epoch's wrapped-key table (and
  whatever prior-epoch wraps the group means to keep offering). Any member
  holds the full wrapped map from the epoch op it read, so any member can
  re-carry it — leaderlessness preserved; wrapped blobs are already sealed
  per-member, so they are safe as cleartext payload fields. Additive
  vocabulary, Bar 2's sanctioned gap class. Without this, any rollup locks
  out a member who lost local key state and orphans mid-handoff joiners;
  with it, rollup is key-distribution-preserving.
- **The sealed baseline keeps the cleartext `topic.baseline` type and
  withholds `Baked` [judgment — design-amendment input].** `apply` requires
  the first record's type to be `topic.baseline` (`topic/view.go:114`) and
  the mid-log checkpoint arm keys on the same type — the type must stay
  cleartext, leaking only that a compaction happened, which the purge
  reveals anyway. And the shipped `BakedState` carries contribution bodies,
  attachment names, and work items in cleartext: a sealed baseline must
  carry all of that *inside* the ciphertext state, keeping at most
  `Lifecycle` visible (the design keeps lifecycle observable). The shipped
  payload shape accommodates this — every field is omitempty — but the
  design doc must say it explicitly.
- **Ciphertext keeping is already sanctioned [measured].** memory.md states
  it verbatim ("a shared historian archives sealed topics only as
  ciphertext — provable *that* an op existed, not what it said"). The
  archivist's `Search` may surface ciphertext snippets as noise but cannot
  leak plaintext; excluding sealed types from search is an archivist-side
  refinement, not substrate [judgment].

Bar 4 not yet run.
