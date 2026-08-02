# Episode 0006 — The door opens: the MCP front door lands (2026-08-02)

Phase 2's local mode shipped the same day its gate opened. Upstream,
soulstream's 018 landed and tagged **v0.7.0** — the remote MCP node
(streamable HTTP in, bearer passthrough to callout admission, one pooled
connection per admitted principal, the public `mcpserver` tool surface
out, custodying nothing) — and the maintainer directed the
incorporation. One consumability step was needed first, the by-now
familiar one: the node module's landing-day `replace => ../` dropped
upstream (soulstream journey 0010 — the tag *is* the change-set its
comment worried about), making the module requireable; SoulNode is its
first consumer, at a pseudo-version until `node/vX` tags (the fourth
tracked pin).

The wiring is deliberately thin (`specs/004-the-front-door/`,
constitution I): `config.json` gained its second plane block
(`planes.door {enabled, listen}`, default `127.0.0.1:8080`, loopback
checked, `init --door-listen` on the founding run); the node holds the
HTTP listener (bind conflicts refuse named, tests get real ports) and
mounts `door.New(Config).Handler()` — local mode, static bearers, the
existing sentinel, the audit logger. The door shuts first at `Stop`, its
pool closing before the connections beneath it. `init` now ends by
telling the owner where to point an MCP client, next to the token.

Measured [the e2e rides `make test`]: a real MCP client (the go-sdk
streamable transport, upstream's own dial pattern) presenting the
founding token forms a session at the door, lists the full tool surface,
and `soulstream_whoami` answers with the *realm-admitted* owner persona
— nothing client-claimed anywhere; a garbage bearer cannot form a
session; the state directory is byte-count identical before and after
the door serves (it custodies nothing — SC-003 asserted); the disabled
arm runs Phase 1 exactly, no HTTP listener at all. The whole vision
sentence is now executable: `soulnode init && soulnode up`, paste the
token into a client, start working.

Out of scope, named: public mode (the OAuth resource metadata, external
authorization server) waits on soulfold upstream — the plane block grows
those fields additively when they exist. HTTPS on a tailnet stays the
deployment story (`tailscale serve` in front of the loopback door);
Phase 3's embedded tsnet keeps its measurement gate.

Refuted/reversed: nothing — the upstream surface fit the seam design
0001 §8 drew, unchanged.

Reversal condition: none — records a completed build (the pin keeps its
flip condition; public mode is upstream-gated, not decided here).

Trail: `specs/004-the-front-door/`; `ceremony/`, `node/node.go`
(startDoor), `cmd/soulnode`; soulstream v0.7.0 + its journey 0010;
design 0001 §8 as-built. Commits: the `004-the-front-door` branch,
merged to main 2026-08-02.
