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

## 2026-07-27 — Bar 4 run: the MLS deferral holds at dogfood scale

Row-by-row verdict table (this entry is the protocol artifact) over the
design doc's threat model — every protected item, every stated exclusion,
both standing caveats, and the epoch-key scheme's own limitations — at the
bar's stated scale: ≤ ~10 members, member devices trusted, metadata
visibility accepted. Verdict per the pre-registered wording: **Bar 4
PASSES — zero requires-MLS rows at that scale; MLS stays an upgrade path,
not a prerequisite.**

| # | Threat-model row (from sealed-topics.md) | Verdict | Why |
|---|---|---|---|
| 1 | Content read by other realm personas | covered-by-epoch-keys | No key, no plaintext; AEAD. |
| 2 | Content read by curators | covered-by-epoch-keys | Curators are blind by design; lifecycle stays visible from metadata. |
| 3 | Content read by the operator / server-disk access | covered-by-epoch-keys | Payloads, baselines, attachments encrypted client-side; bounded by rows 10–11. |
| 4 | Ciphertext spliced to another topic / author / DAG position by the operator | covered-by-epoch-keys | AAD binds (realm, path, id, author, parents, epoch); Bar 1's amendment (epoch+nonce inside the signed record) additionally makes tampering publicly checkable. |
| 5 | Reading content published after leave/eject | covered-by-epoch-keys | Epoch bump excludes the leaver; nothing after the bump decrypts. |
| 6 | Leaver retains what they already saw | out-of-scope-by-design | "There is no retroactive revocation in cryptography" — stated in the design; MLS does not change this either. |
| 7 | Topic existence, `topic_id`, taxonomy position, headers, traffic analysis | out-of-scope-by-design | "Sealed is not hidden" — explicitly excluded; metadata privacy named a different, harder design. |
| 8 | Membership visibility (key distribution names holders; epochs mark joins/leaves) | out-of-scope-by-design | Explicitly excluded; MLS would not hide it (welcome/commit traffic is equally visible). |
| 9 | Mention-notification existence leaks | out-of-scope-by-design | Explicitly excluded; body is sealed (Bar 2 row 6), existence is not. |
| 10 | Key substitution by the adversarial operator (registry is operator-controlled) | out-of-scope-by-design | Standing caveat 1: out-of-band verification is the floor. TOFU pinning + Bar 2's signing-chain endorsement raise the bar; MLS does not close it — authenticating identities is orthogonal to group key agreement. |
| 11 | Member key material on operator-controlled infrastructure | out-of-scope-by-design | Standing caveat 2, stated plainly in the design ("theater for that persona"); no group protocol fixes host compromise. |
| 12 | Member device compromise exposes epoch keys → history readable (no forward secrecy) | out-of-scope at this scale | Excluded by the bar's own premise — member devices trusted. This is the first row that flips beyond dogfood scale; see below. |
| 13 | Compromised member keeps reading within the epoch (no post-compromise security) | out-of-scope at this scale | Same premise as row 12; a membership change already forces a new epoch, which is the scheme's (coarse) recovery lever. |
| 14 | Malicious *member* — leaks plaintext, bogus epoch bumps, wrong-key wraps, ejections | out-of-scope-by-design | Membership trust, stated in the design and in memory.md ("membership trust has to"); epoch ops are signed and attributable. MLS authenticates group operations; it does not prevent an authorized member's abuse. |
| 15 | Non-member destroys the tail with a garbage rollup (availability) | out-of-scope-by-design | Not a confidentiality threat and not sealing's job: identical exposure to open topics since 007, mitigated by signature attribution + keeper recovery (Bar 3), and AEAD makes the vandalism unambiguous. |

Zero rows read requires-MLS [judgment over the table; the table itself is
read off the design doc's own threat-model text]. The honest boundary of
the verdict: rows 12–13 pass *because of* the trusted-member-device
premise, not because epoch keys provide those properties — the design
already says so plainly. The deferral's flip conditions, recorded for the
build gate: membership scale or churn meaningfully beyond ~10, or member
devices no longer assumed trusted (e.g. personas on third-party hosts
joining sealed topics) — either moves MLS from upgrade path to
prerequisite. Every op shape surviving that swap (`sealed.epoch` as
carrier for MLS commit/welcome) is already the design's stated plan, and
nothing found in Bars 1–3 narrows it.

All four pre-registered bars have run: Bar 1 FAIL-with-amendment
(the `{"ct"}` wrapper), Bars 2–4 PASS. The topic is ready for graduation —
the design survives the shipped substrate with recorded amendments — but
graduation waits on the reversal condition's first reading: the dogfood
chafe log runs to 2026-08-10, and a zero-entry "this felt wrong in
plaintext" outcome would reverse sealing's *priority* even with these bars
green.
