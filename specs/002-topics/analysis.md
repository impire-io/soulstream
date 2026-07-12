# Cross-Artifact Analysis: 002-topics

**Date**: 2026-07-12 | **Scope**: spec ↔ plan ↔ tasks ↔ constitution
**Method**: FR/SC inventory → task coverage; constitution alignment; ambiguity/consistency passes.

## Findings & resolutions

| ID | Category | Severity | Finding | Resolution |
|----|----------|----------|---------|------------|
| G1 | Coverage gap | LOW | FR-028 (oversize inline baseline rejected → point to deferred manifest) had only an implementation task, no test. | Folded an assertion into **T011**. |
| D1 | Underspecification | LOW | FR-007 (`expected` is a hint, never a gate) has no enforcement to test. | Accepted — it is a "MUST NOT gate" negative; the announce merely stores the list. Nothing to enforce = nothing to test. |
| D2 | Consistency | INFO | The pure fold (`apply`) sits in Foundational but implements behaviour verified by US2/US3/US4 tests. | Intentional: `apply` is one cohesive pure function (like 001's slug grammar); stories layer NATS I/O + integration tests on top. Not a defect. |

## Constitution alignment

No violations. NATS-native (ordering = stream sequence; replay+live = one ordered consumer; discovery
= subject enumeration + last-msg fetch; no coordinator/consensus). Smallest viable (five op types,
one consumer path, lifecycle derived, sub-topics = subject depth). Every story ships an ELI5 doc.

## Metrics (post-resolution)

- Requirements: 29 FR + 7 SC.
- Tasks: 31.
- Coverage: **100%** of FRs/SCs requiring buildable work have ≥1 task.
- Ambiguity: 0 open markers · Duplication: 0 · Critical issues: 0.

**Verdict**: Ready for implementation.
