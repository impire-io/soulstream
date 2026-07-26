# Implementation Plan: Provisioning Byte Limits

**Branch**: `016-provision-limits` | **Date**: 2026-07-27 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/016-provision-limits/spec.md`

## Summary

Provisioning gains an optional per-artefact storage budget (`realm.Budgets`,
bytes; zero field = unlimited as today) plus a defaults constructor
(`realm.DefaultBudgets()`) carrying the shapes proven by the manual NGS R1
workaround (1 GiB op-log, 64 MiB notify, 64 MiB personas, 512 MiB objects).
`ProvisionOn`/`Client.Provision` accept the budgets as an optional trailing
value (variadic, at most one — existing callers compile unchanged); budgets
apply only at artefact creation; existing artefacts stay untouched and their
byte roofs are now reported. The CLI `provision` command grows `--budgets`
(the defaults switch) and four `--budget-<artefact>` size flags with
human-readable units, and prints each artefact's roof. A new embedded-server
test fixture with `MaxBytesRequired` reproduces the limit-enforcing tier
locally, making both halves of US1 measurable without NGS.

## Technical Context

**Language/Version**: Go 1.26
**Primary Dependencies**: nats.go v1.52 + `nats.go/jetstream` (existing; no new deps)
**Storage**: JetStream stream/KV/object-store configs (`MaxBytes` fields already in the API)
**Testing**: `go test` with embedded `nats-server/v2` via `internal/natstest`; new opts variant to set account `MaxBytesRequired`
**Target Platform**: wherever the library/CLI run today (release matrix unchanged)
**Project Type**: library + CLI (existing packages `realm`, `internal/cli`)
**Performance Goals**: none — provisioning is a one-shot operator action
**Constraints**: create-or-report semantics inviolate; zero-budget call paths byte-identical to v0.4.0
**Scale/Scope**: 4 artefacts, 1 struct, ~2 config constructors touched, ~5 CLI flags, docs page update

## Constitution Check

- **I. NATS-Native First**: PASS. The entire capability is the `MaxBytes`
  field JetStream already defines on stream/KV/object-store configs; the
  limit-enforcing behavior under test is the server's own
  `MaxBytesRequired` account limit. No new infrastructure, no custom
  mechanism, no new server-version requirement (fields predate our floor).
- **II. Smallest Viable Implementation**: PASS. One plain struct, no
  options machinery (the variadic-struct pattern exists to keep the public
  signature source-compatible, not as an extension point); no
  message-count/age knobs (spec rules them out); budgets are not added to
  the identity config file. The defaults are constants, not configuration.
- **III. ELI5 Documentation**: PASS with obligation: `docs/provisioning.md`
  (exists) gains a "storage budgets" section — shelf-space analogy, the
  defaults table, and "provisioning never resizes an existing shelf" — in
  the same change as the behavior.

## Project Structure

### Documentation (this feature)

```text
specs/016-provision-limits/
├── plan.md              # This file
├── research.md          # Phase 0: decisions D1–D6
├── data-model.md        # Phase 1: Budgets, extended ArtefactResult
├── quickstart.md        # Phase 1: NGS R1 walkthrough
├── contracts/
│   └── library.md       # Phase 1: public API deltas (realm, CLI)
└── tasks.md             # Phase 2 (/speckit-tasks — not this command)
```

### Source Code (repository root)

```text
realm/
├── spec.go              # Budgets, DefaultBudgets, budget-aware config constructors
├── provision.go         # ProvisionOn variadic budgets; per-artefact roofs into results
├── report.go            # ArtefactResult.MaxBytes (as-found or as-created)
└── connect.go           # Client.Provision passes budgets through

internal/cli/
├── provision.go         # --budgets switch, --budget-* size flags, roof column in output
└── (size parse/format helpers colocated with the provision command)

internal/natstest/
└── natstest.go          # StartJetStreamWithLimits(t) — MaxBytesRequired account

docs/
└── provisioning.md      # ELI5: storage budgets section
```

**Structure Decision**: no new packages. `realm` keeps owning the mandated
shape; the CLI keeps owning presentation; `natstest` keeps owning embedded
servers. Pure logic (budget validation, roof formatting) stays server-free
and unit-tests without NATS, per the repo's purity convention.

## Complexity Tracking

No constitution violations to justify.
