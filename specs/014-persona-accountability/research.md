# Research: Persona Accountability & Stream Hygiene

All decisions below were reached by reading the current code (`registry/`, `identity/`,
`realm/`, `topic/`, `internal/cli`, `internal/mcpserver`) and the NATS JetStream feature
set mandated by the constitution.

## D1 — How to remove `kind` and enforce strict profiles

**Decision**: Delete `Profile.Kind`, the three `Kind*` constants, and the validation
branch. Introduce one strict decoder `decodeProfile([]byte) (Profile, error)` in
`registry` using `json.Decoder.DisallowUnknownFields`, used by every read path
(`Publish`'s stored-read, `Rotate`, `Lookup`, `All`). `Lookup` (direct target) fails
loudly naming persona and field. `All` (bulk) skips the bad entry and returns it in a
new second return value `[]ProfileWarning{Persona, Err}` so callers can warn without
the read failing; `BuildKeyring` then never sees the bad profile, so its signatures
report unknown-key — exactly the clarified blast-radius rule.

**Rationale**: `DisallowUnknownFields` is the same fail-loud mechanism 013-config
established for config files; reusing it keeps one philosophy. The warning-list return
is the smallest change that lets bulk callers warn loudly (CLI → stderr) without a
callback abstraction.

**Alternatives considered**: silently stripping unknown fields (violates FR-003);
a versioned profile schema (speculative generality — we are the only users).

## D2 — Attestation format and verification rule

**Decision**: Mirror the rotation-proof pattern exactly.

- `identity.AttestationBytes(operator, operated, operatedKeyB64 string) []byte` returns
  the domain-separated statement
  `"soulstream-operator-attestation\n" + operator + "\n" + operated + "\n" + operatedKeyB64`
  (empty `operatedKeyB64` when the operated persona has no key — binding then rests on
  the directory-unique name).
- Profile gains `OperatorAttestation *OperatorAttestation `json:"operator_attestation,omitempty"``
  with `OperatedKey string` (the key the operator saw, `""` if none) and `Sig string`
  (operator's Ed25519 signature over the statement, standard base64). Operator and
  operated names are NOT duplicated inside the stored attestation — they are `OperatedBy`
  and `Name` on the same profile; duplicating them would create a second copy that could
  disagree.
- The portable token (what the operator hands to the operated persona) IS self-contained:
  base64(JSON `{operator, operated, operated_key, sig}`), so it can travel over chat or
  a topic without ambiguity. `registry.NewAttestationToken` / `ParseAttestationToken`.
- **Verification rule** (pure function `registry.AttestationStatus`): a claim is
  - `attested` when the sig verifies against ANY key in the operator's validated chain
    (same rule as op signatures, rotation-proof) AND the bound `OperatedKey` is either
    `""` or a member of the operated persona's own validated chain;
  - `unverified` when there is no attestation, or the operator has no published
    key/chain to verify against;
  - `failed` when a sig is present but does not verify, when the bound key is not in the
    operated persona's chain, or when the operator is distrusted (substitution
    territory poisons vouches too).

**Rationale**: One key type, one signature style, one chain rule — no new cryptographic
concepts. Binding the operated persona's key (not just names) prevents replaying a
vouch onto a hijacked persona whose key changed outside its chain; accepting any
chain-member key means neither party's routine rotation invalidates the vouch.

**Alternatives considered**: signing the whole profile JSON (breaks on every metadata
edit); storing the attestation in the operator's profile (operated persona's entry is
where readers look, and profiles are self-published — the operator cannot write there);
a detached attestation log topic (new machinery; the directory already exists).

## D3 — Where attestations are created and displayed

