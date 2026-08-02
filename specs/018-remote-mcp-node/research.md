# Phase 0 Research: The Remote MCP Node

Sources: the graduated design (`hq/02-DESIGN/extensions/remote-mcp-node.md`),
the research prototype excavated at `56c7a2e`
(`hq/01-RESEARCH/remote-mcp-node/experiment/`), the current tree's
`internal/mcpserver` + `realm` surfaces, the soulidentity checkout at
`5eaf52c` (client, callout, embed), and SoulNode's design 0001 §8 /
roadmap Phase 2 (the downstream consumer of the public MCP surface).

## R1 — Module layout: nested consumer module `node/`

**Decision**: the node is a nested Go module `github.com/impire-io/soulstream/node`
at repo path `node/`, with a **committed** `replace github.com/impire-io/soulstream => ../`
(the node always builds from a repo checkout; it is not `go install @latest`-able,
documented). soulidentity is pinned by **pseudo-version** (no tags exist on that
repo — verified) with `GOPRIVATE=github.com/impire-io` for fetch; local dev may
use an uncommitted `go.work`. **Operational follow-up (Daan)**: CI needs
credentials for the private soulidentity module (org PAT secret or deploy key)
before the node module can build in GitHub Actions; local `make check` needs none.

**Rationale**: FR-014's cycle guard at module granularity — if the node were a
package of the core module, soulstream's `go.mod` would gain a soulidentity
dependency. The prototype proved consumer position works: it reached soulstream
via **public packages only** (`realm`, `topic`, `identity`) [measured at
56c7a2e]. soulidentity's `client` is import-clean (deps: nats.go, nkeys, nuid
only — no server-side deps) [measured].

**Alternatives**: separate repo (like the archivist) — rejected: the design
says the build lives here, and the node co-evolves with the mcpserver
promotion this feature makes; a repo split can happen later without protocol
consequence. Waiting for a soulidentity tag — rejected: pseudo-version is
exact and reproducible; tagging is soulidentity's call.

## R2 — The public MCP surface: promote `internal/mcpserver` → `mcpserver`

**Decision**: `git mv internal/mcpserver mcpserver` (public), keeping
`NewServer(c *realm.Client) *mcp.Server` — the signature already IS the
per-session factory (all 23 tool handlers close over exactly one
`*realm.Client`; persona/signer/realm are read off it) [measured]. Add the one
substantive change: a functional option `WithKeyring(func(context.Context)
(*identity.Keyring, error))` so the reader-verification cache is injectable —
today `keyring()` reads/writes the per-realm **pins file on the local
filesystem**, which is the wrong shape for a multi-principal node. Default
(no option) keeps today's file-backed behavior byte-for-byte, so
`cmd/soulstream-mcp` changes one import line and nothing else. Tests move
with the package (they are in-package and drive handlers directly); the
promoted package may still import `internal/keystore`/`internal/version`
(Go's internal rule keys off the importing *file's* module, which stays
soulstream). `internal/natstest` stays internal.

**Rationale**: FR-015 / SoulNode's fourth upstream ask ("soulstream's public
MCP surface", deliberately held for this feature). Smallest viable: a
visibility change plus one injection point with a concrete need — not a new
abstraction layer.

**Alternatives**: a thin public registration package over a caller-provided
`*mcp.Server` — rejected: more surface, duplicated registration, and every
known consumer wants the whole surface. Reimplementing tools in the node
(what the prototype did with 4 hand-rolled tools) — rejected: that is the
fork FR-015 forbids; the prototype itself flags full reuse as the 018
decision (node.go:314).

## R3 — Node architecture: carry the measured prototype shape

**Decision**: carry the prototype's mechanics as the starting point
[all measured on the rig and/or the BYON]:

- go-sdk `mcp.NewStreamableHTTPHandler(getServer, &mcp.StreamableHTTPOptions{...})`;
  `getServer(r *http.Request)` reads `Authorization: Bearer`, resolves the
  pool entry, returns the per-principal `*mcp.Server` (now
  `mcpserver.NewServer(rc, mcpserver.WithKeyring(...))`).
- Pool keyed by **unverified routing hint**: peek the JWT payload
  (base64url part 2) for `oidc:<iss>:<oid|sub>`; non-JWT → `tok:<token>`.
  Routing only — trust never depends on it.
- Per-entry `nats.Connect` with `nats.TokenHandler` returning the entry's
  current bearer on every (re)connect (the whole refresh mechanism),
  `ReconnectWait(200ms)`, `MaxReconnects(-1)`, optional
  `UserCredentials(sentinelPath)`.
