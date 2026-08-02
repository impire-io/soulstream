# Feature Specification: the fold plane — the bundled sign-in

**Feature Branch**: `006-the-fold-plane`
**Created**: 2026-08-03
**Status**: Implemented (landed 2026-08-03 — journey 0057)
**Input**: soulfold M5's distribution story ("the single-binary
distribution runs the fold in-process, and the distribution story
wiring the issuer at the bundled fold by default") composed into
SoulNode: `planes.fold` runs the deployment's own passkey-first OIDC
provider through soulfold's public embed seam, and public door mode
defaults its authorization server at it.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — One binary, real sign-in (Priority: P1)

An operator enables `planes.fold` and sets `planes.door.public_url`.
Nothing else. The door advertises the bundled fold as its
authorization server; a person's browser walks DCR → passkey sign-in
→ token, and the bearer opens an MCP session whose realm identity is
their fold identity. Zero external services.

**Acceptance scenarios**:

1. **Given** `planes.fold` enabled and door public mode with no
   `auth_issuer`, **Then** the resource metadata names the bundled
   fold's issuer and the identity plane's OIDC lane validates against
   it — the default wiring [measured].
2. **Given** the founding persona (seeded into the fold with the
   `realm` role at first start), **When** they sign in by passkey
   ceremony (first touch enrolls), **Then** their access token forms
   an MCP session at the door, `whoami` names their fold identity, and
   the audit says lane=oidc role=realm [measured].
3. **Given** the founding token or a garbage bearer, **Then**
   coexistence and refusal behave exactly as before [measured].

### User Story 2 — Old realms stay honest (Priority: P2)

A state dir founded before the fold plane existed loads and runs
unchanged (the config block's absence means disabled). Enabling the
plane there is a named refusal pointing at re-init — the fold's
bypass-lane creds are founding artifacts.

## Composition notes (constitution I)

- The fold arrives by tag (`soulfold v0.1.2`) through its public
  `embed` and `authtest` packages; its buckets ride this node's
  JetStream over the fold's own bypass-lane creds; its seal seed lives
  under `<state>/fold/` — outside the JetStream store dir, per its D17.
- Startup order is load-bearing: the fold serves before the identity
  plane, whose OIDC validator discovers its issuer at startup.
- Two consumer-proven upstream additions landed this cycle:
  `embed.Options.NATSCreds` (v0.1.1 — operator-mode parents connect
  authenticated) and persona-shaped user ids (v0.1.2 — `u-hex`;
  soulstream's identity grammar refuses underscores).

## Out of scope

Enabling the fold by default at init (a distribution decision for the
release story); the fold's own lifecycle/admin (soulfold M3); tsnet
(Phase 3, gate unchanged).
