# 0001 — The SoulNode composition

**Status**: graduated from `single-binary-composition` (journey
[episode 0002](../04-JOURNEY/0002-the-composition-gate.md)), 2026-08-02.
Governs Phase 1 of the [roadmap](../03-IMPLEMENTATION/roadmap.md); the
front door (Phase 2) is seamed here but designed later.

Validation tags per the [design README](README.md): **[V]** validated by
the composition rig or a delivered upstream surface, **[D]** specified but
not yet run in SoulNode itself, **[O]** interface fixed, internals open.

## 1. The shape [V]

One binary, one process, five planes — each a Go library consumed through
its public, tagged surface (constitution I), each talking to NATS over an
**ordinary client connection, loopback by default** (constitution III as
ratified). There is no in-process transport anywhere: the bundle is the
deployment where every URL happens to be `127.0.0.1`, which is exactly
what makes decomposition configuration rather than architecture.

| Plane | Surface consumed | Runs as |
|---|---|---|
| Server | `nats-server/v2/server` (embedded, operator mode, JetStream) | goroutine owning the listener |
| Identity | `soulidentity/embed.Run` + `soulidentity/client` | goroutine on its two conns |
| Memory | `soulstream-archivist/keeper` + `archive`, `soulstream/topic.RespondMemory` | goroutine on a realm client |
| Runtime | `soulrealm/{runner,backend/native,minter,declaration}` + `soulstream/{realm,topic}` | goroutine supervising child processes |
| Front door | soulstream MCP surface (Phase 2 — see §8) | goroutine owning the HTTP listener |

Workloads never run in the process: the runtime plane launches them
through soulrealm's backends (native first), and a workload death is an
op on the topic, never a node crash.

## 2. Configuration: planes are declared, connections are URLs [D]

Every plane carries the same connection block:

```
{ enabled: bool, url: string (default nats://127.0.0.1:<port>),
  creds: path (default: minted into the state dir) }
```

- **All defaults** → the bundle: embedded server enabled, every plane on
  loopback with state-dir creds. (As-built from 002: `config.json`
  carries `listen`, `realm` — fixed at founding — and
  `planes.memory.enabled`; each block gains `url`/`creds` when a plane
  can actually be pointed elsewhere.)
