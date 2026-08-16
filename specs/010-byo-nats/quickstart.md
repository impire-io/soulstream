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

## The live run — 2026-08-16, measured

Realm `byon` founded on the Impire DEV system's BYON (agent-connected,
nats-server 2.12.7) with the released binary + the fix stack below:
callout ADMITTED the founding persona on Synadia's infrastructure and
REFUSED a garbage token (audited); `up` served every plane against the
substrate; `init` re-run reported the verified no-op (13 artifacts);
the custody audit passed (no master material, no PAT on disk); the
after-state diff showed the system's two pre-existing callout configs
byte-identical and exactly one new config, two new accounts — additive,
as designed. `sealed_requests=false`: the platform set no callout
xkey, so the unsealed-callout caveat is now a measured fact.

What first contact taught (each measured live, fixed on main, and
replayed as a test):

1. The default state dir collides with the soulstream client's config
   dir — named refusal now (`clientdir_test.go`).
2. `--synadia-system` must take the id as well as the name.
3. The synadia awaiting state (no seeds yet) must load, or a founding
   interrupted mid-account-half cannot resume.
4. A programmatic seed must be persisted the moment it is returned
   (`OnSeed`) — a mid-run failure orphaned a group whose seed died
   with the process; recovery required disable-then-delete (the
   platform refuses deleting an active group).
5. The channel to an agent-connected BYON is lossy — the private-link
   idle watchdog cycles the tunnel while requests are in flight
   (~50% of mutations drew 500 "nats: timeout") — so every driver
   mutation retries bounded through 5xx, list-first.
6. The platform's callout surface is anti-idempotent: re-enabling and
   re-adding a target or user draw a persistent 500 "an unexpected
   error occurred" — enable and wiring are therefore read-first.
