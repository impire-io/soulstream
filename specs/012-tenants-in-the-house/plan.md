# Implementation Plan: Tenants in the house (012)

## Summary

The product's half of hq platform-topology (episode 0133/0134): the
upstream `accounts.*` family (identity ≥ 447ec6b — D47, tenants born
admissible) becomes reachable and durable in the house. Four touches,
all composition: the embedded server's resolver becomes a dir resolver
under the state directory (runtime tenants persist), the identity
plane gains its `SystemConn` (a SYS user and the operator claims both
synthesized in memory from ceremony material — nothing new on disk),
the operator key enters the vault as an ensure at start, and a
`soulstream account` command drives the sealed ops.

## Constitution Check

- **I — Composition, not invention**: PASS. Tenancy logic, the scoped
  signer, and the AUTH coupling live upstream (identity D47); this
  repo wires connections, seeds a resolver, and adds a CLI verb over
  the public client. The identity pin moves to the pseudo-version
  carrying `accounts.*` on the client (the standing-exception pattern)
  until the next upstream tag.
- **II — Same shape as any deployment**: PASS. Operator mode, callout
  admission, dir resolver — exactly what a hosted multi-account
  deployment runs; no dev-only lane anywhere.
- **III — One process, planes by configuration**: PASS. No new plane,
  no new listener; the SYS connection is one more ordinary loopback
  connection, absent on BYO by configuration of the flavour.
- **V — No manual key step**: PASS. The SYS user and operator claims
  are minted in memory per start; the operator-key vault import is an
  ensure the node performs itself.

## Decisions taken in-cycle (design 0001 amendments, propagated to hq)

1. **Dir resolver, always, embedded flavour** (§3 amendment): the
   `MemAccResolver` forgot runtime tenants at shutdown; the dir
   resolver under `<state>/resolver` persists them. Seeding is
   create-if-absent — never overwrite — because the runtime amends
   stored JWTs (AUTH learns each tenant, D47) and re-seeding founding
   shapes would silently unlearn them.
2. **In-memory synthesis over new artifacts**: the SYS user JWT and
   the operator claims the dir resolver requires are built per start
   from seeds the ceremony already persists. No `users/sys.creds`, no
   operator JWT file, nothing to migrate.
3. **Ensure, not founding-only** (F1's posture): the operator key
   reaches the vault at start when absent, so realms founded before
   tenancy gain the capability on their next `up`.

## Test evidence

`node/tenancy_test.go` (the 012 gate): create → usable admission
8.8ms; restart on the same state → tenant still admits and resolves;
suspend refuses, resume restores; audit carries `accounts.create`.
`TestM11Gate` unchanged and green on the dir resolver.
