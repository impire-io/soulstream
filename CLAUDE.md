<!-- SPECKIT START -->
Active feature: **014-persona-accountability** — remove persona `kind` outright
(no backwards compat; strict profile decode rejects unknown fields naming persona +
field; bulk reads skip + warn, direct reads fail), replace the taxonomy with
verifiable accountability: `operated_by` may carry an operator COUNTERSIGNATURE —
statement `identity.AttestationBytes(operator, operated, operatedKeyB64)` =
`"soulstream-operator-attestation\n"+…`, signed with the operator's existing Ed25519
key, valid against ANY key in the operator's validated chain; bound operated-key must
be "" or ∈ operated persona's chain. Status attested/unverified/failed; operator
distrusted ⇒ failed. Portable token (base64 JSON {operator,operated,operated_key,sig}):
CLI `profile attest <persona>` generates, `profile publish --attestation` includes;
profiles stay self-published. `profile show` prints status + operator chain to its
terminal (principal/dangling/cycle). MCP publish_profile: -kind +attestation. Stream
hygiene: main stream narrows to `SOULSTREAM.TOPICS.>` (SVC.> captured by NOTHING —
008 pub-ack gotcha gone); NEW stream `SOULSTREAM_NOTIFY` on
`SOULSTREAM.PERSONA.NOTIFY.>` with MaxMsgsPerSubject 100, DiscardOld, MaxBytes 64MiB
(NGS R1-safe); FetchInbox/FollowInbox read it. Provision converges ONLY the exact
legacy shape (subjects ["SOULSTREAM.>"]): narrow subjects → create notify stream →
migrate newest ≤100 notifies/persona verbatim (same subjects ⇒ sigs still verify) →
purge PERSONA.>/SVC.> residue; OutcomeUpdated. Ships v0.3.0 + plugin/marketplace
bumps in the same delivery (wrapper downloads by its own plugin.json version).

For details read: [specs/014-persona-accountability/plan.md](specs/014-persona-accountability/plan.md)
(spec: `specs/014-persona-accountability/spec.md`, research decisions:
`research.md` D1–D6, contract: `contracts/library.md`, model: `data-model.md`).
Done: `001`–`005` (MVP), `006-signing`, `007-rollup`, `008-discover`, `009-curator`,
`010-work`, `011-vocab`, `012-distribution` (v0.1.0), `013-config` (per-project
`.soulstream.json` + self-installing plugin, v0.2.0) merged + pushed.

Project conventions:
- Go 1.26; module `github.com/impire-io/soulstream`.
- `record` and `identity` import NO NATS; `realm`, `topic`, `registry`, `curator` are NATS-touching.
- Keep pure logic (folds, chain validation, discovery match/merge, curator judgment) separate
  from NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; modern `nats.go/jetstream` API.
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
- Push to origin after merging a completed feature to main.
<!-- SPECKIT END -->
