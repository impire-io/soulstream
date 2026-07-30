# Can a badge-passthrough node give a no-install client full realm membership?

**State:** active
**Started:** 2026-07-30

## Abstract

Some MCP hosts cannot install a binary at all — claude.ai custom connectors,
sandboxed Claude Desktop (observed failing to install `soulstream-mcp` in its
sandbox), locked-down machines. For them the MCP doorway must stand at the
workshop: a shared **remote node** reached by URL. The design direction is
settled across the two repos (SoulIdentity journeys 0013–0016, our 017 seam):
the node passes the caller's bearer token through to NATS, SoulIdentity's auth
callout admits the connection, persona == principal, signing stays in
SoulIdentity custody, and readers build keyrings from the identity plane —
the node holds no creds and no keys. What is *not* settled is whether that
direction survives contact with the deployments and clients we actually have:
callout availability where realms live, token expiry against long MCP
sessions, and the OAuth edge hosted connectors require. A decisive answer
unlocks the 018 build (and closes SoulIdentity's M2 node-half gate); a
refutation redirects it before any node code exists. Consumer-facing sketch:
[docs/mcp-remote.md](../../../docs/mcp-remote.md).

## The question

Can a shared MCP node that validates nothing itself — bearer-token passthrough
to SoulIdentity's auth callout, one pooled NATS connection per user, no
node-held credentials or keys — give a client that cannot install anything
full, attributed, signed realm membership on a deployment class we actually
run?

## Pre-registered bars

- **Bar 1 — admission by badge, both lanes.** Protocol: a prototype node
  (Streamable HTTP via the go-sdk handler, bearer → `nats.Token` passthrough)
  against the operator-mode callout rig (the soulidentity `e2e/` pattern).
  Pass: an MCP `tools/call` carrying (a) a SoulIdentity API token and (b) an
  OIDC access token from the Entra rig each yields a posted turn whose
  attribution equals the token's principal user; a revoked and a garbage
  token each draw a refusal at the NATS edge with no realm write; the node's
  configuration contains a URL and a realm name and nothing credential-shaped.
- **Bar 2 — custody end to end through the node.** Protocol: same rig; the
  persona key exists nowhere but the vault (materialised on first touch).
  Pass: a turn posted through the node reads `SigVerified` by a reader whose
  keyring comes from one `keys.public` answer — with the negative control
  (no keyring → `unknown-key`) showing the verdict is earned; grep of the
  node process's environment and filesystem shows no seed, no creds file.
- **Bar 3 — expiry and revocation against a living session.** Protocol: the
  5s-TTL callout rig; an MCP session older than the TTL keeps calling tools
  while the client presents a refreshed bearer; then the token is revoked
  mid-session. Pass: refreshed-badge calls keep succeeding across at least
  3× TTL with at most the single in-flight call failing per re-proof;
  post-revocation writes are refused within 2× TTL (soulidentity measured
  5.25s on this rig — the node must not widen that bound).
- **Bar 4 — a genuinely no-install client connects.** Protocol: node exposed
  over HTTPS; attempted from a hosted MCP client that cannot run binaries
  (claude.ai custom connector and/or Claude Desktop remote connector), trying
  the API-token bearer first, then the Entra OAuth discovery path (RFC 9728
  protected-resource metadata, `401` + `WWW-Authenticate`). Pass: at least
  one badge lane completes `initialize` → `tools/list` →
  `soulstream_board` → `soulstream_post_turn` end to end; the README records
  *which* lane(s) worked and the exact friction of the one(s) that did not.

## Reversal condition

Two observable readings would refute the direction rather than the details:
the deployment class the realms actually live on cannot host the badge office
(observable: auth callout never firing for a connection on that class — e.g.
Synadia Cloud/NGS account limitations recorded from the operator portal), which
would force verification into the node itself — a different trust class than
"the server is the verifier" and grounds to redesign, not persist; or no badge
lane in Bar 4 passes from any no-install client (observable: both lanes
recorded failing with their friction), which removes the population this node
exists to serve and sends the effort back to making local installs viable in
sandboxes instead.

## Verdict

_Empty until graduation (`/research-graduate remote-mcp-node --to
design|artifact|abandoned`)._
