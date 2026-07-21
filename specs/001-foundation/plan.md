# Implementation Plan: Foundation — Realm Provisioning & the Operation Record

**Branch**: `001-foundation` | **Date**: 2026-07-12 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/001-foundation/spec.md`

## Summary

Deliver the Soulstream **wire layer** as a Go module: connect to NATS from a named context,
provision a realm's one stream and one object store idempotently (**create-or-report, never
mutate in place**), and provide the operation record with a lossless wire ⇆ struct mapping and a
deterministic RFC 8785 (JCS) canonical form, plus persona-name validation and write-side
attribution enforcement. No topic vocabulary, lifecycle, baselines, materialisation, mentions,
discovery, client, adapter, or signing — those are later features. The whole record surface is
unit-testable with no server; provisioning is tested against an in-process JetStream server.

## Technical Context

**Language/Version**: Go 1.26 (toolchain present: go1.26.2; embedded nats-server 2.14 needs ≥1.24 — satisfied)
**Module path**: `github.com/impire-io/soulstream` (assumption; trivially changeable, no external resolution needed for local builds/tests)
**Primary Dependencies** (all verified against pkg.go.dev / source, Jan 2026):
- `github.com/nats-io/nats.go` + `.../jetstream` — connection and the modern JetStream management API (`jetstream.New`, `CreateStream`, `Stream`, `CreateObjectStore`, `ObjectStore`; sentinels `ErrStreamNotFound`, `ErrStreamNameAlreadyInUse`, `ErrBucketNotFound`, `ErrBucketExists`).
- `github.com/synadia-io/orbit.go/natscontext` — `Connect(name, ...nats.Option) (*nats.Conn, Settings, error)` to connect from a named context.
- `github.com/google/uuid` (v1.6.0) — `uuid.NewString()` / `uuid.NewRandom()` for UUIDv4 op-ids.
- `github.com/gowebpki/jcs` (v1.0.1) — `Transform([]byte) ([]byte, error)` for RFC 8785 canonicalisation (chosen over `cyberphone/json-canonicalization` for clean module path + semver).
- `github.com/nats-io/nats-server/v2` (v2.14.x) — **test-only** in-process JetStream server.

**Storage**: JetStream only — the `SOULSTREAM` file-backed stream and the `soulstream-objects` object store. No external database (Constitution I).
**Testing**: `go test`; table-driven unit tests for record/canonical/identity (no server); integration tests for provisioning against an in-process `nats-server` with a `t.TempDir()` store dir.
**Target Platform**: any Go-supported OS; verified on darwin/arm64.
**Project Type**: single Go library (three cohesive public packages), consumed by later CLI/MCP features.
**Performance Goals**: not performance-critical at this layer; record build/parse/canonicalise are in-memory microsecond operations. Target: "instant" to the caller, no explicit budget.
**Constraints**: NATS server 2.12+ (limits retention w/o age expiry, subject rollup, duplicate window, disk storage). Messages stay text/references-only; headers small.
**Scale/Scope**: one realm per account; the record surface is pure functions (arbitrary op volume); provisioning touches exactly two artefacts.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. NATS-Native First** — ✅ PASS. Every capability is a built-in JetStream primitive: the stream
  (op-log), the object store (blobs), `Nats-Msg-Id` dedup (idempotent publish), subject rollup
  headers (provisioned, exercised later). No database, coordinator, or API tier. Minimum server
  version stated (2.12+). `Nats-Expected-Last-Subject-Sequence` is noted for later (rollup) and
  deliberately unused here — no custom concurrency code.
- **II. Smallest Viable Implementation** — ✅ PASS. The realm shape is fixed and non-configurable (no
  speculative tuning knobs); provisioning is create-or-report (no reconfiguration engine); signing
  is a carried-through slot, not an implementation. Non-goals (FR-027) keep topics/lifecycle out.
  The one config surface (context name, realm, optional persona) is the minimum needed to connect
  and to bind the canonical record.
- **III. ELI5 Documentation** — ✅ PASS (planned). Ships `docs/` pages in plain words with everyday
  analogies for each new concept: *the realm* (a private workshop), *the operation record* (a
  delivery slip whose details live on the label, not inside the box), *the canonical form* (the
  slip re-typed on a standard form so two copies always match byte-for-byte), and *provisioning*
  (setting up the empty workshop, and checking — never silently rearranging — one that already
  exists). Docs ship in the same change as the code (a per-story task, not a polish phase).

**Result**: PASS — no violations; Complexity Tracking not required.

### Post-design re-check

Re-evaluated after Phase 1 (data-model + contracts): still PASS. The design adds no dependency
beyond the NATS client, the context loader, a UUID generator, and a canonicaliser — each maps
directly to a functional requirement. `record` and `identity` import nothing from NATS, keeping
the pure surface server-free (SC-007). No new abstraction was introduced "for later".

## Project Structure

### Documentation (this feature)

```text
specs/001-foundation/
├── plan.md              # This file
├── research.md          # Phase 0 output — dependency API decisions (verified)
├── data-model.md        # Phase 1 output — entities & rules
├── quickstart.md        # Phase 1 output — provision + build-a-record walkthrough
├── contracts/           # Phase 1 output — public Go API surface
│   ├── record.md
│   ├── realm.md
│   └── identity.md
└── tasks.md             # Phase 2 output (/speckit-tasks — not created here)
```

### Source Code (repository root)

```text
go.mod
go.sum
Makefile                        # fmt / test / lint targets (constitution quality gates)
.golangci.yml                   # linter config

