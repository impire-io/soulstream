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

Bars 3–4 not yet run.
