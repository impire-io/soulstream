# Contract: `soulstream` CLI

## Global

```
soulstream [--context <name>] [--realm <name>] [--persona <name>] <command> [args…]
```

Env fallbacks: `SOULSTREAM_CONTEXT`, `SOULSTREAM_REALM`, `SOULSTREAM_PERSONA` (flags override).
Exit 0 = success; non-zero + stderr message = error/usage.

## Commands

| Command | Args / flags | Does |
|---|---|---|
| `provision` | — | Ensure the realm's stream + object store; print per-artefact result. |
| `board` | `--json` | List every topic (path, name, lifecycle). |
| `start <name>` | `--subject`, `--tag` (repeat), `--parent` | Announce a topic; print its path. |
| `show <path>` | `--json` | Print the materialised topic (baseline, contributions+mentions, attachments, lifecycle). |
| `post <path> <body>` | — | Post a turn (mentions handled by the library). |
| `comment <path> <op-id> <body>` | — | Post a comment anchored to `<op-id>`. |
| `attach <path> <file>` | `--type`, `--anchor` | Store the file; print its object key. |
| `get <object> <outfile>` | `--force` | Write bytes, verify digest; no clobber without `--force`. |
| `close <path>` | — | Post a close transition. |
| `watch <path>` | — | Stream contributions live until SIGINT (exit 0). |
| `inbox` | — | Stream this persona's notifications live until SIGINT (exit 0). |

Write commands (`start`,`post`,`comment`,`attach`,`close`) require `--persona`; read commands
(`board`,`show`,`get`) do not.

## Testable core

```go
package cli

type Config struct { Context, Realm, Persona string }
type Connector func(ctx context.Context, cfg Config) (*realm.Client, error)

// Run parses args, resolves config, connects via connect, dispatches; returns the exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, connect Connector) int

// Main wires os streams + a SIGINT context + the natscontext connector.
func Main(args []string) int
```

Tests call `Run` with an injected `Connector` that returns a `realm.NewClient` over an in-process
server (FR-018), asserting stdout + exit code.
