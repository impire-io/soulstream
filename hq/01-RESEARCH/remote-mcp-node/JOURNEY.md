# remote-mcp-node — investigation journal (started 2026-07-30)

## 2026-07-30 — the prototype and Bars 1–3, in one sitting

**Setup.** Built the prototype node and its rig in [experiment/](experiment/)
(a consumer-position module importing both repos; soulstream at the working
tree ≈ v0.6.0, soulidentity at the working tree, post-journey-0016). The rig
is the one combination neither repo had ever run: **admission through auth
callout** (the Entra e2e's front door) **with the both-subject-space scope
template** (the m2 gate's shape) — SoulIdentity user ops beside
`SOULSTREAM.>`/JetStream, plus `$SYS.REQ.USER.INFO` — on one team account
(ACME) hosting both the SoulIdentity service and the realm. The node itself
is ~250 lines: `mcp.NewStreamableHTTPHandler` with a per-request hook, a pool
keyed by an *unverified* routing hint (OIDC `iss`+`oid` peek, else the token
string), `nats.TokenHandler` reading the freshest bearer stored by the HTTP
layer, principal → `PersonaSigner` → `realm.NewClient` per user.

**Bar 1 — PASS.** Both lanes admit through the node with attribution read
back from the realm, not the node's word: the API-token session's turn is
authored `daan-ext`, the Entra-shaped session's turn is authored by the oid.
Garbage and revoked tokens refuse (MCP session never forms, `callout REFUSED`
in the audit) and the topic's contribution count is unchanged — no realm
write without admission.

- **Finding (Bar 1 wording amended openly): the sentinel is required in
  operator mode.** The bare token option is refused
  (`nats: Authorization Violation` with no callout decision); the sentinel
  creds file (soulidentity journey 0008: a public, deny-all bearer artifact)
  is the rung that routes a connection to the callout. So the node's honest
  config is **URL + realm + sentinel** — the sentinel is credential-*shaped*
  but grants nothing by itself. Bar 1's "nothing credential-shaped" is
  amended to "nothing that grants access by itself"; measured basis recorded
  here.
- **Finding: OIDC-lane personas are oids.** The issuer names the minted user
  `sub.OID`, so realm attribution reads as a UUID (a legal soulstream slug);
  the human display (`preferred_username`) survives only in the audit line.
  Fine for machines, poor for humans reading a board — makes the
  persona-presentation op on SoulIdentity's surface (display metadata beside
  `keys.public`) load-bearing, exactly the gap named in the pre-registration.
- **Finding: the principal is server-asserted, discoverable by the node.**
  `$SYS.REQ.USER.INFO` returns the *resolved* permission set; the expanded
  `{{account-subject()}}.{{name()}}` in the `sign.record` grant names
  (account, user) — a whoami with no client-claimed identity anywhere. It is
  a parse of a permission subject, though; an explicit whoami surface is an
  018 niceness.

**Bar 2 — PASS.** The negative control first: with no keyring the node's
announcement reads `unknown-key`. Then the reader builds its keyring from
**one** `keys.public` answer and announce + turns read `SigVerified`. Zero
per-user acts anywhere; the persona key materialised in the vault at
`PersonaSigner` construction; the node's process held URL + realm + sentinel
and no key material.

**Bar 3 — PASS, after the investigation's best scare.**

- *Phase 1 (expiry, refreshed badges):* on a 5 s-TTL rig, **33/33 writes over
  17 s with zero failures** across three `authentication expired`
  disconnects. The client refreshed its bearer per request; the HTTP layer
  stored it; `TokenHandler` presented it on each reconnect; nats.go's
  reconnect buffering absorbed even the in-flight window. Better than the
  registered allowance (≤1 in-flight failure per re-proof).
- *Phase 2 (revocation at the IdP):* after stripping the role from fresh
  tokens, **last successful write +3.1 s, persistent failure from +3.9 s**
  (bound: 2×TTL = 10 s). Mechanism, from the audit: at the first TTL
  boundary the reconnect presents the stripped token, callout refuses
  (`no roles claim`), nats.go refuses twice and **closes the connection
  permanently** — signing stops dead.
- *The scare:* two earlier runs showed writes "succeeding" 20–30 s
  post-strip. The full audit refuted the test, not the design: `sign.record`
  entries stopped at the refusal, while the test loop counted `err == nil`
  as success — **MCP tool errors are in-band** (`result.IsError`), and 54
  tool-level failures on a dead connection had been logged as "ok". Lesson
  for every MCP harness we write: a transport-level success is not a call
  success.
- *Design note for 018:* after nats.go's two-strikes close, the prototype's
  pool entry is a permanent corpse. A real node must evict dead entries and
  rebuild on the next authenticated request (and answer 401, not 400, while
  refused).

**Where the bars stand:** 1–3 PASS [measured, rig in experiment/]. **Bar 4
(a genuinely no-install client) remains** — it needs the node exposed over
HTTPS and a claude.ai / Claude Desktop connector pointed at it, which is an
operator act. The reversal-condition question (does the deployment class the
realms actually live on — NGS/Synadia Cloud — host auth callout?) also
remains open; nothing measured today touches it.

## 2026-07-31 — Bar 4 prepared: the operator's constraints turn out benign

Operator input reshaped Bar 4's plan: tailscale is available for HTTPS,
there is **no Entra tenant** (Auth0 proposed instead), and — on the
reversal condition — **auth callout is available for BYON environments on
Synadia Cloud/NGS**, with that setup in progress [operator report; the
measurement stays open until admission runs there].

**Finding: no IdP is required for a no-install client.** Verified against
the connector auth docs: **static bearer headers are supported on Claude
Desktop, claude.ai (`static_headers`, beta), and Claude Code** — so Bar 4's
API-token lane runs with nothing but a `sit_` token in a header. The OAuth
lane requires RFC 9728 resource metadata, S256 PKCE, and a registration
story (DCR — which Auth0 supports — or CIMD, or Anthropic-held
credentials); bare `client_credentials` is unsupported (interactive consent
always), and tokens in URLs are prohibited.

**Finding: the OIDC lane is issuer-agnostic; "Entra" is three claim
names.** Read from `internal/callout/validator.go`: go-oidc discovery +
JWKS, RS256 only, pinned audience — any compliant issuer fits. The shape is
the claims: `oid` (required, becomes the persona), `roles` (exactly one
declared team), `preferred_username` (display). Auth0 therefore works via a
post-login Action emitting those three — with two pre-known traps recorded
in [bar4-runbook.md](bar4-runbook.md): the `oid` value must be a
soulstream-legal slug (raw `auth0|…` subjects are not), and Auth0's
custom-claim namespacing may drop bare claim names, which would justify a
small issuer-profile config in soulidentity with a measured refusal.

**Node made runnable for the operator act.** `experiment/cmd/node` (flags:
listen, nats-url, sentinel, realm, public-url, auth-issuer); the OAuth edge
(RFC 9728 metadata + 401 `WWW-Authenticate` challenges, off unless
`--public-url` is set, covered by `edge_test.go`); **corpse eviction** — the
2026-07-30 design note built in: a failed or permanently closed pool entry
is evicted and admission retried with the next request's badge; and a
`board` tool so the pre-registered pass protocol (initialize → tools/list →
board → post_turn) maps onto the prototype as written. Bars 1–3 re-run
green after the changes [measured]. The operator steps live in
[bar4-runbook.md](bar4-runbook.md).

## 2026-08-01 — the BYON is reachable; the realm stands on it

The reversal-condition environment exists [measured, probes]: Synadia Cloud
BYON, context `impire-dev-platform`, server `beno1` (nats-server 2.12.7) at
`nats://100.108.7.14:4222` — on the tailnet already, RTT 8.4 ms. JetStream
enabled on the account; the bootstrap user carries `Deny:
$SYS.REQ.USER.AUTH`, the guard consistent with callout availability on this
deployment class. Realm `proof` provisioned through the context with the
default budgets (op-log 1 GiB, inbox 64 MiB, objects 512 MiB, personas
64 MiB) — soulstream's realm shape stands on the BYON without incident.

Still unmeasured, and honestly so: **callout admission itself.** The AUTH
side (external authorization, issuer creds, the scoped signing key with the
both-subject-space template) exists only in the Synadia Cloud console's
gift; the exact remaining acts are itemised in
[bar4-runbook.md](bar4-runbook.md) §0a. Lane B's local-IdP question was
also answered this week: Pocket ID (Go, SQLite, configurable claim keys —
needs a small RFC 7591 shim) or Ory Hydra (native DCR, proven with Claude,
bring-your-own consent page) are the recorded candidates, with ZITADEL
ruled out for the dance (DCR open issue #9810).

## 2026-08-01 (later) — the console acts become one command: the Cloud API

The operator pointed at the Synadia Cloud API, and it covers everything §0a
listed as console work — with two findings worth their own lines:

- **The Cloud API has first-class auth-callout endpoints**
  (`control-plane-sdk-go` v0.9.0): `EnableAuthCallout(system,
  {control_account})`, target-account wiring that explicitly names a
  signing-key group on each side, and callout users registered from
  control-account NATS users. Synadia's own model matches the rig's shape
  one-to-one.
- **Programmatic signing-key groups solve the custody question in
  soulidentity's favour.** `CreateAccountSkGroup(…, {programmatic: true,
  scope})` returns the SEED exactly once — the sanctioned export the vault
  import needs; the console-only worry recorded on 2026-07-31 is void. The
  scoped group carries our both-subject-space template verbatim
  (`{{account-subject()}}.{{name()}}` templating included).

Built `experiment/cmd/byon-setup` (plan/apply, idempotent re-runs, seeds
written 0600 before anything else can fail, next-step soulidentity commands
printed). Compiles against the SDK; the live run awaits a Personal Access
Token — the one input only the operator can mint. Unmeasured until then,
honestly: every call shape is compile-checked but none has met the real API;
the callout-id addressing (system id?) is an explicit guess the tool logs
loudly about.
