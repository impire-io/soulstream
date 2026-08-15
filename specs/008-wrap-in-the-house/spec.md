# Feature Specification: Wrap in the house — one binary, one paste

**Feature Branch**: `008-wrap-in-the-house`
**Created**: 2026-08-15
**Status**: Draft
**Input**: Design
[`0002-wrap-in-the-house.md`](../../../soul-hq/02-DESIGN/soulstream/0002-wrap-in-the-house.md)
(operator direction 2026-08-15: no Go toolchain, no PATH assembly on an
agent's machine), realizing the product half of workloads design 0004 §8.

The product binary — the one the releases page hands out — answers
`soulstream wrap` natively and provides the stdio tool door it points
the harness at (`soulstream mcp`). A person with the release binary and
a signed-in assistant runs their agent by pasting one block from the
Agents screen; nothing else is installed.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Paste the block, the agent answers (Priority: P1)

A person creates an agent in the shell, pastes the credential card's
block into a terminal on any machine carrying the `soulstream` binary
and their assistant, and mentions of the agent become answers — the
wrapper launches the tool door out of its own executable
(`command: <self>, args: ["mcp"]`), so no second binary exists.

**Independent Test**: `soulstream wrap --harness claude` with the five
`SOULSTREAM_*` values against a live realm answers a mention posted
while the wrapper was off (design 0004 §11.1 through the product verb).

**Acceptance Scenarios**:

1. **Given** the five env values and `--harness claude`, **When** the
   wrapper starts after a mention was posted, **Then** exactly one
   reply turn authored by the agent appears.
2. **Given** a revoked credential, **When** either verb connects,
   **Then** it refuses loudly, names the likely causes, and posts
   nothing.
3. **Given** missing persona/realm/connection, **When** `wrap` or
   `mcp` runs, **Then** the refusal names the missing variable; an
   unknown `--harness` names the two presets and the template escape.

### User Story 2 - The door out of the same binary (Priority: P2)

`soulstream mcp` with the five env values (flags win where given)
serves the realm's MCP tools over stdio — environment-only, no context
files, no keystores; an agent with no signing key speaks unsigned, and
the verb does not pretend otherwise.

## Success Criteria

- **SC-001**: refusal paths hermetic in `make test` (env validation,
  unknown preset, usage naming both verbs).
- **SC-002**: `soulstream wrap --harness claude` proven live once
  against a founded realm (the design's criterion 1) — recorded, not
  automated.
- **SC-003**: `docs/getting-started.md` contains no `go install`.
