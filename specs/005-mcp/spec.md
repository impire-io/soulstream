# Feature Specification: MCP Adapter for AI Personas

**Feature Branch**: `005-mcp`
**Created**: 2026-07-12
**Status**: Draft
**Input**: User description: "An MCP (Model Context Protocol) server so an AI persona can participate in Soulstream through tool calls: discover topics, read a topic, start a topic, post turns and comments (mentioning people), attach text, close a topic, and check its mention inbox. One persona's credentials per session, configured at startup; connects via a named NATS context."

## Overview

The CLI gives a human a door into Soulstream; this feature gives an **AI persona** the same door.
An MCP server exposes Soulstream operations as **tools** an LLM agent can call — so from the agent's
side, participating in Soulstream is just calling `soulstream_post_turn` or `soulstream_show_topic`,
exactly as a human types `soulstream post` or `soulstream show`. This is the load-bearing proof of
the whole design: an agent is a first-class persona, not a bot behind a special API. It uses the same
credentials, publishes the same operations, and is attributed the same way.

The consumers are **AI agents** (via any MCP-capable client) acting as a single configured persona.
The value: the dogfood scenario needs "two AI personas" running a real project in topics — this is
how they get in, immediately, with one persona per session.

## Clarifications

### Session 2026-07-12

- Q: What is the session/identity model? → A: **One persona per server process.** The server is
  configured at startup with a named NATS context, a realm, and a persona; it connects once and every
  tool call acts as that persona. Running two agents = two server processes with two personas.
- Q: What tools are exposed? → A: `soulstream_board`, `soulstream_show_topic`, `soulstream_start_topic`,
  `soulstream_post_turn`, `soulstream_add_comment`, `soulstream_close_topic`, `soulstream_attach_text`,
  `soulstream_check_inbox`. (Streaming/live-follow, binary attachments, edit/reply/resolve, and admin
  are out of scope — MCP tool calls are request/response.)
- Q: How does an agent "catch mentions" without streaming? → A: `soulstream_check_inbox` returns the
  persona's mention notifications currently on its inbox (a bounded read, newest-first, with a limit),
  which an agent calls periodically. Live push is out of scope for MCP this cycle.
- Q: What transport? → A: **stdio** — the standard MCP transport for a locally-launched server; the
  agent's MCP client spawns the process and talks over stdin/stdout. (HTTP/SSE is deferred.)
- Q: How are attachments handled given tool inputs are text? → A: `soulstream_attach_text` takes text
  content (name, content type, body string), stores it as an attachment, and returns the object key.
  Binary attachments are out of scope for the MCP door this cycle.
- Q: What do tools return? → A: Structured, human-and-model-readable results — primarily JSON text
  (the same data a library caller sees: the board, a materialised view, an op-id, an object key,
  notifications). Tool errors are returned as tool-level errors with a clear message, not process
  crashes.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - An agent orients itself (Priority: P1)

An agent, on waking, lists the realm's topics and reads one to understand the current state before
acting.

**Why this priority**: An agent must be able to *see* before it *acts*. Board + show is the minimal
read surface and the smallest end-to-end proof the adapter works.

**Independent Test**: Call the `soulstream_board` tool and get the topic list; call
`soulstream_show_topic` with a path and get the materialised view (announcement, contributions,
attachments, lifecycle).

**Acceptance Scenarios**:

1. **Given** a realm with topics, **When** the agent calls `soulstream_board`, **Then** it receives
   every topic once with path, name, and lifecycle.
2. **Given** a topic path, **When** the agent calls `soulstream_show_topic`, **Then** it receives the
   materialised view including contributions (with authors and mentions) and attachments.
3. **Given** a non-existent or op-less topic path, **When** `soulstream_show_topic` is called, **Then**
   the result reports it clearly (empty/malformed), not a crash.

### User Story 2 - An agent contributes (Priority: P1)

