# Cross-Artifact Analysis: 013-config

**Date**: 2026-07-21 · **Artifacts**: spec.md, plan.md, tasks.md (+ research,
data-model, contracts, quickstart) · **Result**: no critical findings; ready to
implement.

## Requirements coverage

| FR | Covered by tasks | Note |
|---|---|---|
| FR-001 chain, per-field | T001–T002 (engine), T004–T005 (both entry points) | |
| FR-002 `.soulstream.json`, nearest-only walk-up | T001, T003 | |
| FR-003 user `config.json` beside keys/pins | T001 | R4 fixes location |
| FR-004 fail-loud / skip-absent | T001, T003, T006, T010 | strict decode per R1 |
| FR-005 config-dir-relative paths | T001, T003 | resolved at load (R5) |
| FR-006 `soulstream config`, offline | T009–T011 | contract fixes output shape |
| FR-007 no credentials in config | structural (File has no credential fields) + docs T007 | nothing executable — verified by absence |
| FR-008 unchanged flag/env behaviour | T003 (unset baseline), T006 (precedence through real Run) | SC-005 |
| FR-009 wrapper resolve→download→verify→cache | T012 | contract §wrapper |
| FR-010 no partial cache, named failures | T012, T014 | temp dir + atomic mv (R8) |
| FR-011 plugin 0.2.0 + matching release | T013, T018 | release pairing per R9 |
| FR-012 docs everywhere | T007, T008, T011, T015, T016 | Principle III inside stories |

All 12 FRs mapped; all 5 SCs have a proving task (SC-001→T006, SC-002→T010,
SC-003/004→T014+T018, SC-005→T003/T006).

## Consistency checks

- **Terminology**: "five fields", source kinds, `$DATA` layout, and archive naming
  are identical across spec, data-model, contract, and tasks. No drift found.
- **Constitution**: no NATS involvement (gate I trivially holds); no new knobs beyond
  spec (gate II — the one temptation, a version-override env for testing the
  download, was rejected in favour of test-setup-only scratch copies, T014); docs are
  in-story tasks (gate III).
- **Duplication**: none — resolution logic exists once (`internal/config`), both
  entry points consume it.

## Watch items (minor, non-blocking)

1. **T006 cwd injection**: `Run` reads the process working directory; tests use
   `t.Chdir` (Go ≥1.24 — fine on 1.26). If flake-prone under parallel tests, thread
   cwd through `Run` explicitly instead — decide during T004.
2. **T014 download test** uses the v0.1.0 release (only one published pre-merge);
   the real 0.2.0 path is re-proven in T018 — acceptable ordering, explicitly listed.
3. **Env-var empty-string semantics**: an env var set-but-empty counts as unset
   (chain skips it) — matches today's `os.Getenv` behaviour; T003 pins it.
