# Cross-Artifact Analysis: 007-rollup

**Date**: 2026-07-21 | **Scope**: spec ↔ plan ↔ tasks ↔ constitution

## Findings

No critical/high findings. Constitution: rollup and its race guard are the NATS
primitives the constitution explicitly prefers (`Nats-Rollup`,
`Nats-Expected-Last-Subject-Sequence`); no coordinator anywhere (FR-013); additive
payload fields, no new op types; ELI5 docs are in-story tasks (T009, T013, T018).

| ID | Severity | Note | Resolution |
|----|----------|------|------------|
| B1 | MEDIUM | Two readers meet baselines in places the 006 code never did: the **board** derives lifecycle by folding each topic's log (a manifest baseline hides lifecycle inside the object), and a **live follower** that retained pre-rollup history receives the landed rollup as a mid-log `baseline` op (would hit the "unknown type" warning). | Both encoded in tasks: T011 explicitly covers the board's manifest resolution; T002 amended with the mid-log-baseline checkpoint rule (skip with a specific benign warning) and T006 asserts follower consistency across a mid-follow rollup. |
| L1 | LOW | FR-004 ("never implicit except close/archive") is a negative requirement — nothing to build, only to not build. | Verified by absence at review time plus the T020 sweep; the only rollup call sites are `Handle.Rollup`, `Close`, `Archive`, and the two client commands/tools. |
| L2 | LOW | `Transition(ctx, Archived)` (the primitive) archives without compacting — a valid but unusual terminal-uncompacted topic. | Deliberate: the primitive stays orthogonal; `Archive()` is the mandated act, and the contract documents the bare-transition caveat. The fold's terminal rule makes even this path safe (writes refuse once observed). |
| L3 | LOW | The view structs' JSON casing change (T001) alters `show --json` and MCP result keys from 006's accidental `"Sig"`-style casing. | Intentional and pinned in contracts (baked state makes the shape wire); pre-1.0 single-consumer; all affected tests updated in T001 itself so the suite never passes through a red state. |

## Coverage

| Requirement | Tasks | | Requirement | Tasks |
|---|---|---|---|---|
| FR-001 | T005, T006 | | FR-008 | T003, T006 |
| FR-002 | T002, T006 | | FR-009 | T014, T015 |
| FR-003 | T005, T006 | | FR-010 | T014, T015, T016, T017 |
| FR-004 | T014, T020 (absence) | | FR-011 | T011, T012 |
| FR-005 | T001, T005, T010 | | FR-012 | T007, T008, T016, T017 |
| FR-006 | T010, T012 | | FR-013 | T020 (sweep) |
| FR-007 | T002, T005, T006 | | | |

SC-001→T002/T006 · SC-002→T006 · SC-003→T006 · SC-004→T012 · SC-005→T015/T016/T017 ·
SC-006→full suite at every checkpoint. US1–US3 each close with a docs task.

## Metrics

- Requirements: 13 FR + 6 SC · Tasks: 20 · Coverage: 100% · Critical: 0 · High: 0 ·
  Medium: 1 (resolved in tasks before implementation).
