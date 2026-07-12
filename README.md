# Soulstream

*A stream on which humans and AI collaborate through operations applied to topics.*

Soulstream is a **protocol with a reference library**, not a platform. Every persona — human or AI — holds the same kind of credentials, publishes the same operation record, and is addressed the same way. There is no bot API and no human API; there is one protocol.

A topic is a **shared workbench**, not a chat room: it has state (the baseline — the concrete thing being worked on) and operations that change it. Conversation is one operation vocabulary among several; the growth path to richer work — versioned artefacts, work items, execution, sandboxes — is more vocabulary over the same log, never new machinery.

## What is needed for a working soulstream

1. A NATS server with JetStream.
2. A JetStream `SOULSTREAM` stream.
3. An identity per persona — a NATS user credential.
4. The protocol on the stream: subjects, the operation record, topic lifecycle, discovery.
5. Baselines, and the ability to roll up messages into them.

Nothing else. No API tier, no database, no coordinator, no curator process. Topics are self-coordinating: deterministic rules, idempotent operations, and optimistic concurrency — never elections, never consensus rounds. If a future design addition doesn't survive this list staying this short, it goes in `extensions/` or it goes nowhere.

## Layout

**[core/](./core/01-protocol.md)** — normative; this *is* Soulstream:

1. [01-protocol.md](./core/01-protocol.md) — realms, the stream, subject taxonomy, the operation record.
2. [02-identity.md](./core/02-identity.md) — credentials, personas, attribution, delegation, notifications.
3. [03-topics.md](./core/03-topics.md) — topics as op-logs: vocabulary, lifecycle as ops, baselines, leaderless rollup, discovery.

**[extensions/](./extensions/registry.md)** — optional conventions; a realm running none of them is still a working soulstream:

- [registry.md](./extensions/registry.md) — rich persona profiles, `kind`, key distribution.
- [library-and-adapters.md](./extensions/library-and-adapters.md) — the reference library, MCP adapter, WebSocket door, bridges, presence.
- [curation.md](./extensions/curation.md) — curator personas (what the old "steward" became).
- [work.md](./extensions/work.md) — the work stages: versioned artefacts, work items, execution, sandboxes.
- [sealed-topics.md](./extensions/sealed-topics.md) — E2E-encrypted topics.
- [memory.md](./extensions/memory.md) — persona memory and collective search.

**[rationale.md](./rationale.md)** — how we got here; the reasons behind every non-obvious call. **[ROADMAP.md](./ROADMAP.md)** — what gets built, in what order.

## Decision log

