# Contracts: 013-config

## `internal/config` (Go, stdlib only, imports no NATS)

```go
// File is the shape of both config files and of explicitly-passed values.
type File struct {
    Context  string `json:"context"`
    Realm    string `json:"realm"`
    Persona  string `json:"persona"`
    KeyFile  string `json:"key_file"`
    PinsFile string `json:"pins_file"`
}

type SourceKind string // "flag" | "env" | "project" | "user" | "unset"

type Source struct {
    Kind   SourceKind
    Detail string // env var name, or config file path; "" for flag/unset
}

type Value struct {
    V      string
    Source Source
}

type Resolved struct {
    Context, Realm, Persona, KeyFile, PinsFile Value
}

// Resolve assembles the five fields: explicit > env > nearest project file
// (walk-up from cwd) > user file > unset. explicit holds ONLY values the user
// passed as flags (callers use flag.Visit to fill it). Load errors (malformed
// JSON, unknown fields, unreadable file) abort with the file path in the error;
// absent files are skipped. Relative key/pins paths in a file are resolved
// against that file's directory.
func Resolve(explicit File, cwd string) (Resolved, error)

// Fields returns the five values in stable order with their canonical names —
// what the CLI `config` command prints.
func (r Resolved) Fields() []struct{ Name string; Value }
```

Guarantees:

- Pure: no NATS, no network; reads at most two config files plus directory stats.
- `Resolve(File{}, cwd)` with no env and no files ⇒ all fields `unset`, no error.
- Precedence is per-field: sources mix freely across fields.
- Byte-for-byte compatibility: when no config files exist, resolved values equal
  today's flag-else-env behaviour exactly.

## CLI: `soulstream config`

```
$ soulstream config
context   personal    env SOULSTREAM_CONTEXT
realm     soulstream  project /Users/daan/impire/soulstream/.soulstream.json
persona   daan        flag
key_file  (unset)     — keystore default applies
pins_file (unset)     — keystore default applies
```

- Never connects; exit 0 even fully unset; exit 1 only on a config-file load error
  (same error every other command would give).
- Columns: field, effective value (`(unset)` when empty), source description.

## Both entry points (CLI + soulstream-mcp)

- Flags no longer default to `os.Getenv(...)`; resolution happens after parse via
  `config.Resolve`. Help text documents the chain. All existing errors ("persona
  required", keystore messages) unchanged in wording.

## Plugin wrapper (`scripts/soulstream-mcp.sh`)

Resolution order (first hit wins):

1. `SOULSTREAM_MCP_BIN` — used as-is, no verification (developer override).
2. `soulstream-mcp` on PATH.
3. `$DATA/bin/v<ver>/soulstream-mcp` where recorded sha256 verifies; else:
4. Download `soulstream_<ver>_<os>_<arch>.tar.gz` + `checksums.txt` from
   `github.com/impire-io/soulstream/releases/download/v<ver>/`, verify archive
   against checksums.txt, extract, record binary sha256, atomic move, exec.

`<ver>` = the plugin's own version read from `.claude-plugin/plugin.json`.
Failure contract: any missing tool, failed fetch, or checksum mismatch ⇒ stderr
message naming the failure + the three manual install options, exit 1, temp dir
removed, cache untouched. Windows: not served by this script (manual install).
