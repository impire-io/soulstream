# Extension: The Remote MCP Node

*A URL into a realm, for clients that cannot install anything. The first
adapter that holds **no** credentials — auth callout supplies each session's
identity, so the node custodies nothing and multiplexes personas across
connections. Nothing here is protocol-normative: the node speaks the same core
subjects any client does; it adds reach, never capability.*

Graduated from research topic `remote-mcp-node`
([episode 0008](../../04-JOURNEY/0008-remote-mcp-node.md)), Bars 1–3 and the
reversal-condition measurement PASS on a Synadia Cloud BYON; Bar 4 partial —
the OAuth authorization-server story below is the open build decision.

---

## Why it exists

The [MCP adapter](../../../docs/mcp.md) runs next to the agent and holds one
persona's credentials ([library-and-adapters.md](library-and-adapters.md),
Layer 2). Some hosts cannot run it at all: sandboxed Claude Desktop (observed
failing to install the binary), claude.ai custom connectors, locked-down
machines. For them the door must stand *at the workshop* — a shared node
reached by URL. This extension is that node.

It also reframes Layer 2. The existing adapters are credential custodians. The
remote node holds **no credentials and no keys**: the caller presents a bearer
token, the node passes it straight to NATS, and SoulIdentity's auth callout
admits or refuses. Identity is the server's decision, per connection; the node
is dumb plumbing on purpose. That is what makes one shared door safe for many
people.

## The shape (measured)

```
  hosted client ──bearer──▶ [ node ] ──nats.Token(bearer)──▶ NATS server
   (Claude Desktop,          per-user                          │ auth callout
    claude.ai, …)            pooled conn                       ▼
                                                        SoulIdentity issuer
                                                        (validate → mint JWT)
```

1. **One pooled NATS connection per principal.** The node keys a pool by an
   *unverified* routing hint (the OIDC `iss`+`oid` peek, else the token
   string) — routing only; a forged hint just builds a connection the callout
   refuses. Trust is decided at the NATS edge, never in the node.
2. **Bearer → `nats.Token`, freshest wins.** The HTTP layer stores the newest
   bearer the client presents; a `nats.TokenHandler` reads it on every
   (re)connect. This is the whole token-refresh mechanism — a session outlives
   the callout JWT's TTL by re-presenting a fresh badge (Bar 3: 33/33 writes
   across 3× TTL).
3. **Principal is server-asserted.** The node derives (account, user) from its
   own resolved permissions via `$SYS.REQ.USER.INFO` — the expanded
   `sign.record` grant names the principal. No client-claimed identity is ever
   trusted. `persona == principal's user`.
4. **Signing delegated, custody intact.** The per-user connection carries both
   subject spaces (SoulIdentity user ops + the realm), so
   `client.PersonaSigner` slots into `realm.Config.Signer` unmodified; the
   persona key materialises in the vault on first touch and never reaches the
   node. Readers verify from `keys.public` — the identity plane is the
   directory (SoulIdentity journeys 0015–0016); the node publishes no profile.
5. **Revocation is the callout's, not the node's.** A revoked or role-stripped
   token is refused at the next reconnect within ~TTL (Bar 3: +3.9 s on a 5 s
   TTL). Dead pool entries are evicted and admission retried on the next
   authenticated request.

### The scope template (deployment requirement)

The represented-user signing key the callout mints must grant, on the caller's
own prefix: `soulidentity.status`, `soulidentity.xkey`,
`soulidentity.{{account-subject()}}.{{name()}}.sign.record`,
`…keys.public`, plus `SOULSTREAM.>`, `$JS.API.>`, `$KV.>`, `$O.>`,
`$SYS.REQ.USER.INFO` (pub) and `_INBOX.>`, `SOULSTREAM.>` (sub). On Synadia
Cloud this is a **programmatic scoped signing-key group** (its seed returned
once, for the vault import); a non-programmatic group hosts the callout issuer
user. `cmd/byon-setup` scripts the whole wiring via the Cloud API.

## The OAuth edge — the open build decision (018)

The transport works; the last gap is *what a hosted client authenticates
against*. Measured 2026-08-01: **Claude Desktop's remote-connector dialog
offers only OAuth** (Client ID/Secret, else Dynamic Client Registration) —
there is **no static-header field**, so the API-token (`sit_`) bearer lane,
though it works over the wire, is unreachable from that UI. A no-install hosted
client therefore *requires* the OAuth lane. The node already serves the
resource side (RFC 9728 `/.well-known/oauth-protected-resource`, `401` +
`WWW-Authenticate`, PKCE-ready); what 018 must choose is the authorization
server:

- **Node-embedded minimal AS (single-persona bridge).** The node serves DCR +
  PKCE + a small login/consent page and issues a token it maps to a configured
  identity; it connects to NATS as *one* persona. Needs no external IdP, works
  with the exact dialog today — but the node then holds that identity's
  credential and fronts a single persona. This is the "personal node" shape:
  honest, useful, not the multi-user passthrough.
- **External OIDC AS (multi-user passthrough).** A real authorization server
  (Auth0, or self-hosted Pocket ID / Ory Hydra) emits `oid` + `roles` claims;
  Claude does DCR+PKCE against it; the node passes the OIDC access token
  through; the callout OIDC lane admits per-user. This is the designed
  end-state. Constraints, researched: the AS must support DCR (or a pre-
  registered client / CIMD / Anthropic-held credentials); `oid` becomes the
  persona so it must be a legal slug (`^[a-z0-9]+(-[a-z0-9]+)*$`, ≤64 — raw
  `auth0|…` subjects are illegal); the AS must emit the claim *names* the
  validator reads, or SoulIdentity gains a small issuer-claim-profile config.

The two are not exclusive: the bridge unblocks a personal node now, the
external AS is the multi-user target.

## Deployment shape (non-negotiables 018 inherits)

- **Always proxy-fronted.** A production node sits behind HTTPS
  (tailscale serve/funnel, or a reverse proxy). The go-sdk's DNS-rebinding
  guard rejects a loopback server with a non-loopback Host unless the proxy
  shape is declared — the node must declare it.
- **Persona presentation.** OIDC-lane personas are oids; human display names
  live only in the audit today. Readable boards need a persona-presentation op
  beside `keys.public` on SoulIdentity's surface (a SoulIdentity D-decision) —
  otherwise identity-plane realms lose display metadata the realm phone book
  carries.
- **The node is stateless trust.** It stores no per-user secret and no trust
  state; restart is free. Its only durable config is a URL, a realm name, and
  the operator-mode sentinel that routes connections to callout.

## Relationship to the roadmap

This is the "node half" of SoulIdentity's M2 (its ROADMAP): one pooled
connection per user, no node-held creds — the identity-plane side (callout,
first-touch key materialisation, `keys.public` directory, `PersonaSigner`)
shipped in SoulIdentity journeys 0013–0016 and is proven cross-service. The
build lives here as feature **018** — a consumer submodule of soulstream (the
cycle guard: it imports both repos, neither core repo imports the other). The
research prototype (node, rig, `byon-setup`, `cmd/probe`, the OAuth edge) is
the starting point, recoverable from git history.
