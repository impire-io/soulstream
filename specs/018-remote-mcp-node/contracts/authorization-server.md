# The AS-Facing Contract

*What an external authorization server must provide for its tokens to open
the remote MCP node's door. This document is the interface (SC-005): a
conforming AS — soulfold, or any other — is built from this page, not from
node or identity-plane internals. Everything here is measured against the
identity plane at soulidentity `5eaf52c` (`internal/callout/validator.go`)
and the hosted-client behavior observed 2026-08-01.*

## Roles in the flow

```
hosted client ──(1) GET {node}/.well-known/oauth-protected-resource
              ◄── {"resource": <node URL>, "authorization_servers": [<issuer>]}
              ──(2) OIDC discovery at <issuer>/.well-known/openid-configuration
              ──(3) Dynamic Client Registration (RFC 7591)          [AS]
              ──(4) authorization-code + PKCE sign-in               [AS]
              ◄── access token (a JWT, claims below)
              ──(5) Authorization: Bearer <token> ──► node ──passthrough──►
                                            NATS auth callout ──► identity plane
                                            validates per this contract
```

The node is only steps 1 and 5 — it never talks to the AS. The identity
plane is the validator. The AS's obligations:

## 1. Discovery & endpoints

- Serve an OIDC discovery document at
  `<issuer>/.well-known/openid-configuration` naming `jwks_uri`,
  `authorization_endpoint`, `token_endpoint`, `registration_endpoint`.
- The **issuer value is matched exactly** by the validator (string
  equality with the deployment's configured issuer). No trailing-slash
  slack; pick one form and keep it.
- Discovery is fetched by the identity plane **at startup and fails
  closed** — the AS must be reachable when the plane boots.

## 2. Client registration & sign-in

- **Dynamic Client Registration (RFC 7591)** for the zero-config hosted
  dialog (Claude Desktop offers OAuth only; DCR is what makes "paste a
  URL" sufficient). A pre-registered client id is the acceptable fallback
  for ASes without DCR.
- **Authorization-code flow with PKCE (S256)**. The hosted client drives
  it; the AS must not require a client secret for public clients.

## 3. The access token (the load-bearing part)

The access token presented to the node MUST be a **JWT** with:

| Requirement | Value | On violation |
|---|---|---|
| Signature alg | **RS256 only** — key published in JWKS | refused (alg downgrade is refused explicitly) |
| `iss` | exactly the configured issuer | refused |
| `aud` | exactly the configured audience (see below) | refused |
| validity window | `iat`/`exp` current | refused |
| `oid` claim | REQUIRED; stable per person; **a legal persona slug**: `^[a-z0-9]+(-[a-z0-9]+)*$`, length ≤ 64 | missing → refused; illegal slug → the persona is unusable in the realm — mint conformant values (raw `auth0\|123` subjects are illegal) |
| `roles` claim | array of strings; **exactly one** value must name a role declared on the identity plane | zero matches → refused; two+ matches → ambiguous, refused; unmatched extras are inert |
| `preferred_username` | optional; display for audit only, never keyed | — |

**`oid` is identity.** It keys the minted NATS user and becomes the realm
persona (`persona == principal's user`); the persona's signing key
materialises in the identity plane's vault on the person's first touch. Two
people must never share an `oid`; an `oid` must never be reassigned.

**Audience is fixed per deployment.** The validator compares `aud` against
one configured value. With DCR every client has its own id — so the AS MUST
stamp the deployment's fixed audience (recommended: the node's canonical
public URL as a resource identifier, RFC 8707-style) on access tokens
regardless of which registered client requested them. Commercial ASes call
this an "API audience"; soulfold should treat it as such.

**JWKS rotation is free.** The validator refetches JWKS on unknown `kid` —
the AS may rotate signing keys without any identity-plane restart. New algs
are not free: RS256 only.

## 4. Lifetime & revocation semantics

- Keep access tokens short-lived; the hosted client re-presents fresh
  tokens and the node's freshest-bearer mechanism carries the session — the
  AS does not need refresh-token gymnastics for the node's benefit, only a
  working refresh grant for its clients.
- Revocation reaches the realm at the **next admission** (a callout
  re-validation: signature, window, and `roles`-still-declared are all
  re-checked every time). Stripping the person's role at the AS — or
  deleting the declared role on the identity plane — refuses their next
  connection; a live pooled connection persists at most one callout TTL.

## 5. What the AS must NOT rely on

- The node never validates tokens, never calls the AS, and never sends the
  AS anything — do not expect introspection traffic, back-channel logout,
  or resource-side sessions.
- The `iss`/`oid` the node peeks pre-admission is **routing only**; the AS
  gains nothing from crafting them and cannot harm an admitted principal by
  them (non-interference is the node's guarantee, FR-005).
- No per-user provisioning act exists anywhere (identity-plane D26): the
  first token that admits IS the onboarding. The AS must not assume a
  registration step on the realm side.

## 6. Deployment binding checklist (operator)

| Knob | Must equal |
|---|---|
| identity plane `OIDCIssuer` | the AS's exact issuer string |
| identity plane `OIDCAudience` | the fixed audience the AS stamps |
| node `AuthIssuer` | the same issuer (it is what the 9728 metadata advertises) |
| identity plane declared role(s) | the role name(s) the AS emits in `roles` |

Mismatch symptoms: metadata points clients at an AS whose tokens the
callout refuses (issuer/audience skew), or admissions refuse with role
errors (role not declared / ambiguous).

## 7. Conformance gate

The node repository carries a minimal AS stand-in **written from this
document alone** (discovery, JWKS, RS256 mint, DCR) and a scripted client
driving discovery → registration → sign-in → admitted session →
signed-and-verified first operation (SC-001/SC-005). An AS that matches
this page passes that flow unchanged; soulfold's live pairing is the
follow-up measurement (spec Q1).
