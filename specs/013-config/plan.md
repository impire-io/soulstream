# Implementation Plan: Config-file identity resolution & self-installing plugin binary

**Branch**: `013-config` | **Date**: 2026-07-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/013-config/spec.md`

## Summary

Two client-side conveniences that complete the "identity follows the project" story.
(1) The five who-acts-where fields (context, realm, persona, key file, pins file) gain
two new sources below flags and environment: a project `.soulstream.json` found by
walking up from the working directory (nearest file only), and a user
`config.json` in the existing Soulstream config dir. Each field resolves
independently: **flag > env > project file > user file > unset**. A new
`soulstream config` command shows every field's effective value and true source. (2)
The Claude Code plugin's wrapper script becomes self-installing: override var > PATH >
per-version cached binary in the plugin data dir, else download the release archive
matching the plugin's version for this OS/arch, verify against `checksums.txt`, cache,
exec. Plugin bumps to 0.2.0 and a v0.2.0 release ships so the wrapper has matching
binaries to fetch.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire-io/soulstream`); POSIX shell (bash) for the wrapper
**Primary Dependencies**: stdlib only for the new Go code (`encoding/json`, `os`, `flag`, `path/filepath`); wrapper uses curl-or-wget + shasum-or-sha256sum + tar
**Storage**: two JSON files on disk (`.soulstream.json` per project, `<UserConfigDir>/soulstream/config.json`); cached binary under `${CLAUDE_PLUGIN_DATA}` (fallback `${XDG_DATA_HOME:-~/.local/share}/soulstream-plugin`)
**Testing**: `go test` — pure unit tests over temp dir trees (no NATS server needed for resolution); existing in-process-server tests for CLI integration
**Target Platform**: macOS/Linux for the self-install path (amd64+arm64); Windows keeps manual install
**Project Type**: library + two thin clients + plugin packaging
**Performance Goals**: resolution adds no measurable startup cost (a few stat calls + ≤2 small file reads); cached wrapper start adds one checksum of a ~10 MB file
**Constraints**: byte-for-byte behaviour preservation when no config files exist (SC-005); config files must never carry credentials (FR-007); failed downloads leave no cache (FR-010)
**Scale/Scope**: 1 new Go package, 2 entry points rewired, 1 new CLI command, 1 wrapper rewrite, 1 new ELI5 doc + 4 doc updates

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. NATS-Native First**: PASS — nothing here touches the wire. Config resolution is
  client-side assembly of the same five values that already exist; no new
  infrastructure, no state beside two tiny JSON files the user owns. No specific NATS
  server capability is relied on; minimum server version unchanged.
- **II. Smallest Viable Implementation**: PASS — no new knobs (the five fields keep
  their names and meanings), no `--config` flag, no cross-file merging on the walk-up,
  no format negotiation (JSON only), no Windows self-install. The `config` command
  prints and exits — it is the acceptance probe for the feature, not a new surface.
  The wrapper grows only the download-verify-cache path the spec demands.
- **III. ELI5 Documentation**: PASS — new `docs/configuration.md` (plain-words page
  with an everyday analogy: the sticker on the project folder that says who you are
  there), plus updates to `docs/cli.md`, `docs/mcp.md`, plugin README, and the setup
  skill. Docs are tasks inside the stories, not a polish phase.

Post-design re-check (after Phase 1): PASS — design added nothing beyond the above;
the only internal type is the resolved-field set both entry points already need.

## Project Structure

### Documentation (this feature)

```text
specs/013-config/
├── plan.md              # This file
├── research.md          # Phase 0: decisions + rationale
├── data-model.md        # Phase 1: config file shape, resolution model, cache layout
├── quickstart.md        # Phase 1: try-it walkthrough
├── contracts/
│   └── library.md       # internal/config API, `soulstream config` output, wrapper contract
└── tasks.md             # Phase 2 (/speckit-tasks — not created here)
```

### Source Code (repository root)

```text
internal/config/               # NEW — pure resolution logic, stdlib only, no NATS
├── config.go                  # File shape, strict load, walk-up discovery, user path
├── resolve.go                 # per-field precedence chain + Source provenance
└── config_test.go, resolve_test.go
internal/cli/
├── cli.go                     # flags default to "" + flag.Visit; resolve via internal/config;
│                              #   new `config` command + usage text
└── config_cmd.go, config_cmd_test.go   # NEW — the `soulstream config` printer
cmd/soulstream-mcp/main.go     # same resolution before connecting
plugins/soulstream/
├── .claude-plugin/plugin.json # version 0.2.0
├── scripts/soulstream-mcp.sh  # self-install rewrite
└── README.md                  # config file + self-install docs
.claude-plugin/marketplace.json  # plugin entry version 0.2.0
docs/configuration.md          # NEW ELI5 page
docs/cli.md, docs/mcp.md       # updated
plugins/soulstream/skills/setup/SKILL.md  # updated (config file is now the primary path)
```

**Structure Decision**: `internal/config` (not a public package): both entry points
share it, but the resolution chain is a client convenience, not protocol surface —
`record`/`identity`/`realm`/`topic`/`registry` are untouched. It imports no NATS,
keeping the pure/IO split the codebase already follows.

## Design essentials

**Resolution.** `config.Resolve(explicit File, cwd string) (Resolved, error)`:
`explicit` carries flag-set values only (detected via `flag.Visit`, so a flag left at
its default no longer swallows the env var — today's `flag.String("context",
os.Getenv(...))` pattern is replaced). Chain per field: explicit → `SOULSTREAM_*` env
(non-empty) → nearest project file walking up from `cwd` → user file → unset.
`Resolved` records value + provenance (`flag` / `env NAME` / `file PATH` / `unset`)
per field; `soulstream config` prints exactly that. Files load with
`DisallowUnknownFields`; any load error aborts with the file path in the message
(absent files skip). Relative `key_file`/`pins_file` values resolve against the config
file's own directory at load time; the resolved value then feeds
`keystore.ResolveKeyFile`/`ResolvePinsFile` unchanged (their env+default logic keeps
working when config contributes nothing).

**Wrapper.** Resolution order `SOULSTREAM_MCP_BIN` → `command -v soulstream-mcp` →
`$DATA/bin/v<version>/soulstream-mcp` (verified against the sha256 recorded at install
time; mismatch = re-download). Download path: plugin version parsed from the plugin's
own `plugin.json`; `uname -s/-m` → `darwin|linux` / `amd64|arm64`; fetch archive +
`checksums.txt` from the GitHub release into a temp dir under `$DATA`; verify; extract
`soulstream-mcp`; record its sha256; atomic `mv` into place; exec. Any failure exits
with the manual-install message and removes the temp dir (no partial cache).

## Complexity Tracking

No violations — table intentionally empty.
