# Episode 0002 — The composition gate: three bars PASS, the ecosystem opens its seams (2026-07-31 → 2026-08-02)

The founding question — can the full ecosystem shape run correctly composed
in one process, and what must each component expose — was opened with three
pre-registered bars and closed with all three measured PASS, plus a chain
of consequences that reshaped four sibling repos in two days.

**Bar 1 — embedded admission parity: PASS** [measured]. A throwaway rig
provisioned an operator-mode server with auth callout entirely through
`server.Options` (no config file, no `nsc`) and ran the SoulIdentity
service + callout issuer against it. Token-lane admission in 4 ms with the
server-asserted principal (`$SYS.REQ.USER.INFO`'s expanded `sign.record`
grant) scoped to its own prefix; bare, garbage, and revoked tokens refused
with the refusals in the audit — 3/3 observations on 3 consecutive runs,
plus a TCP control arm. One real divergence found: a *refused* connect over
the in-process pipe transport blocks a fixed ~10.0 s and surfaces
`io: read/write on closed pipe` instead of `nats: Authorization Violation`
— isolated to the pipe (not DontListen mode), matching the server's flush
deadline [measured; mechanism unconfirmed at source level].

**Bar 2 — the embed surface is small: PASS**, and mostly *delivered before
graduation*. The recon found the ecosystem's pattern: provisioning was
public everywhere, serve assemblies were `internal/`-only, and two
consumers had already ridden the module-namespace dodge to reach them. The
maintainer's direction ("expose the right constructs downstream") turned
the enumeration into landings: soulidentity's public `embed.Run(ctx,
Options)` (its D29, feature 002-embed-seam, journey 0018 — with a
compiler-proof consumer gate whose module path sits outside the namespace),
the archivist's public `archive`/`keeper` packages on soulstream v0.6.0
(the pre-registered OnServed bump plus a second typed-nil-Signer guard fix
the bump surfaced), and soulrealm pinned to tagged soulstream v0.6.0 with
its dev `replace` dropped (its journey 0011). The fourth ask — soulstream's
public MCP surface — is deliberately held: the maintainer's open
`remote-mcp-node` topic is its natural vehicle, and it gates only Phase 2.

**Bar 3 — the ceremony is code: PASS** [measured]. Every first-boot
artifact — operator, SYS/AUTH/realm accounts, signing keys, xkeys,
sentinel, buckets, first token — generated from an empty directory by one
provision function, with every founding administrative act running through
the public `client` over the node's own connection. The enumerated
inventory (kept in this episode's trail via git history) is design 0001's
ceremony section.

**Refuted and superseded, openly:** the first gate run refuted the
assumption that `Drain()` completes — soulidentity's `embed.Run` now
flush-confirms so "returned means silent" (its episode 0018). And the
2026-08-01 judgment that the refusal divergence forces a transport *split*
(internal planes in-process, front door loopback) was superseded by the
maintainer's stronger call: **everything through loopback** — every plane
connects over an ordinary NATS connection, so running only parts (BYO NATS
server, soulrealm on another machine) is configuration, never architecture.
The rig's in-process arm is thereby a finding of record, not the product
shape; the divergence demotes to a candidate upstream issue. Constitution
III was reworded accordingly and the constitution ratified at 1.0.0 with
this graduation.

Opened: design [`0001-soulnode-composition.md`](../02-DESIGN/0001-soulnode-composition.md)
— the all-loopback composition, per-plane connection configuration, the
persisted ceremony (`soulnode init`), and Phase 1's acceptance criteria.
Phase 1 is unblocked.

Reversal condition: the topic's own, updated by the transport decision — a
component fork or a refused embed ask still reverses the one-process bet to
a supervised multi-process installer; and a deployment class where any
local listener is untenable (or measured loopback overhead at node scale)
reopens in-process transport, gated on the upstream refusal fix landing
first.

Trail: the topic folder `hq/01-RESEARCH/single-binary-composition/`
(README with per-bar verdicts, JOURNEY, ceremony.md, embed-surfaces.md,
experiment/ rig — removed at graduation, full history in git);
soulidentity journey 0018 + D29; soulstream-archivist commit `643bb77`;
soulrealm journey 0011; constitution 1.0.0.
