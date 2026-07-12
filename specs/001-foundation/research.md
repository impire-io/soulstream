# Research: Foundation — Dependency & API Decisions

**Feature**: 001-foundation | **Date**: 2026-07-12
**Method**: Verified against pkg.go.dev and library source (Jan 2026); no guesses carried into the plan.

Each decision below resolves a Technical Context unknown from the plan.

## 1. Connecting from a named NATS context

- **Decision**: Use `github.com/synadia-io/orbit.go/natscontext` and its
  `Connect(name string, opts ...nats.Option) (*nats.Conn, Settings, error)`.
- **Rationale**: Mandated project convention (global instruction: use orbit `natscontext` for
  context-based connections). It reads the standard `nats context` files
  (`~/.config/nats/context/<name>.json`), so credentials/URLs are configuration, not code (FR-001).
  The returned `Settings` exposes `JSDomain` for `jetstream.NewWithDomain` if a realm ever needs a
  JetStream domain.
- **Gotcha captured**: `Connect` returns **three** values — a missing/invalid context yields an
  error before any server contact (satisfies FR-002 fail-fast, feeds US1 scenario 3).
- **Alternatives considered**: raw `nats.Connect(url, ...)` — rejected (hard-codes server/creds,
  violates FR-001). Legacy separate `Load`+connect — does not exist in this package; `Connect` is
  the single entry point.

## 2. JetStream management (stream + object store)

- **Decision**: Use the modern `github.com/nats-io/nats.go/jetstream` API. Provision with a
  **lookup-then-create** flow, never `CreateOrUpdateStream`.
  - Get context: `jetstream.New(nc)`.
  - Stream: `js.Stream(ctx, "SOULSTREAM")` → on `errors.Is(err, jetstream.ErrStreamNotFound)` call
    `js.CreateStream(ctx, cfg)`; otherwise read `stream.CachedInfo().Config` and diff.
  - `StreamConfig{Name, Subjects:["SOULSTREAM.>"], Retention:LimitsPolicy, MaxAge:0,
    AllowRollup:true, Duplicates:2*time.Minute, Storage:FileStorage}`.
  - Object store: `js.ObjectStore(ctx, "soulstream-objects")` → on
    `errors.Is(err, jetstream.ErrBucketNotFound)` call
    `js.CreateObjectStore(ctx, jetstream.ObjectStoreConfig{Bucket:"soulstream-objects"})`.
- **Rationale**: FR-008 forbids modifying an existing artefact in place. `CreateOrUpdateStream`
  **updates** an existing stream (and some fields are immutable server-side, which would surface as
  confusing errors), so it is the wrong primitive for create-or-report. Lookup-then-create gives
  exact control: create only when absent, otherwise compare-and-report. Sentinel errors are checked
  with `errors.Is` (never string matching).
- **Field mapping verified**: `Retention→retention`, `MaxAge→max_age` (0 = no limit),
  `AllowRollup→allow_rollup_hdrs`, `Duplicates→duplicate_window`, `Storage→storage`. These are the
  exact settings the spec mandates (FR-003/004/005).
- **Alternatives considered**: legacy `nc.JetStream()` API — rejected (older, being superseded;
  new code should use the `jetstream` package). `CreateOrUpdateStream` — rejected per FR-008 above.

## 3. Operation-identity generation (UUIDv4)

- **Decision**: `github.com/google/uuid` (v1.6.0); generate with `uuid.NewString()` (or
  `uuid.NewRandom()` when a caller wants to handle a `crypto/rand` failure without a panic).
- **Rationale**: Produces exactly the lowercase hyphenated `8-4-4-4-12` form the clarification
  fixed (FR-013), from `crypto/rand`, unique without coordination. `NewString()` is a single call;
  it panics only on entropy-source failure (vanishingly rare) — acceptable, with `NewRandom` as the
  graceful variant if we choose to surface the error.
- **Alternatives considered**: hand-rolled short IDs / the reference design's abbreviated
  examples — rejected: reproducing an arbitrary truncation adds risk for no benefit; a standard
  UUIDv4 is unambiguous, token-safe, and trivially unique within the dedup window.

## 4. Canonicalisation (RFC 8785 / JCS)

- **Decision**: `github.com/gowebpki/jcs` (v1.0.1); `jcs.Transform([]byte) ([]byte, error)`. Flow:
  build the canonical record struct → `json.Marshal` → `jcs.Transform`.
- **Rationale**: Clean module path and proper semver (vs. `cyberphone/json-canonicalization`'s deep
  nested import and pseudo-version), same underlying algorithm. JCS sorts object keys
  lexicographically (UTF-16 code units) recursively and preserves array order — so field-order
  independence (FR-020, SC-004) is guaranteed regardless of Go struct field order, and `parents`
  array order is preserved. Numbers normalise via the ES6 algorithm.
- **Gotchas captured**:
  - Input to `Transform` must be valid UTF-8 JSON (an object/array at the root). `json.Marshal` of
    the canonical struct always satisfies this.
  - Integers beyond 2^53 lose precision (all JSON numbers are treated as float64). Not a concern
    here — `v` is `1` and everything else is strings/arrays; the payload's own numeric fidelity is
    the payload author's concern, documented as a size/discipline note.
- **Alternatives considered**: emitting sorted keys ourselves via `json.Marshal` — rejected: does
  not implement full JCS number/string normalisation, so signatures produced later could mismatch.
  `cyberphone/json-canonicalization` — rejected on packaging only (behaviour identical).

## 5. In-process JetStream server for tests

- **Decision**: Use `github.com/nats-io/nats-server/v2/server` directly in a `internal/natstest`
  helper: `server.NewServer(&server.Options{JetStream:true, StoreDir:t.TempDir(), Host:"127.0.0.1",
  Port:-1, NoLog:true, NoSigs:true})` → `go ns.Start()` → `ns.ReadyForConnections(10*time.Second)`
  → return `ns.ClientURL()` and `ns.Shutdown` as cleanup.
- **Rationale**: Canonical embedded-server approach with full control over JetStream + a temp store
  dir; provisioning is verifiable with no external server (SC-007). `t.TempDir()` auto-cleans.
- **Gotchas captured**: `NewServer` returns `(*Server, error)` — the error must be checked.
  `Port:-1` picks a free port; pair with `ClientURL()`. `NoSigs`/`NoLog` keep tests quiet and
  signal-handler-free. nats-server 2.14 needs Go ≥1.24 (we have 1.26).
- **Alternatives considered**: `.../v2/test` helpers (`RunServer`) — rejected: they do **not**
  enable JetStream for you (you still set `JetStream`+`StoreDir`), so they add a dependency for
  almost no gain. `DontListen:true` + `nats.InProcessServer(ns)` — noted as a faster no-TCP option,
  but the URL-based path is simpler and lets connection code stay identical to production.

## Version alignment

Keep `nats-server/v2` (v2.14.x) and `nats.go` (v1.x current) roughly co-current to avoid JetStream
protocol skew. Go toolchain 1.26 satisfies both. Exact pinned versions land in `go.mod` during
implementation (`go get` resolves latest compatible).
