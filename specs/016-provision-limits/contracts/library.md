# Library Contract Deltas: Provisioning Byte Limits

All deltas are additive or source-compatible; no existing caller changes.

## Package `realm`

```go
// Budgets are optional per-artefact byte roofs applied when provisioning
// CREATES an artefact. Zero fields mean "no budget" (notify: the mandated
// NotifyMaxBytes). Budgets never modify an existing artefact.
type Budgets struct {
    OpLog    int64
    Notify   int64
    Personas int64
    Objects  int64
}

// DefaultBudgets returns the shapes proven on the known limit-enforced
// free tier (NGS R1): 1 GiB op-log, 64 MiB notify, 64 MiB personas,
// 512 MiB objects.
func DefaultBudgets() Budgets

// Signature changes (source-compatible: variadic, at most one value;
// a second value or any negative field is an error).
func ProvisionOn(ctx context.Context, js jetstream.JetStream, budgets ...Budgets) (*ProvisionReport, error)
func (c *Client) Provision(ctx context.Context, budgets ...Budgets) (*ProvisionReport, error)

// ArtefactResult gains the roof reading (0 = unlimited):
type ArtefactResult struct {
    // ...existing fields unchanged...
    MaxBytes int64
}
```

Error contract: negative budget → `realm: budget for <artefact> must be
positive`-class error before any server call; the limit-enforcing tier's
refusal without budgets keeps surfacing exactly as today (server error
10113 wrapped with the artefact-naming context, FR-003).

## CLI `soulstream provision`

```text
soulstream provision [--budgets]
                     [--budget-oplog SIZE] [--budget-notify SIZE]
                     [--budget-personas SIZE] [--budget-objects SIZE]
```

- `SIZE`: bytes, or KiB/MiB/GiB suffix (binary powers). Explicit 0 or
  negative → flag error naming the artefact (FR-005).
- `--budgets` alone → `DefaultBudgets()`. Flags alone → zero `Budgets`
  with only the named fields set. Both → defaults with named fields
  overwritten (clarification 2026-07-27).
- Report output gains a roof column per artefact: the budget applied
  (`created`) or found (`conformant`/`nonconformant`), `unlimited` when 0.

## MCP

No change. Provisioning is not an MCP tool today and does not become one.

## Compatibility invariants

- `ProvisionOn(ctx, js)` (no budgets) is byte-identical in behavior to
  v0.4.0 for every artefact (SC-002); the existing provisioning test suite
  passes unmodified.
- The legacy-shape convergence path (014) is untouched: it preserves every
  tuned setting including an operator's MaxBytes, and budgets do not apply
  to it (it is an update of an existing stream, and budgets only apply at
  creation).
