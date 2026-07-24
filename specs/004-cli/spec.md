# Feature Specification: CLI Client for Humans

**Feature Branch**: `004-cli`
**Created**: 2026-07-12
**Status**: Shipped (v0.1.0)
**Input**: User description: "A minimal command-line client so a human can use Soulstream from a terminal: provision a realm, list the board, start a topic, show/watch a topic, post turns and comments (with @mentions), attach and download files, close a topic, and watch their notification inbox. Connects via a named NATS context."

## Overview

The library makes Soulstream usable *from Go*; this feature makes it usable *from a terminal*. A
person runs `soulstream <command>` to do everything a persona does — start topics, converse,
mention colleagues, attach files, follow along live, and catch mentions — without writing code.

The consumers are **humans operating a persona** from the shell (and scripts). The value: the
dogfood scenario ("one human, two agents, one real project run in topics") needs a human door, and
the terminal is the smallest one. It also proves the library is ergonomic enough to drive.

## Clarifications

### Session 2026-07-12

- Q: How is the CLI configured (which server, realm, persona)? → A: A named NATS context supplies
  the server + credentials (`--context`, env `SOULSTREAM_CONTEXT`); the realm and persona come from
  `--realm`/`SOULSTREAM_REALM` and `--persona`/`SOULSTREAM_PERSONA`. Flags override env. The persona
  is required for write commands and optional for read-only ones.
- Q: What commands are in scope? → A: `provision`, `board`, `start`, `show`, `watch`, `post`,
  `comment`, `attach`, `get`, `close`, `inbox`. (`edit`, `reply`, `resolve`, sub-topic-specific
  helpers, and admin/registry commands are out of scope.)
- Q: What output format? → A: Human-readable text by default; `board` and `show` also accept
  `--json` for scripting. Errors go to stderr with a non-zero exit code; success is exit 0.
- Q: How does `watch`/`inbox` terminate? → A: They stream until interrupted (Ctrl-C / SIGINT), then
  exit 0 cleanly. All other commands are one-shot.
- Q: How are mentions handled by the CLI? → A: The CLI passes bodies through to the library, which
  already parses `@name` and fires notifications — the CLI adds nothing, so `post`/`comment` bodies
  containing `@name` "just work".
- Q: Dependency policy for arg parsing? → A: Standard library `flag` only (global flags + a
  subcommand + per-subcommand flags); no third-party CLI framework, per the smallest-viable
  principle.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Set up and see what's there (Priority: P1)

A person points the CLI at a realm, provisions it if needed, and lists the topics on the board.

**Why this priority**: The first thing anyone does is connect and look around. Provision + board is
the smallest end-to-end proof the CLI can talk to a realm.

**Independent Test**: Against a running server + context, run `provision` then `board`; confirm the
realm is ready and the board lists existing topics (or is empty) with names and lifecycle.

**Acceptance Scenarios**:

1. **Given** a context and realm, **When** the person runs `provision`, **Then** the realm's stream
   and object store are ensured and a per-artefact result is printed.
2. **Given** a provisioned realm, **When** they run `board`, **Then** each topic is listed once with
   its path, name, and lifecycle; `--json` prints the same as machine-readable JSON.
3. **Given** an empty realm, **When** they run `board`, **Then** an empty list is printed (exit 0),
   not an error.
4. **Given** a missing context or unreachable server, **When** any command runs, **Then** it prints
   a clear error to stderr and exits non-zero.

### User Story 2 - Start a topic and converse (Priority: P1)

A person starts a topic, posts turns and comments (mentioning colleagues), and prints the topic's
current state.

**Why this priority**: This is the actual work — starting and using a topic from the terminal. It's
the core of the human door.

**Independent Test**: Run `start`, capture the topic path, `post` a turn with an `@mention`,
`comment` on it, then `show` the topic and see the contributions and mention.

**Acceptance Scenarios**:

1. **Given** a persona, **When** they run `start "Q2 VAT"` with optional subject/tags/parent, **Then**
   the topic is announced and its path is printed.
2. **Given** a topic path, **When** they run `post <path> "…@bookkeeper-agent…"`, **Then** the turn
   is posted and the mentioned persona is notified (by the library).
3. **Given** an operation to anchor to, **When** they run `comment <path> <op-id> "…"`, **Then** the
   comment is posted anchored to that op.
4. **Given** a topic with activity, **When** they run `show <path>`, **Then** the baseline, ordered
   contributions (with authors and any mentions), attachments, and lifecycle are printed; `--json`
   emits the materialised view.
5. **Given** a write command with no persona configured, **When** it runs, **Then** it errors
   clearly that a persona is required.

### User Story 3 - Follow along and catch mentions (Priority: P2)

A person watches a topic update live in their terminal, and separately watches their inbox for
mentions, until they interrupt.

**Why this priority**: Live presence is what makes collaboration feel real, but conversing (P1)
works without it, so it's P2.

**Independent Test**: In one process run `watch <path>`; from another, post a turn; confirm the
watcher prints the new turn. Run `inbox`; from another persona, mention this one; confirm the
notification prints. Interrupt each and confirm a clean exit.

**Acceptance Scenarios**:

1. **Given** a topic, **When** `watch <path>` is running and someone posts, **Then** the new
   contribution is printed live.
2. **Given** a persona, **When** `inbox` is running and the persona is mentioned, **Then** the
   notification (topic, op-id, author) is printed live.
3. **Given** a running `watch` or `inbox`, **When** the person presses Ctrl-C, **Then** it exits 0
   without a stack trace.

### User Story 4 - Exchange files and close out (Priority: P2)