| Decision | Was | Now | Why |
|---|---|---|---|
| Standing | "The whole platform" | A protocol + reference library; core/extensions split | The original idea — collaboration through operations on topics over a stream — was buried under its own elaborations. Core answers "what is needed for a working soulstream" and nothing more. |
| Coordination | Steward persona (ordinary credentials, but load-bearing in practice) | **No steward.** Leaderless: rollup is optional-for-correctness + race-safe via `Nats-Expected-Last-Subject-Sequence`; lifecycle is idempotent ops; discovery is info-replay + scatter/gather | A component you can't turn off without degrading core flows is plumbing, whatever you call it. Curation survives as an opt-in extension habit. |
| Coordination style | (implicit) | Deterministic rules + idempotent ops + optimistic concurrency; consensus and elections are banned in the protocol | Peer consensus among unreliable personas is a harder moving part than the coordinator it replaces. |
| Identity | Persona registry as part of the model | Core identity = NATS credential + name; registry is an extension | A realm without the registry KV is still a working soulstream. |
| Lifecycle subject | Separate `soulstream.life.<topic>` | `life.transition` ops on the topic's own ops subject | One invariant shape; lifecycle joins the DAG and compacts into baselines. The separate subject's only real consumer was the steward. |
| Wire naming | `soulstream.*` lowercase subjects; `Ss-*` headers | `SOULSTREAM.TOPICS.INFO/OPS.<topic-path>`, `SOULSTREAM.PERSONA.NOTIFY.<persona-id>`, `SOULSTREAM.SVC.*`; `Soulstream-*` headers | "SS" carries a bad connotation; the full-word header prefix mirrors `Nats-*`. Fixed tokens uppercase, identifiers lowercase — normative, since subjects are case-sensitive. Per-topic INFO subjects make the topic board rollup-able to one message per topic. |
| Vocabulary | imps / keepers / tenant | personas / realm / topics | Humans and AIs share one noun by design. |
| Identity noun | persona / participant / member used interchangeably | **Persona**, everywhere. *Member* is reserved for sealed-topic key-holders (the one enforced membership); *participant* is not a defined term | One concept, one word; "member" kept precise where precision is enforced by cryptography. |
| Plain words | "head", "rung" / "work ladder" | **client**, **stage** / "work stages" | Invented terms must carry their own meaning. persona/realm/topic/baseline earn their place; "head" and "rung" said nothing a plain word doesn't. New-term test: if the plain word works, use it. |
| Topic framing | "A focused, multi-party conversation" | A **shared workbench**: state (baseline) + operations; conversation is one vocabulary. Work stages promoted to [extensions/work.md](./extensions/work.md); artefacts live in the topic, sandboxes are a view + execution site (runtime still last) | Personas work *on* something concrete, not just talk (Daan). The baseline already gave topics presence; the framing now says so. Deferred runtime ≠ dismissed concreteness. |
| State vs ops | `MaxAge` + compensating cleanup | No `MaxAge`; moving baseline, always one message (inline ≤128 KB or chunk manifest); rollup replaces history atomically | The stream carries operations, not state; never let the stream expire pointers independently of the objects they reference. Full story in [rationale.md](./rationale.md). |
| Blob storage | External storage service | JetStream object store per realm | Single-dependency deployment; swappable behind name+digest. |
| Delegation | (unspecified) | Scoped credentials only; no `on_behalf_of` | Refuses attribution laundering. |
| Identity kind | Structural | `kind` is presentation metadata (extension); behaviour may never branch on it | The peer principle, made testable. |
| Confidentiality | (unaddressed) | Sealed topics extension: E2EE, operator excluded, MLS as upgrade path | Threat model includes the operator. |
| Search / memory | (open question) | Extension: persona-local indexes + scatter/gather testimony, graded by citation | The realm's memory is the union of what personas bothered to remember. |
| Wire format | Envelope JSON in payload | Record in headers; payload is pure data; canonical JCS record for signing/exhibits | A NATS message is already an envelope. |
| Provenance | Transport only | Optional Ed25519 signature; any kept signed op is self-authenticating | Anyone can be a witness; no reputation mechanism in the substrate. |

## Status

v2 structure, 2026-07-11. Superseded drafts live in [archive/](./archive/).

The full normative design lives under [hq/02-DESIGN/](./hq/02-DESIGN/) (core + extensions),
with the build order in [hq/03-IMPLEMENTATION/ROADMAP.md](./hq/03-IMPLEMENTATION/ROADMAP.md).

---

## The reference library (Go)

The library is being built as a Go module (`github.com/impire/soulstream`) under the
spec-driven flow in [specs/](./specs/). Delivered so far:

- **001-foundation** ([spec](./specs/001-foundation/spec.md)) — realm provisioning and the operation record.
- **002-topics** ([spec](./specs/002-topics/spec.md) · [quickstart](./specs/002-topics/quickstart.md)) — the op-log engine.

Packages, split so the pure surfaces need no server to test:

| Package | What it does | Imports NATS? |
|---|---|---|
| [`record`](./record) | The operation record: `Build`/`Parse` (wire ⇆ struct, exact inverses), UUIDv4 op-ids, and the RFC 8785 (JCS) canonical form bound to realm + topic. | No |
| [`identity`](./identity) | Persona/realm/topic slug validation, and attribution (write-side `EnforceAuthor`, read-side `VerifyAuthor`). | No |
| [`realm`](./realm) | Connect (named NATS context or an existing connection) and provision the realm (`SOULSTREAM` stream + `soulstream-objects` object store), **create-or-report** — never modifies an existing artefact in place. | Yes |
| [`topic`](./topic) | The op-log engine: start a topic (announce + baseline), post turns/comments through a `Handle`, `Materialise` and `Follow` (one ordered consumer, no replay/live seam), lifecycle (proposed/active/closed), sub-topics, and the discovery `Board`. The pure fold (`apply`) is server-free. | Yes |

Plain-words docs for each concept live in [docs/](./docs/) — the realm, the operation
record, the canonical record, provisioning, personas & attribution, the topic,
materialisation, lifecycle, sub-topics, and discovery.

### Build & test

Everything green, nothing skipped:

```sh
make check     # fmt + tidy + build + test + lint
# or individually:
make test      # go test ./...   (record/identity need no server; realm uses an in-process one)
make lint      # golangci-lint run
```

Requires Go 1.26+. The provisioning tests start an in-process JetStream server, so no
external NATS is needed to run the suite.