record/                         # the operation record — imports NO nats
├── record.go                   # Record type; wire build/parse (headers ⇆ struct); parents rule
├── canonical.go                # CanonicalRecord: JCS canonicalisation; wire⇆canonical mapping
├── id.go                       # UUIDv4 op-id generation
├── errors.go                   # field-specific parse/validation errors (sentinels)
├── record_test.go              # round-trip matrix (no/one/many parents; sig/no-sig; unknown hdrs)
├── canonical_test.go           # determinism + losslessness (SC-004)
└── id_test.go

identity/                       # persona names & attribution — imports NO nats
├── name.go                     # slug grammar (personas, realms, topic ids)
├── author.go                   # write-side enforcement + read-side check (+ resolver hook)
└── identity_test.go

realm/                          # connection + provisioning — the only nats-touching package
├── connect.go                  # Connect from a named NATS context (natscontext) → Client
├── spec.go                     # RealmSpec constants (mandated stream/object-store settings)
├── provision.go                # create-or-report: Stream() lookup, CreateStream if missing
├── conformance.go              # compare existing stream/store config to RealmSpec → drift list
├── report.go                   # ProvisionReport type
├── connect_test.go             # missing context / unreachable / no-jetstream errors
└── provision_test.go           # fresh, idempotent re-run, partial (store missing), drift

internal/natstest/              # test-only in-process JetStream server helper
└── server.go                   # StartJetStream(t) (url, cleanup) — JetStream + t.TempDir()

docs/                           # ELI5 documentation (Constitution III)
├── realm.md
├── operation-record.md
├── canonical-record.md
└── provisioning.md
```

**Structure Decision**: A single Go library split into three cohesive public packages mirroring the
three spec areas — `record` (pure), `identity` (pure), `realm` (touches the server). Keeping
`record`/`identity` free of any NATS import is deliberate: it makes the record surface unit-testable
with zero infrastructure (SC-007) and localises the NATS dependency to `realm`. **Connection is
decoupled from provisioning**: `Connect` uses `natscontext`, while `Provision` accepts a JetStream
handle, so provisioning tests run against an in-process server via a direct connection without
needing a configured context. A test-only `internal/natstest` helper runs the embedded JetStream
server (`server.NewServer` with `JetStream:true`, `StoreDir:t.TempDir()`, `Port:-1`).

### Key implementation note (create-or-report)

`CreateOrUpdateStream` is deliberately **not** used: it updates an existing stream in place, which
FR-008 forbids. Provisioning instead: `js.Stream(ctx, "SOULSTREAM")` → on `ErrStreamNotFound` call
`CreateStream` (report `created`); otherwise read `CachedInfo().Config`, diff against `RealmSpec`,
and report `conformant` or `nonconformant` (with the specific drift) — never `UpdateStream`. Object
store is symmetric: `js.ObjectStore` lookup → `CreateObjectStore` on `ErrBucketNotFound`.

## Complexity Tracking

> No Constitution violations. This section intentionally left empty.
