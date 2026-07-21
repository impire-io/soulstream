# Research: Op Signing & Key Distribution

All Technical Context unknowns resolved. Decisions below; each with rationale and
alternatives considered.

## R1. Signature algorithm & encoding

- **Decision**: Ed25519 (stdlib `crypto/ed25519`), signature carried in `Soulstream-Sig`
  as standard base64 (`encoding/base64.StdEncoding`) of the 64-byte signature. Public
  keys are base64 of the 32-byte key; secret material stored as the 32-byte seed.
- **Rationale**: Ed25519 is mandated by the design (`hq/02-DESIGN/core/01-protocol.md`,
  `extensions/registry.md`). Stdlib implementation means zero new dependencies.
  Base64 std (not URL) matches the design docs' `"ed25519": "<base64>"` examples.
  Seeds (not expanded private keys) are the canonical Ed25519 storage form —
  `ed25519.NewKeyFromSeed` reconstructs deterministically.
- **Alternatives considered**: raw hex (longer, no design precedent); nkeys-style
  encoding (couples signing identity to NATS credential tooling — the design keeps the
  transport anchor and the durable signing key separate on purpose).

## R2. Signing input

- **Decision**: the bytes of `record.Record.Canonical(realm, topicBinding)` computed
  with `Signature` empty. Verification re-parses the wire record, clears `Signature`,
  recomputes canonical bytes, and verifies. No changes to `record` needed: `sig` is
  `omitempty` in the canonical form, so an empty signature produces exactly the
  pre-signature bytes.
- **Rationale**: the canonical record was built in 001 for precisely this purpose; the
  `omitempty` behaviour makes "canonical bytes of the unsigned record" and "signing
  input" the same thing with no second serialisation path.
- **Alternatives considered**: signing the raw wire headers+payload (unstable — header
  order and whitespace are transport artifacts); adding a dedicated `SigningBytes`
  method (redundant with `Canonical`).

## R3. Canonical topic binding per subject family

- **Decision**: the `topic` value bound into the canonical record is the subject suffix
  after the fixed prefix — the topic path for `SOULSTREAM.TOPICS.OPS.<path>` and
  `SOULSTREAM.TOPICS.INFO.<path>`, the persona name for
  `SOULSTREAM.PERSONA.NOTIFY.<persona>`. One helper (`canonicalBinding(subject)`) owns
  the rule.
- **Rationale**: deterministic and verifiable by any reader from the subject the op
  arrived on (the reader always knows the subject it consumed); keeps the design's
  anti-splicing property for every op family with one rule and no new headers.
- **Alternatives considered**: binding a payload-derived topic for notify ops
  (verifier would need payload schema knowledge per type — violates "one record shape");
  leaving info/notify ops unsigned (fails FR-002).

## R4. Where signing hooks in

- **Decision**: `realm.Config` gains an optional `Signer *identity.SigningKey`;
  `realm.Client.Signer()` exposes it. `topic.publishOp` — the single choke point every
  publish already flows through (turns, comments, attachments, announce, baseline,
  lifecycle, notify) — signs when `c.Signer() != nil`: build record, compute canonical
  bytes, set `Signature`, rebuild headers.
- **Rationale**: one code path signs everything (FR-002, SC-002); personas without a
  signer hit the identical pre-feature path (FR-003). `Build()` is cheap; re-running it
  after setting `Signature` avoids a second wire-assembly implementation.
- **Alternatives considered**: signing inside `record.Build` (record must stay
  crypto-free and realm/topic-unaware); per-call-site signing (n paths to forget one).

## R5. Persona directory storage & concurrency

- **Decision**: JetStream KV bucket `soulstream-personas` (history 10), key = persona
  name, value = profile JSON. First publish uses KV `Create` (fails if the key exists);
  rotation and profile edits use `Update` with the read revision. Provisioned as the
  realm's third artefact, create-or-report, same pattern as stream and objects bucket
  (`Stream()` lookup + create, never update-in-place).
- **Rationale**: constitution I — KV is the native watchable map with history and
  optimistic concurrency built in; `Create`/`Update(rev)` implement "refuse to silently
  replace a published key" and the two-clients-race edge case with zero custom
  coordination.
- **Alternatives considered**: profiles as ops on a registry topic (replay cost and
  lifecycle noise for a lookup table; KV *is* the stream-backed projection, maintained
  by the server); a separate bucket per persona (namespace sprawl, nothing gained).

## R6. Profile shape & rotation proof

- **Decision**: profile JSON carries `name`, `display_name`, `kind`, `description`,
  `operated_by`, `created_at`, `signing_key {ed25519, since}`, and `rotations` — an
  ordered list of `{from, to, proof}` entries. The chain is derived: root = first key
  (or `rotations[0].from`), each rotation's `proof` is Ed25519 by `from` over the
  domain-separated bytes `"soulstream-key-rotation\n" + persona + "\n" + to`, and the
  final `to` must equal the current `signing_key.ed25519`. No separate key-history
  field; the chain is the history.
