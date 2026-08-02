# Episode 0004 — The realm remembers: provisioning and the memory plane land (2026-08-02)

M1.2 shipped through the spec-kit flow (`specs/002-realm-joins/`): `init`
now founds the record substrate too (`realm.ProvisionOn` joins the
founding acts, create-or-report; `up` re-guards it so the keeper's
"is the realm provisioned?" failure is structurally unreachable), the
ceremony gained the archivist's transport credential and the realm name
(`config.json`: `listen`, `realm` — default `home`, fixed at founding —
and the first design-§2 plane block, `planes.memory.enabled`), and the
node gained the memory plane: the archivist's public `keeper`/`archive`
packages on a realm client whose signer is the identity plane's
`PersonaSigner("archivist")` — the persona key materialized in the vault
on first touch, nothing persona-shaped on disk [measured, SC-003's
assertion].

The e2e is the owner's real path, not a bypass lane [measured, rides
`make test` in ~5 s]: sentinel + token admission → `PersonaSigner("owner")`
→ a topic started and a turn posted on the admitted connection → the
archivist keeps it (the kept exhibit's author reads `owner`) → a memory
query over the realm's own transport returns the archivist's attributed
answer citing the turn. Restart continuity measured exactly-once: the
archive grows by precisely the ops posted after the restart, nothing
re-captured. The disabled arm holds: `planes.memory.enabled: false` runs
admission exactly as M1.1 and creates no archive directory.

The dependency ledger: soulstream enters at its real tag (v0.6.0); the
archivist joins soulidentity in the tracked pseudo-version class (its
public seam landed after its only tag) — two pins waiting on upstream
release acts, no `replace` anywhere.

Refuted/reversed: nothing this time — the upstream seams delivered on
2026-08-01 fit exactly as the composition research said they would; the
plane went in without an upstream change of any kind.

Opened: M1.3 — an agent runs (the runtime plane): soulrealm's public
runner/backend/minter on a declared workload, and with it the known
design question of the workload signing key (the realm account needs a
*plain* signing key for the minter beside the scoped one — scoped keys
reject carried permissions, soulrealm's fleet research measured it).

Reversal condition: none — records a completed build against measured
acceptance criteria (the realm-name default `home` and the plane-block
shape are recorded as-built in design 0001; the pseudo-version pins keep
their flip conditions in the feature plans).

Trail: `specs/002-realm-joins/`; `ceremony/`, `node/node.go` (the plane),
`node/found.go` (ProvisionOn), `cmd/soulnode`; design 0001 §§2/4/6
propagated. Commits: the `002-realm-joins` branch, merged to main
2026-08-02.
