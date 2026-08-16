# Quickstart: BYO NATS (010)

## Automated coverage (runs in `make test`)

- `ceremony/byo_test.go` — phase-1 material, round-trip, kit content
  (no secrets, byte-identical regeneration), the §6 refusals, the
  state-dir custody scan.
- `node/byo_test.go` — the rig plays the operator: a stock nats-server
  stood up from a CONFIG FILE in operator mode, its account JWTs
  authored from nothing but the kit's public keys; full founding, M1.1
  semantics through the callout, restart, custody audit (by file name
  and by seed content); conf-auth and partial-kit named refusals.
- `cmd/soulstream/byo_test.go` — the two-phase CLI flow and its flag
  refusals.
- `internal/synadia/synadia_test.go` — the driver's sequence against a
  control-plane API stub: creation, idempotence-by-name, the
  lost-seed refusal.

## The kit's nsc sequence — validated against real nsc [measured 2026-08-16]

The command shapes the kit emits were run verbatim against nsc
(2.8.x, local sandbox): `nsc add account`, `nsc edit account
--js-*-storage -1 --sk <pub>`, `nsc edit signing-key --account …
--sk <pub> --role soulstream-user --allow-pub '<scope>' --allow-sub
'<scope>'` (the `{{account-subject()}}`/`{{name()}}` templates
survive), `nsc edit authcallout --account … --auth-user <pub>
--allowed-account <pub> --curve <xpub>`, and `nsc describe account
<name> --field sub | tr -d '"'` for the hand-back. All `[ OK ]`.

## Manual runbook: the live Synadia Cloud run (SC-004's live half)

The stub proves the sequence; the platform's behavior was measured once
in journey 0038 (remote MCP node) — this runbook re-proves it for the
full product founding when an operator next has a BYON at hand. Never
part of `make test` (the Entra-lane precedent, identity spec 001).

1. Mint a PAT in Synadia Cloud (Personal Access Tokens); note the
   system name and the BYON's client URL.
2. `SOULSTREAM_SYNADIA_TOKEN=uat_… soulstream init --state /tmp/byon \
   --byo synadia-cloud --url nats://<byon>:4222 --synadia-system <name> \
   --realm <realm>`
3. Expect: the two accounts `soulstream-<realm>`/`soulstream-<realm>-auth`
   in the console; three programmatic signing-key groups + one on-demand;
   the issuer user; callout enabled with the AUTH account as control
   account; the founding token printed once; the callout smoke round
   passing (a garbage token refused).
4. If the platform sets a callout xkey, the founding says so out loud
   and the callout runs unsealed on our side — record what the console
   shows; this is the one seam 0038 flagged and no automated test can
   reach.
5. `soulstream up`, then the M1.1 observations by hand: sentinel +
   token admits, `soulstream-shell` reachable, a turn posts.
6. Re-run step 2 — it must report the resources as existing and create
   nothing (idempotence on the live platform).
