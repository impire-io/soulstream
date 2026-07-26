# Episode 0004 — Provisioning byte limits: the strict landlord gets a one-command realm (2026-07-27)

Since Soulstream first touched NGS (2026-07-21), one debt stood in every setup
note: limit-enforced account tiers (NGS R1 mandates a byte roof on every
stream — "Stream Requires Max Bytes Set") refused `soulstream provision`
outright with err 10113, and the operator hand-created the op-log stream,
persona directory, and object store before provisioning would call the realm
conformant. `016-provision-limits` retires that workaround: an optional
`realm.Budgets` value (variadic on `ProvisionOn`/`Client.Provision`, so every
existing caller compiles unchanged) sets creation-time roofs, and
`DefaultBudgets()` carries the empirically proven shapes — 1 GiB op-log,
64 MiB inbox, 64 MiB personas, 512 MiB objects. The CLI grew
`provision --budgets` plus four `--budget-*` size flags (binary units only).

The R1 refusal itself is now reproduced in the suite: `internal/natstest`
gained a server variant whose account sets `MaxBytesRequired: true` — the
same switch NGS runs — so both halves of the headline scenario are
[measured]: budget-less provisioning fails naming the refused artefact, and
`DefaultBudgets()` creates all four artefacts with their roofs verified on
the server, not just in the report. Create-or-report stayed inviolate
[measured]: re-provisioning with different budgets mutates nothing and
reports roofs **as found** (read from backing streams — where the server
taught us it spells "no roof" as `-1`, normalised to the report's
`0 = unlimited`). The live SC-003 check against the real NGS realm came back
all-conformant with every roof visible — the as-found values exactly equal
the defaults, because the defaults ARE the hand-created shapes [measured].
One deliberate asymmetry [judgment]: the inbox stream never goes unlimited —
an unbudgeted notify keeps its mandated 64 MiB, because 014 bounded that
store by design. 342 tests, 0 skipped, lint 0.

Nothing was refuted; the one surprise worth keeping is the server's `-1`
spelling for "unlimited", which would have leaked into reports as a
nonsense roof had the fixture not caught it.

What it opened: provisioning on hosted tiers is now a first-run experience
instead of a wiki page, which matters the week the dogfood run starts.

Reversal condition: none — records a completed build/measurement. (The
default sizes are constants, not commitments: if real usage outgrows them,
they change as plain numbers with no design consequence.)

Trail: `specs/016-provision-limits/` (spec, plan, research D1–D6,
data-model, contracts, quickstart, tasks), `docs/provisioning.md` (storage
budgets, ELI5), `realm/` + `internal/cli/` + `internal/natstest/`; branch
`016-provision-limits` (spec 22d8aa7 → feat ecab2b5).
