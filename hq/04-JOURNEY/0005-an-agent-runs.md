# Episode 0005 — An agent runs: Phase 1 is complete (2026-08-02)

M1.3 shipped through the spec-kit flow (`specs/003-an-agent-runs/`), and
with it every Phase 1 exit criterion of design 0001 §9 is met. The
ceremony gained the **workload minting key** — a *plain* realm signing
key beside the scoped one, because the runtime's minter embeds
per-workload permissions in the user JWTs it signs and a scoped key
rejects them (the two-keys split soulrealm's fleet research measured) —
plus the `runner` transport credential (inventory now 20 artifacts).
`soulnode workload start <declaration>` is the runtime plane,
invocation-scoped by design: it composes upstream's public surfaces
(declaration, minter, native backend, runner) on a realm client speaking
as persona `runner`, signing through the identity plane. The claim-race
node supervisor stays upstream with Fleet — composition, not invention.

The e2e is soulrealm's own founding proof re-run inside the composition
[measured, ~1 s of a 5.5 s suite]: upstream's `agent-echo`, built from
the module cache and declared unchanged, launches with a minted
TTL-bounded credential, posts "hello from echo" — the turn authored by
`echo` on the topic — while the lifecycle lands as a completed work item
authored and owned by `runner`; the archivist keeps all of it; after end
of life no credential of any kind lingers in scratch. The refusal paths
are named and fast: node down (points at `soulnode up`), bad
declaration, missing artifact.

**The composition caught an upstream bug — the loop working as
designed.** Under the enforcing operator-mode server, agent-echo's own
`realm.NewClient` refused to construct: the availability probe
(`js.AccountInfo`) publishes to `$JS.API.INFO`, which the minted agent
permission set never granted — invisible to upstream's open-server
suites, measured here on the first enforcing run [measured]. The fix
landed upstream first (soulrealm `3fee11f`: one subject in the agent
allow-list, tools untouched), the pin bumped, the e2e went green. One
test assumption was also refuted openly: scratch does *not* retain the
workload's creds file after end of life — upstream's backend removes the
scratch dir, which is better hygiene than the assertion assumed; the
assertion now checks that nothing credential-shaped lingers.

Phase 1, in whole [measured]: `soulnode init` founds a realm in ~0.15 s
(20 artifacts, owner-only modes, one token printed once); `soulnode up`
runs server + identity + memory planes on ordinary loopback connections;
the owner posts through admission and memory answers with citations; a
declared agent runs and the realm remembers it. Three pseudo-version
pins await upstream tags (soulidentity, archivist, soulrealm) — the one
standing constitution-I exception, flip conditions recorded per plan.

Reversal condition: none — records completed builds against measured
acceptance criteria (the invocation-scoped runtime keeps its named
growth path: upstream's Fleet supervisor, consumed when it lands).

Trail: `specs/003-an-agent-runs/`; `ceremony/`, `node/workload.go`,
`cmd/soulnode`; soulrealm commit `3fee11f` (the consumer-proven fix);
design 0001 §§4/6 propagated. Commits: the `003-an-agent-runs` branch,
merged to main 2026-08-02.
