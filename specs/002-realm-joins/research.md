# Research — 002-realm-joins

## R1 — Realm name: `home` by default, founded into config

- **Decision**: `config.json` gains `"realm"`; default `home`; `init
  --realm` sets it on the founding run only (same file-is-configuration
  rule as `--listen`).
- **Rationale**: a personal node's realm is its home; the name is
  load-bearing on every stream subject, so it belongs to the founding,
  not to a flag on every `up`.
- **Alternatives**: hostname-derived (surprising, unstable); mandatory
  flag (violates two-commands-zero-decisions first boot).

## R2 — Provisioning: `ProvisionOn` at founding, guarded again at `up`

- **Decision**: the founding acts run `realm.ProvisionOn(ctx, js)` (the
  connection-decoupled create-or-report surface); `up` runs it again
  before starting the memory plane.
- **Rationale**: founding owns creation (FR-001); the `up` guard makes
  the keeper's "is the realm provisioned?" failure structurally
  unreachable and gives FR-001 its create-or-verify semantics on every
  path. `ProvisionOn` avoids `realm.NewClient`'s connection-ownership
  semantics for a transient act.
- **Alternatives**: provision on first `up` only (moves a founding act
  out of founding); `realm.NewClient(...).Provision` (takes ownership of
  a connection the founding acts still need).

## R3 — The archivist's credential is ceremony; its persona key is not

- **Decision**: ceremony gains the `archivist` bypass-lane user
  (`users/archivist.creds`, inventory 17→18 files + sentinel). Its
  *persona signing key* is deliberately absent from disk: the plane
  builds `client.PersonaSigner("archivist")` at startup and the key
  materializes in the identity-plane vault on first touch (upstream
  D26). Pre-release, the inventory change is wholesale — no migration
  shim for M1.1-era directories (none exist outside tests).
- **Rationale**: transport credential vs persona custody are different
  things (the ecosystem's own split); SC-003 is the observable.

## R4 — The plane block lands with its first plane

- **Decision**: `config.json` becomes
  `{"listen", "realm", "planes": {"memory": {"enabled": true}}}`.
  Only `enabled` is honored in M1.2; URL/creds keep their loopback +
  state-dir defaults implicitly (design §2 notes the full block; fields
  land when a consumer can point them elsewhere).
- **Rationale**: design §2 requires the shape; smallest honest instance
  of it.

## R5 — Plane wiring and failure semantics

- **Decision**: after the identity plane is ready (its signer surface
  must answer before `PersonaSigner` construction), the node opens the
  archivist connection, builds the signer, wraps the connection in
  `realm.NewClient` (the plane owns that connection; `Close` on stop),
  opens `archive.Open(<state>/archive)`, then runs `keeper.Run` and
  `topic.RespondMemory` under the node ctx. Startup errors abort
  `Start` with the plane named; runtime exits of either loop surface on
  the node's error channel and are logged loud at `Stop` (FR-006).
  `OnServed` (3-arg) logs to the audit stream.
- **Rationale**: design §6's wiring, the upstream OnServed lesson
  already applied, constitution III's fail-loud rule.

## R6 — The e2e proves the *owner's* path, not a bypass lane

- **Decision**: the M1.2 test drives everything a real owner does:
  sentinel + token admission → `client.PersonaSigner("owner")` →
  `realm.NewClient` on the admitted connection → start topic, post turn
  → `topic.MemoryQuery` → assert a citation to the posted op; then
  restart continuity (archive holds each op exactly once — counted via
  the public `archive.Open` on the stopped node) and the
  `enabled: false` arm (M1.1 admission e2e still green, no `archive/`
  created).
- **Rationale**: SC-001/002/004 verbatim; the scoped template the
  ceremony grants is exactly what this path exercises — if the template
  is wrong, this test is what catches it.
