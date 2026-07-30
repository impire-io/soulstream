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

That is the **remote node**. It is designed but **not built yet** — it's the next
feature cycle, and this page describes the design so you know what's coming and what
to wait for.

## How the designed door works: show a badge

Everything follows from one rule: **you show a badge, and the badge is checked by the
badge office — never by the door.**

- **The badge** is a bearer token in the HTTP header your MCP client already knows how
  to send: an **Entra access token** (your organisation's normal sign-in) or a
  **SoulIdentity API token** (for clients that can't do the OAuth dance). One header,
  either shape.
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
twenty-three buttons — attributed, signed, revocable.

## What still needs building

Being precise about the gap, since most of the hard parts already exist:

- **Shipped on the SoulIdentity side**: the badge office (auth callout with both
  token lanes), one-identity-one-persona, keys materialising on first touch, custody
  signing that satisfies Soulstream's [signer seam](./signing.md) directly, and the
  cross-service proof that a record signed there verifies here.
- **The node itself** (Soulstream's next cycle): the HTTP door, the per-user
  connection pool, reading keys from the identity plane, re-proving on token expiry.
- **The connector-friendly edge**: the OAuth discovery handshake hosts like claude.ai
  expect (API-token badges are the fallback while that settles).
- **On SoulIdentity's plate**: a home for persona *presentation* (display name,
  [operator attestations](./operators.md)) beside the keys, so identity-plane realms
  don't lose what the realm phone book carries today.

The full pre-registration — bets, open questions, and what would change the design —
lives in the research topic in [`hq/01-RESEARCH/`](../hq/01-RESEARCH/).

## Related

- [MCP quickstart](./mcp-quickstart.md) — the local install this page is the
  exception to.
- [The MCP adapter](./mcp.md) — what any door, local or remote, opens onto.
- [Signing](./signing.md) · [The persona directory](./persona-directory.md) ·
  [Operators](./operators.md)