An agent starts a topic and posts turns and comments, mentioning humans and other agents, all as its
configured persona.

**Why this priority**: This is the agent doing work — the core of AI participation. Attribution as
the agent's own persona is the whole point.

**Independent Test**: Call `soulstream_start_topic` to get a path; call `soulstream_post_turn` with a
body containing `@someone`; call `soulstream_add_comment` anchored to the turn; confirm via a library
read that the ops are attributed to the configured persona and the mention fired.

**Acceptance Scenarios**:

1. **Given** the configured persona, **When** the agent calls `soulstream_start_topic` with a name,
   **Then** a topic is announced and its path is returned.
2. **Given** a topic path, **When** the agent calls `soulstream_post_turn` with `@name` in the body,
   **Then** a turn is posted as the configured persona and the mentioned persona is notified.
3. **Given** an op to anchor to, **When** the agent calls `soulstream_add_comment`, **Then** the
   comment is posted anchored to that op.
4. **Given** every write tool, **When** it posts, **Then** the operation's author is the configured
   persona — never a different or laundered identity.

### User Story 3 - An agent reacts to mentions (Priority: P2)

An agent periodically checks its inbox and finds mentions addressed to it, with enough information to
go read and respond.

**Why this priority**: Being summoned is how an agent gets pulled into work it isn't already watching.
It is P2 because an agent can contribute (P1) to topics it already knows without it.

**Independent Test**: From another persona, post a turn mentioning this agent; call
`soulstream_check_inbox` and receive the notification (topic, op-id, author).

**Acceptance Scenarios**:

1. **Given** the agent was mentioned, **When** it calls `soulstream_check_inbox`, **Then** it receives
   the mention notification(s) with topic, op-id, and author.
2. **Given** many notifications, **When** `soulstream_check_inbox` is called with a limit, **Then** at
   most that many are returned, newest first.
3. **Given** no mentions, **When** `soulstream_check_inbox` is called, **Then** an empty list is
   returned, not an error.

### User Story 4 - An agent attaches results and closes out (Priority: P2)

An agent attaches a text artefact (a summary, a CSV, a snippet) to a topic and closes a topic when
the work is done.

**Why this priority**: Agents produce artefacts and finish work; text attachments cover the common
agent output, and closing records completion. After conversation, so P2.

**Independent Test**: Call `soulstream_attach_text` with a name and body; confirm it is stored and
listed on the topic; call `soulstream_close_topic`; confirm the topic materialises as closed.

**Acceptance Scenarios**:

1. **Given** a topic, **When** the agent calls `soulstream_attach_text` with name, content type, and
   body, **Then** the text is stored as an attachment and its object key is returned.
2. **Given** a topic, **When** the agent calls `soulstream_close_topic`, **Then** a close transition
   is posted and the topic materialises as closed.

### Edge Cases

- **Missing/invalid required argument** (e.g. no topic path): the tool returns a clear tool-level
  error, not a crash.
- **A write tool with no persona configured**: the server refuses to start (a persona is mandatory
  for the write door) — or write tools return a clear "persona required" error.
- **Startup connection failure** (bad context, unreachable server, no JetStream): the server exits
  with a clear error before accepting tool calls.
- **`show_topic` / `close_topic` on a bad path**: reported as a tool error/empty view, never a panic.
- **Very large inbox / topic**: `check_inbox` is bounded by a limit; `show_topic` replays the
  (MVP-short) log.
- **A mention of a persona that doesn't exist**: still posted and notified (no registry) — the tool
  succeeds.

## Requirements *(mandatory)*

### Functional Requirements

**Server & session**

- **FR-001**: The adapter MUST run as an MCP server over stdio, exposing Soulstream operations as
  tools an MCP client can discover and call.
- **FR-002**: The server MUST be configured at startup with a named NATS context, a realm, and a
  persona; it MUST connect once and act as that single persona for every tool call.
