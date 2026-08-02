# Implementation Plan: The Remote MCP Node

**Branch**: `018-remote-mcp-node` | **Date**: 2026-08-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/018-remote-mcp-node/spec.md`

## Summary

A URL into a realm for clients that cannot install anything. Two deliverables,
one feature: (1) **the public MCP surface** — `internal/mcpserver` promoted to
public `mcpserver` with one new injection point (a keyring provider), so any
host process that brings a connected, signer-wired `realm.Client` can serve
the full 23-tool surface (FR-015; SoulNode's fourth upstream ask); (2) **the
node** — a nested consumer module `node/` that serves that surface over
streamable HTTP behind bearer passthrough: per-principal pooled NATS
connections admitted by SoulIdentity's auth callout, freshest-bearer refresh
hardened with session-binding + candidate-probe (non-interference, FR-005),
server-asserted principals via `$SYS.REQ.USER.INFO`, delegated signing via
`client.PersonaSigner` through the 017 seam, and the OAuth resource side
(RFC 9728 + 401 challenge) pointing at an **external** authorization server
only — the AS-facing contract is a first-class artifact
([contracts/authorization-server.md](contracts/authorization-server.md))
gated by a stand-in built from it (SC-005). The research prototype at
`56c7a2e` is the measured starting point ([research.md](research.md) R3);
the trust-model fix over it is R4.

## Technical Context

**Language/Version**: Go 1.26 (both modules)
**Primary Dependencies**:
- Core module: no new dependencies (MCP go-sdk v1.6.1, nats.go v1.52 already present)
- Node module (`github.com/impire-io/soulstream/node`, new):
  `github.com/impire-io/soulstream` (committed relative `replace => ../`),
  `github.com/impire-io/soulidentity` (client pkg only; **pseudo-version pin**,
  no tags exist; `GOPRIVATE=github.com/impire-io`),
  `modelcontextprotocol/go-sdk v1.6.1`, `nats.go v1.52`;
  test-only: `nats-server/v2 v2.14.x`, `jwt/v2`, `nkeys` (operator-mode
  callout rig), stdlib `crypto/rsa` + `net/http/httptest` (AS stand-in)
**Storage**: none in the node (stateless trust — FR-002); realm-side is the
existing JetStream artifacts; identity plane's vault/token KV belong to soulidentity
**Testing**: `go test` per module; node rig = embedded operator-mode
nats-server + auth callout + soulidentity `embed.Run` + contract-built OIDC
AS stand-in; adversarial non-interference suite; token-material log audit
**Target Platform**: same six as the CLI (darwin/linux/windows, amd64/arm64);
node deploys proxy-fronted (HTTPS terminator in front, FR-012)
**Project Type**: library promotion + a new service binary in a nested consumer module
**Performance Goals**: workshop scale — tens of concurrent principals per
node instance; admission latency owned by the callout; no node-side queueing
**Constraints**: node holds no credentials/keys/durable state; token material
never in logs (FR-013); cycle guard — neither core repo imports the other
(FR-014); minimum NATS server 2.12+ (callout + JetStream)
**Scale/Scope**: 2 packages touched in core (`mcpserver` promotion, one cmd
import line), 1 new module (~6 source files + rig + stand-in), 2 carried
tools (`byon-setup`, `probe`), docs

## Constitution Check

- **I. NATS-Native First — PASS with one justified edge.** Everything
  realm-side is NATS: admission is the NATS auth callout, identity ops are
  NATS request/reply, state is JetStream, revocation is JWT TTL + callout
  re-admission. The node adds an HTTP listener — justified in Complexity
  Tracking: hosted MCP clients speak streamable HTTP and OAuth *by
  measurement* (the connector dialog offers nothing else); the HTTP tier is
  a Layer-2 adapter carrying zero protocol semantics (the node is
  explicitly non-normative, FR-001). No databases, no coordinators, no
  queues; pool state is in-memory and rebuildable from nothing. Minimum
  server version stated: 2.12+.
- **II. Smallest Viable Implementation — PASS.** The public surface is a
  visibility change (`git mv`) plus ONE option with a concrete need today
  (keyring injection for multi-principal hosts). The node carries only
  measured mechanisms; the trust-model change (R4) is the smallest that
  satisfies clarified FR-005. No embedded AS (resolved decision), no config
  file, no speculative options — `--prefix` exists because soulidentity's
  D14 forces it, not "for later". byon-setup/probe are carried, not
  productized (spec Q2).
- **III. ELI5 Documentation — PASS.** New `docs/remote-node.md` ("a front
  desk that holds no keys — your badge does the talking"), updates to
  `docs/mcp.md` (the two doors: local adapter vs remote node) and
  `docs/operators.md` cross-reference; the AS contract doc is written for
  an implementer audience but the concept page carries the plain-words
  version. All ship in the same change (tasked per story).

*Post-design re-check (after Phase 1): PASS — no new violations introduced;
the only non-NATS component remains the justified HTTP edge; contracts add
no machinery beyond the stated surfaces.*

## Project Structure

### Documentation (this feature)

```text
specs/018-remote-mcp-node/
├── plan.md              # This file
├── research.md          # Phase 0 — R1..R11
├── data-model.md        # Phase 1 — pool/entry/session/bearer model
├── quickstart.md        # Phase 1 — consumer + operator view
├── contracts/
│   ├── library.md       # public mcpserver + node package/config contract
│   ├── authorization-server.md  # THE AS-facing contract (soulfold's interface)
│   └── http.md          # node HTTP surface: 9728 metadata, 401 challenge, MCP endpoint
└── tasks.md             # Phase 2 (/speckit-tasks — not created by /speckit-plan)
```

### Source Code (repository root)

```text
mcpserver/                     # PROMOTED from internal/mcpserver (public; tests move too)
│                              # + WithKeyring option (server.go); NewServer(c, opts...)
cmd/soulstream-mcp/main.go     # one import line changes; behavior identical
node/                          # NEW nested module: github.com/impire-io/soulstream/node
├── go.mod, go.sum             # replace soulstream => ../ ; soulidentity pseudo-version
├── node.go                    # Node type, getServer factory, HTTP handler assembly
├── pool.go                    # pool, entry lifecycle, session binding, candidate probe (R4)
├── principal.go               # $SYS.REQ.USER.INFO derivation
├── oauth.go                   # RFC 9728 metadata + 401 challenge (PublicURL mode)
├── config.go                  # flags/env, validation, teaching errors
├── *_test.go                  # unit + rig-backed integration + adversarial suite
├── rigtest/                   # embedded operator-mode server + callout + embed.Run
│                              # + the contract-built OIDC AS stand-in (SC-005)
└── cmd/
    ├── soulstream-node/       # the binary
    ├── byon-setup/            # carried operator tooling (best-effort, Synadia Cloud)
    └── probe/                 # carried live-verification driver (follow-up measurement)
docs/
├── remote-node.md             # NEW ELI5 concept page
└── mcp.md                     # updated: local adapter vs remote node
```

**Structure Decision**: nested consumer module per R1 — FR-014's cycle guard
at module granularity; the core module's dependency set is unchanged. The
`mcpserver` promotion lives in the core module (it is soulstream surface);
everything that knows about bearers, HTTP, or soulidentity lives in `node/`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| HTTP listener in the node (non-NATS edge) | Hosted MCP clients speak streamable HTTP + OAuth only [measured 2026-08-01: connector dialog has no other lane] | "NATS-only door" serves zero no-install clients — the feature's entire audience |
| External OAuth AS in the deployment picture | Same measurement: the dialog requires an AS; decision resolved — external only, node stays credential-free | Node-embedded AS makes the node a credential custodian and single-persona — the opposite of the graduated design |
| Candidate-probe connection (one extra connect per new badge) | Clarified FR-005: forged hints must not displace a principal's freshest token (prototype behavior was a DoS hole) | Trusting the hint violates the spec; keying by full token kills pooling and session continuity across refreshes |
