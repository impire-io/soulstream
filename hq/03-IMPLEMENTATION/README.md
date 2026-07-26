# 03-IMPLEMENTATION — what gets built, in what order

| File | Role |
|---|---|
| [`ROADMAP.md`](ROADMAP.md) | The live plan: the MVP criterion, the one-way doors, day-2 order, and the work stages. What gets built and in what order, behind which gate. |
| [`DOGFOOD.md`](DOGFOOD.md) | The dogfood run's protocol: the two-week scenario, how each persona launches, and the evidence duty that feeds the eg-walker and sealed-topics gates. |

## Conventions

- **Roadmap ↔ journey ↔ specs mapping:** a roadmap item is built as a numbered
  feature through the spec-kit flow (`/speckit-specify` → plan → tasks →
  implement, artifacts frozen in `specs/NNN-*/`), and lands together with a
  numbered episode in [`../04-JOURNEY/`](../04-JOURNEY/README.md). Feature numbers
  come from git branches; episode numbers from the journey sequence; release
  versions from git tags (`v*`).
- **Landing a feature means, in the same merge:** quality gate green
  (`make fmt && make test && make lint`), the roadmap updated with the shipped
  outcome, the journey episode written (`/journey-log`), and behavioral changes
  propagated into the [`../02-DESIGN/`](../02-DESIGN/README.md) docs they touch.
- **Frozen specs.** A `specs/NNN-*/` body is a point-in-time artifact and is not
  rewritten after the feature ships; only its `Status` field is updated to record
  the shipping version. New capabilities that aren't yet decided start as research
  in [`../01-RESEARCH/`](../01-RESEARCH/README.md), not as a spec.
