# Data Model: Foundation — Realm Provisioning & the Operation Record

**Feature**: 001-foundation | **Date**: 2026-07-12 | **Source**: [spec.md](./spec.md)

This feature has no persisted domain database — its "data" is the shape of what travels on
the wire and the shape of what the library reports. The entities below are value types
(immutable once constructed) plus one configuration type and one result type.

## Entity: Record (the operation record)

The unit of everything. Carried on the wire as NATS message headers + a pure-data payload.

| Field | Type | Wire header | Canonical key | Rules |
|---|---|---|---|---|
| ID | string (UUIDv4) | `Nats-Msg-Id` | `id` | Required. Lowercase hyphenated `8-4-4-4-12`. Doubles as dedup identity (FR-012/013). |
| Version | int | `Soulstream-Version` | `v` | Required. Must equal `1`; any other value rejected on parse (FR-016). |
| Author | string (persona name) | `Soulstream-Author` | `author` | Required. Valid persona slug (FR-024). |
| Parents | []string (op-IDs) | `Soulstream-Parents` | `parents` | Ordered. **Absent header ⇔ empty slice** (FR-015). Comma-separated on the wire. Each entry a UUIDv4. |
| Type | string | `Soulstream-Type` | `type` | Required. Non-empty. Namespaced token (e.g. `turn.post`); this feature does not enumerate types. |
| Timestamp | time (RFC 3339) | `Soulstream-Ts` | `ts` | Required. Author-claimed, informational; never an ordering authority (FR-018). |
| Signature | string (optional) | `Soulstream-Sig` | `sig` (omitted if empty) | Optional. Carried through untouched; never produced/verified here (FR-023). |
| Payload | []byte (pure data) | message body | `data` (parsed JSON value) | Text/references only; no blobs (FR-026). Opaque bytes on the wire; a JSON value in the canonical record. |

Derived / bound at canonicalisation time (not stored on the wire message, supplied by context):

| Field | Type | Canonical key | Rules |
|---|---|---|---|
| Realm | string (slug) | `realm` | From client config (FR-028). Bound into canonical record (FR-022). |
| Topic | string (topic-path) | `topic` | The ops/info subject's topic path. Bound into canonical record (FR-022). |

**Unknown `Soulstream-*` headers**: preserved verbatim on parse, never stripped (FR-017). Stored
alongside the known fields as an ordered extras map so a round-trip re-emits them.

**State**: a Record is immutable after construction. There are no lifecycle transitions in this
feature (lifecycle is a later feature). Validity is binary: a Record either parses/validates or
is rejected with a field-specific error.

## Entity: CanonicalRecord

The deterministic, field-order-independent serialisation of a Record used for future signing
input, exported exhibits, and sealed-topic inner ops (FR-019).

- Logical shape (object with keys `v, realm, topic, id, author, parents, ts, type, data`, plus
  `sig` when present), serialised then canonicalised per RFC 8785 (JCS).
- **Determinism rule**: two Records equal in content produce byte-identical canonical output
  regardless of source field order (FR-020, SC-004).
- **Losslessness rule**: wire ⇆ canonical maps each field exactly once, both directions (FR-021).
- `parents` serialises as a JSON array (empty array when there are no parents — distinct from the
  wire's *absent header*, but semantically the same empty set; the mapping is defined once).

## Entity: PersonaName (validated slug)

- Grammar: `^[a-z0-9]+(-[a-z0-9]+)*$`, length 1–64 (FR-024).
- Rejections carry a reason (uppercase, dot, whitespace, wildcard `*`/`>`, leading/trailing/double
  hyphen, empty, too long).
- The same grammar validates **realm names** and **topic-id slugs**.

## Entity: RealmSpec (mandated provisioning target)

The fixed, non-configurable shape a realm must have. Not user-tunable (Smallest Viable — no
speculative options).

| Artefact | Identity | Mandated settings |
|---|---|---|
| Stream | name `SOULSTREAM` | Subjects `SOULSTREAM.>`; Retention = Limits; **no age-based expiry**; subject rollup permitted; duplicate window ≥ 2 min; disk storage. |
| Object store | bucket `soulstream-objects` | Exists. (Usage vocabulary is a later feature.) |

## Entity: ProvisionReport (structured provisioning result)

Returned by a provisioning run (FR-009). Per artefact:

| Field | Type | Meaning |
|---|---|---|
| Artefact | enum {stream, object_store} | Which artefact. |
| Outcome | enum {created, conformant, nonconformant} | created = was missing, now made; conformant = already present & correct; nonconformant = present but drifts. |
| Nonconformities | []string | Present only when `nonconformant`; each names one specific drift (e.g. "MaxAge is set", "AllowRollup disabled", "storage is memory not file"). Never mutated in place (FR-008). |

The overall run succeeds even when an artefact is `nonconformant` — the report is informational;
the operator decides. The run fails only on connection/JetStream errors (FR-002).

## Entity: ClientConfig

The required inputs to construct the library client.

| Field | Type | Rules |
|---|---|---|
| ContextName | string | Named NATS context to connect from (FR-001). Missing context → fail fast (FR-002). |
| Realm | string (slug) | Required; validated (FR-028); bound into every canonical record. |
| Persona | string (slug), optional | The persona this client publishes as; when set, write-side attribution is enforced against it (FR-025). Read-only clients may omit it. |

## Relationships

- A **ClientConfig** connects to one **Realm** (one NATS account) holding one **Stream** and one
  **Object store**.
- A **Record** is authored by one **PersonaName**, references zero-or-more parent Records by ID,
  and canonicalises to one **CanonicalRecord** bound to its Realm + Topic.
- **Provisioning** reads/creates the Realm's artefacts and returns one **ProvisionReport**.
