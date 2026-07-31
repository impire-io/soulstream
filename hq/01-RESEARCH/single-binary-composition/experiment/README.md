# soulnoderig — the composition rig (throwaway)

The discriminating experiment for the `single-binary-composition` topic: the
whole first-boot ceremony as code (`provision.go`, no config file, no `nsc`,
no manual step), booting an embedded operator-mode NATS server with
auth-callout admission and the SoulIdentity service + callout issuer
in-process, then exercising the pre-registered Bar 1 observations in the
SoulNode shape (`DontListen` + `nats.InProcessServer`) against a TCP control
arm.

**The module path is a deliberate dodge:** it sits under
`github.com/impire-io/soulidentity/` so importing soulidentity's `internal/`
service/callout/vault packages is legal — the same consumer-position trick
soulstream's `remote-mcp-node` experiment used. SoulNode proper cannot do
this (constitution I); needing it here is a Bar 2 reading, not a precedent.
Never published; removed at graduation. The `replace` directive assumes the
sibling working tree at `../../../../../soulidentity`.

Run:

```
go test -v ./...
```

~2 minutes: every *refused* in-process connect blocks ~10 s (that finding is
the point — see `timeout_probe_test.go` and the topic `JOURNEY.md`).
