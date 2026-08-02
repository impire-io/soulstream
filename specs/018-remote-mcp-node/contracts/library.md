# Library Contract: public MCP surface + node module

## 1. `mcpserver` (promoted, core module)

```go
// Package mcpserver serves soulstream's MCP tool surface for any host
// that brings a connected realm client.
package mcpserver

// NewServer builds the full tool surface (23 tools) bound to c.
// The client carries the session's whole identity: realm, persona, signer.
// Options customize host-specific concerns; zero options == the stdio
// adapter's exact behavior (file-backed pins keyring), byte-for-byte.
func NewServer(c *realm.Client, opts ...Option) *mcp.Server

// WithKeyring injects the reader-verification keyring provider, replacing
// the default per-realm pins file. Hosts that multiplex principals (the
// remote node, the single-binary house) MUST inject one — a shared
// filesystem cache is the wrong shape for multi-tenant verification.
// The provider may return (nil, nil): verification degrades to
// unknown-key exactly as a missing pins file does today.
func WithKeyring(func(ctx context.Context) (*identity.Keyring, error)) Option
```

**Guarantees**:
- Transport-agnostic: the returned `*mcp.Server` binds to any go-sdk
  transport (`StdioTransport`, `StreamableHTTPHandler`, in-process).
- No handler touches the filesystem, environment, or process identity —
  everything flows from `c` and the injected option. (The default keyring
  is the one legacy exception, preserved for the stdio adapter.)
- Tool names, schemas, and behavior are IDENTICAL to v0.6.0's
  `internal/mcpserver` — promotion moves code, it does not edit surface —
  with ONE deliberate addition (analysis I1): `soulstream_whoami`, which
  reports the session's persona, realm, and signer public key, so a
  remote user can see who the admission edge decided they are. 24 tools
  total.
- BREAKING for nobody: `internal/mcpserver` had no external importers
  (it could not, by definition).

## 2. `node` (nested module `github.com/impire-io/soulstream/node`)

```go
// Package node is the remote MCP door: streamable HTTP in, bearer
// passthrough to NATS auth-callout admission, one pooled connection per
// admitted principal. It custodies nothing.
package node

type Config struct {
    Listen       string // HTTP bind address (default 127.0.0.1:8080)
    PublicURL    string // canonical fronted URL; "" = local mode (no OAuth metadata)
    AuthIssuer   string // external AS issuer; REQUIRED when PublicURL != ""
    Realm        string // realm name (required)
    NATSURL      string // NATS server URL (required)
    SentinelPath string // optional creds file routing connections to callout
    Prefix       string // soulidentity subject root (default "soulidentity")
}

// New validates cfg (teaching errors) and returns a ready-to-serve node.
func New(cfg Config) (*Node, error)

// Handler returns the http.Handler: MCP endpoint + well-known metadata.
// The caller owns the listener (the binary wraps it; embedders — the
// single-binary house — mount it on their own mux).
func (n *Node) Handler() http.Handler

// Close drains the pool: closes every pooled connection and realm client.
func (n *Node) Close()
```

**Guarantees** (each test-asserted):
- **Custody**: no method persists a token, key, or per-user secret; `Close`
  + restart loses nothing a re-presented bearer can't rebuild (SC-004).
- **Non-interference** (FR-005/R4): serve-only-over-own-principal's-entry;
  `latest` updates only via bound-session or admitted-probe paths; forged
  hints cause zero adoption/eviction/displacement (SC-002).
- **Freshness**: TokenHandler presents the entry's `latest` on every
  (re)connect; a session presenting fresh bearers survives ≥3× callout TTL
  (SC-003, prototype Bar 3 re-run through the full tool surface).
- **Eviction**: corpses (build error, auth-closed) evicted on next touch;
  refusals non-sticky.
- **Silence about tokens**: log lines carry principal + hint class + cause,
  never token material (SC-006 grep audit).
- **Cycle guard** (FR-014): this module imports soulstream and soulidentity
  public packages; neither core repo imports it or each other — asserted by
  a go.mod check in tests (mirror of 017's rule).

## 3. `cmd/soulstream-node`

Flags mirror `Config` (env fallbacks `SOULSTREAM_NODE_*`), `-version`.
Serves `Handler()` on `Listen`. Exit codes: config error 2, serve error 1.
Startup teaching checks: realm stream present (R9), issuer set when public.

## 4. Carried tooling (best-effort, spec Q2)

- `node/cmd/byon-setup`: Synadia Cloud callout wiring (prototype shape:
  scoped programmatic sk-group for users, non-programmatic issuer group,
  callout enablement + XKey surfaced loudly, creds download). Inputs:
  `SYNADIA_PAT`, `--system`, `--app-account`, `--control-account`, `--out`,
  `--apply`.
- `node/cmd/probe`: live pass protocol (`initialize → tools/list → whoami →
  board → start_topic → post_turn`) + realm-side verification through an
  independent reader (negative control → keyring → `SigVerified`,
  `Author == persona`). The follow-up measurement's driver.

## 5. Consumer expectations (SoulNode Phase 2)

The embedding host brings: its own listener/mux (mounts `Handler()`), its
own NATS URL (loopback there), its own sentinel, and its own bearer story
(static tokens on the loopback lane — the passthrough is lane-agnostic,
spec Q4). It gets: the same pool, the same admission trust model, the same
public tool surface. Nothing in `node` assumes remote-ness beyond config.