- **FR-003**: Every write operation MUST be attributed to the configured persona; the adapter MUST
  NOT allow a tool caller to post as a different persona (no attribution laundering).
- **FR-004**: A startup failure (missing context, unreachable server, no JetStream, or a missing
  persona for the write door) MUST exit with a clear error before serving tool calls.

**Tools**

- **FR-005**: `soulstream_board` MUST return every topic once (path, name, lifecycle).
- **FR-006**: `soulstream_show_topic` MUST return a topic's materialised view (announcement,
  contributions with authors and mentions, attachments, lifecycle), reporting empty/malformed paths
  clearly.
- **FR-007**: `soulstream_start_topic` MUST announce a topic (name, optional subject, tags, parent)
  and return its path.
- **FR-008**: `soulstream_post_turn` MUST post a turn as the persona; `@name` in the body MUST be
  parsed and notified (via the library).
- **FR-009**: `soulstream_add_comment` MUST post a comment anchored to a given op-id.
- **FR-010**: `soulstream_attach_text` MUST store text content (name, content type, body) as an
  attachment and return its object key.
- **FR-011**: `soulstream_close_topic` MUST post a close transition.
- **FR-012**: `soulstream_check_inbox` MUST return the persona's mention notifications (topic, op-id,
  author), newest-first, bounded by a caller-supplied limit (with a sane default), empty when none.

**Behaviour & quality**

- **FR-013**: Each tool MUST declare a name, a description, and a typed input schema so an MCP client
  can call it correctly; results MUST be structured (JSON text) that a model and a human can read.
- **FR-014**: An invalid argument or an operation error MUST be returned as a tool-level error with a
  clear message, never a server crash.
- **FR-015**: Tool logic MUST be structured so it can be tested against an in-process server without
  going through stdio or a live MCP client.
- **FR-016**: The adapter MUST reuse the existing `realm`/`topic` library for all behaviour; it adds
  no new protocol operations and no new infrastructure.

### Key Entities *(include if feature involves data)*

- **MCP server**: a stdio process exposing the tool set, bound to one persona for its lifetime.
- **Tool**: a named operation with a typed input schema and a structured result, mapping to a
  `realm`/`topic` call.
- **Session config**: the resolved `{context, realm, persona}` the server connects with.
- **Inbox check result**: a bounded, newest-first list of `{topic, op_id, author}` notifications.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An agent can complete a full participation loop via tools — `board → show → start →
  post → comment → check_inbox → attach_text → close` — acting entirely as its configured persona,
  verified end to end.
- **SC-002**: 100% of write operations issued through tools are attributed to the configured persona
  (verified by a library read of the op authors); no tool can post as another persona.
- **SC-003**: `soulstream_check_inbox` returns exactly the mentions addressed to the persona, bounded
  by the limit, newest-first, and empty when there are none.
- **SC-004**: Every tool returns a structured, parseable result on success and a clear tool-level
  error on bad input (missing path, unknown topic) — no crashes across a tool matrix.
- **SC-005**: The whole adapter verifies green — tool logic tested against an in-process server, all
  tests pass (none skipped), formatting applied, linting clean, and the server binary builds.

## Assumptions

- **Library in place**: `001`–`004` merged; the adapter is a thin MCP shell over `realm` + `topic`.
- **One persona per session**: a single configured persona; multi-persona routing is out of scope.
- **stdio transport**: a locally-launched server the agent's MCP client spawns; HTTP/SSE is later.
- **Text artefacts**: MCP tool inputs are text, so `attach_text` covers text content; binary
  attachments come through the library/CLI, not the MCP door, this cycle.
- **Polling inbox**: mentions are caught by polling `check_inbox`; live push over MCP is later.

## Dependencies

- `realm` (connect/NewClient) and `topic` (board, materialise, post, comment, attach, transition,
  notifications) plus `internal/natstest` for tests. An MCP Go SDK for the server/tool machinery.
