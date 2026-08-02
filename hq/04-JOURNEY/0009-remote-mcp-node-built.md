# Episode 0009 — The remote MCP node, built: the door that holds nothing (2026-08-02)

**What landed.** Feature 018 turned episode 0008's proven prototype into shipped
code — released **v0.7.0** — in two pieces. First, the tool surface every host
serves became public: `internal/mcpserver` promoted to `mcpserver`, its
`NewServer(c *realm.Client)` already the per-session factory (the handlers close
over exactly one realm client), plus a `WithKeyring` option so a multi-tenant host
stops sharing one on-disk pins file, plus a new `soulstream_whoami` (24 tools) so a
remote user can read who the edge decided they are. This is the "public MCP surface"
[SoulNode](https://github.com/impire-io/soulnode) held its Phase-2 front door for —
its fourth upstream ask, delivered. Second, the node itself: a nested consumer module
`node/` (own `go.mod`, `replace soulstream => ../`, soulidentity by pseudo-version)
serving that surface over streamable HTTP — bearer passthrough onto per-principal
pooled NATS connections that SoulIdentity's auth callout admits, principals asserted
server-side from `$SYS.REQ.USER.INFO`, signing delegated through the 017 seam
(`PersonaSigner`), and the RFC 9728 + 401 OAuth resource edge.

**The open 018 decision, resolved: external OIDC only** [judgment, maintainer's
call]. Episode 0008 left "which authorization server" as the build's first question —
a node-embedded mini-AS (single-persona bridge) versus an external OIDC AS
(multi-user passthrough). The maintainer chose external-only: the node is never an
authorization server, it points at one and stays agnostic about which, with
[soulfold](https://github.com/impire-io/soulfold) the intended default. What makes
that real is that the AS-facing contract — discovery, DCR, PKCE, and the exact claims
(`oid` a legal slug that *becomes* the persona, exactly-one declared `roles`, RS256,
fixed audience) — is written down and **proven to be the interface**: the module's
test rig admits a scripted client through the full flow against a stand-in built from
that document alone (SC-005). The contract, not any particular server, is the seam.

**The one thing we changed from the prototype: the trust model** [measured]. The
research node stored each presented bearer on its pool entry *before* checking
admission. Under a shared multi-user door that is a denial-of-service hole: a forger
crafting a JWT with a victim's `iss`+`oid` routes to the victim's entry and overwrites
their freshest token with garbage, breaking the victim's next reconnect. The shipped
node (research decision R4) lets a bearer influence an entry only *after* it has been
admitted **for that entry's principal** — via the build, a prior admission, a
bound-session refresh, or a short-lived candidate probe whose server-asserted
principal must match. The adversarial suite fires forged tokens carrying the victim's
exact routing hint (bad signature, no role, garbage) and confirms zero adoption, zero
eviction, and an uninterrupted victim session. Routing hints route; they never
authorize.

**Measured, all five stories, on an in-process edge.** The rig runs the *real*
admission edge — soulidentity's public `embed.Run` (vault + callout issuer) on an
embedded operator-mode nats-server — so these are not mock assertions: a scripted
no-install client joins and its turn verifies as its persona on an independent reader
that trusts nothing the node said (US1); two principals interleave with correct
attribution, the node writes zero files under a redirected HOME, and no token value or
JWT payload appears in any log line (US2); a session survives more than three callout
TTLs by reconnect re-admission and a revoked token refuses within the window,
non-stickily (US3); the full DCR+PKCE flow admits and all seven contract violations
refuse (US4); a killed node's client resumes on a fresh instance with only a
re-presented bearer, and a fronted `Host` is served (US5). One rig bug taught its own
lesson: RS256 tokens silently failed the whole OIDC lane until the stand-in's signer
passed `crypto.SHA256` (not hash `0`) so the PKCS#1 DigestInfo prefix was present —
the same class of in-band-failure trap episode 0008 flagged, caught here by an
end-to-end verify, not a transport check. A second architectural fact, worth banking:
node-authored content verifies from the identity plane's `keys.public`, **not** the
soulstream profile registry — the node publishes no profile, so the reader keyring is
built from the vault directory.

**What it opened.** The cycle guard now has a running test (neither core repo imports
the other or the node; `mcpserver` imports no node/soulidentity code). CI and release
are wired but **dormant** — the node module fetches the private soulidentity module,
so its jobs are guarded on `NODE_CI_ENABLED` + a read credential and stay skipped
(green) until an operator flips them, keeping the core release self-contained.
`byon-setup` and `probe` carried over as best-effort operator tooling (spec Q2): build
from source, not in the release archive. Still on SoulIdentity's plate, unchanged from
0008: a persona-presentation surface beside `keys.public`, since OIDC-lane personas
are oids and human display names live only in the audit.

Reversal condition: the external-AS-only call rests on a measurement — Claude
Desktop's hosted connector dialog is OAuth-only (0008) — so it reopens if a hosted
client grows a static-header lane (observable: a no-install client admitting through
the `sit_` passthrough with no OAuth), which would restore the embedded-bridge option
as the simpler personal-node path. The credential-free / AS-agnostic property reopens
if soulidentity ever special-cases an issuer or the node ever validates a token itself
(observable: an issuer-aware branch in the node, or the node reading JWKS) — either
makes the node a trust holder and re-opens the custody model. Neither is measured
today.

Trail: [`../02-DESIGN/extensions/remote-mcp-node.md`](../02-DESIGN/extensions/remote-mcp-node.md),
[`../03-IMPLEMENTATION/ROADMAP.md`](../03-IMPLEMENTATION/ROADMAP.md) (018),
[`../../specs/018-remote-mcp-node/`](../../specs/018-remote-mcp-node/) (spec, plan,
research R1–R11, contracts/{library,authorization-server,http}, data-model, quickstart,
tasks), [`../../docs/mcp-remote.md`](../../docs/mcp-remote.md),
[`../../docs/mcp.md`](../../docs/mcp.md); the `node/` module; episode
[0008](0008-remote-mcp-node.md) (the research proof) and the prototype at pre-graduation
commit `56c7a2e`; SoulIdentity journey 0019 (soulfold, the sibling IdP). Feature branch
`018-remote-mcp-node`, released **v0.7.0**.