A person attaches a file to a topic, another downloads it, and someone closes the topic when done.

**Why this priority**: Files and closing complete the workbench lifecycle, but come after
conversation.

**Independent Test**: `attach <path> file.csv`; `show` lists it; `get <object> out.csv` reproduces
the bytes; `close <path>` then `show` reports the topic closed.

**Acceptance Scenarios**:

1. **Given** a file, **When** they run `attach <path> <file> [--type] [--anchor]`, **Then** it's
   stored and referenced, and its object key is printed.
2. **Given** an attachment's object key, **When** they run `get <object> <outfile>`, **Then** the
   original bytes are written to the outfile and verified against the digest.
3. **Given** a topic, **When** they run `close <path>`, **Then** a close transition is posted and
   `show` reports lifecycle `closed`.

### Edge Cases

- **Unknown command / missing args**: prints usage to stderr, exits non-zero.
- **`--json` on a streaming command** (`watch`/`inbox`): each event is emitted as one JSON object per
  line (JSON Lines) so streams stay scriptable. (Optional; text is the default.)
- **`get` to an existing file**: overwrites only with `--force`; otherwise errors (don't clobber).
- **`get` of a digest that doesn't match**: reports the mismatch and exits non-zero (don't write
  corrupt bytes silently — or writes then warns; the CLI verifies).
- **Posting to a closed topic**: the library warns; the CLI surfaces the warning but still posts.
- **SIGINT mid-one-shot**: the command is abandoned; partial publishes already acknowledged remain
  (idempotent by op-id) — acceptable.

## Requirements *(mandatory)*

### Functional Requirements

**Configuration & connection**

- **FR-001**: The CLI MUST take its server + credentials from a named NATS context (`--context` or
  `SOULSTREAM_CONTEXT`), the realm from `--realm`/`SOULSTREAM_REALM`, and the persona from
  `--persona`/`SOULSTREAM_PERSONA`; flags override environment.
- **FR-002**: Write commands MUST require a persona and error clearly when none is set; read-only
  commands (`board`, `show`, `get`) MUST work without one.
- **FR-003**: On a connection failure (missing context, unreachable server, no JetStream), the CLI
  MUST print a clear error to stderr and exit non-zero.

**Commands**

- **FR-004**: `provision` MUST ensure the realm's artefacts and print the per-artefact result.
- **FR-005**: `board` MUST list every topic once (path, name, lifecycle); `--json` prints JSON.
- **FR-006**: `start <name>` MUST announce a topic (optional `--subject`, `--tag` (repeatable),
  `--parent`) and print the new topic path.
- **FR-007**: `show <path>` MUST print the materialised topic (baseline, contributions with authors
  and mentions, attachments, lifecycle); `--json` emits the view.
- **FR-008**: `post <path> <body>` MUST post a turn (mentions handled by the library).
- **FR-009**: `comment <path> <anchor-op-id> <body>` MUST post an anchored comment.
- **FR-010**: `attach <path> <file>` (optional `--type`, `--anchor`) MUST store the file and print
  its object key.
- **FR-011**: `get <object> <outfile>` MUST write the attachment's bytes and verify the digest;
  it MUST NOT overwrite an existing file without `--force`.
- **FR-012**: `close <path>` MUST post a close transition.
- **FR-013**: `watch <path>` MUST stream contributions live until interrupted, then exit 0.
- **FR-014**: `inbox` MUST stream the persona's notifications live until interrupted, then exit 0.

**Behaviour & quality**

- **FR-015**: Unknown commands or missing required arguments MUST print usage to stderr and exit
  non-zero.
- **FR-016**: Success MUST exit 0; any error MUST exit non-zero with a message on stderr.
- **FR-017**: The CLI MUST use only the standard library for argument parsing (no third-party CLI
  framework).
- **FR-018**: Command logic MUST be structured so it can be tested against an in-process server
  without going through the named-context connection path.

### Key Entities *(include if feature involves data)*

- **CLI invocation**: global flags (`--context`, `--realm`, `--persona`, `--json`), a subcommand,
  and per-subcommand flags/args.
- **Command**: a single verb (`board`, `start`, …) mapping to a library call, with text or JSON
  output.
- **Config**: the resolved `{context, realm, persona}` from flags + environment.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A person can go from nothing to a used topic entirely from the terminal —
  `provision → start → post → comment → show` — and see their contributions, verified end to end.
- **SC-002**: `board` and `show` produce valid JSON under `--json` that round-trips to the same data
  a library caller would see.
- **SC-003**: `attach` then `get` reproduces a file byte-for-byte with a verified digest in 100% of
  cases; `get` refuses to clobber without `--force`.
- **SC-004**: `watch` and `inbox` print a live event within a second of it being posted from another
  process, and exit 0 on SIGINT.
- **SC-005**: Every command exits 0 on success and non-zero with a stderr message on error; unknown
  commands print usage — verified across a command matrix, with the whole CLI green (tests pass,
  none skipped; formatting; lint).

## Assumptions

- **Library in place**: `001`–`003` merged; the CLI is a thin shell over `realm` + `topic`.
- **One persona per invocation**: the CLI acts as a single persona (from config); multi-persona
  sessions are out of scope.
- **Terminal output**: plain text; no TUI/curses layer this cycle (a richer TUI is later).
- **Context management**: creating NATS contexts (`nats context add`) is the operator's job, done
  outside the CLI.

## Dependencies

- `realm` (connect/provision, `NewClient`) and `topic` (all of it), plus `internal/natstest` for
  tests. A reachable/embedded NATS server for the integration scenarios.
