# Data Model: The Remote MCP Node

The node holds **no durable data**. Everything below is in-memory, rebuilt
from nothing on restart (FR-002). The model's job is the trust boundary:
which bearer may influence which connection.

## Entities

### Node

The process. Fields (all from config, R8): `Listen`, `PublicURL`,
`AuthIssuer`, `Realm`, `NATSURL`, `SentinelPath`, `Prefix`. Owns one `Pool`
and the HTTP handler assembly. No identity of its own beyond the optional
sentinel credential (deny-all, routes connections to the callout).

### Pool

`map[hint]*entry` + lock. The **hint** is routing-only (R3): `oidc:<iss>:<oid|sub>`
peeked unverified from a JWT-shaped bearer, else `tok:<token>`. A hint
collision or forgery routes; it never authorizes (FR-004).

### Entry (one per admitted principal)

| Field | Meaning |
|---|---|
| `latest` | the freshest **admitted-path** bearer (see Bearer lifecycle) — what `nats.TokenHandler` presents on every (re)connect |
| `nc` | the pooled NATS connection (ReconnectWait 200ms, MaxReconnects -1, optional sentinel creds) |
| `principal` | server-asserted `(account, persona)` from `$SYS.REQ.USER.INFO` after admission — never client-claimed |
| `rc` | `realm.Client` wired with `PersonaSigner(persona)` (017 seam; key stays in the vault) |
| `srv` | the per-principal `mcpserver.NewServer(rc, WithKeyring(...))` |
| `sessions` | the set of MCP session ids bound to this entry |
| `err` | build failure, marks the entry a corpse |

**Entry states**: `absent → building → live → corpse → absent`

- `absent → building`: first request for a hint; the presented bearer is the
  build token. The build IS the admission probe — a garbage token fails the
  build and hurts only its presenter.
- `building → live`: NATS connect admitted + principal derived + signer +
  realm client + server constructed.
- `building/live → corpse`: build error, or `nc.IsClosed()` (nats.go gives
  up after consecutive auth violations — revoked/expired badge).
- `corpse → absent`: evicted on next touch; the next authenticated request
  rebuilds (refusals are non-sticky, US3).

### Session binding

`session id → entry`, established when `getServer` admits the session
(entry was built by, or probe-verified for, that session's bearer).
Invariant (FR-005): **a session is only ever served by the entry it is bound
to, and only a bound session's requests may update that entry's `latest`.**

### Bearer lifecycle (the R4 state machine)

```
presented ──(no entry)──────────────► build token ──admitted──► latest
    │                                        └─refused──► 401, entry evicted
    ├─(bound session, same entry)──────────► latest        (refresh path)
    ├─(unbound, == entry.latest)───────────► bind session  (same badge)
    └─(unbound, differs, entry live)──► PROBE connect
                 ├─ admitted ∧ principal == entry.principal ► adopt as latest + bind
                 ├─ admitted ∧ principal ≠ entry.principal  ► serve via OWN entry (rekey), victim untouched
                 └─ refused ────────────────────────────────► 401, entry untouched
```

The probe is a short-lived NATS connection dialed with the candidate bearer;
it exists so adoption is always evidence-backed. Forged-hint outcomes
[test-asserted]: zero adoption, zero eviction, zero displacement of the
imitated principal (SC-002).

### Principal

`(account, persona)` parsed from the resolved publish permissions returned
by `$SYS.REQ.USER.INFO` — the 5-part subject
`<prefix>.<account>.<persona>.sign.record` names it. Uniqueness: one entry
per principal in steady state (hint collisions converge on probe/rekey).
`persona` is the realm identity; legality is the AS's conformance burden
(`oid` must be a legal slug — contract).

### Protected-resource metadata (served, not stored)

RFC 9728 document, computed from config:
`{"resource": PublicURL, "authorization_servers": [AuthIssuer],
"bearer_methods_supported": ["header"]}`. Exists only when `PublicURL` is
set; its absence means local/no-OAuth mode (400s, prototype Bars-1-3
behavior).

## Validation rules

- Config: `PublicURL` set ⇒ `AuthIssuer` required (metadata must name the
  AS); `Realm` and `NATSURL` always required; teaching errors name the flag.
- Startup: realm stream must exist (read-only check; the node never
  provisions — R9).
- Every entry invariant above is test-asserted; the log audit (SC-006)
  asserts no bearer value ever appears in any log line, error, or metadata.

## Relationships to existing models

- `realm.Config{Realm, Persona, Signer}` — consumed per entry; typed-nil
  signer guard (017) still applies: the signer is assigned only when the
  `PersonaSigner` construction succeeded.
- `mcpserver` handlers — unchanged; they see one `realm.Client` and cannot
  tell the node from the stdio adapter (FR-001: reach, not capability).
- Identity plane objects (vault keys, token records, minted user JWTs,
  scope templates) — soulidentity's model; the node observes only their
  effects (admission, permissions, signatures).
