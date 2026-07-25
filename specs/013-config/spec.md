# Feature Specification: Config-file identity resolution & self-installing plugin binary

**Feature Branch**: `013-config`
**Created**: 2026-07-21
**Status**: Shipped (v0.2.0)
**Input**: User description: "Config-file resolution for Soulstream identity, plus a self-installing plugin binary. Today the CLI and MCP server resolve who-acts-where (NATS context, realm, persona, key file, pins file) from flags and environment variables only. Add a third and fourth source: a project-level config file and a user-level default. Per-field precedence flag > env > nearest project file > user file. The Claude Code plugin's wrapper script should self-install the soulstream-mcp binary from GitHub releases, verified, cached in the plugin data dir. Bump the plugin to 0.2.0."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Per-project identity from a config file (Priority: P1)

Daan works in two directories. `~/work/client-a` contains a `.soulstream.json` naming
the client's realm and his consultant persona; `~/impire/soulstream` contains one naming
the impire realm and his `daan` persona. Every Soulstream tool — the CLI and the MCP
server the Claude Code plugin spawns — picks up the identity of whichever project it
runs in, with no flags and no environment changes. His machine-wide defaults (the NATS
context he always uses) live once in a user-level file beside his keys.

**Why this priority**: This is the feature's reason to exist — identity should follow
the project, not the shell. The plugin spawns the MCP server with the project as its
working directory, so per-project identity makes the plugin usable across projects.

**Independent Test**: Create two directories with different `.soulstream.json` files;
run a read command in each; observe each connects with its own realm/persona without
flags or environment variables.

**Acceptance Scenarios**:

1. **Given** a directory whose `.soulstream.json` names realm `acme` and persona `ada`,
   **When** a command runs there with no flags and no environment, **Then** it acts as
   `ada` on `acme`.
2. **Given** the same directory is nested three levels below where `.soulstream.json`
   lives, **When** a command runs in the nested directory, **Then** the walk-up finds
   the file and the same identity applies.
3. **Given** a project file that names only `realm` and `persona`, and a user file that
   names only `context`, **When** a command runs, **Then** all three resolve — each
   field independently from the nearest source that provides it.
4. **Given** both a project file and an environment variable set for `persona`,
   **When** a command runs, **Then** the environment variable wins; **and given** a
   flag is also passed, **Then** the flag wins over both.
5. **Given** no config files, flags, or environment, **When** a command that needs a
   persona runs, **Then** the existing "persona required" failure appears unchanged.

---

### User Story 2 - Seeing where a value came from (Priority: P2)

When something connects to the wrong realm, Daan runs `soulstream config` and sees the
five resolved values side by side with the source of each — this flag, that environment
variable, this project file (with its path), the user file, or unset.

**Why this priority**: Four sources mean misconfiguration is now possible in four
places; without visibility, debugging is guesswork. It is also the natural acceptance
probe for Story 1.

**Independent Test**: Set overlapping values via file and environment, run
`soulstream config`, and check each row names the winning source truthfully.

**Acceptance Scenarios**:

1. **Given** persona set by flag, realm by environment, context by project file, and
   pins file by nothing, **When** `soulstream config` runs, **Then** each row shows the
   effective value and its true source, and unset fields say so.
2. **Given** no configuration at all, **When** `soulstream config` runs, **Then** it
   prints all fields as unset and exits successfully — it never requires a connection.

---

### User Story 3 - The plugin installs its own binary (Priority: P2)

A new user installs the Soulstream plugin in Claude Code on a machine with no Go
toolchain and nothing pre-installed. On first connection the plugin fetches the exact
release build matching the plugin's own version for their OS and architecture, verifies
it against the published checksums, caches it in the plugin's data directory, and runs
it. Later plugin updates fetch their matching build; a developer with their own binary
on PATH (or `SOULSTREAM_MCP_BIN` set) is left alone.

**Why this priority**: Removes the last manual step between "install plugin" and
"working tools". Depends on nothing from Story 1, but the two together complete the
per-project story.

**Independent Test**: With an empty plugin data dir, no binary on PATH, and no
override set, launch the wrapper; observe a verified download, a cached binary, and a
working server. Launch again; observe no second download.

**Acceptance Scenarios**:

1. **Given** no binary anywhere, **When** the wrapper starts, **Then** it downloads the
   release archive matching the plugin version and this OS/arch, verifies the checksum,
   caches the binary, and serves.
2. **Given** a cached binary from a previous start, **When** the wrapper starts,
   **Then** it uses the cache and performs no network access.
3. **Given** `SOULSTREAM_MCP_BIN` points at a binary, **When** the wrapper starts,
   **Then** that binary is used — no PATH search, no download.
4. **Given** a binary on PATH and nothing cached, **When** the wrapper starts, **Then**
   PATH wins and no download happens.
5. **Given** the download or checksum verification fails, **When** the wrapper starts,
   **Then** it exits with a message naming what failed and the manual install options,
   and caches nothing.

---

### Edge Cases

- A config file with invalid JSON, or a field name it does not recognise: the command
  fails immediately, naming the file path and the problem — a typo must never silently
  fall through to a different realm.
