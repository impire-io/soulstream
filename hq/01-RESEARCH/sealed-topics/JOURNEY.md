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

Bars 2–4 not yet run.
