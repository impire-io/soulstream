# Feature Specification: One console, one vocabulary

**Feature Branch**: `009-one-console-one-vocabulary`
**Created**: 2026-08-15
**Status**: Draft
**Input**: Design
[`0001-soulnode-composition.md`](../../../soul-hq/02-DESIGN/soulstream/0001-soulnode-composition.md)
§2 as amended (planes named by function; legacy keys read forever) and
soulstream-idp design D31 (the bundled sign-in plane serves its admin
API, not its HTML console — the product's console is the shell).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The names say what things do (Priority: P1)

A person founding a realm reads `--signin-listen` and `--mcp-listen`,
finds `planes.signin` and `planes.mcp` in their config, and `up` prints
"sign-in" and "MCP" — no byname anywhere. A realm founded before the
rename keeps working untouched: its `planes.fold`/`planes.door` keys
are read forever, and its on-disk artifacts (`users/fold.creds`) are
found by fallback.

**Acceptance Scenarios**:

1. **Given** a fresh `init`, **Then** `config.json` carries
   `planes.signin`/`planes.mcp` and the state dir carries
   `users/signin.creds`; `up` prints functional labels only.
2. **Given** a state dir written with the legacy keys and
   `users/fold.creds`, **When** `up` runs, **Then** it verifies and
   serves identically — pinned by a migration fixture in `make test`.
3. **Given** `--fold-listen`/`--door-listen` on `init`, **Then** they
   are accepted as the older spellings of the new flags.

### User Story 2 - The shell is the console (Priority: P1)

The bundled sign-in plane mounts its admin API and not its HTML
console: `/admin` on the sign-in listener answers 404, `up` prints no
admin-console line, and administration happens in the shell.

## Success Criteria

- **SC-001**: the migration fixture (legacy keys + legacy creds name)
  boots in `make test`; a fresh found writes only functional names.
- **SC-002**: `/admin` on the bundled sign-in plane answers 404 while
  `/api/admin/*` still guards and serves — pinned in the node tests.
- **SC-003**: no "door"/"fold" prose in `cmd` output, `node` logs, or
  `ceremony` refusals; docs mirror the real `up` output.
