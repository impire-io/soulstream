# Episode 0003 — First boot is real: init and up land (2026-08-02)

SoulNode's first code shipped M1.1 through the spec-kit flow
(`specs/001-init-and-up/`): `soulnode init` founds a realm — the complete
ceremony of design 0001 §4 generated and persisted into one state
directory, the founding acts performed through soulidentity's public
`client` over the node's own loopback connection, the first token printed
exactly once — and `soulnode up` runs the composition: embedded
operator-mode server (pure `server.Options`, resolver preloaded from the
persisted account JWTs, JetStream in the state dir, loopback bind) with
the identity plane inside the process via the public `embed.Run`, every
plane on an ordinary loopback connection (constitution III as ratified).

The honest numbers [measured]: the real binary founds a realm — transient
boot, vault sealed, sentinel and token minted, 17 artifacts on disk with
owner-only modes — in **0.15 s** (SC-001's budget was a minute); the
end-to-end gate (found → admit with server-asserted persona confined to
its own prefix → garbage refused with `callout REFUSED` audited → revoked
refused → stop → restart on the same state) runs in ~1 s inside
`make test`; re-`init` is a verified no-op reporting 17 artifacts and
never re-mints; the whole quality gate is green.

Three packages hold the shape the plan drew: `ceremony` (pure — generate,
persist 0700/0600, load, verify with a named-first-failure damage matrix;
`sentinel.creds` written last as the founding-complete marker), `node`
(the composition: server, three bypass-lane connections, identity plane,
founding acts), `cmd/soulnode` (flags, env, signals, printing). The
dependency story is constitution I with one tracked exception:
soulidentity rides a pseudo-version pin of its main until upstream tags —
no `replace` anywhere, flip condition recorded in the plan's Complexity
Tracking.

Refuted along the way: "refuse any directory with group/other bits" — the
first test run showed `t.TempDir()` (and any user's umask-made `mkdir`)
hands out 0755, so a refusal there is hostile, not safe. The behavior
became *tighten then verify*: `init` chmods the directories it owns to
0700 and refuses only when the filesystem will not hold the mode
[measured, the failing run and the green one after].

Opened: M1.2 (the realm joins — provisioning + the archivist plane) on
the same composition seam; the config schema gains its per-plane blocks
there, as the design's growth path prescribes.

Reversal condition: none — records a completed build against measured
acceptance criteria (the pseudo-version pin carries its own flip
condition in the feature plan; the founding-persona name "owner" is a
deliberate default awaiting a consumer's chafe, not a decision to
reverse).

Trail: `specs/001-init-and-up/` (spec, plan, research R1–R8, data model,
contracts, tasks all checked); `ceremony/`, `node/`, `cmd/soulnode/`;
design 0001 §§3–6/§9-M1.1. Commits: the `001-init-and-up` branch, merged
to main 2026-08-02.
