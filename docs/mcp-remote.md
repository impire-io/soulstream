# Remote MCP: the door that lives at the workshop

"Can I run Soulstream's MCP server remotely?" hides two different questions, and the
first one usually answers itself.

## The workshop is already remote

The realm — every topic, turn, attachment, seal — lives **on the NATS server**, which
can be anywhere (your LAN, a VPS, NGS). The [MCP adapter](./mcp.md) is not the
workshop; it's a **doorway**: a small stateless program that carries three names and,
optionally, your seal. So "I want to work from another machine" doesn't need anything
remote at all: install the same small binary there
([quickstart](./mcp-quickstart.md)), point it at the same context, walk in.

That's the honest default answer, and for most setups the best one: the door is
lighter than anything you'd build to avoid installing it.

## When the door itself has to move

Some places can't install a binary, full stop: claude.ai custom connectors, sandboxed
Claude Desktop, a phone, a locked-down corporate laptop. For them the doorway has to
stand **at the workshop** instead of on the desk — one shared door, reached by URL,
that many people walk through as themselves.

That is the **remote node** (`soulstream-node`), and it is **built** — feature 018.
This page describes how it works and how to run it.

## How the door works: show a badge

Everything follows from one rule: **you show a badge, and the badge is checked by the
badge office — never by the door.**

- **The badge** is a bearer token in the HTTP header your MCP client already knows how
  to send: an **OIDC access token** (your organisation's normal sign-in, through
  whatever authorization server the operator chose) or a **SoulIdentity API token**
  (for clients that can paste a header instead of doing the OAuth dance). One header,
  either shape — the node passes it through regardless of how you got it.
- **The door passes the badge through.** The node validates nothing itself: your badge
  becomes the token on a NATS connection opened *for you*, and
  [SoulIdentity](https://github.com/impire-io/soulidentity)'s auth callout — the badge
  office — checks it against the issuer and admits (or refuses) the connection. One
  pooled connection per person; the node holds **no credentials of its own**.
- **You are who your badge says.** One identity, one persona: the persona *is* the
  authenticated principal. There is no mapping table, no sign-up step, no per-user
  provisioning act anywhere — a persona is born the first time it shows up.
- **Seals stay in custody.** The node keeps no keys. When your persona signs, the
  signing happens inside SoulIdentity (`sign.record`); your key materialises in its
  vault on first touch and never leaves. Records signed this way verify like any
  other — byte-identical, already proven end-to-end.
- **The phone book is the identity plane.** On a realm fronted by SoulIdentity,
  readers fetch verification keys from it directly — no
  [pin notebooks](./persona-directory.md), no profile publishing. SoulIdentity *is*
  the directory; the realm's own phone book remains the standalone fallback for
  realms without an identity plane.
- **Badges expire.** Admission is TTL-bounded; the connection re-proves itself with
  the freshest badge, so a revoked token stops working within its TTL — not at the
  end of a long session.

The result for the person on the locked-down laptop: **a URL, not an install**. Add
the connector, sign in as yourself, and you're a realm member with the same
twenty-four buttons — attributed, signed, revocable.

## The sign-in question, answered

A hosted connector (Claude Desktop, claude.ai) authenticates a remote server by
**OAuth only** — there is no static-token field to paste a badge into. So a no-install
client needs an authorization server to sign in against. Feature 018 resolved this the
honest way: the node is **never** an authorization server itself. It points at an
**external** one, and stays agnostic about which — it only publishes, at the standard
discovery spot, *where* to sign in, and passes the resulting token through. The
intended default is [soulfold](https://github.com/impire-io/soulfold), the ecosystem's
own small embeddable OIDC provider; any conforming server works. What "conforming"
means is written down precisely in the
[AS-facing contract](../specs/018-remote-mcp-node/contracts/authorization-server.md):
discovery, dynamic client registration, PKCE, and the exact claims a token must carry
(notably a legal-slug `oid` that becomes your persona). The node repository proves that
contract *is* the interface — its tests drive the whole flow against a stand-in built
from that document alone.

Clients that can send their own headers (like Claude Code) skip OAuth entirely: paste a
SoulIdentity API token as the bearer. Same door, same admission edge.

## Running the door

One binary, `soulstream-node`, config small enough to read at a glance:

```sh
soulstream-node \
  --listen 127.0.0.1:8080 \
  --public-url https://node.example.com \   # front it with HTTPS; enables OAuth
  --issuer https://auth.example.com \       # your authorization server
  --realm workshop \
  --nats-url nats://your-server:4222 \
  --sentinel-creds ~/.config/soulstream-node/sentinel.creds
```

Front it with an HTTPS terminator (`tailscale serve`, Caddy, nginx — the node never
terminates TLS itself). Omit `--public-url` for a loopback/local door with static
bearers only (the shape the [single-binary house](https://github.com/impire-io/soulnode)
embeds). Restart it whenever: it holds **no state** — no tokens, no keys, no
per-user secret on disk — so a restart loses nothing a re-presented badge can't rebuild.
The realm must already be provisioned; the node never provisions. The full walkthrough,
including the operator's admission-edge prerequisites, is the
[018 quickstart](../specs/018-remote-mcp-node/quickstart.md).

This is proven, not hypothetical: on a live Synadia Cloud deployment a prototype node
admitted a client through auth callout and posted a signed, correctly-attributed turn —
end to end, over an HTTPS front door — and the shipped node's test rig runs the whole
admission edge (auth callout + identity plane) in-process, verifying every claim above.
The design and its measured evidence live in
[the remote-MCP-node design](../hq/02-DESIGN/extensions/remote-mcp-node.md) and
[journey episode 0008](../hq/04-JOURNEY/0008-remote-mcp-node.md).

## Related

- [MCP quickstart](./mcp-quickstart.md) — the local install this page is the
  exception to.
- [The MCP adapter](./mcp.md) — what any door, local or remote, opens onto.
- [Signing](./signing.md) · [The persona directory](./persona-directory.md) ·
  [Operators](./operators.md)
