# Data Model: Persona Accountability & Stream Hygiene

## Profile (registry) — updated

KV value in `soulstream-personas`, keyed by persona name. Decoded STRICTLY
(unknown field ⇒ invalid, error names persona + field).

| Field | JSON | Type | Rules |
|---|---|---|---|
| Name | `name` | string | slug; must equal the KV key |
| DisplayName | `display_name,omitempty` | string | presentation only |
| Description | `description,omitempty` | string | free-form self-presentation (replaces any use `kind` had) |
| OperatedBy | `operated_by,omitempty` | string | slug; MUST NOT equal `name`; audit fact, never a permission |
| OperatorAttestation | `operator_attestation,omitempty` | *OperatorAttestation | only valid when `operated_by` is set |
| CreatedAt | `created_at` | time | set on first publish, preserved on update |
| SigningKey | `signing_key,omitempty` | *SigningKeyInfo | unchanged (006) |
| Rotations | `rotations,omitempty` | []Rotation | unchanged (006) |

**REMOVED**: `kind` (and constants `KindHuman`/`KindAgent`/`KindService`). A stored
profile still carrying `kind` is invalid (strict decode).

## OperatorAttestation (registry) — new

Stored inside the operated persona's profile. Operator/operated names live on the same
profile (`operated_by` / `name`) and are not duplicated here.

| Field | JSON | Type | Rules |
|---|---|---|---|
| OperatedKey | `operated_key,omitempty` | string | base64 Ed25519 key the operator saw; `""` = operated persona had no key |
| Sig | `sig` | string | operator's base64 Ed25519 signature over the attestation statement |

**Statement** (identity, pure): `identity.AttestationBytes(operator, operated, operatedKeyB64)` =
`"soulstream-operator-attestation\n" + operator + "\n" + operated + "\n" + operatedKeyB64`.

## Attestation token — new (transport-only, never stored as-is)

base64(JSON): `{"operator": slug, "operated": slug, "operated_key": b64-or-empty, "sig": b64}`.
Produced by `registry.NewAttestationToken`, consumed by `registry.ParseAttestationToken`
at `profile publish` time; `operator` must equal the profile's `operated_by`, `operated`
must equal the publishing persona.

## Claim status — derived, never stored

`registry.AttestationStatus(p Profile, operatorChain []string, operatorDistrusted bool, operatedChain []string) string`

| Status | When |
|---|---|
| *(none)* | `operated_by` empty — no claim exists |
| `unverified` | claim present but no attestation, OR operator has no validated chain |
| `attested` | sig verifies against ANY operator-chain key AND (`operated_key` is `""` OR ∈ operated persona's own chain) |
| `failed` | sig present but verifies against no operator-chain key; or bound key ∉ operated chain; or operator distrusted |

## Operator chain — derived, never stored

`registry.OperatorChain(profiles map[string]Profile, start string) (links []string, terminal ChainTerminal)`

Walk `operated_by` links with a visited set. Terminals: `principal` (persona with no
claim), `dangling` (named operator absent from directory), `cycle` (self-reference or
loop), `invalid` (a profile on the chain failed strict decode — surfaced by the caller
that loaded profiles). Individual profiles stay readable regardless of terminal.

## Bulk-read warnings (registry) — new

`All` returns `([]Profile, []ProfileWarning, error)`;
`ProfileWarning{Persona string; Err error}` — entries skipped by strict decode.
Callers warn loudly (CLI stderr); keyrings simply never learn the skipped persona
(signatures → unknown-key).

## Realm retention shape (realm) — updated

| Stream | Name | Subjects | Retention law |
|---|---|---|---|
| Op-log | `SOULSTREAM` | `SOULSTREAM.TOPICS.>` | limits, MaxAge 0, AllowRollup, file, dupes ≥2m — unchanged otherwise |
| Inboxes | `SOULSTREAM_NOTIFY` | `SOULSTREAM.PERSONA.NOTIFY.>` | limits, MaxAge 0, **MaxMsgsPerSubject 100**, Discard Old, **MaxBytes 64 MiB**, file, dupes ≥2m |

`SOULSTREAM.SVC.>` is captured by neither stream (transient by construction).
KV `soulstream-personas` and object store `soulstream-objects` unchanged.

New constants: `realm.NotifyStreamName`, `realm.NotifyStreamSubject`,
`realm.InboxWindow = 100`, `realm.NotifyMaxBytes`. `realm.StreamSubject` becomes
`"SOULSTREAM.TOPICS.>"` (legacy value `"SOULSTREAM.>"` recognised only by the
convergence path).

## Provision outcomes (realm) — updated

`OutcomeUpdated` added: reported when the recognised legacy stream shape
(`subjects == ["SOULSTREAM.>"]`) was converged (subjects narrowed, notify stream
created, ≤100 newest notifications per persona migrated verbatim, `PERSONA.>` and
`SVC.>` residue purged). Every other divergent shape stays report-only.
