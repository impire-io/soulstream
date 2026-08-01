# Bar 4 runbook — a no-install client through the node

The operator half of Bar 4 ([README](README.md)): the node exposed over
HTTPS, a hosted MCP client pointed at it, both badge lanes tried. Facts about
Claude connector auth verified 2026-07-31 against
[the connector auth docs](https://claude.com/docs/connectors/building/authentication):
**static bearer headers are supported on Claude Desktop, claude.ai
(`static_headers`, beta), and Claude Code (`--header`)**; the OAuth path
requires RFC 9728 resource metadata (the node serves it), S256 PKCE, and a
registration story (DCR, CIMD, or Anthropic-held credentials). Tokens in
URLs are prohibited — headers or OAuth only.

## 0a. The BYON concretes (probed 2026-08-01)

The deployment exists: Synadia Cloud BYON, context **`impire-dev-platform`**
(`DEV-PLATFORM-daan` is the same target), server `beno1` (v2.12.7) at
`nats://100.108.7.14:4222` — already on the tailnet, RTT ~8 ms. JetStream is
enabled on the account; realm **`proof`** is provisioned with default
budgets. The bootstrap user carries `Deny: $SYS.REQ.USER.AUTH` (the callout
guard). What the account does NOT yet have is the callout wiring below —
**scripted in `experiment/cmd/byon-setup`** against the Synadia Cloud API
(first-class auth-callout endpoints; programmatic signing-key groups whose
seed is returned exactly once — the sanctioned custody export). It needs a
Personal Access Token from
`https://cloud.synadia.com/profile/personal-access-tokens`:

```sh
cd hq/01-RESEARCH/remote-mcp-node/experiment
SYNADIA_PAT=uat_… go run ./cmd/byon-setup --system dev-impire-platform          # discover + plan
SYNADIA_PAT=uat_… go run ./cmd/byon-setup --system dev-impire-platform --apply  # wire it, seeds → byon-secrets/
```

It performs, idempotently: the control (AUTH) account, a programmatic
**scoped** sk group on the app account (the both-subject-space template) and
one on the control account, `EnableAuthCallout`, the target-account wiring,
and the issuer user (registered as callout user, creds downloaded). What
that automates, spelled out:

1. **The AUTH side**: an account with external authorization enabled
   (allowed account = this app account), its callout xkey, and creds for
   the issuer user — these become `--callout-creds`/`--auth-account`.
2. **A scoped signing key on the app account** whose template carries the
   represented-user allows (see `experiment/rig_test.go`): the
   `soulidentity` user ops on the own prefix, `SOULSTREAM.>`, `$JS.API.>`,
   `$KV.>`, `$O.>`, `$SYS.REQ.USER.INFO` (pub) and `_INBOX.>`,
   `SOULSTREAM.>` (sub). Generate the keypair locally, register the public
   half as the scoped signing key in the console, keep the seed for the
   vault import.
3. Same pattern for an AUTH-account signing key (the issuer's signing key,
   vault name `auth/issuer`).

Then, on any tailnet machine:

```sh
# the service + issuer (xkey seeds via SOULIDENTITY_FIRST_KEY / _SURFACE_KEY / _CALLOUT_KEY)
soulidentity serve --context impire-dev-platform \
  --callout-creds auth-issuer.creds --auth-account <AUTH-public-key> \
  --callout-ttl 5m [--oidc-issuer … --oidc-audience …]

# provision the trust material (as the admin principal)
soulidentity key import --context impire-dev-platform --as <acct>/<user> \
  --name acme --kind nats-account-signing-key --seed-file team.seed --account <acct-pub>
soulidentity key import … --name auth/issuer … --account <AUTH-pub>
soulidentity token create --context impire-dev-platform --as <acct>/<user> \
  --account <acct-pub> --user daan --label "claude desktop"
soulidentity sentinel --context impire-dev-platform --as <acct>/<user> > sentinel.creds
```

## 0. Prerequisites

- A NATS deployment running the SoulIdentity service **and** the callout
  issuer, with the both-subject-space scope template on the team account
  (the shape `experiment/rig_test.go` builds; BYON on Synadia Cloud/NGS once
  that environment lands — callout is available there per the operator,
  which also answers the topic's reversal-condition question when measured).
- The realm provisioned (`soulstream provision` as an operator identity).
- The sentinel creds exported to a file.

## 1. Run the node

```sh
cd hq/01-RESEARCH/remote-mcp-node/experiment
go run ./cmd/node --realm <realm> --nats-url <url> --sentinel <sentinel.creds> \
  --public-url https://<machine>.<tailnet>.ts.net \
  --auth-issuer https://<tenant>.auth0.com/     # only needed for the OAuth lane
```

## 2. HTTPS via tailscale

- **Claude Desktop on a tailnet machine** (the cheap first run):
  `tailscale serve --bg 8080` — tailnet-only HTTPS, nothing public.
- **claude.ai web** connects from Anthropic's servers, so it needs
  `tailscale funnel --bg 8080` — genuinely public; prefer the Desktop run
  first.

## 3. Lane A — API-token bearer (no IdP at all)

Create a SoulIdentity API token for the persona
(`tokens.create` → `sit_…`), then in Claude Desktop add a custom connector
to `https://<machine>.<tailnet>.ts.net` with header
`Authorization: Bearer sit_…` (Claude Code equivalent:
`claude mcp add --transport http node <url> --header "Authorization: Bearer sit_…"`).

Pass protocol: initialize → tools/list → `board` → `post_turn`, then verify
the turn's author and `SigVerified` from a reader (the Bar 1/2 checks, by
hand or with the experiment's reader snippet).

## 4. Lane B — OAuth via Auth0

The callout's OIDC lane is issuer-agnostic (go-oidc discovery + JWKS,
RS256, pinned audience); the *Entra shape* is only three claim names the
validator reads: `oid` (required; becomes the persona), `roles` (must name
exactly one declared team), `preferred_username` (display, audit-only).

1. Auth0: create an **API** (identifier = the audience, e.g.
   `soulstream-node`), RS256.
2. A **post-login Action** must emit those three claims on the access
   token. Two traps, both known in advance:
   - **The `oid` value becomes the NATS user and the soulstream persona** —
     it must match `^[a-z0-9]+(-[a-z0-9]+)*$` (≤64). Auth0's raw
     `sub` (`auth0|…`) is illegal (the `|`); emit a slug-safe stable id
     (e.g. a UUID pinned in `app_metadata`).
   - **Auth0 may drop non-namespaced custom claims** on access tokens.
     Verify in the tenant whether bare `oid`/`roles` survive an Action's
     `setCustomClaim`; if not, the validator's claim names need a small
     issuer-profile config in soulidentity (a new D-decision there), which
     this run would justify with a measured refusal.
3. Enable **dynamic client registration** on the tenant (Claude supports
   DCR out of the box; CIMD or Anthropic-held credentials are the
   alternatives if DCR is off the table).
4. Run the callout issuer with `--oidc-issuer https://<tenant>.auth0.com/`
   `--oidc-audience <api identifier>`, and the node with `--public-url` +
   `--auth-issuer` so discovery works: Claude hits the node → 401 +
   `WWW-Authenticate: Bearer resource_metadata=…` → RFC 9728 document →
   Auth0 → S256 PKCE consent → bearer on every request → callout admits.

## 5. What to record in JOURNEY.md

Which lane(s) completed the pass protocol end to end, the exact friction of
any that did not, and — once the BYON environment is up — whether callout
admission works there (the reversal-condition measurement). Then the topic
is ready for `/research-graduate remote-mcp-node --to design`.
