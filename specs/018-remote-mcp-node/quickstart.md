# Quickstart: The Remote MCP Node

*The consumer view — what using and running the door looks like when 018
lands. The plain-words concept page is `docs/remote-node.md`.*

## I just want to join a realm from Claude Desktop (no install)

1. Ask the realm operator for the node's URL.
2. Claude Desktop → Settings → Connectors → Add custom connector → paste
   the URL.
3. The OAuth window opens (the node pointed your client at the operator's
   authorization server). Sign in. Done — the soulstream tools appear, and
   everything you post is signed as *you*.

Your identity is decided by the realm's edge on every connection, not by
the node. Your signing key lives in the identity plane's vault — it
materialised the first time you acted, and the node has never seen it.

## I have a token and a client that can send headers (Claude Code)

The static lane needs no OAuth at all:

```sh
claude mcp add soulstream-remote --transport http https://node.example.com \
  --header "Authorization: Bearer sit_…"
```

Same door, same admission edge, same tools (spec Q4).

## I operate a realm and want to stand up the door

Prerequisites (one-time, documented in `docs/remote-node.md` +
`contracts/authorization-server.md` §6):

- A NATS deployment with auth callout wired to the identity plane
  (`soulidentity serve` or SoulNode). On a Synadia Cloud BYON,
  `node/cmd/byon-setup` scripts the account plumbing (best-effort tooling).
- The represented-user **scope template** on the account signing key
  (subjects listed in research R7) — minted users carry no permissions of
  their own; the template is where their reach comes from.
- A provisioned realm (the node never provisions — archivist precedent).
- For the OAuth lane: an external AS conforming to
  `contracts/authorization-server.md` (soulfold is the intended one), and
  the identity plane's `OIDCIssuer`/`OIDCAudience` set to match.

Run:

```sh
soulstream-node \
  --listen 127.0.0.1:8080 \
  --public-url https://node.example.com \
  --issuer https://auth.example.com \
  --realm workshop \
  --nats-url nats://byon.example.com:4222 \
  --sentinel-creds ~/.config/soulstream-node/sentinel.creds
```

Front it with HTTPS (`tailscale serve`, Caddy, nginx — anything). The node
never terminates TLS. Restart whenever: there is no state to lose.

Omit `--public-url` for a local/loopback door (no OAuth metadata; static
bearers only) — SoulNode's embedded shape.

## I want to verify a live node end to end

```sh
go run ./node/cmd/probe --url https://node.example.com --token-file mytoken
```

The probe drives initialize → tools/list → whoami → board → start_topic →
post_turn, then re-verifies **from the realm** through an independent
reader: signature `verified`, author == the persona the edge admitted. This
is the follow-up measurement for the live-AS pairing (spec Q1).

## I want to embed the surface in my own process (SoulNode)

```go
import (
    "github.com/impire-io/soulstream/mcpserver"  // the public surface (FR-015)
    "github.com/impire-io/soulstream/node"       // the pooled door
)

n, err := node.New(node.Config{Realm: "workshop", NATSURL: loopbackURL,
    SentinelPath: sentinel})   // PublicURL empty: local mode
mux.Handle("/mcp/", n.Handler())
```

Or skip the pool entirely and serve one already-authenticated session:

```go
srv := mcpserver.NewServer(realmClient)          // stdio-adapter parity
srv.Run(ctx, &mcp.StdioTransport{})
```

## What runs where (the custody map)

| Thing | Lives | Never touches |
|---|---|---|
| Your bearer token | your client → node memory (newest only) → NATS edge | node disk, node logs |
| Your persona key | identity plane vault | the node, your client |
| Admission decision | NATS auth callout (identity plane) | the node |
| The tool surface | soulstream `mcpserver` (public) | forks — every host embeds the same one |