- A config file that is valid but empty (`{}`): contributes nothing; resolution
  continues to the next source.
- Two `.soulstream.json` files on the walk-up path (nested projects): only the nearest
  is consulted — project files do not stack.
- Relative `key_file`/`pins_file` paths in a config file: resolved relative to the
  directory containing that file, so a project file's `./keys/ci.ed25519` means the
  project's own directory, not wherever the command happened to run.
- The walk-up reaches the filesystem root without finding a project file: not an error;
  the user file (then defaults) apply.
- A project file names a persona whose signing key is absent on this machine: identical
  to today's behaviour for a persona without a key — operations are unsigned; naming an
  identity in a file grants nothing, because keys are resolved locally per
  realm+persona and never travel in config.
- Corrupted cached binary (partial earlier download): the wrapper re-verifies the
  cached file's checksum before trusting it and re-downloads on mismatch.
- Machine without either checksum tool or without a download tool: the wrapper reports
  which tool is missing and falls back to the manual install message.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: All Soulstream entry points (terminal client and MCP server) MUST resolve
  each of the five identity fields — context, realm, persona, key file, pins file —
  independently through the precedence chain: command-line flag, then environment
  variable, then nearest project config file, then user config file, then unset.
- **FR-002**: The project config file MUST be named `.soulstream.json` and MUST be
  discovered by walking from the working directory upward to the filesystem root,
  consulting only the nearest file found.
- **FR-003**: The user config file MUST be `config.json` inside the existing Soulstream
  user configuration directory (the one already holding `keys/` and `pins/`).
- **FR-004**: A config file that cannot be parsed, or that contains an unrecognised
  field, MUST fail the command with an error naming the offending file path; an absent
  file MUST be silently skipped.
- **FR-005**: Relative key-file and pins-file paths in a config file MUST resolve
  against the directory containing that config file.
- **FR-006**: A `soulstream config` command MUST print each field's effective value and
  its source (flag, environment variable name, config file path, or unset) without
  requiring a server connection, and exit successfully.
- **FR-007**: Config files MUST NOT be able to carry credentials or signing material;
  they name an identity only. Signing keys continue to resolve locally per
  realm+persona.
- **FR-008**: Existing flag and environment behaviour MUST be unchanged for users who
  have no config files.
- **FR-009**: The plugin wrapper MUST resolve the server binary in order: explicit
  override variable, PATH, cached copy in the plugin data directory; when none exists
  it MUST download the release archive matching the plugin's own version and the
  current OS/architecture, verify it against the release's published checksums, cache
  the binary per version, and run it.
- **FR-010**: A failed download or failed verification MUST leave no cached binary and
  MUST produce an error naming the failure and the manual install alternatives.
- **FR-011**: The plugin version MUST be bumped to 0.2.0, and a matching release of the
  binaries MUST exist so the wrapper has something to fetch.
- **FR-012**: Plain-words documentation MUST cover the config file (both levels, the
  precedence chain, the security boundary) and the plugin's self-install behaviour;
  the plugin README and setup skill MUST be updated to match.

### Key Entities

- **Identity configuration**: the five who-acts-where fields (context, realm, persona,
  key file, pins file); exists per invocation, assembled from up to four sources.
- **Project config file**: `.soulstream.json`, committed alongside a project, naming
  some or all fields for work done under that directory tree.
- **User config file**: machine-wide defaults in the user's Soulstream config
  directory, same shape as the project file.
- **Cached server binary**: a per-version, checksum-verified copy of the MCP server
  kept in the plugin's data directory, surviving plugin updates.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user with two configured project directories switches realms by
  changing directory — zero flags, zero environment edits, in both the terminal client
  and the plugin-spawned server.
- **SC-002**: For any misresolved field, one invocation of the inspection command
  reveals the winning source, including the full path of the file responsible.
- **SC-003**: On a machine with no Go toolchain and no pre-installed binary, installing
  the plugin and connecting yields working tools with zero manual binary steps beyond
  plugin install itself.
- **SC-004**: A second connection performs no downloads (cache hit), and a tampered or
  truncated cached binary is detected and replaced rather than executed.
- **SC-005**: Every pre-existing invocation style (flags only, environment only)
  behaves byte-for-byte as before when no config files are present.

## Assumptions

- The five fields keep their existing names and meanings; no new knobs are introduced.
- JSON is the config format — it matches the operation record and the persona
  directory, and needs nothing new.
- Only the nearest project file applies (no cross-file merging on the walk-up); the
  project and user layers do merge per-field, which is what makes "context in the user
  file, realm/persona in the project file" work.
- An explicit `--config`-style flag pointing at an arbitrary file is out of scope —
  discovery by directory is the feature; a sixth source would blur precedence.
- The wrapper's self-install targets macOS and Linux (both architectures each);
  Windows users install the binary manually as today. The wrapper is a shell script
  and Claude Code on Windows does not run it through the same path.
- The plugin data directory is provided by the host (Claude Code) and survives plugin
  updates; when absent (e.g. very old host), the wrapper falls back to a stable
  location under the user's home directory.
- Release archives and checksums continue to be published in the shapes 012
  established (`soulstream_<version>_<os>_<arch>.tar.gz` + `checksums.txt`).
