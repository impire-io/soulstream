# Specification Analysis Report: 010-work

Cross-artifact consistency analysis (spec.md, plan.md, tasks.md + research.md,
data-model.md, contracts/library.md, quickstart.md) against constitution v1.0.0.
Date: 2026-07-21.

## Findings

| ID | Category | Severity | Location(s) | Summary | Recommendation |
|----|----------|----------|-------------|---------|----------------|
| I1 | Inconsistency | MEDIUM | data-model.md (WorkEvent/WorkItem field tags) vs topic/view.go conventions | data-model.md prescribes `json:"-"` for `StreamSeq` on work structs, but existing view structs (Contribution, Attachment) serialise StreamSeq/Sig with tags and the baking path *zeroes* them (cleanBaked\*) rather than omitting — prescribing different tags would fork the convention and complicate baked round-trip equality. | Treat data-model tags as indicative; T009 mirrors Attachment's exact field shapes. Note added to data-model.md. |
| C1 | Coverage | MEDIUM | FR-015; tasks.md T012 | No task explicitly asserts per-op signature status on work items/events (and revisions) when published by a signed client — FR-015 says "same per-op signature status for readers". | Extend T012's integration test: signed clients ⇒ items/events/revisions carry verified status; unsigned ⇒ unsigned. Applied. |
| C2 | Coverage | MEDIUM | FR-019; data-model.md invariants; tasks.md T010/T012 | data-model says "lifecycle transitions never modify items (test guards it)" but no task contained that test; likewise "closed topic permits work ops with warning" (spec edge case) was untasked. | Extend T010 (fold: close op leaves items untouched) and T012 (closed topic accepts work ops). Applied. |
| A1 | Naming | LOW | tasks.md T007; plan.md structure | Artefact MCP tools live in `internal/mcpserver/work_tools.go` — the file name is broader than "work items" but keeps all seven 010 tools together. | Accept (one feature, one file). No change. |
| D1 | Template drift | LOW | tasks.md Phase 1 | Template expects Setup tasks; 010 has none (existing module, zero new deps). | Accept — explicitly documented in tasks.md. No change. |

No duplication, ambiguity, or constitution findings.

## Coverage Summary

| Requirement | Has Task? | Task IDs |
|---|---|---|
| FR-001 revise via existing op | ✓ | T002, T005, T006 |
| FR-002 lineage grouping rule | ✓ | T002, T003 |
| FR-003 deterministic tip + history | ✓ | T002, T003, T004 |
| FR-004 identity root / display name | ✓ | T002, T003 |
| FR-005 fetch tip/revision + digest | ✓ | T005, T006, T007 |
| FR-006 artefacts survive compaction | ✓ | T004 |
| FR-007 four additive op types | ✓ | T001, T009 |
| FR-008 open creates item | ✓ | T009, T010, T012 |
| FR-009 first claim wins, void later | ✓ | T009, T010, T012 |
| FR-010 state machine, malformed≠void | ✓ | T009, T010, T017 |
| FR-011 author-agnostic, trail | ✓ | T009, T010 |
| FR-012 anchored evidence | ✓ | T010, T014 |
| FR-013 mentions in work bodies | ✓ | T012 |
| FR-014 items bake + old baselines | ✓ | T011 |
| FR-015 signing unchanged | ✓ | T009, T012 (post-remediation) |
| FR-016 CLI surface | ✓ | T006, T014, T018 |
| FR-017 MCP surface | ✓ | T007, T015, T018 |
| FR-018 additive-only | ✓ | T010, T019 |
| FR-019 content ops / curator / lifecycle decoupled | ✓ | T009, T013, T010+T012 (post-remediation) |
| SC-001 race convergence | ✓ | T010, T011, T012 |
| SC-002 revision convergence | ✓ | T003, T004 |
| SC-003 compaction equality | ✓ | T004, T011 |
| SC-004 pre-010 suite + baselines | ✓ | T010, T011, T019 |
| SC-005 end-to-end CLI+MCP loop | ✓ | T014, T015, T019 |

**Constitution Alignment Issues:** none. (I: no new infrastructure or server
features; II: stage 1 adds zero op types and derived-only state, stage 2 adds
exactly the four named ops; III: docs tasks T008/T016/T018 ship inside their
stories.)

**Unmapped Tasks:** none — T001–T019 all trace to FRs/SCs above.

**Metrics:**
- Total Requirements: 19 FR + 5 SC
- Total Tasks: 19
- Coverage: 100% (after remediation of C1/C2)
- Ambiguity Count: 0 | Duplication Count: 0 | Critical Issues: 0

## Resolution

MEDIUM findings I1/C1/C2 remediated directly in data-model.md and tasks.md in
this same change (autonomous cycle convention); LOW findings accepted as-is.
Proceed to `/speckit-implement`.
