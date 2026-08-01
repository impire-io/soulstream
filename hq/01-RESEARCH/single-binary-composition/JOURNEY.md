# single-binary-composition — investigation journal (started 2026-07-31)

Opened under the maintainer's standing go-ahead ("create the repo, then go
ahead autonomously", 2026-07-31): the bars above were pre-registered by the
assistant from the roadmap's Phase 0 scoping, not drafted live with the
maintainer — any amendment he wants lands here openly before it binds.

## 2026-07-31 — the rig runs: admission parity holds, refusal parity does not

**Setup.** Recon across the siblings first (three read-only agents). The
load-bearing discovery: soulstream's `remote-mcp-node` experiment runs the
SoulIdentity service and callout issuer **in-process** by naming its module
`github.com/impire-io/soulidentity/researchnode` — under soulidentity's own
namespace, which is what makes its `internal/` imports legal [measured, its
go.mod]. This rig uses the same consumer-position dodge
(`…/soulidentity/soulnoderig` in [experiment/](experiment/)) — and the
*necessity* of the dodge is the Bar 2 headline confirmed from day one:
SoulNode proper cannot wire the service today; the public embed seam is a
real upstream ask.

**The ceremony is code** ([experiment/provision.go](experiment/provision.go),
the Bar 3 shape): `Provision(dir, inProcess)` generates from an empty
directory — operator, SYS, AUTH (external authorization, callout xkey,
allowed accounts), REALM (JetStream unlimited, scoped signing-key template
with the sign.record/keys.public grants), issuer/service/ops users, vault
first key + surface xkey, the two KV buckets, the vault, the service +
callout issuer in-process, both account signing keys imported, one `sit_`
token + sentinel creds. Pure `server.Options` + `MemAccResolver` — no config
file (the sibling rig used `ProcessConfigFile`; this arm proves the
callout/scoped-key machinery works through code-built options too
[measured]). No `nsc`, no external binary, no manual step. The `Inventory`
list in provision.go is the enumerated ceremony.

**Measured (Bar 1 protocol).** DontListen server, every connection via
`nats.InProcessServer` — the SoulNode shape — three consecutive fresh-rig
runs, plus a TCP control arm (`go test -v` in experiment/):

