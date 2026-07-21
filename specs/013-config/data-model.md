# Data Model: 013-config

## Config file (both levels, same shape)

`.soulstream.json` (project) and `<UserConfigDir>/soulstream/config.json` (user):

```json
{
  "context": "personal",
  "realm": "soulstream",
  "persona": "daan",
  "key_file": "./keys/daan.ed25519",
  "pins_file": ""
}
```

- Every field optional; absent/empty fields contribute nothing.
- Unknown fields are an error (strict decode), naming the file path.
- `key_file`/`pins_file`: if relative, resolved against the config file's directory
  at load time; the stored value thereafter is absolute.
- The file can never carry credentials — values are names/paths only; signing seeds
  stay in the local keystore, resolved per realm+persona.

## Resolution model (in-memory, per invocation)

| Type | Fields | Notes |
|---|---|---|
| `config.File` | Context, Realm, Persona, KeyFile, PinsFile (strings) | the five values; also the JSON shape |
| `config.Resolved` | one `Value` per field | final assembly both entry points consume |
| `config.Value` | `V string`, `Source Source` | provenance travels with the value |
| `config.Source` | `Kind` (flag / env / project / user / unset) + `Detail` | Detail = env var name or file path |

Chain per field, first non-empty wins:

1. explicit (flag the user actually passed — `flag.Visit`)
2. environment (`SOULSTREAM_CONTEXT`, `SOULSTREAM_REALM`, `SOULSTREAM_PERSONA`,
   `SOULSTREAM_KEY_FILE`, `SOULSTREAM_PINS_FILE`)
3. nearest `.soulstream.json` walking up from the working directory
4. user `config.json`
5. unset (`""` — downstream behaviour unchanged: persona-required errors,
   keystore default paths)

State: none persisted; resolution is a pure function of (explicit values, environ,
filesystem at cwd).

## Cached server binary (plugin data dir)

```text
$DATA/                             # ${CLAUDE_PLUGIN_DATA:-${XDG_DATA_HOME:-$HOME/.local/share}/soulstream-plugin}
└── bin/
    └── v0.2.0/
        ├── soulstream-mcp         # verified binary, 0755
        └── soulstream-mcp.sha256  # hex digest recorded at install time
```

- Lifecycle: created only after checksum verification succeeds (temp dir + atomic
  move); re-verified against the recorded digest on every start; deleted and
  re-fetched on mismatch; new plugin version ⇒ new `v<version>/` dir, old dirs left
  behind (harmless, tiny, debuggable).
