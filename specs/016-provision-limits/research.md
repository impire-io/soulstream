# Research: Provisioning Byte Limits

Decisions resolving every open point in the Technical Context. No external
unknowns — the feature sits entirely on JetStream config fields the codebase
already uses; research here is decision-recording, not discovery.

## D1 — Budgets surface: one plain struct, optional-by-variadic

**Decision**: `type Budgets struct { OpLog, Notify, Personas, Objects int64 }`
(bytes; zero = no budget for that artefact) passed as
`ProvisionOn(ctx, js, budgets ...Budgets)` / `Client.Provision(ctx, budgets ...Budgets)`
with more than one value rejected as a programming error.

**Rationale**: Existing callers (tests, CLI, any external user) compile
unchanged — source compatibility without a second entry point. A functional
options API for a single option is exactly the speculative machinery
Article II prohibits; a plain value the caller can construct, print, and
compare is the smallest thing that works.

**Alternatives considered**: breaking the signature (`ProvisionOn(ctx, js,
Budgets{})` everywhere) — pointless churn for callers who want today's
behavior; `WithBudgets()` option type — machinery for one knob; a separate
`ProvisionOnWithBudgets` — two names for one operation.

## D2 — Zero means "no budget" in the struct; the CLI rejects explicit zero

**Decision**: at the library level a zero field is indistinguishable from
"not set" and means unlimited (today's behavior); negative values are
rejected before any server contact with an error naming the artefact. The
CLI, which can see that a flag was explicitly passed, rejects an explicit
`--budget-* 0` (and negatives) at parse time per FR-005.

**Rationale**: Go zero values make "absent" and "0 bytes" one state in a
plain struct; pushing explicitness detection to the layer that has it (flag
parsing) keeps the library honest and the struct plain. A `*int64` field
per artefact would restore the distinction at the cost of a pointer-heavy
API for a case (an intentional 0-byte artefact) that is meaningless anyway
— JetStream treats 0 as unlimited, so an explicit zero could only mislead.

**Alternatives considered**: pointer fields (`*int64`) — ugly to construct,
and the distinguished case has no valid use; sentinel `-1` for unlimited —
inverts the zero-value convention every Go reader expects.

## D3 — Notify keeps its mandate; a notify budget overrides at creation only

**Decision**: `notifyStreamConfig()` keeps `NotifyMaxBytes` (64 MiB) as its
roof whenever `Budgets.Notify` is zero; a non-zero `Budgets.Notify` replaces
it for a notify stream being created. `DefaultBudgets().Notify ==
NotifyMaxBytes`, so the defaults switch changes nothing about notify.

**Rationale**: 014 mandated the notify roof deliberately (bounded store on
R1 tiers); budgets must not silently strip it. Overriding at creation is
just choosing a different roof for the same bounded design; existing notify
streams are report-only like every other artefact.

## D4 — Reporting roofs: read the artefact's backing stream config

**Decision**: `ArtefactResult` gains `MaxBytes int64` (0 = unlimited). For
created artefacts it is the budget applied; for existing ones it is read
from the artefact's backing stream config (`SOULSTREAM`,
`SOULSTREAM_NOTIFY`, `KV_soulstream-personas`, `OBJ_soulstream-objects`)
via the same `js.Stream(...).CachedInfo()` shape the op-log path already
uses.

**Rationale**: KV/object-store status interfaces expose usage, not limits;
the backing stream's config is where the roof actually lives, and reading
it is one lookup with machinery already in use. The report stays
create-or-report: reads only.

**Alternatives considered**: type-asserting `KeyValueStatus` to its
concrete type for `StreamInfo()` — couples to an implementation detail of
nats.go for no fewer round trips.

## D5 — CLI sizes are human units both ways

**Decision**: `--budget-oplog 1GiB`-style values (plain bytes, or
KiB/MiB/GiB suffixes, binary powers only) parsed by a small colocated
helper; the provision report prints roofs the same way (`1.0 GiB`,
`unlimited`). No new dependency.

**Rationale**: FR-006's "at a glance" already forces a formatter; accepting
what we print is symmetry, and raw byte counts for gigabyte values invite
off-by-three-orders mistakes. Binary powers only because JetStream limits
are byte counts and mixed SI/binary parsing is a known confusion source.

**Alternatives considered**: bytes-only flags — hostile UX for the primary
use case; importing a units library — a dependency for twenty lines.

## D6 — Reproducing the limit-enforcing tier in tests

**Decision**: extend `internal/natstest` with a variant that starts the
embedded server with an account whose JetStream limits set
`MaxBytesRequired: true` — the exact server switch behind NGS R1's
"Stream Requires Max Bytes Set: true". US1's two acceptance scenarios run
against it: budget-less provisioning fails with the per-artefact error
(same 10113 class as observed on NGS), budgeted provisioning creates all
four artefacts.

**Rationale**: The whole feature exists because of this tier; a test that
cannot express the tier would prove nothing. The server option is public
nats-server configuration, not a mock — the failure is the real failure
[measured, once implemented].

**Alternatives considered**: asserting only that configs carry MaxBytes —
would pass even if the server still refused them; testing against NGS
itself — external dependency in the suite, forbidden by convention.
