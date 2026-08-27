# Tasks: Tenants in the house (012)

All complete 2026-08-27.

- [x] T001 Pin identity at the pseudo-version carrying the `accounts.*`
      client surface and `KindNATSOperatorKey` (95a8d9e).
- [x] T002 `node/node.go`: dir resolver under `<state>/resolver`,
      seeded create-if-absent from the founding JWTs; operator claims
      synthesized in memory (`trustedOperator`) — the dir resolver
      refuses bare TrustedKeys.
- [x] T003 `node/node.go`: `connectSys` (in-memory SYS user) +
      `SystemConn` into `embed.Options` (nil on BYO, ops off honestly).
- [x] T004 `node/node.go`: operator-key ensure at start
      (`operator/root`, read-first, import when absent).
- [x] T005 `cmd/soulstream/account.go`: `account
      create|list|show|suspend|resume` over the sealed ops; usage text.
- [x] T006 `node/tenancy_test.go`: the 012 gate — usable admission,
      restart survival, suspend/resume, audit line.
- [x] T007 `make check` fully green; M1.1 gate re-verified on the dir
      resolver.