- **BYO NATS** → embedded server `enabled: false`, every URL points at
  the user's server. The ceremony's account half runs against that server
  (see §4's [O]).
- **Remote plane** → that plane `enabled: false` here, enabled on the
  other machine with a non-loopback URL. Same binary, same config schema,
  both sides.

A plane MUST NOT behave differently because its URL is or is not
loopback. The embedded server binds `127.0.0.1` only, port configurable
(default the NATS conventional port; refusing to start on a bind conflict
with a message naming the config key).

## 3. The embedded server [V]

Provisioned entirely in code — `server.Options` + `MemAccResolver`, no
config file, no `nsc` [measured, the rig]: `TrustedKeys` = the operator,
`SystemAccount` = SYS, resolver preloaded from the persisted account JWTs,
JetStream on `<state>/jetstream`. Boot order: server ready → identity
plane → realm provisioning → memory + runtime planes → front door.

## 4. The ceremony and the state directory [V for generation, D for persistence]

`soulnode init` generates, in dependency order, the inventory the research
enumerated and the rig proved executable from an empty directory
[measured]:

1. **Operator nkey** — the trust root (`TrustedKeys`).
2. **SYS account** nkey + JWT (grants `$SYS.REQ.USER.INFO` — the
   server-asserted whoami everything downstream leans on).
3. **AUTH account** nkey + JWT: `EnableExternalAuthorization(<issuer
   user>)`, `Authorization.AllowedAccounts` = the realm account,
   `Authorization.XKey` = the callout curve key; plus its **signing key**
   (vault name `auth/issuer` — signs admitted users and the sentinel).
4. **Realm account** nkey + JWT: JetStream unlimited locally, plus a
   **plain workload signing key** (`SigningKeys.Add` — the runtime
   minter's key; per-workload permissions ride in the JWTs it signs, a
   scoped key rejects them; as-built in 003) and the
   **scoped signing key** whose `jwt.UserScope` template *is* the admitted
   persona's permission set — `soulidentity.{{account-subject()}}.{{name()}}.sign.record`
   / `.keys.public`, `soulidentity.status` / `.xkey`, `SOULSTREAM.>`,
   `$JS.API.>`, `$KV.>`, `$O.>`, `$SYS.REQ.USER.INFO` (pub);
   `_INBOX.>`, `SOULSTREAM.>` (sub).
5. **Bypass-lane users**: AUTH issuer user; realm service, ops, and
   archivist users (account-key signed). These never leave the state
   dir. (The archivist's entry is transport only — its *persona* signing
   key is vault-held, materialized on first touch; as-built in 002.)
6. **Curve keys**: callout xkey (public in the AUTH JWT, seed to the
   issuer), vault first key, service surface key.
7. **Buckets** `SOULIDENTITY_VAULT` + `SOULIDENTITY_TOKENS`; **realm
   provisioning** (`realm.NewClient` + `Provision` — stream, notify,
   personas, objects).
8. **Founding acts through the public `client` over the node's own
   loopback connection** [measured — no in-process admin API exists
   upstream, and none is wanted]: import the realm scoped signing key and
   the AUTH signing key into the vault, mint the **sentinel** (public by
   design), create the **first API token** — the one secret `init` prints.

**Persistence and idempotence** [D — the delta beyond the rig]: seeds,
account JWTs, sentinel, and config persist under the state dir (`0700`
dirs, `0600` files); `init` on a non-empty state dir verifies and reports
— it MUST NOT regenerate a trust root that JetStream state already
depends on. Moving the realm to a new machine is copying the state dir
(vision: day 2). Custody follows soulidentity's D13: raw seeds on the
node's own disk are the accepted trust class, stated without euphemism.

**[O]** — the BYO-NATS ceremony subset: against a user-supplied server,
which of steps 1–4 apply (their operator, their accounts) and what
SoulNode refuses to touch. Interface: the config block above; default:
BYO mode requires the accounts to exist and only runs steps 5–8. The
exact split needs its own pass before BYO ships.

## 5. Admission [V]

Sentinel creds + `sit_` token; the callout issuer mints a TTL-bounded
scoped user; the principal is server-asserted (the expanded `sign.record`
grant names persona and account — no client-claimed identity anywhere).
Refusals (garbage, revoked) land in the audit log. All measured through
the public surfaces end to end. Revocation propagation bound = token TTL,
inherited from soulidentity's D22.

## 6. Plane wiring [V for identity/memory, D for runtime]

- **Identity**: `embed.Run(ctx, Options)` on two loopback conns (service
  account, AUTH account); options straight from the state dir. `Run`
  returned ⇒ surface silent (upstream contract, its journey 0018).
- **Memory**: `archive.Open(<state>/archive)` → `keeper.Run` +
  `keeper.Witness` + `topic.RespondMemory` on a realm client
  (`realm.NewClient` over loopback, archivist persona, signer from the
  identity plane's `client.PersonaSigner`).
- **Runtime** [V as-built in 003]: soulrealm public packages, native
  backend, workloads as declarations — **invocation-scoped**: `soulnode
  workload start` supervises one declared workload as persona `runner`
  (transport creds from the ceremony, signer vault-held), minting with
  the ceremony's *plain* workload signing key (a scoped key rejects the
  minter's carried permissions — the two-keys split). The long-running
  node supervisor (claim-race placement, sweeper) is soulrealm's own
  unbuilt Fleet milestone — SoulNode consumes it when it lands upstream
  and MUST NOT invent one here (constitution I).

## 7. Shutdown and failure [D]

One signal context fans out to every plane; planes drain their
subscriptions, the runtime stops workloads through backend handles, the
server shuts down last. A plane failure is surfaced and named (log +
non-zero exit if fatal at boot; runtime restarts of planes are Phase 1's
[O] — default: fail loud, no silent restarts).

## 8. The front door — seamed, not designed [named for Phase 2]

MCP over streamable HTTP, static-bearer admission through the callout,
per-user pooled loopback connections, corpse eviction on dead pools —
the shape soulstream's `remote-mcp-node` research measured. Gated on that
topic's outcome and soulstream's public MCP surface (the fourth upstream
ask, held for the maintainer). Nothing in §§1–7 may depend on its
internals; it consumes the same admission lane as any client.

## 9. Acceptance criteria (Phase 1, made precise per feature in specs/)

- **M1.1 — the server and the identity plane**: on an empty state dir,
  `soulnode init && soulnode up` reaches: sealed surface answers
  `status`; sentinel + printed token admits with the persona scoped to
  its own prefix; garbage and revoked tokens refuse with audited
  refusals; `init` re-run is a verified no-op; every artifact of §4 is on
  disk with the stated modes. Zero manual steps, zero external binaries.
- **M1.2 — the realm joins**: realm provisioned on the embedded server;
  the archivist keeps ops and answers the memory convention, attributed
  to its persona with a vault-held key.
- **M1.3 — an agent runs**: a declared agent workload launches through
  the runtime plane (native backend), posts a turn attributed to its
  persona, and its lifecycle appears as work ops — the soulrealm M1.1
  proof, re-run inside SoulNode.

## Requirement language

MUST / MUST NOT / MAY per the [design README](README.md); values marked
*default* ship unless configuration overrides them.