- 3/3 runs green in-process; TCP control green. Token-lane admission
  (sentinel creds + `sit_` token) in **4 ms**; `$SYS.REQ.USER.INFO` names
  `daan-ext@<realm account>` from the expanded `sign.record` grant; every
  resolved `soulidentity.*` grant stays inside the persona's own prefix.
  Bare token without the sentinel refused (consistent with the sibling
  rig's amended Bar 1 finding). Garbage token refused with
  `callout REFUSED` in the audit; revoked token refused after
  `RevokeToken`. [measured]
- **Divergence found — refusals are slow and mute in-process.** Every
  *refused* connect over the in-process pipe blocks a fixed **~10.0 s** and
  surfaces `io: read/write on closed pipe`; the same refusal over TCP
  returns immediately (3 ms) as `nats: Authorization Violation`. Client
  `nats.Timeout` does not change it (default vs 1 s → 10.002 s / 10.005 s).
  Isolation probe (same TCP-listening server, both client transports):
  TCP 3 ms with the proper error, pipe 10.002 s with the closed-pipe error —
  so the variable is the **in-process pipe transport**, not DontListen mode.
  [measured, experiment/timeout_probe_test.go] The 10 s matches the
  server's `DEFAULT_FLUSH_DEADLINE` and the -ERR payload never reaches the
  pipe client [mechanism-argument — exact code path not traced; candidate
  upstream issue against nats-server/nats.go].
- Bar 1 (b)/(c) still PASS as pre-registered (refusal happens, audit
  records it, no connection forms) — the bar text never demanded refusal
  *latency* or *error identity* parity. Recorded as a named finding rather
  than a bar amendment: admission parity holds; refusal parity does not.
- Design consequence to argue at graduation: internal planes ride
  in-process connections (they are never refused in normal operation); the
  front door's per-user pooled connections should ride loopback TCP until
  the upstream behavior is fixed — otherwise every bad badge costs 10 s and
  loses its refusal reason. [judgment]

Suite cost: ~90 s for the 3-run in-process arm — entirely the nine 10 s
refusal waits; admission itself is milliseconds throughout.

## 2026-07-31 — recon lands: the surfaces enumerated, all three bars stand

The two remaining recon reports (soulidentity serve path;
archivist/soulstream/soulrealm surfaces) completed the Bar 2 and Bar 3
evidence:

- **[ceremony.md](ceremony.md)** (Bar 3): the enumerated first-boot
  inventory, 1:1 with the rig's `Provision`. The recon confirmed the same
  sequence in soulidentity's own canonical e2e
  (`client/callout_e2e_test.go`, `TestM4GateAgainstOperatorModeServer`) —
  the rig's ceremony is that test's, re-expressed through pure
  `server.Options` instead of a config file. One structural reading worth
  keeping: soulidentity has **no in-process provisioning API** — every
  mutating op (`keys.import`, `tokens.create`, `sentinel.mint`, …) is
  NATS-surface only. For SoulNode that is a feature, not a gap: `soulnode
  init` provisions over its own in-process connection with the public
  `client`, exactly as the rig does [measured].
- **[embed-surfaces.md](embed-surfaces.md)** (Bar 2): per component —
  soulidentity needs one public `embed.Run` package (serve assembly is
  `internal/`; the rig's namespace dodge is the proof); soulstream's realm
  plane is public already (`realm.NewClient(nc, cfg)` is the in-process
  seam) but the MCP front door is `internal/mcpserver`; the archivist's
  keeper/witness/store glue is all `internal/` over public primitives;
  soulrealm embeds launch-one-workload with zero asks, while the
  long-running node supervisor is its own unbuilt Fleet milestone and the
  `replace ../soulstream` on its main blocks version pinning. No component
  needs more than an embed seam [mechanism-argument, signature-level
  recon]. Four upstream asks, listed in the doc.

**Where the bars stand:** Bar 1 PASS [measured, 3/3 × 3 runs + TCP control,
with the named refusal-parity finding]. Bar 2 PASS [enumerated; threshold
met — no component beyond an embed seam]. Bar 3 PASS [measured — the
ceremony runs from an empty dir in code; inventory 1:1 reviewed].

**Held for the maintainer** (working agreement — teach-back before the
direction is recorded): graduation to design, which will carry two judgment
calls — (1) the front door's client-facing connections ride loopback TCP
while internal planes ride in-process (the refusal-parity finding); (2)
which upstream asks to file first and whether the constitution ratifies as
drafted. `/research-graduate single-binary-composition --to design` is
ready to run once those survive teach-back.

## 2026-08-01 — upstream ask 1 of 4 delivered: soulidentity's embed seam

The maintainer confirmed the direction ("expose the right constructs in
the downstream projects") and the first ask landed the same day, through
soulidentity's own process: D29 + feature `002-embed-seam` (its journey
episode 0018, merged to its main). `embed.Run(ctx, Options)` is public —
value-only options, custody unchanged, provisioning still wire-only — and
its proof module is the sharper version of this topic's own trick
inverted: a module path *outside* the namespace, so `internal/` imports
cannot compile [measured, `soulidentity/e2e/embedgate` rides its
`make test`]. Consequence for this topic: the rig's namespace dodge is now
deletable — the rig can re-wire through the public seam when the topic
graduates (or be retired with the folder, since embedgate now proves the
same shape upstream). Asks remaining: soulstream public `mcp` package
(interacts with its open `remote-mcp-node` topic — the natural vehicle),
archivist keeper/archive seam + OnServed bump, soulrealm×soulstream
release tagging.

## 2026-08-01 — asks 3 and 4 delivered the same day; only the front door remains

- **Archivist (ask 3)**: `archive/` and `keeper/` are public — the embed
  seam as this topic proposed (open a Store, hand it to `keeper.Run` +
  `keeper.Witness` + `topic.RespondMemory`). Rode the soulstream
  v0.4.0→v0.6.0 bump; the pre-registered OnServed change landed, plus a
  second 017-guard fix the bump surfaced (typed-nil `*SigningKey` must
  leave `Config.Signer` unset — in the daemon and its test helper both;
  the guard caught it exactly as designed) [measured, its gate green
  end-to-end against v0.6.0].
- **soulrealm (ask 4)**: the `../soulstream` replace is gone — pinned at
  v0.6.0 (its journey 0011). Measured first: soulstream main is
  tag-current (v0.6.0 + docs-only commits); full gate green against the
  pin. soulrealm is now consumable as a tagged dependency the moment it
  tags.
- **Remaining (ask 2)**: soulstream's public `mcp` package — deliberately
  untouched: the `remote-mcp-node` research topic is the maintainer's
  open investigation (Bar 4 in flight) and its graduation is the natural
  vehicle for that surface. Held for him.

Consequence for Phase 1: every embed surface M1.1–M1.3 needs now exists
publicly (identity plane via soulidentity `embed`, realm client via
`realm.NewClient`, memory via archivist `keeper`/`archive`, workload
launch via soulrealm's public packages) — the front-door surface gates
only Phase 2.
