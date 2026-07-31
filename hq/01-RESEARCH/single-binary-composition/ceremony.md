# The first-boot ceremony, enumerated (Bar 3)

Every artifact `soulnode init` must generate, in dependency order. This list
and [`experiment/provision.go`](experiment/provision.go) agree 1:1 — the
`Inventory` variable there names the same entries, and the rig proves the
whole sequence executable from an empty directory with zero manual steps and
zero external binaries [measured: `TestBar3CeremonyFromEmptyDir`,
`TestBar1InProcess` — admission works at the end of it].

## Trust root

1. **Operator nkey.** The deployment's trust root. The embedded server
   trusts it via `server.Options.TrustedKeys` — no operator JWT string, no
   config file. The seed must persist in the state dir (everything below is
   signed by it or by keys it endorses).

## Accounts (JWTs stored in the server's account resolver)

2. **System account** (`SYS`): nkey + JWT signed by the operator;
   `server.Options.SystemAccount`. Grants `$SYS.REQ.USER.INFO` answers — the
   server-asserted whoami the front door depends on.
3. **AUTH account**: nkey + JWT signed by the operator, carrying the
   admission machinery:
   - `EnableExternalAuthorization(<issuer user pub>)` — routes connections
     to the callout;
   - `Authorization.AllowedAccounts` — the account(s) callout may admit
     into (the realm account);
   - `Authorization.XKey = <callout xkey pub>` — seals auth
     requests/responses;
   - a **signing key** whose seed enters the vault as `auth/issuer` — it
     signs every admitted-user JWT and the sentinel.
4. **Realm account**: nkey + JWT signed by the operator:
   - JetStream limits (unlimited locally);
   - a **scoped signing key** added via `SigningKeys.AddScopedSigner` whose
     `jwt.UserScope.Template` is the admitted persona's permission set —
     `soulidentity.{{account-subject()}}.{{name()}}.sign.record` /
     `.keys.public`, `soulidentity.status`, `soulidentity.xkey`,
     `SOULSTREAM.>`, `$JS.API.>`, `$KV.>`, `$O.>`, `$SYS.REQ.USER.INFO`
     (pub) and `_INBOX.>`, `SOULSTREAM.>` (sub). The template's expansion
     *is* the persona's server-asserted identity. Its seed enters the vault
     bound to the realm account.

The resolver is `server.MemAccResolver`, populated in code — the rig proves
the callout + scoped-key machinery needs no config file [measured]. A real
SoulNode persists account JWTs in the state dir and re-populates the
resolver on boot (or moves to a directory resolver — a design-time choice).

## Users (the bypass lane — creds that exist before admission works)

5. **AUTH issuer user**: nkey + JWT signed by the AUTH account key — the
   callout issuer's own connection.
6. **Realm service user** and **realm ops user**: JWTs signed by the realm
   account key (full-account users, no scope). The service user runs the
   SoulIdentity service; the ops user performs the founding administrative
   acts below. These are the "creds bypass lane" — SoulNode holds them
   internally; they never leave the state dir.

## Curve keys (x25519)

7. **Callout xkey**: public half in the AUTH JWT (step 3), seed to the
   callout issuer.
8. **Vault first key**: seals the vault at rest (`SOULIDENTITY_VAULT`).
9. **Service surface xkey**: seals the service's request/reply surface.

## JetStream state

10. **KV buckets** `SOULIDENTITY_VAULT` (the sealed vault) and
    `SOULIDENTITY_TOKENS` (API-token digests), created on the service
    connection.

## Founding administrative acts (through the public client, as ops)

11. **Import the realm scoped signing key** into the vault
    (`kind nats-account-signing-key`, bound to the realm account) — callout
    minting depends on it.
12. **Import the AUTH signing key** as `auth/issuer` — admitted-JWT and
    sentinel signing depend on it.
13. **Mint the sentinel** — the public, deny-all bearer creds artifact
    every client presents alongside its token; written to the state dir
    (the one ceremony artifact that is *meant* to be handed out).
14. **Create the first API token** (`sit_…`) + its revocation digest — what
    the user pastes into a Claude client.

## What `soulnode init` adds beyond the rig

The rig regenerates everything per run; `init` must also **persist** (seeds,
account JWTs, sentinel, state dir layout) and be **idempotent** on an
existing state dir — re-running never regenerates a trust root that
JetStream state already depends on. That behavior is Phase 1 (M1.2) design
work, not ceremony discovery.
