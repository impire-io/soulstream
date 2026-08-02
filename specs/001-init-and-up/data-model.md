# Data Model — 001-init-and-up

## The state directory (the realm's physical home)

```text
<state>/                     0700
├── config.json              0600   {"listen": "127.0.0.1:4222"}
├── keys/                    0700
│   ├── operator.nk          0600   operator seed (trust root)
│   ├── sys.nk / sys.jwt     0600   system account seed + operator-signed JWT
│   ├── auth.nk / auth.jwt   0600   AUTH account (external authorization,
│   │                               allowed accounts, callout xkey pub)
│   ├── auth-signing.nk      0600   AUTH signing key seed (vault: auth/issuer)
│   ├── realm.nk / realm.jwt 0600   realm account (JetStream limits, scoped
│   │                               signing key with the persona template)
│   ├── realm-signing.nk     0600   realm scoped signing key seed
│   ├── callout.xk           0600   callout curve seed (pub is in auth.jwt)
│   ├── vault-first.xk       0600   vault first key seed
│   └── surface.xk           0600   service surface curve seed
├── users/                   0700
│   ├── service.creds        0600   identity-plane service connection
│   ├── issuer.creds         0600   callout issuer connection (AUTH account)
│   └── ops.creds            0600   founding/ops acts connection
├── sentinel.creds           0600   deny-all bearer; the founding-complete
│                                   marker (written LAST); public by design,
│                                   0600 anyway — distribution is deliberate
└── jetstream/               0700   the message store (server-managed)
```

Not on disk, ever: the first token's plaintext (printed once; digest
lives in the identity plane's token bucket inside `jetstream/`).

## Ceremony (in-memory, `ceremony.State`)

One struct holding the parsed form of everything above: operator keypair,
three account (seed, JWT) pairs plus the two signing-key seeds, the three
curve seeds, the three user creds. `Generate(listen string)` fills it;
`Save(dir)` persists with the modes above; `Load(dir)` re-parses;
`Verify(dir)` = Load + the invariant checks below.

**Verification invariants** (first failure named in the error):

1. Every inventory path exists with the stated mode (or stricter).
2. Seeds decode as their kind (operator/account/user/curve).
3. JWTs parse; each account JWT's subject matches its seed's public key;
   the AUTH JWT carries external authorization and the callout xkey
   matching `callout.xk`'s public half; the realm JWT carries the scoped
   signing key matching `realm-signing.nk`.
4. `config.json` parses and `listen` is a host:port on a loopback host.
5. Completion: `sentinel.creds` present ⇒ complete; keys present without
   it ⇒ "incomplete ceremony" refusal (R4).

## Node configuration and lifecycle

`node.Config`: state dir + the loaded `ceremony.State` + listen address.
Lifecycle: `Start` → server ready (loopback listener) → identity plane
(`embed.Run`, service + issuer connections from `users/*.creds`) → ready
(status answers) — then `Stop` (planes drained via ctx, server shutdown,
connections closed; state dir untouched and reusable).

States: `created → serving → stopped`; `Start` failures name the failing
stage (bind conflict, vault-key mismatch, plane startup) and leave no
goroutines behind.
