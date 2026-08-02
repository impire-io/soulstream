# Research — 001-init-and-up

No NEEDS CLARIFICATION markers existed. The decisions below close the
plan-time choices; the measured base is the composition rig (topic
`single-binary-composition`, in git history) and the delivered upstream
seams.

## R1 — soulidentity pinned at a pseudo-version until its first tag

- **Decision**: `require github.com/impire-io/soulidentity
  v0.0.0-<timestamp>-<hash>` (main; carries the embed seam). No
  `replace`, ever. The pin flips to `v0.x.y` the day upstream tags.
- **Rationale**: constitution I's intent is reproducible consumption of
  public surfaces — a pseudo-version is exactly that; only the *letter*
  ("tagged releases") awaits upstream's release act, which is not
  SoulNode's to perform. Tracked in the plan's Complexity Tracking and
  the roadmap's external dependencies.
- **Alternatives considered**: waiting for the tag (gates Phase 1 on a
  calendar, violating gates-not-calendars); asking for the tag now (a
  release act on the maintainer's repo — flagged to him, not assumed);
  `replace` (forbidden outright).

## R2 — Package split: pure `ceremony`, talking `node`, thin `cmd`

- **Decision**: `ceremony` generates/persists/loads/verifies the state
  directory and never opens a connection or starts a server; `node`
  composes (server boot, identity plane, founding acts); `cmd/soulnode`
  parses flags/env, owns signals, prints.
- **Rationale**: the sibling discipline (pure logic separate from I/O)
  — ceremony tests run in microseconds with no ports; the e2e lives
  where the composition lives.
- **Alternatives considered**: one `soulnode` package (couples key
  generation to server lifecycle in tests); putting founding acts in
  `ceremony` (they need a live connection — wrong side of the split).

## R3 — `init` boots the composition transiently for the founding acts

- **Decision**: `init` = `ceremony.Generate` + persist, then
  `node.Start` (same code path `up` uses), founding acts through the
  public `client` over the node's own loopback connection (import both
  signing keys, mint sentinel to disk, create the first token), then
  stop. The KV buckets need no separate step — the identity plane's
  `embed.Run` creates them on start (upstream contract).
- **Rationale**: one composition, two commands — `init` proves boot
  works before the user ever runs `up` (fail at init time, not first-use
  time), and the founding acts run in the exact environment they'll live
  in. The rig measured this shape end to end.
- **Alternatives considered**: founding acts on first `up` (makes `up`
  non-idempotent and moves the one-time token print into a long-running
  command's log — wrong place for a secret); a separate `soulnode
  provision` command (a third command for no user benefit —
  constitution V wants two).

## R4 — Completion marker and damage semantics

- **Decision**: `sentinel.creds` is the founding-complete marker — it is
  the last artifact `init` writes. Keys present + sentinel absent ⇒
  "incomplete ceremony" refusal naming the fact, with the documented
  recovery (delete the never-booted directory, run `init` fresh).
  `Verify` checks the full inventory: presence, modes, parseability
  (seeds decode, JWTs parse, account pubs match their seeds); first
  failure is named in the error. `init` on a complete directory =
  `Verify` + report; `up` on anything but a complete directory = refusal
  pointing at `init`.
- **Rationale**: the marker must be an artifact the ceremony needs
  anyway (no bookkeeping file to drift), ordered last so its presence
  implies everything before it. Resume-on-partial was rejected: trust
  roots are cheap to regenerate before first boot and dangerous to
  half-trust after.

## R5 — First-token semantics

- **Decision**: the token is created only on the run that generated the
  state (fresh `init`), printed to stdout with a shown-once warning;
  only the digest lives on (in the identity plane's token bucket).
  Re-`init` never mints. Additional tokens are the identity plane's
  existing client surface, out of M1.1 scope.
- **Rationale**: spec FR-005/FR-006; matches soulidentity's own
  plaintext-once, digest-is-the-handle design.

## R6 — Configuration: `config.json`, listener default, state-dir default

- **Decision**: `config.json` in the state dir, written by `init`,
  currently `{"listen": "127.0.0.1:4222"}` — the conventional NATS port,
  loopback-only bind. Flag `--listen` at `init` time sets it; `up` reads
  it (no `up`-time override — the file is the configuration). State dir:
  `--state` flag, `SOULNODE_STATE` env, default
  `os.UserConfigDir()/soulnode` (the archivist's precedent). Bind
  conflict at `up` ⇒ refusal naming the address and the config file. The
  schema grows a per-plane `{enabled, url, creds}` block as planes land
  (design §2) — added when the second plane arrives (M1.2), not
  speculatively.
- **Rationale**: design §2's contract with constitution III; smallest
  viable schema today, named growth path.

## R7 — Logging

- **Decision**: human lines (`state dir`, `listening`, `identity plane
  serving`, shutdown) on stdout via the cmd; the identity plane's
  audit/serving log (slog text, includes `callout REFUSED`) on stderr —
  the daemon convention upstream already set. The e2e captures stderr's
  audit for the refusal assertions.
- **Rationale**: secrets never in logs; audit greppability matches the
  upstream e2e patterns.

## R8 — Test shape

- **Decision**: `ceremony_test.go` — generate/persist/load roundtrip,
  mode assertions, damage matrix (missing file, corrupted seed, JWT/seed
  mismatch), idempotent re-verify. `node_test.go` — the M1.1 gate:
  fresh dir → `Init` → `Start` → the three admission observations
  (sentinel + token admits, `$SYS.REQ.USER.INFO` names the persona and
  confines it to its own prefix; garbage refused with `callout REFUSED`
  in the captured audit; revoked refused) → interrupt/restart works.
  `main_test.go` — CLI contract: fresh `init` prints exactly one token
  line; re-run prints none and exits 0; `up` on uninitialized dir exits
  non-zero naming `init`. All tests hermetic (loopback, temp dirs).
- **Rationale**: SC-002/003/004/005 mapped one-to-one; the observations
  reuse the rig's proven protocol.