**Decision**: Creation is CLI-only: `soulstream profile attest <operated>` (requires a
signer; reads the operated persona's current key via `Lookup`, prints the token).
`profile publish` gains `--attestation <token>`; MCP `publish_profile` drops `kind` and
gains an `attestation` token parameter (agent personas are typically the operated side).
Display: `profile show` prints the claim status and walks the operator chain to the
principal (pure walk `registry.OperatorChain` over profiles from `All`, cycle-guarded).
MCP has no profile-display tool, so no MCP display change.

**Rationale**: FR-010's token flow; the operator is a human at the CLI in every real
scenario we have. Smallest surface that satisfies FR-008 ("every surface that displays
a profile" — today that is `profile show` alone).

## D4 — Bounding inboxes: a second stream, not per-message TTLs

**Decision**: New stream `SOULSTREAM_NOTIFY` capturing exactly
`SOULSTREAM.PERSONA.NOTIFY.>` with `MaxMsgsPerSubject: 100` (the clarified bound),
`Discard: Old`, limits retention, no age expiry, file storage, 2m duplicate window,
and a mandated `MaxBytes` of 64 MiB. The main `SOULSTREAM` stream narrows its capture
to `SOULSTREAM.TOPICS.>`. Publishing is untouched (same subjects — JetStream routes by
subject); `FetchInbox`/`FollowInbox` switch their stream lookup to the notify stream.

**Rationale**: JetStream's `MaxMsgsPerSubject` is exactly the "keep the last N per
subject" primitive — but it is stream-wide, so it can never sit on the same stream as
the never-truncated topic op-logs. A second stream is the NATS-native way to give one
subject family its own retention law. Constitution-mandated evaluation of newer server
features: per-message TTLs (2.11+) express time-based expiry, but the spec bound is
count-based (100 most recent), so TTLs don't fit; subject transforms move subjects but
don't change retention; rollup requires a writer to volunteer compaction and bounds
nothing. `MaxBytes` on the notify stream is coherent (the store is bounded by design)
and incidentally makes provisioning work on account tiers that require an explicit
byte cap (the NGS R1 gotcha from 012) — for this stream only; the main stream's gap
stays deferred. Minimum server version: unchanged (2.12+ per constitution;
`MaxMsgsPerSubject` and purge-by-subject predate it by years).

**Alternatives considered**: `MaxMsgsPerSubject` on the single stream (destroys topic
history — hard no); client-side self-rollup of inboxes (needs every reader's
cooperation, bounds nothing for absent personas); per-message TTLs (time-based, spec
wants count-based).

## D5 — Excluding service traffic and converging existing realms

**Decision**: With the main stream narrowed to `SOULSTREAM.TOPICS.>`, service subjects
(`SOULSTREAM.SVC.>`) fall outside every stream — request/reply becomes genuinely
transient, and the 008 "JetStream pub-ack lands in the reply inbox" gotcha disappears
(the malformed-reply filter stays as harmless defence). Provisioning converges a legacy
realm deliberately: if the existing stream's subjects are exactly the legacy
`["SOULSTREAM.>"]`, it (1) updates ONLY the `Subjects` field to
`["SOULSTREAM.TOPICS.>"]` (preserving every other setting, including any
operator-added `MaxBytes`), (2) creates the notify stream, (3) migrates the newest
≤100 notify messages per persona by republishing them verbatim (headers + data; the
canonical binding derives from the unchanged subject, so signatures still verify),
(4) purges `SOULSTREAM.PERSONA.>` and `SOULSTREAM.SVC.>` residue from the main
stream. Any other non-conformant shape is reported, never touched — same as today.
A new `OutcomeUpdated` joins the provision report vocabulary.

**Rationale**: FR-013 demands convergence; the existing "create-or-report" stance
survives for every shape we don't positively recognise. The ban on
`CreateOrUpdateStream` (blind in-place mutation) stands — this is an explicit,
recognised-legacy-only, single-field update. Migration must run AFTER the narrow (the
notify stream cannot be created while the main stream still claims the overlapping
subject; JetStream forbids overlapping capture) and reads the old messages from the
main stream, where they remain stored after the subject change. The brief window where
neither stream captures a just-published notify is accepted and documented (spec edge
case: mid-provision traffic may be lost; provisioning is a maintenance moment).

**Alternatives considered**: keeping `SOULSTREAM.>` and adding a notify stream
(overlapping subjects — JetStream refuses); subject-transforming SVC into a dead-end
(still stored somewhere or config contortion); leaving legacy realms non-conformant
with a manual runbook (fails FR-013).

## D6 — Release and plugin

**Decision**: Ship as v0.3.0 in the same delivery (tag after merge, as 013 did with
v0.2.0): the MCP `publish_profile` schema changes shape (drops `kind`), and the plugin
wrapper downloads the release matching its own `plugin.json` version — so plugin
0.3.0 + marketplace bump + tag `v0.3.0` must move together, or the wrapper would chase
a release that doesn't exist.

**Rationale**: Established 013 precedent; anything else strands the plugin.
