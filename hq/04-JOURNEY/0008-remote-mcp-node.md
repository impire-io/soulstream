# Episode 0008 — The remote MCP node: a URL into the realm, proven on the BYON (2026-07-30 → 08-01)

**The question:** can a shared MCP node that validates nothing itself —
bearer-token passthrough to SoulIdentity's auth callout, one pooled NATS
connection per user, no node-held credentials or keys — give a client that
cannot install anything full, attributed, signed realm membership on a
deployment class we actually run? It exists because Claude Desktop's sandbox
was observed failing to install `soulstream-mcp`: when the door can't come to
the machine, the door must stand at the workshop. Four pre-registered bars, a
prototype node (~250 lines over the go-sdk's `StreamableHTTPHandler`), and a
rig that ran the one combination neither soulstream nor soulidentity had run
before — **callout admission with a scope template spanning both subject
spaces** (SoulIdentity user ops beside the Soulstream realm).

**Bars 1–3: PASS** [measured, local operator-mode rig]. Both token lanes
admit through the node with attribution read back *from the realm*, not the
node's word — `daan-ext` for the API-token session, the oid for the
Entra-shaped one; garbage and revoked tokens form no session and write
nothing (Bar 1). A turn reads `unknown-key` without a keyring and
`SigVerified` from one `keys.public` answer, the node holding no key material
(Bar 2). On a 5 s TTL, 33/33 writes across 3× TTL with zero failures on
refreshed badges, and IdP-side revocation biting at +3.9 s inside the 2× TTL
bound (Bar 3). The investigation's one scare — writes "succeeding" 20–30 s
after revocation — was **a test bug, not a design failure**: MCP tool errors
are in-band (`result.IsError`), and the loop had counted transport success as
call success; the audit showed signing had stopped dead at the refusal
[measured]. Lesson banked for every MCP harness we write.

**The reversal condition — the topic's central bet — did NOT trigger; it is
CLOSED** [measured, BYON]. The operator stood up a Synadia Cloud BYON
(nats-server 2.12.7, context `impire-dev-platform`), and via the Cloud API
(`control-plane-sdk-go`, scripted in `cmd/byon-setup`) auth callout enabled,
wired, and *fired*: the whole stack ran live — `soulidentity serve` against
the BYON, the node, and `cmd/probe` driving the pass protocol
(`initialize → whoami: daan@… → board → start_topic → post_turn`), the
reader verifying `SigVerified` from one `keys.public` read — including through
a `tailscale serve` HTTPS front door. Verification stays in the server, never
the node; the "force verification into the node, redesign not persist" trigger
did not fire. Two Cloud-API facts earned along the way: **programmatic
signing-key groups return the seed exactly once** — the sanctioned custody
export that lets soulidentity's vault hold the key (killing a console-only
worry), and the platform **refuses NATS users under programmatic groups** (the
issuer needs an on-demand group).

**Bar 4 — a genuinely no-install client: PARTIAL, and this is the finding that
shapes 018** [measured, screenshot]. The transport path passes end to end from
a real MCP client library. But the *hosted* client — the population the node
exists for — hit a wall: **Claude Desktop's Add-custom-connector dialog
exposes only OAuth** (Client ID/Secret, else Dynamic Client Registration).
There is **no static-header field**, so the `sit_` bearer lane, the cheapest
path, is unreachable on that surface. A no-install hosted client therefore
*requires* the OAuth lane — validating why the node already serves RFC 9728
resource metadata and 401 challenges, and turning "which authorization server"
from a nicety into a first build decision.

**What it opened / reversed.** The node reframes the adapter model of
[`../02-DESIGN/extensions/library-and-adapters.md`](../02-DESIGN/extensions/library-and-adapters.md):
that Layer-2 adapter is defined as "a credential custodian… holds the
credentials of the personas it fronts," and the remote node is the first
adapter that holds **no credentials at all** — callout supplies each session's
identity, so the node multiplexes personas *across connections* without ever
custodying one. Also reversed in passing: my 2026-07-31 claim that static
bearer headers work on Claude Desktop connectors — true for *local* MCP config
and claude.ai's enterprise connector *API*, false for the hosted
add-a-remote-connector UI. Also surfaced: a real node is *always*
proxy-fronted (the go-sdk's DNS-rebinding guard 403s a loopback server behind
tailscale until told the proxy shape is intended), and OIDC-lane personas are
oids — human display names survive only in the audit, making a
persona-presentation op on SoulIdentity's surface load-bearing for readable
boards. The design graduates to
[`../02-DESIGN/extensions/remote-mcp-node.md`](../02-DESIGN/extensions/remote-mcp-node.md);
the prototype, rig, `byon-setup`, probe, and OAuth edge carry into the 018
build (recoverable from git — the experiment module lived at
`hq/01-RESEARCH/remote-mcp-node/experiment/`, last at the pre-graduation
commit).

Reversal condition: the node design assumes callout is the verifier and the
node holds nothing — reopened if a target deployment cannot host callout
(observable: callout never firing for a connection on that class, as
Synadia Cloud BYON was the test and passed), which would force verification
into the node, a different trust class; or if no OAuth authorization server
can emit the `oid`/`roles` claims the callout OIDC lane needs while completing
Claude's DCR+PKCE flow (observable: the 018 OAuth spike recording both a
node-embedded AS and an external IdP failing the hosted connector), which
would send the no-install effort back to making sandbox installs viable.

Trail: [`../02-DESIGN/extensions/remote-mcp-node.md`](../02-DESIGN/extensions/remote-mcp-node.md),
[`../03-IMPLEMENTATION/ROADMAP.md`](../03-IMPLEMENTATION/ROADMAP.md) (018),
[`../../docs/mcp-remote.md`](../../docs/mcp-remote.md),
[`../../docs/mcp-quickstart.md`](../../docs/mcp-quickstart.md); the concluded
research topic (removed on graduation, full history in git — commits
`ba8f47c` … the graduation commit); SoulIdentity journeys 0013–0016 (the
identity-plane side) and its ROADMAP M2.
