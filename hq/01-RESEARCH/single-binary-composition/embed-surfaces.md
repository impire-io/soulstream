# The embed surface per component (Bar 2)

What SoulNode needs each component to export so the composition is
constitution-I clean (public tagged surfaces, no `internal/` reaches), what
exists today, and the exact ask where something is missing. Readings from
the 2026-07-31 recon across all four sibling repos plus this topic's own rig
([`experiment/`](experiment/README.md)).

**The measured baseline.** The rig proves the split empirically: every
*provisioning* act of the ceremony — key imports, token create/revoke,
sentinel mint — runs through soulidentity's **public** `client` package over
an in-process connection [measured: `experiment/provision.go` uses only
`client` for admin acts]. Only the *serve-side assembly* (service + callout
issuer wiring) forced the rig to ride soulidentity's module namespace to
reach `internal/` — the dodge soulstream's `remote-mcp-node` experiment
pioneered. The missing surfaces are all serve-side seams.

## SoulIdentity

- **Public today:** `client/` — the complete consumer surface: `New(nc,
  account, user)`, `ImportKey`, `CreateToken`/`Tokens`/`RevokeToken`,
  `MintSentinel`, `Mint`/`MintCreds`/`MintEphemeral`, `SignRecord`,
  `PersonaPublicKey`, `PersonaSigner` (structurally satisfies soulstream's
  `identity.Signer`). The entire first-boot admin ceremony and all
  steady-state signing/directory reads are covered [measured, the rig].
- **Missing:** the serve path. `service.New/Start`, `callout.NewIssuer/
  Start`, `vault.New/NewKVStore`, `callout.NewKVTokenStore` all live under
  `internal/`; the only assembly of them is `cmdServe` in
  `cmd/soulidentity/main.go`.
- **The ask (one package):** a public `embed` (or `soulidentity`) package
  lifting `cmdServe` behind a value-only options struct —

  ```go
  type Options struct {
      Conn        *nats.Conn    // service connection (realm account)
      CalloutConn *nats.Conn    // optional; presence enables the issuer
      VaultBucket, TokenBucket string
      FirstKey, SurfaceKey, CalloutKey string // SX… seeds
      AuthKeyName, AuthAccount string
      CalloutTTL  time.Duration
      Prefix      string
      OIDCIssuer, OIDCAudience string
      Logger      *slog.Logger
  }
  func Run(ctx context.Context, o Options) error
  ```

  Config stays by-string/by-value so no internal type (`vault.Vault`,
  `callout.Store`, `OIDCValidator`) leaks. Everything `Run` assembles
  already exists inside `cmdServe` — the seam is "expose the assembled
  entrypoint", not restructuring. [mechanism-argument, from the recon's
  signature-level read]
- **Bar 2 threshold:** PASS — one new public package, Run-shaped.

## soulstream

- **Public today:** the whole realm-participation plane. The in-process
  seam exists: `realm.NewClient(ctx, nc *nats.Conn, cfg)` builds a client
  on a connection SoulNode already holds; `topic` (post/work/materialise/
  memory/discovery), `record`, `identity` (structural `Signer`),
  `registry`, `curator` are all public.
- **Missing:** the MCP front door. `mcpserver.NewServer(c *realm.Client)`
  and every tool registration live in `internal/mcpserver` (+
  `internal/config`, `internal/keystore`).
- **The ask (one package):** a public `mcp` package exposing
  `NewServer(c *realm.Client) *mcp.Server` (the key/config seams stay
  SoulNode-side — it has its own state dir). Alternatively the
  remote-mcp-node prototype graduates into this surface — same ask, one
  home.
- **Bar 2 threshold:** PASS — one new public package.

## soulstream-archivist

- **Public today:** nothing — the repo has no library packages. But every
  primitive it glues is public soulstream surface (`realm.NewClient`,
  ordered consumer via `Client.JetStream()`, `record.Exhibit`,
  `topic.MemoryWitness`/`RespondMemory`).
- **Missing:** the glue: `keeper.Run` (op-log capture), `keeper.Witness`,
  `archive.Store` — all `internal/`.
- **The ask (one seam):** promote `keeper` + `archive` to public packages,
  or a single `archivist.Serve(ctx, c *realm.Client, store) error`
  entrypoint. Plus the already-pre-registered 2-line `OnServed(kind, n,
  err)` signature bump on its next soulstream dependency bump.
- **Bar 2 threshold:** PASS — at most two promoted packages, Serve-shaped.

## soulrealm

- **Public today:** everything SoulNode Phase 1 needs. `runner.Runner`
  (Minter/Backend/Realm), `backend` + `native` (also msb, k8s),
  `minter.NewSigningKeyMinter`, `declaration.Parse/Validate`; soulstream's
  public `topic.Handle` satisfies `runner.TopicClient` structurally.
  Launch-one-workload embeds with zero asks [recon, signature-level].
- **Missing (fleet-era, not Phase 1):** a long-running node supervisor
  (observe topics → sweep stale claims → claim-race → launch) exists
  nowhere yet — it is soulrealm's own unbuilt Fleet milestone (its design
  0003), not a seam SoulNode may invent (constitution I).
- **Release blocker, not a surface:** soulrealm's main carries `replace
  github.com/impire-io/soulstream => ../soulstream` — SoulNode cannot pin
  it until soulstream tags a release soulrealm builds against.
- **Bar 2 threshold:** PASS for the Phase 1 scope — zero new packages; the
  node supervisor is a named upstream milestone, and the tagging gap is
  release discipline, not restructuring.

## The upstream asks, in one list

1. **soulidentity:** public `embed.Run(ctx, Options)` for the service +
   callout issuer (one package, config by value).
2. **soulstream:** public `mcp.NewServer(c *realm.Client)` (one package;
   natural landing for the remote-mcp-node prototype).
3. **soulstream-archivist:** public `keeper`/`archive` (or
   `archivist.Serve`) + the 2-line OnServed bump.
4. **soulrealm × soulstream:** tagged releases so the `replace` drops;
   the fleet node supervisor lands upstream on its own roadmap.

No component requires more than an embed seam; none requires internal
restructuring. Bar 2's threshold is met across the board.
