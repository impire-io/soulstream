# Research: 013-config

Decisions resolving everything the plan needed. No open questions remain.

## R1 — Config format: JSON, strict

- **Decision**: JSON with `json.Decoder.DisallowUnknownFields`; field names
  `context`, `realm`, `persona`, `key_file`, `pins_file`.
- **Rationale**: JSON is already the project's lingua franca (operation record, JCS,
  persona directory) and costs zero dependencies. Strict decoding turns a typo
  (`presona`) into a loud error naming the file instead of a silent fall-through to a
  different identity — the worst failure mode this feature could have.
- **Alternatives considered**: TOML/YAML (new dependency, violates smallest-viable);
  lenient decoding with warnings (a warning on stderr is invisible under an MCP
  client — fail loud is the only observable behaviour).

## R2 — Distinguishing "flag set" from "flag defaulted"

- **Decision**: flags default to `""`; after `Parse`, `flag.Visit` collects the flags
  the user actually passed. Only visited flags enter the chain as `flag` source.
- **Rationale**: today's pattern (`flag.String("context", os.Getenv(...), …)`) bakes
  the env into the default, making flag and env indistinguishable — fine when they
  were the only two sources, wrong once `soulstream config` must name the true source
  and files sit below env. `flag.Visit` is stdlib and exact.
- **Alternatives considered**: sentinel defaults (fragile); a custom flag.Value
  recording set-ness (more code for the same answer).

## R3 — Walk-up semantics: nearest file only

- **Decision**: from the working directory upward to the filesystem root, the first
  `.soulstream.json` found is the only project file consulted. No stacking, no
  cross-file merging between nested project files. The project layer and the user
  layer DO merge per-field (that is the whole chain).
- **Rationale**: mirrors how git finds `.git` — one project, one identity file; a
  mental model users already have. Stacking nested files would make "which realm am I
  on?" depend on an unbounded set of files.
- **Alternatives considered**: merge all files on the path (rejected: unpredictable);
  stop at a repo boundary like `.git` (rejected: Soulstream should not care whether a
  project is a git repo).

## R4 — User-level file location

- **Decision**: `<os.UserConfigDir()>/soulstream/config.json` — beside the existing
  `keys/` and `pins/` directories.
- **Rationale**: `internal/keystore` already established that directory as
  Soulstream's home; a second root would fragment it.

## R5 — Relative paths in config files

- **Decision**: `key_file`/`pins_file` relative values resolve against the directory
  containing the config file, at load time.
- **Rationale**: a project file saying `./ci/key.ed25519` must mean the project's own
  tree regardless of which subdirectory the command runs in; resolving at load keeps
  provenance simple (the resolved absolute path is the value).

## R6 — MCP server working directory

- **Decision**: the MCP server resolves from its own working directory; no special
  flag. Claude Code spawns plugin MCP servers with the project directory as cwd,
  which is exactly the per-project behaviour the spec wants.
- **Rationale**: verified against Claude Code's documented plugin behaviour during
  012; also matches any other MCP client that sets cwd per workspace.

## R7 — Wrapper download tooling

- **Decision**: `curl -fsSL` with `wget -qO-` fallback; `shasum -a 256` with
  `sha256sum` fallback; `tar -xzf` (universal). Plugin version parsed from the
  plugin's own `.claude-plugin/plugin.json` with `sed` (no jq dependency). Missing
  tool → named error + manual-install message.
- **Rationale**: macOS ships curl+shasum, mainstream Linux ships curl-or-wget +
  sha256sum; none of this needs installing in practice, and every failure path
  degrades to the manual instructions that exist today.
- **Alternatives considered**: requiring jq (extra dependency for one field); a Go
  self-installer binary (chicken-and-egg).

## R8 — Cache layout & integrity

- **Decision**: `$DATA/bin/v<version>/soulstream-mcp` + `soulstream-mcp.sha256`
  beside it, where `$DATA` is `${CLAUDE_PLUGIN_DATA:-${XDG_DATA_HOME:-$HOME/.local/share}/soulstream-plugin}`.
  The recorded sha256 is re-checked on every start; mismatch or absence of the
  `.sha256` file forces re-download. Downloads happen in a temp dir under `$DATA`
  and reach the final path only via `mv` after verification.
- **Rationale**: per-version dirs make plugin upgrades fetch matching binaries with
  no invalidation logic; re-checking a ~10 MB file costs milliseconds and closes the
  truncated-download and tampered-cache windows (SC-004); temp-dir + atomic move
  guarantees FR-010's "no partial cache".

## R9 — Release pairing

- **Decision**: bump plugin to 0.2.0 and tag `v0.2.0` in the same delivery; the
  wrapper fetches the release equal to its plugin version.
- **Rationale**: version-pinned fetches make plugin and binary a matched pair —
  no "latest" drift, reproducible installs, and the 012 release pipeline already
  publishes exactly the needed artifact names.
