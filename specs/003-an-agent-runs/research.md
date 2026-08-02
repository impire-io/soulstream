# Research — 003-an-agent-runs

## R1 — Two realm signing keys, two jobs

- **Decision**: the realm account carries *both* the scoped signing key
  (admission: callout-minted personas get the account's template) and a
  new **plain** signing key (the workload minter embeds per-workload
  permissions in the user JWT it signs).
- **Rationale**: measured upstream [soulrealm journey 0010, spike 3]: a
  user JWT carrying its own permissions but signed by a *scoped* key is
  rejected at connection time. One key cannot serve both lanes.
- **Alternatives**: minting workloads through the identity plane's
  `mint.ephemeral` (D28) — the fleet design's preferred future; it needs
  per-role scoped keys and tag templates and is upstream's own milestone.
  The local plain-key minter is upstream's measured fallback and what
  its public surface ships today.

## R2 — The runtime plane is invocation-scoped, by design

- **Decision**: `soulnode workload start` supervises exactly one
  declared workload — upstream's own command semantics, composed. No
  loop, no claim race, no sweeper in SoulNode.
- **Rationale**: constitution I — the long-running node supervisor is
  soulrealm's unbuilt Fleet milestone; building it here would be
  invention. Design 0001 §6's wording is propagated at landing to say
  "invocation-scoped until upstream's supervisor lands".

## R3 — The runner is a persona like everyone else

- **Decision**: ceremony gains the `runner` bypass-lane user; the
  command signs lifecycle ops through `PersonaSigner("runner")`
  (vault-held, first-touch). Persona name == user name (upstream D27/D6
  discipline).
- **Rationale**: work ops are realm activity and deserve attribution;
  upstream's own cmd ran unsigned in its M1.1 era — the composition can
  afford the full shape because the identity plane is already in the
  process… next door on loopback.

## R4 — The e2e artifact is upstream's own agent-echo

- **Decision**: the test builds
  `github.com/impire-io/soulrealm/cmd/agent-echo` from the module cache
  (`go build` in-test) and declares it. Assertions: the turn's author is
  the workload persona (materialised read), `work.open/claim/done` on
  the topic, everything in the archive.
- **Rationale**: byte-identical to upstream's proof — the point of M1.3
  is *their* workload running unchanged inside *our* composition; a
  local fake agent would prove less. The toolchain-in-test pattern is
  upstream's own integration style.

## R5 — URL is a parameter; `config.json` is the source for the command

- **Decision**: `RunWorkload(ctx, cfg, url, declPath)`; the cmd passes
  `nats://<listen>` from the verified state; tests pass the ephemeral
  URL. Unreachable server → immediate named refusal pointing at
  `soulnode up` (no retry loop).
- **Rationale**: the same one-assembly rule as 001; refusal-fast is
  spec FR-004.
