# Specification Analysis Report: 011-vocab

Cross-artifact consistency analysis (spec.md, plan.md, tasks.md + research.md,
data-model.md, contracts/library.md, quickstart.md) against constitution v1.0.0.
Date: 2026-07-21.

## Findings

| ID | Category | Severity | Location(s) | Summary | Recommendation |
|----|----------|----------|-------------|---------|----------------|
| C1 | Coverage | MEDIUM | US3 scenario 2 / SC-003 ("live views and cold replays alike"); tasks T012 | Dormant reactivation was tested cold and baked but no task asserted it through `Follow` (live). The fold identity makes live ≡ cold by construction, but SC-003 names it explicitly. | T012 extended with a Follow-based reactivation assertion. Applied. |
| A1 | Consistency | LOW | spec FR-009 vs data-model fold table | FR-009 lists "proposed/active → dormant"; data-model allows dormant→dormant (idempotent duplicate marks). Not a conflict — idempotence is required by scenario 6 — but the FR wording could be read as excluding it. | Accept: data-model states "idempotent" explicitly; spec scenario 6 demands it. No change. |
| A2 | Naming | LOW | contract CLI `detach` vs op `attachment.remove` | Two names for one act (command word vs wire word). | Accept: house precedent (`work done` vs `work.done`); docs pair them. No change. |
| D1 | Template drift | LOW | tasks.md Phase 1 | No setup tasks (existing module). | Accept — documented. |

No duplication, ambiguity, or constitution findings. The same-author edit rule
(the one place 011 deviates from author-agnostic folding) is argued in spec
Clarifications and research D3, and is a pure projection rule — no Principle
conflict.

## Coverage Summary

| Requirement | Tasks |
|---|---|
| FR-001 reply | T002, T003, T005, T006, T007 |
| FR-002 same-author edit, render latest | T002, T003, T004, T005 |
| FR-003 chains survive compaction | T004 |
| FR-004 edit warnings (foreign/unknown/empty) | T002, T003 |
| FR-005 resolve marks | T002, T003, T005, T006, T007 |
| FR-006 remove marks | T009, T010 |
| FR-007 artefact tip fallback | T009 |
| FR-008 archival blob GC | T010 |
| FR-009 dormant state + reactivation | T012 |
| FR-010 pure rules | T013 |
| FR-011 manual mark-dormant | T014 |
| FR-012 opt-in curator sweeps | T014 |
| FR-013 compaction + back-compat | T003, T004, T009, T012 |
| FR-014 signing | T002, T005 |
| FR-015 CLI/MCP surfaces | T006, T007, T011, T014 |
| FR-016 additive-only | T003 |
| SC-001…SC-005 | T003/T004 · T009/T010 · T012/T013/T014 · T003/T016 · T016 |

**Unmapped Tasks:** none (T001–T016 all trace).

**Metrics:** 16 FR + 5 SC · 16 tasks · coverage 100% · ambiguity 0 ·
duplication 0 · critical 0.

## Resolution

C1 remediated in tasks.md in this change; LOW findings accepted. Proceed to
implementation.
