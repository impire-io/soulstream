# Implementation Plan: Op Signing & Key Distribution

**Branch**: `006-signing` | **Date**: 2026-07-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-signing/spec.md`

## Summary

Give personas Ed25519 signing identities: `Soulstream-Sig` becomes real (Ed25519 over the
already-implemented JCS canonical record, `record.Record.Canonical`), a minimal persona
directory (`soulstream-personas` KV bucket) distributes public keys with TOFU chain
pinning and old-key-signs-new-key rotation, and every read path annotates ops with a
verification status (unsigned / verified / failed / unknown-key) that flags but never
drops. Technical approach: signing primitives and the keyring go in the NATS-free
`identity` package; a new NATS-touching `registry` package owns profile publish/lookup
and pure chain validation; `topic`'s single publish choke point (`publishOp` in
`topic/wire.go`) signs when the client carries a signer; a pure annotate pass computes
statuses for materialise/follow/inbox; clients share key/pin file handling via a new
`internal/keystore`. No new module dependencies: `crypto/ed25519` is stdlib.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire/soulstream`)
**Primary Dependencies**: existing only — `nats.go` v1.52 + `nats.go/jetstream` (KV),
`gowebpki/jcs`, `google/uuid`, `synadia-io/orbit.go/natscontext`; new crypto is stdlib
`crypto/ed25519` + `crypto/rand` + `encoding/base64`
**Storage**: JetStream KV bucket `soulstream-personas` (history ≥ 10) provisioned
create-or-report beside the stream and objects bucket; client-side files — Ed25519 seed
(0600) and a pins JSON per realm under the user config dir
**Testing**: `go test ./...`; pure sign/verify/chain/annotate tests are serverless;
KV + end-to-end tests use the embedded server via `internal/natstest.StartJetStream(t)`
**Target Platform**: anywhere Go runs (CLI + MCP stdio adapter; darwin/linux)
**Project Type**: library + two thin clients (existing layout)
**Performance Goals**: Ed25519 verify ≈ tens of µs/op — immaterial for MVP-scale topics;
one KV read per persona per materialise, no watchers
**Constraints**: wire format unchanged except populating the existing optional
`Soulstream-Sig` header; unsigned publishing byte-identical to pre-feature; read paths
never fail because of signature state
**Scale/Scope**: one realm, handfuls of personas and topics (dogfood scale)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. NATS-Native First** — PASS. Key distribution is a plain JetStream KV bucket
  (native history, native optimistic concurrency: `Create` for first publish, `Update`
  with revision for rotation — no lock service, no coordinator). No new infrastructure;
  the only non-NATS state is the persona's own secret key and pin file, which by
  definition cannot live on the substrate (the secret must never travel; the pin must
  survive substrate compromise). No new server capability is relied on beyond KV —
  minimum server version unchanged from MVP (2.12+ target).
- **II. Smallest Viable Implementation** — PASS. No new op types, no new headers
  (`Soulstream-Sig` exists in the wire spec since 001), no watch API, no service
  announcements, no sealing keys, no key servers; the registry slice is exactly
  profile-with-signing-key. Rotation proof is a fixed domain-separated byte string, not
  a new envelope format. Verification is an annotate pass, not a pipeline stage.
- **III. ELI5 Documentation** — PASS. Two new pages (`docs/signing.md`,
  `docs/persona-directory.md` — wax-seal and phone-book analogies) plus updates to
  `canonical-record.md`, `provisioning.md`, `cli.md`, `mcp.md`, `README.md` index, all
  shipping inside the story tasks.

## Project Structure

### Documentation (this feature)

```text
specs/006-signing/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── library.md       # new/changed Go API surface
│   ├── wire-and-kv.md   # sig encoding, profile JSON, rotation proof, pin file
│   └── clients.md       # CLI command + MCP tool contract
└── tasks.md             # Phase 2 output (/speckit-tasks — not created by plan)
```

### Source Code (repository root)

```text
identity/                # NATS-free (unchanged rule)
├── sign.go              # NEW SigningKey: generate, seed encode/decode, Sign, Verify
├── keyring.go           # NEW Keyring: persona → validated key chain (+ distrusted set)
└── (existing files)

registry/                # NEW package — NATS-touching (KV); pure parts separated
├── profile.go           # Profile struct, JSON shape, validation (pure)
├── chain.go             # rotation-proof chain validation, pin extension check (pure)
├── kv.go                # Publish (create/rotate), Lookup, All over the KV bucket
└── doc.go

realm/
├── provision.go         # + third artefact: soulstream-personas KV (create-or-report)
├── spec.go              # + bucket name/shape constants
└── connect.go           # Config gains optional Signer; Client.Signer() accessor

topic/
├── wire.go              # publishOp signs canonical bytes when client has a signer
├── verify.go            # NEW pure annotate: recs + realm + path + Keyring → statuses
├── view.go              # Contribution/Attachment/Announcement/Notification gain Sig field
├── materialise.go       # thread statuses through the fold output
├── follow.go            # annotate live records
└── notify.go            # inbox records annotated; notify publishes already signed via publishOp

internal/keystore/       # NEW shared by both clients
├── keystore.go          # seed file load/save (0600), pins JSON load/save, default paths
└── keystore_test.go

internal/cli/
├── commands.go          # + key init|show|rotate, profile publish|show
├── render.go            # status glyphs (✓ ✗ ?) + substitution-attack banner
└── connect.go           # Config gains KeyFile/PinsFile resolution, loads signer + keyring

internal/mcpserver/
├── server.go            # session signer + keyring wiring
└── tools.go             # + publish_profile tool; sig status in read-tool results

docs/
├── signing.md           # NEW (ELI5, wax-seal analogy)
├── persona-directory.md # NEW (ELI5, phone-book + handwriting-pinning analogy)
└── (updates: canonical-record.md, provisioning.md, cli.md, mcp.md, README.md)
```

**Structure Decision**: extend the existing library-plus-thin-clients layout. `identity`
takes the pure crypto (it already owns attribution; keys are identity material and it
imports no NATS). `registry` joins `realm`/`topic` as the third NATS-touching package
rather than growing `realm` beyond connect+provision; its chain logic is pure and
serverless-testable, mirroring the `topic` fold pattern. `internal/keystore` exists so
CLI and MCP share one implementation of "where keys and pins live on disk".

## Complexity Tracking

No constitution violations to justify. The one judgment call: a third top-level package
(`registry`) instead of folding profiles into `realm`. Chosen because `realm` is
documented as "connect + provision the realm's artefacts" and profile CRUD/chain
validation is a different concern; the split keeps both packages single-purpose and the
pure chain code separately testable.
