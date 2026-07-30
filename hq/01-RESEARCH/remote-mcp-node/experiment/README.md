# The prototype node and its rig

Part of the [remote-mcp-node](../README.md) investigation's permanent trail —
committed so the rig survives to Bar 4 and into the 018 cycle; it is **not**
product code.

- `node.go` — the prototype node: Streamable-HTTP MCP server, bearer
  passthrough, one pooled NATS connection per user, principal derived from the
  server's own user-info answer, signing via `client.PersonaSigner`.
- `rig_test.go` — the operator-mode rig: AUTH (external authorization + xkey)
  and one team account (ACME) hosting both the SoulIdentity service and the
  realm, with the scope template spanning both subject spaces.
- `bars_test.go` — Bars 1–3 as tests (`go test ./...`, ~25 s; Bar 3 writes
  `bar3-audit.log` beside itself, untracked).

A separate module in the consumer position (imports both repos; the cycle
guard). The `replace` directives assume the sibling checkout layout
(`soulstream/` and `soulidentity/` side by side). Findings and honest numbers:
[../JOURNEY.md](../JOURNEY.md).
