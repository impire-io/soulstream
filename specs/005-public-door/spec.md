# Feature Specification: the public door — OAuth for hosted clients

**Feature Branch**: `005-public-door`
**Created**: 2026-08-02
**Status**: Implemented (landed 2026-08-03 — journey 0055)
**Input**: Roadmap Phase 2's public-mode clause: "`planes.door` grows
`public_url`/`auth_issuer` additively when it exists"; upstream
soulstream 018's public mode (RFC 9728 resource metadata, external
OIDC only, AS-agnostic); soulfold M4 proving the intended default AS
admits users through the realm's callout. Unblocked 2026-08-02 by
soulfold M4 (journey 0054).

## User Scenarios & Testing *(mandatory)*

### User Story 1 — A hosted client signs in from nothing (Priority: P1)

A person points a hosted MCP client (claude.ai connector, sandboxed
desktop) at their node's public door URL. The client discovers
everything itself: the 401 challenge names the resource metadata, the
metadata names the deployment's authorization server, the client
registers (DCR), the person signs in there, and the resulting bearer
opens an MCP session whose realm identity is the token's subject.

**Acceptance scenarios**:

1. **Given** a node with `planes.door.public_url`, `auth_issuer`, and
   `auth_audience` set, **When** a cold request hits the door,
   **Then** it draws a 401 naming the resource metadata, and the
   metadata advertises exactly the configured public URL and AS.
2. **Given** an access token from the AS whose roles value names a
   declared role, **Then** the bearer forms an MCP session, `whoami`
   names the token's subject, and the audit attributes lane=oidc and
   the role.
3. **Given** a token naming an undeclared role, or a garbage bearer,
   **Then** no session forms.
4. **Given** public mode, **Then** the founding-token lane still
   works — the owner's badge is not displaced.

### User Story 2 — Config stays honest (Priority: P2)

Public mode is a package deal: `public_url`, `auth_issuer`, and
`auth_audience` together or not at all; declaring it with the door
disabled is a config error, not a silent no-op. The three fields
survive the Save/Load round-trip. The door listener stays loopback —
HTTPS is deployment fronting (`tailscale serve`), and Phase 3's tsnet
gate is untouched.

## Success Criteria

- **SC-001** [measured]: the full discovery-to-session walk in
  `make test` against the upstream contract's AS stand-in (rigtest) —
  the door is AS-agnostic; soulfold is the intended default, not a
  dependency (its admission proof is soulfold M4's own gate).
- **SC-002** [measured]: coexistence (founding token) and refusals
  (undeclared role, garbage bearer).
- **SC-003** [measured]: config validation and round-trip.

## Consumer-caught upstream-shaped bug, fixed here

`listen: 127.0.0.1:0` pre-flighted a random port but nats-server reads
port 0 as its default 4222 — the probe and the server disagreed, and
parallel test packages collided on 4222. Port 0 now maps to the
server's random-port spelling (-1) so "any free port" means what it
says everywhere.

## Out of scope

Bundling the fold in-process and defaulting `auth_issuer` at it
(soulfold M5's wiring story); tsnet (Phase 3, gate unchanged);
per-role scope templates beyond the founding "realm" role (a future
lifecycle concern).
