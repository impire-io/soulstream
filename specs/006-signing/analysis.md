# Cross-Artifact Analysis: 006-signing

**Date**: 2026-07-21 | **Scope**: spec ↔ plan ↔ tasks ↔ constitution

## Findings

One HIGH coverage gap found and remediated in the same change; the rest are LOW notes.
Constitution: NATS-native (directory = plain KV with native Create/Update(rev)
concurrency; the only client-side state — seed and pins — *must* live off-substrate by
threat model), smallest-viable (no new deps, no new op types, no watch API, one new
tool), ELI5 docs are per-story tasks (T010, T018, T023, T026).

| ID | Severity | Note | Resolution |
|----|----------|------|------------|
| C1 | HIGH | FR-004 mandates publishing **and updating** one's own profile from both clients, but contracts made `registry.Publish` create-only (`ErrProfileExists`) — the only update path was `Rotate`, so metadata edits were impossible. | Remediated: `Publish` is now create-or-metadata-update — creates when absent; when present, updates metadata with `Update(rev)` while stored key material is authoritative; an incoming *different* key without rotation proof is refused (`ErrKeyConflict`, "rotate instead"), which also implements the second-client-different-key edge case. contracts/library.md, contracts/clients.md, data-model.md, tasks T014/T016/T017 updated. |
| A1 | LOW | FR-013 "every tool result that returns ops": `Board`/`BoardEntry` is a projection over announcements, not an op view. | Scope defined: sig status attaches to the op views (contributions, attachments, announcement, notifications). Board entries stay unannotated this cycle — no acceptance scenario touches them (constitution II). |
| T1 | LOW | Spec says "persona directory", plan/package say `registry`. | Deliberate: the design extension is named registry (`hq/02-DESIGN/extensions/registry.md`); user-facing docs say "persona directory" (plain-words rule), the package keeps the design name. Noted in docs task T018. |
| I1 | LOW | `topic.Notification` is constructed by direct struct conversion from `NotifyPayload` (`Notification(np)`); adding the `Sig` field breaks that conversion. | Implementation note attached to T020 — replace the conversion with explicit field assignment. |

Verified during analysis (not assumed): every publish path — announce, baseline, turn,
comment, attachment, lifecycle, **and mention.notify** — flows through
`topic/wire.go:publishOp` (`publishNotify` delegates to it), so T005 alone makes
FR-002's "every op it publishes" true.

## Coverage

| Requirement | Tasks | | Requirement | Tasks |
|---|---|---|---|---|
| FR-001 | T001, T007, T008 | | FR-008 | T012, T024 |
| FR-002 | T004, T005, T006 | | FR-009 | T019, T020 |
| FR-003 | T005, T006 | | FR-010 | T019, T020 |
| FR-004 | T011, T013, T014, T016, T017 | | FR-011 | T019 |
| FR-005 | T011, T028 | | FR-012 | T008, T016, T021, T025 |
| FR-006 | T012, T015, T016 | | FR-013 | T009, T017, T022 |
| FR-007 | T012, T021, T022 | | FR-014 | T006 |

SC-001→T006 · SC-002→T006 · SC-003→T006 · SC-004→T012/T021/T024 · SC-005→T020 (+ suite
stays green) · SC-006→T024 · SC-007→T014/T020. US1–US4 each have a closing docs task.

## Metrics

- Requirements: 14 FR + 7 SC · Tasks: 28 · Coverage: 100% after C1 remediation ·
  Critical: 0 · High: 1 (remediated pre-implement).
