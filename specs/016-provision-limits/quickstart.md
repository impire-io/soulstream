# Quickstart: Provisioning a Limit-Enforced Account (NGS R1)

The scenario this feature exists for, end to end.

## Before (v0.4.0): the manual workaround

On an NGS R1 account, `soulstream provision` fails with err 10113
("stream requires max bytes") and the operator must pre-create the op-log
stream, persona KV, and object store by hand with the nats CLI before
provisioning will report the realm conformant.

## After: one command

```sh
# Everything with proven defaults (1 GiB op-log, 64 MiB notify,
# 64 MiB personas, 512 MiB objects):
soulstream --context personal --realm soulstream provision --budgets

# Opinionated: defaults except a bigger attachment store:
soulstream provision --budgets --budget-objects 2GiB

# Only the op-log budgeted, everything else unlimited (self-hosted):
soulstream provision --budget-oplog 10GiB
```

The report shows each artefact with its outcome and its byte roof
(`unlimited` where none). Re-running against an existing realm — including
one hand-created during the workaround era — changes nothing and reports
the roofs as found.

## Verifying locally without NGS

The test suite reproduces the tier with an embedded server whose account
sets `MaxBytesRequired: true` (the switch behind NGS's "Stream Requires
Max Bytes Set"). To see it by hand: run that server config, provision with
no budgets (fails exactly like NGS), then with `--budgets` (succeeds).