- Principal via `$SYS.REQ.USER.INFO`: scan resolved publish allows for the
  5-part `<prefix>.<account>.<persona>.sign.record` subject → (account,
  persona). Server-asserted; `persona == principal's user`.
- Per-entry wiring: `siclient.New(nc, account, persona).PersonaSigner(persona)`
  → `realm.NewClient(ctx, nc, realm.Config{Realm, Persona, Signer})` →
  `mcpserver.NewServer`. PersonaSigner construction fails fast if the
  persona belongs to another principal [measured, sid `client.go`].
- Corpse eviction: entry with build error or `nc.IsClosed()` (nats.go stops
  reconnecting after consecutive auth violations) is closed, deleted, and
  admission retried on the next authenticated request.
- OAuth resource side only when `PublicURL` is set:
  `GET /.well-known/oauth-protected-resource` → RFC 9728
  `{resource, authorization_servers:[issuer], bearer_methods_supported:["header"]}`;
  401 + `WWW-Authenticate: Bearer resource_metadata="…", error="invalid_token"`.
- Proxy declaration: `DisableLocalhostProtection: PublicURL != ""` (the
  HTTPS front is the shape; the SDK's DNS-rebinding guard must yield)
  [measured: tailscale-serve front].

**Rationale**: every mechanism above has a measurement behind it; 018 should
re-house, not re-derive.

## R4 — Non-interference: session binding + candidate-bearer probe (NEW vs prototype)

**Problem**: the prototype stores the presented bearer into the entry's
`latest` on **every request, pre-admission**. Under the spec's clarified
FR-005 (strong non-interference) that is a hole: a forger who crafts a JWT
with a victim's `iss`+`oid` routes to the victim's entry and **overwrites
their freshest token with garbage** — the victim's next reconnect then fails.
Denial of service against a named principal by an unauthenticated actor.

**Decision**: a bearer may become an entry's `latest` only through one of two
admitted paths:

1. **Session binding (the refresh path)**: an MCP session is bound at
   creation to the entry that admitted it. A request carrying an
   established session's id may update that entry's `latest` — the hosted
   client refreshing its token mid-session is exactly this.
2. **Candidate probe (the new-badge path)**: a request with no bound session
   whose bearer differs from the entry's current `latest` gets a
   short-lived **probe connection** dialed with the candidate bearer. Only
   if the probe is admitted AND its server-asserted principal equals the
   entry's principal is the bearer adopted (and the session bound);
   otherwise the request is refused with the 401 challenge and the entry is
   untouched. A brand-new entry (no live connection) needs no probe — the
   build IS the probe; a garbage token there hurts only its presenter.

**Rationale**: satisfies FR-005/SC-002 measurably (adversarial test: forged
hints, zero displacement). Probe cost is one extra connect per *new badge
per principal* — token-TTL cadence, negligible.

**Alternatives**: trusting the hint (prototype) — now spec-illegal. Keying
the pool by full token string only — kills pooling across refreshes (every
new token = new connection + no session continuity). Verifying JWTs in the
node — forbidden by design (the node must not hold trust or validator
config; admission is the edge's).

## R5 — The AS-facing contract (what soulfold builds against)

**Decision**: the identity plane's OIDC lane is the admission target; the
contract for any external AS is therefore [all measured in soulidentity at
`5eaf52c`, `internal/callout/validator.go`]:

- **Trust config**: exactly `OIDCIssuer` + `OIDCAudience` (both-or-neither)
  on the identity plane. Exact issuer match; OIDC discovery fetched at
  startup (fails closed); JWKS signature check with refetch on unknown kid
  (AS key rotation needs no restart); **RS256 only** (alg downgrade
  refused); validity window enforced.
- **Access token**: a JWT. Claims read: `oid` (REQUIRED — keys the minted
  user; becomes the persona, so it MUST be a legal slug
  `^[a-z0-9]+(-[a-z0-9]+)*$`, ≤64); `roles` ([]string — exactly ONE value
  must name a declared role on the identity plane; zero → refuse, more than
  one match → ambiguous, refuse); `preferred_username` (optional, audit
  display only, never keyed). `aud` MUST equal the configured
  `OIDCAudience` — a **fixed deployment-chosen identifier** (recommended:
  the node's canonical public URL). With Dynamic Client Registration each
  client gets its own id, so the AS MUST stamp the fixed audience
  regardless of requesting client (resource-indicator or fixed-audience
  config — soulfold requirement, standard "API audience" on commercial ASes).
- **Client side** (AS ↔ hosted client, node not involved): OIDC discovery
  document, authorization-code + PKCE, and Dynamic Client Registration
  (RFC 7591) for the zero-config hosted dialog (or a pre-registered
  client id as fallback).
- **Node side**: RFC 9728 metadata names the issuer; 401 challenge points
  at the metadata. `AuthIssuer` config MUST equal the identity plane's
  `OIDCIssuer` or hosted clients will authenticate against an issuer the
  callout refuses.

**Static lane unchanged**: `sit_`-prefixed tokens dispatch to the token
lane by shape (digest lookup, expiry, role binding); `eyJ` dispatches to
OIDC; anything else refused. The node's passthrough is identical for both
(spec Q4).

## R6 — Test rig: embedded server + public `embed.Run` + contract-built AS stand-in

**Decision**: the node module gets its own rig (it cannot import soulstream's
`internal/natstest` nor soulidentity's internals — module boundary):
embedded `nats-server/v2` in operator mode with auth callout (the
prototype's `rig_test.go`, recoverable from `56c7a2e`, is the reference —
rebuilt on **public** surfaces: `jwt/v2` + `nkeys` for the operator/account
ceremony, soulidentity's public **`embed.Run`** for the identity plane +
callout issuer in-process — the seam SoulNode's composition gate just
proved). The OIDC AS stand-in is written in the node module's test code
**from the contract document alone** (SC-005): discovery doc + JWKS +
RS256 mint + DCR; driven by a scripted client for the full
discovery→register→sign-in→admitted flow.

**Rationale**: SC-005 makes the stand-in a first-class deliverable; using
`embed.Run` keeps the rig on public tagged-ish surfaces (and doubles as a
consumer proof of SoulNode's embed seam). Scope templates on the account
signing key are the rig's job (they are deployment config, not soulidentity
code — verified: minted user JWTs carry no permissions, `SetScoped(true)`).

## R7 — Scope template + deployment wiring: documented prerequisite, carried tooling

**Decision**: the represented-user scope template (the subjects from the
design: `<prefix>.status`, `<prefix>.xkey`,
`<prefix>.{{account-subject()}}.{{name()}}.sign.record`, `…keys.public`,
`SOULSTREAM.>`, `$JS.API.>`, `$KV.>`, `$O.>`, `$SYS.REQ.USER.INFO` pub;
`_INBOX.>`, `SOULSTREAM.>` sub) is a **deployment requirement documented in
the node's operator docs**; `cmd/byon-setup` (Synadia Cloud API scripting:
programmatic scoped sk-group for users, non-programmatic group for the
callout issuer user, callout enablement, XKey surfacing) is **carried as
best-effort operator tooling** under the node module, per spec Q2. The
callout XKey case stays a loud, surfaced error (prototype byon-setup:143).

## R8 — Node configuration (FR-002's "glance-sized" set)

**Decision**: flags/env only, no config file: `--listen` (default
`127.0.0.1:8080`), `--public-url` (empty = local/no-OAuth mode),
`--issuer` (required when `--public-url` set), `--realm`, `--nats-url`,
`--sentinel-creds` (optional path), `--prefix` (soulidentity subject root,
default `soulidentity` — must match the plane or ops time out, D14).
Version flag per house convention. No secrets in config; the sentinel
credential is deny-all by construction (Pub/Sub deny `>`), it routes to
callout and grants nothing.

**Rationale**: matches the design's "durable config is a URL, a realm name,
and the operator-mode sentinel"; `--prefix` is forced by soulidentity's D14.

## R9 — Provisioning: the node does not provision

**Decision**: like the archivist, the node requires an existing, provisioned
realm; it never calls `Provision` (minted users' scope may not even allow
stream creation). Documented prerequisite + a teaching error at startup if
the realm's stream is absent (read-only check).

## R10 — Revocation semantics: TTL-bounded, stated honestly

**Decision**: adopt the identity plane's semantics as the spec's "admission
window": revocation (token deleted, role stripped, vault role removed) bites
at the **next admission** — fresh connects refuse immediately; an already-
live pooled connection lives until its minted JWT expires (callout TTL,
default 15m; measured biting within 2×TTL on the rig). The node's part is
only: evict on auth-closed connections, never extend service beyond the
admitted JWT. Operator docs state TTL sizing = revocation responsiveness.

## R11 — Observability without tokens

**Decision**: structured log lines at admission build/refuse/evict/probe
carrying principal (account+persona), hint *class* (oidc/opaque — never the
values), and cause; bearer values, JWT payloads, and sentinel contents never
logged (FR-013/SC-006 audited by a test that greps a captured full-run log).

## Minimum NATS server version

Auth callout + JetStream: NATS **2.12+** (constitution floor; callout
available earlier, 2.12 is the stated project minimum). Embedded test server:
nats-server v2.14.x (prototype pin).