- **Rationale**: smallest structure that makes the chain self-validating offline;
  binding the persona name into the proof prevents replaying a rotation onto another
  persona's profile; excluding `since` from proof bytes honours the clarification that
  `since` is informational.
- **Alternatives considered**: JCS-canonicalised proof object (more machinery for the
  same bytes); signing the whole new profile with the old key (couples display-metadata
  edits to key custody); relying on KV history for the chain (KV history is
  operator-truncatable; the profile must be self-contained for TOFU).

## R7. Pinning model

- **Decision**: clients pin, per realm, per persona, the *validated chain* (ordered
  list of base64 public keys) as first seen — stored in one JSON file client-side. On
  every directory read: validate the profile's internal chain; then require the pinned
  chain to be a prefix of the profile chain. Extension → accept and re-pin the longer
  chain. Anything else (shorter, diverged, invalid proof) → persona is *distrusted*:
  its signatures report `failed` and clients surface a substitution-attack warning.
  Verification accepts a signature matching **any** key in the validated chain.
- **Rationale**: implements the clarified TOFU-of-the-chain semantics (FR-006/FR-008)
  with one invariant — "pinned is a prefix of published" — that is trivially testable;
  any-chain-key verification keeps superseded-era ops verifiable without era
  bookkeeping.
- **Alternatives considered**: pinning only the current key (breaks old-era
  verification and cannot distinguish rotation from substitution); pinning in a KV
  bucket (the pin's whole purpose is surviving substrate compromise — it must be
  client-side); per-op era matching by `since` (clarification explicitly rules it out).

## R8. Verification surfacing in the library

- **Decision**: a pure annotate pass in `topic` (`verify.go`): given `[]SeqRecord`, the
  realm name, the subject binding, and an `identity.Keyring` (persona → validated chain
  + distrusted set), produce per-op `SigStatus` ∈ {`unsigned`, `verified`, `failed`,
  `unknown-key`}. View structs (`Contribution`, `Attachment`, `Announcement`,
  `Notification`) gain a `Sig SigStatus` field. Materialise/Follow/FetchInbox thread
  statuses through; a nil keyring degrades signed ops to `unknown-key` (SC-007). The
  keyring is built by the client from the directory + pins (via `registry`), keeping
  `topic`'s projection logic server-free as mandated by project conventions.
- **Rationale**: keeps the fold pure and serverless-testable; statuses are a function
  of (records, keyring), not of I/O; nil-keyring degradation makes the directory a true
  extension rather than a dependency.
- **Alternatives considered**: verification inside `Materialise` doing its own KV reads
  (breaks the pure-fold convention); a separate `Verifier` service type (machinery the
  scale doesn't need).

## R9. Client key & pin storage

- **Decision**: new `internal/keystore` shared by CLI and MCP. Defaults under
  `os.UserConfigDir()/soulstream/`: seed at `keys/<realm>-<persona>.ed25519` (written
  0600), pins at `pins/<realm>.json`. Overridable via `--key-file` / `--pins-file`
  flags and `SOULSTREAM_KEY_FILE` / `SOULSTREAM_PINS_FILE` env vars (MCP uses the env
  forms). Missing key file ⇒ client runs unsigned, exactly as today.
- **Rationale**: both clients need identical semantics (FR-012/FR-013); one tiny
  package prevents drift; config-dir default keeps the dogfood flow zero-flag.
- **Alternatives considered**: OS keychain integration (platform-specific machinery,
  deferred until a concrete need); keys in the NATS credential file (mixes transport
  credential lifecycle with durable signing identity, contra design).

## R10. Client surface

- **Decision**: CLI gains `key init|show|rotate` and `profile publish|show`;
  `show`/`watch`/`inbox` render per-op glyphs (`✓` verified, `✗` failed, `?`
  unknown-key, nothing for unsigned) plus a loud one-line banner naming any distrusted
  persona. MCP gains one tool (`publish_profile`) and sig-status fields in every
  tool result that returns ops; it signs automatically when the env-configured key
  exists. Key rotation stays CLI-only this cycle (an agent's operator rotates keys).
- **Rationale**: FR-012/FR-013 exactly; the banner satisfies "loud,
  machine-distinguishable" without new output modes; rotation-via-CLI keeps the MCP
  tool count at the minimum the spec requires.
- **Alternatives considered**: MCP rotate tool (no acceptance scenario requires it —
  cut per constitution II); always-on verification column for unsigned ops (noise in a
  realm that hasn't adopted signing yet).
