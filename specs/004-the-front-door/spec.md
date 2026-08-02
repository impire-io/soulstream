# Feature Specification: the front door — the MCP door plane

**Feature Branch**: `004-the-front-door`
**Created**: 2026-08-02
**Status**: Draft
**Input**: User description: "Phase 2 (design 0001 §8, roadmap): the MCP door plane — soulstream's remote node (landed upstream as 018, v0.7.0, module made consumable) runs inside `soulnode up`: streamable HTTP in, bearer passthrough to the realm's own admission, one pooled connection per admitted principal, the public tool surface out. Local mode, static bearers — the token `init` printed is the badge."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Point a client at the node and just work (Priority: P1)

The owner runs `soulnode up` and points an MCP client (Claude Code, a
desktop client) at the door URL with the token `init` printed as a
bearer. The session forms; `whoami` answers with *their* persona as the
realm admitted it — never as the client claimed it; the full soulstream
tool surface is there. A wrong or revoked token gets a refusal, and the
door itself custodies nothing: no keys, no per-user secrets, restart
free.

**Why this priority**: This is the product's mouth — the vision's
"point a client at the URL SoulNode prints and start working." Phase 1
built the realm; this makes it reachable by the tools people actually
use.

**Independent Test**: With the node up, an MCP client presenting the
founding token connects to the door, lists tools, and `whoami` names
the owner persona; a garbage bearer cannot form a session.

**Acceptance Scenarios**:

1. **Given** the node is up with the door enabled (the bundle default),
   **When** an MCP client connects with `Authorization: Bearer <token>`,
   **Then** the session initializes, the tool list is the public
   soulstream surface, and `whoami` answers with the server-admitted
   persona.
2. **Given** a garbage or revoked bearer, **When** a client tries to
   connect, **Then** no session forms and the refusal is visible in the
   node's log.
3. **Given** `init` completed, **Then** its output tells the owner where
   to point a client (the door URL) alongside the token.
4. **Given** the door disabled in configuration, **Then** `up` serves
   exactly as Phase 1 (no HTTP listener exists).

---

### Edge Cases

- The door's HTTP port is taken: `up` refuses naming the address and the
  config key — same rule as the NATS listener.
- The door binds loopback only in this feature; fronting it publicly
  (HTTPS, `tailscale serve`) is deployment documentation, and the
  external-AS OAuth story (public mode) stays upstream-gated on soulfold
  — out of scope here, config shape must not preclude it.
- Interrupt during active sessions: the door closes its pool and the
  node drains as before.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `config.json` MUST gain the door's plane block
  (`planes.door`: `enabled`, default true; `listen`, default loopback on
  the conventional alternate-HTTP port) — design §2's shape, second
  instance.
- **FR-002**: `up` MUST run the door in-process when enabled: upstream's
  node library on the node's own loopback NATS listener and sentinel,
  local mode (static bearers), the identity-plane audit logger receiving
  its admission/refusal/eviction events.
- **FR-003**: The door MUST custody nothing: no new key material, no
  per-user state in the state directory; the sentinel it reads is the
  existing public artifact.
- **FR-004**: A door startup failure (bind conflict included) MUST
  prevent `up` from reporting healthy, named — never a silently missing
  door.
- **FR-005**: `init`'s founding output and `up`'s serving lines MUST
  name the door URL when enabled.
- **FR-006**: Admission at the door MUST be the realm's own: bearer
  passthrough to the callout — a bad badge refuses; nothing
  client-claimed decides identity.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In the test suite: MCP client + founding token → session
  forms, tools list non-empty, `whoami` names the owner persona; garbage
  bearer → no session; all against the running composition.
- **SC-002**: Door disabled → Phase 1 e2e behavior byte-identical, no
  HTTP listener.
- **SC-003**: No new secrets on disk; the state-dir inventory is
  unchanged by this feature.
- **SC-004**: Full quality gate green.

## Assumptions

- Local mode only (no `PublicURL`/`AuthIssuer`): the OAuth/OIDC public
  story arrives with soulfold upstream; the plane block gains fields
  then, additively.
- The upstream node module is consumed at a pseudo-version until it tags
  (`node/v0.x` — upstream's release act), the fourth tracked pin.
