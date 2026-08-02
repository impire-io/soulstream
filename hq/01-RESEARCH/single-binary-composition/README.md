# Can the full ecosystem shape run correctly composed in one process?

**State:** graduated
**Started:** 2026-07-31
**Graduated:** 2026-08-02 — to design (`hq/02-DESIGN/0001-soulnode-composition.md`)

## Abstract

This topic gates Phase 1 of the roadmap
([`roadmap.md`](../../03-IMPLEMENTATION/roadmap.md)). It measures the three
unknowns standing between the founding bet ([episode
0001](../../04-JOURNEY/0001-genesis.md)) and a buildable design: whether an
embedded operator-mode NATS server with auth-callout admission behaves
identically to the wire deployments the ecosystem already measures against
(soulstream's `remote-mcp-node` rig, Bars 1–3 PASS there 2026-07-30); what
each component must export so SoulNode can wire it without `internal/`
reaches (constitution I); and whether the first-boot ceremony can be
generated entirely by code (constitution V). A decisive PASS opens the first
design document (the SoulNode composition); a FAIL on parity or surfaces
triggers the founding reversal condition.

## The question

Can the full ecosystem shape — operator-mode NATS, callout admission, sealed
vault, realm, memory — run correctly composed in one process, and what does
each component need to expose for SoulNode to wire it without `internal/`
reaches?

## Pre-registered bars

- **Bar 1 — Embedded admission parity.** An embedded operator-mode NATS
  server (JetStream on a fresh temp state dir), provisioned entirely by rig
  code, with the SoulIdentity service and callout issuer running against it,
  reproduces the wire rig's admission readings: (a) the API-token lane
  through sentinel creds admits, and the persona's server-resolved
  permissions (`$SYS.REQ.USER.INFO`) scope it to its own prefix; (b) a
  garbage token is refused with a callout refusal in the audit and no
  connection formed; (c) a token revoked at the store is refused the same
  way. **Threshold:** 3/3 observations green on 3 consecutive rig runs.
  **Protocol:** a throwaway consumer-position rig under `experiment/` in this
  topic folder; the SoulIdentity service wired in-process if a public surface
  permits, otherwise supervised as a subprocess — *which of the two it had to
  be is itself a recorded reading feeding Bar 2*.
- **Bar 2 — The embed surface is small.** For each component SoulNode
  bundles — SoulIdentity (service + callout issuer), soulrealm (node),
  archivist (memory), soulstream (realm client) — a written enumeration of
  the exported entry points SoulNode needs, each either existing today
  `[measured]` or a named upstream ask with a proposed signature.
  **Threshold (PASS):** every component's ask fits an embed seam — a
  Run/Serve-shaped entry point plus an options struct, at most three new
  public packages per component, no restructuring of component internals.
  **FAIL:** any component whose composition demands more than an embed seam.
- **Bar 3 — The ceremony is code, end to end.** The rig's provision step
  generates every first-boot artifact — operator, system account, AUTH
  account, realm account, signing keys, xkeys, sentinel creds, vault first
  key, KV buckets — starting from nothing but an empty directory: zero
  manual steps, zero external binaries (no `nsc`), and the ceremony
  inventory document and the provision code agree 1:1. **Threshold:** a
  fresh state dir reaches Bar 1's passing observations with no step
  performed outside the rig, and every artifact in the inventory doc is
  traceable to a line of provision code.

## Reversal condition

If Bar 1 can only pass by forking a component (patching admission behavior to
tolerate the embedded server), or a Bar 2 upstream ask is refused or grows
beyond an embed seam, the one-process bet reverses: SoulNode becomes a
supervisor of separate component processes shipped as one installer — the
distribution promise survives, the embedding bet does not (episode 0001's
founding reversal condition, restated here as the topic's own).

## Verdict

- **Bar 1 — Embedded admission parity: PASS** [measured]. 3/3 observations
  green on 3 consecutive fresh-rig runs in the DontListen+in-process arm,
  plus the TCP control arm: token-lane admission 4 ms with the
  server-asserted principal scoped to its own prefix; bare, garbage, and
  revoked tokens all refused with `callout REFUSED` in the audit. The
  protocol's recorded reading for Bar 2: in-process wiring required the
  module-namespace dodge. One named finding, not a bar failure: refusals
  over the in-process pipe block a fixed ~10.0 s and lose the -ERR reason
  (transport-isolated to the pipe, not DontListen) [measured] — superseded
  as a design constraint by the maintainer's 2026-08-02 all-loopback
  decision, and demoted to a candidate upstream issue.
- **Bar 2 — The embed surface is small: PASS** [measured readings +
  mechanism-argument on the proposed seams]. Every component fits an embed
  seam; the four upstream asks were enumerated in `embed-surfaces.md` and
  three were *delivered* before graduation: soulidentity `embed.Run`
  public (its D29/journey 0018, compiler-proof gate), archivist
  `archive`/`keeper` public on soulstream v0.6.0, soulrealm pinned to
  tagged v0.6.0 (its journey 0011). The fourth (soulstream public `mcp`
  surface) is held for the maintainer's open `remote-mcp-node` topic and
  gates only Phase 2.
- **Bar 3 — The ceremony is code, end to end: PASS** [measured]. The
  rig's `Provision` generates every inventory artifact from an empty
  directory — pure `server.Options`, no config file, no `nsc`, no manual
  step — and reaches Bar 1's passing observations; `ceremony.md` and the
  provision code agree 1:1, and every founding administrative act runs
  through the public `client` on the node's own connection.

The founding reversal condition was not triggered: no component required
forking, no embed ask was refused. The one-process bet holds, with the
transport refined by the maintainer to all-loopback so decomposition stays
configuration.
