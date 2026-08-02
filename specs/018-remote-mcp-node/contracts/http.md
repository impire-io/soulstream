# HTTP Contract: the node's wire surface

Three routes. Nothing else. All other paths 404.

## 1. `GET /.well-known/oauth-protected-resource`

Served **only when `PublicURL` is configured** (public mode). RFC 9728:

```json
{
  "resource": "<PublicURL>",
  "authorization_servers": ["<AuthIssuer>"],
  "bearer_methods_supported": ["header"]
}
```

`Content-Type: application/json`. No auth required. In local mode
(`PublicURL == ""`) the route is absent (404) — Bars-1-3 behavior.

## 2. The MCP endpoint (`/` by default)

Streamable HTTP (MCP go-sdk `StreamableHTTPHandler`), session semantics per
the SDK. Per-request rules:

- `Authorization: Bearer <token>` is the only credential lane
  (`bearer_methods_supported: ["header"]`). Any bearer shape is passed
  through — OIDC JWT or static `sit_` token alike (spec Q4); the admission
  edge dispatches by shape.
- **No/invalid bearer, public mode** → `401` with
  `WWW-Authenticate: Bearer resource_metadata="<PublicURL>/.well-known/oauth-protected-resource", error="invalid_token"`
  — the challenge that steers a hosted client into the OAuth flow (FR-008).
- **No bearer, local mode** → the SDK's bare `400` (no OAuth story to point
  at).
- **Bearer refused at admission** (build/probe refusal) → same `401`
  challenge; the entry is left per the non-interference rules
  (data-model bearer lifecycle).
- Established sessions ride their bound entry; a session whose entry died
  gets tool-call errors naming the auth cause and recovers by
  re-presenting a valid bearer (no session teardown required — spec edge
  case).

## 3. Proxy-fronted shape (FR-012)

The node binds loopback/private and is fronted by an HTTPS terminator whose
public name is `PublicURL`. Setting `PublicURL` declares the shape:
`DisableLocalhostProtection` on the SDK handler (the DNS-rebinding guard
must yield to the declared front). TLS is the front's job; the node serves
plain HTTP on `Listen` and never terminates TLS itself.

## Headers & logging

- Bearer values never appear in logs, error bodies, or the metadata
  document (SC-006).
- Error bodies are generic (`invalid_token`); cause detail goes to the
  operator log, keyed by principal or hint class — not to the wire.
