# Contract: CLI Commands & MCP Tools

## CLI (`cmd/soulstream`, `internal/cli`)

Global flags gain `--key-file` and `--pins-file` (env: `SOULSTREAM_KEY_FILE`,
`SOULSTREAM_PINS_FILE`; defaults per contracts/wire-and-kv.md). When the key file
exists, every publishing command signs automatically; when absent, publishing is
unsigned — no flag, no prompt.

### New commands

| Command | Behaviour | Exit |
|---|---|---|
| `key init` | Generate a keypair for the session persona; write seed (0600). Refuses if the file exists. Prints the public key. | 0 / 1 |
| `key show` | Print the persona's public key (from the key file). | 0; 1 if no key |
| `key rotate` | Generate a new keypair, publish the rotation (proof by old key) to the directory, then replace the local seed file (old seed kept as `<file>.prev`). Requires an existing key and a published profile. | 0 / 1 |
| `profile publish` | Publish the persona's profile (create-only). Flags: `--display-name`, `--kind` (human\|agent\|service, default human), `--description`, `--operated-by`. Includes the public key when a key file exists. Error if the profile already exists (message points at `key rotate` / metadata update path). | 0 / 1 |
| `profile show <persona>` | Print a persona's directory profile, its validated chain, and its pin state (pinned / extended / distrusted). | 0; 1 unknown persona |

### Changed commands

- `provision` — reports the third artefact (`personas` KV bucket) in its
  create-or-report output.
- `show`, `watch`, `inbox` — each rendered op line carries a status glyph: `✓`
  verified, `✗` failed, `?` unknown-key, no glyph for unsigned. If any persona in the
  rendered set is distrusted, the first output line is a banner:
  `!! possible key substitution for <persona> — signatures from this persona are not trusted`
  written to stderr as well, so it is machine-distinguishable and unmissable (FR-007).
- All reading commands build the keyring: load pins → read directory (`registry.All`)
  → reconcile → persist extended pins → pass keyring to the library. A realm without
  the bucket yields a nil keyring (statuses degrade, nothing fails).

## MCP adapter (`cmd/soulstream-mcp`, `internal/mcpserver`)

Session env additions: `SOULSTREAM_KEY_FILE`, `SOULSTREAM_PINS_FILE` (same defaults).
Key present ⇒ every publishing tool signs automatically. The adapter never generates
keys (`key init`/`rotate` are operator actions via the CLI).

### New tool (total: 9)

| Tool | Input | Output |
|---|---|---|
| `publish_profile` | `display_name?`, `kind?` (human\|agent\|service, default agent), `description?`, `operated_by?` | the published profile incl. public key; error if the profile already exists |

### Changed tools

- Every tool result that returns ops (`read_topic`, `watch`-style reads, `fetch_inbox`,
  board/topic listings that include announcements) gains a `sig` field per op with
  `unsigned` / `verified` / `failed` / `unknown-key`, plus a top-level
  `distrusted_personas` list when non-empty — the loud surface an AI persona can act on.
- Keyring lifecycle mirrors the CLI: pins loaded at session start, reconciled per
  read, extended pins persisted.

## Shared behaviours (both clients)

- FR-005 guard: no client logic branches on `kind` or any profile field for anything
  but rendering. (Greppable: `kind ==` must only appear in presentation code.)
- Missing directory, missing profile, missing key file: all degrade silently to
  unsigned/unknown-key — never an error on read paths.
- Secret seed never appears in logs, errors, tool results, or stdout (only `key init`
  prints the *public* key).
