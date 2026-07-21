<!-- SPECKIT START -->
Active feature: **007-rollup** — re-baselining (rollup), manifest baselines, archived lifecycle.
Rollup = publish the folded baseline with `Nats-Rollup: sub` (`jetstream.MsgRollup`/`MsgRollupSubject`)
guarded by `jetstream.WithExpectLastSequencePerSubject(lastConsumedOp.StreamSeq)` — rejected publish
⇒ `ErrRollupLost`, nothing changed. `BaselinePayload` grows additively: `baked *BakedState`
(contributions/attachments/lifecycle folded in) beside opaque `state`, or `manifest *ManifestRef`
(one object `baseline/<path>/<op-id>`, digest, size) above the 128 KB threshold. Fold seeds from
`baked`; `payload.frontier` non-empty ⇒ those ids are the frontier candidates (baseline id only at
birth); baked interior ids anchor-resolvable, never frontier. Baked elements inherit the baseline
op's sig status. `Archived` terminal: fold ignores later transitions; all Handle writes refuse
(`ErrTopicArchived`); `Handle.Close` = transition + 1 best-effort rollup, `Handle.Archive` =
transition + rollup ×3 retries. View structs gain lowercase json tags (baked state is wire — MCP
result casing changes). CLI: `rollup`, `archive`; MCP: `soulstream_rollup_topic` (10th tool).

For technologies, project structure, shell commands, and other context, read the
current plan: [specs/007-rollup/plan.md](specs/007-rollup/plan.md)
(spec: `specs/007-rollup/spec.md`, contracts: `specs/007-rollup/contracts/`).
Done: `001`–`005` (MVP), `006-signing` merged.

Project conventions:
- Go 1.26; module `github.com/impire/soulstream`.
- `record` and `identity` packages import NO NATS; `realm`, `topic`, and `registry` are the
  NATS-touching packages.
- Keep pure projection logic (materialise/board fold over `record.Record` slices, registry chain
  validation) separate from the NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; use the modern
  `github.com/nats-io/nats.go/jetstream` API (ordered consumers, `PublishMsg`).
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
<!-- SPECKIT END -->
