# Data Model: Provisioning Byte Limits

Two shapes change, both in `realm`. Nothing touches the wire, the op-log,
or any persisted format — budgets live only in JetStream artefact
configuration and in the in-memory provision report.

## Budgets (new)

| Field | Type | Meaning |
|---|---|---|
| `OpLog` | `int64` | Byte roof for the op-log stream (`SOULSTREAM`). 0 = unlimited. |
| `Notify` | `int64` | Byte roof for the inbox stream (`SOULSTREAM_NOTIFY`). 0 = keep the mandated `NotifyMaxBytes` (64 MiB) — never unlimited (D3). |
| `Personas` | `int64` | Byte roof for the persona directory KV (`soulstream-personas`). 0 = unlimited. |
| `Objects` | `int64` | Byte roof for the attachment store (`soulstream-objects`). 0 = unlimited. |

**Validation** (before any server contact, FR-005 / D2): any negative field
→ error naming the artefact. Zero fields are legal and mean "as today".
More than one `Budgets` value passed variadically → error (programming
mistake, not operator input).

**Defaults** (`DefaultBudgets()`, FR-002 — the proven workaround shapes):

| Artefact | Default |
|---|---|
| Op-log | 1 GiB |
| Notify | 64 MiB (= `NotifyMaxBytes`) |
| Personas | 64 MiB |
| Objects | 512 MiB |

**Composition rule** (clarification 2026-07-27): the CLI defaults switch
constructs `DefaultBudgets()` and explicit flags overwrite single fields;
explicit flags without the switch start from the zero `Budgets`. The
library only ever sees the composed value.

## ArtefactResult (extended)

| Field | Type | Meaning |
|---|---|---|
| existing fields | — | unchanged (artefact, outcome, nonconformities) |
| `MaxBytes` | `int64` | The artefact's byte roof: as applied for `created`, **as found** for every other outcome. 0 = unlimited. |

**Lifecycle**: populated on every provisioning run for every artefact;
reads come from the artefact's backing stream config (D4). No state
transitions — the report remains a point-in-time reading.

## Explicitly not modelled

- Budgets in `.soulstream.json` / `config.json` — identity files carry
  identity (spec assumption).
- Message-count, age, or per-message caps — out of scope (spec assumption).
- Resizing existing artefacts — provisioning reports, never mutates
  (FR-004); there is no "desired vs actual reconciliation" entity on
  purpose.
