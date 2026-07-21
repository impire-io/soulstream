# Implementation Plan: Scatter/Gather Topic Discovery

**Branch**: `008-discover` | **Date**: 2026-07-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/008-discover/spec.md`

## Summary

Discovery is plain NATS request-reply over `SOULSTREAM.SVC.DISCOVER` — no JetStream,
no provisioning change, nothing stored. `topic.Discover` publishes a signed
`topic.discover` record with a reply inbox, gathers `topic.discover.reply` records
until its deadline, and merges them (one entry per topic path, all answerers
credited, per-answer verification status). `topic.RespondDiscovery` is the
any-persona responder: subscribe, rebuild the board projection per request, match
(case-insensitive substring over name/subject-matter/tags, empty query = all, capped
at limit), reply only when there are matches. Signing rides the 006 path with one new
canonical-binding case: service messages bind to the *service name* (`DISCOVER`) —
requests and replies alike — never the ephemeral inbox, so the record build is
factored to accept an explicit binding. `realm.Client` gains `Conn()` (the raw
connection accessor request-reply needs). Clients: CLI `discover` + long-running
`respond`; MCP `soulstream_discover` (11th tool, ask-only). Zero new dependencies.

## Technical Context

**Language/Version**: Go 1.26 (module `github.com/impire-io/soulstream`)
**Primary Dependencies**: existing only — core `nats.go` request-reply (`NewInbox`,
`SubscribeSync`, `PublishMsg` with `Reply`); no JetStream involvement
**Storage**: none — discovery traffic is ephemeral by design
**Testing**: `go test ./...`; matcher and merge are pure (serverless); ask/answer
round-trips on the embedded server with sub-second deadlines
**Target Platform**: unchanged
**Project Type**: existing library + two thin clients
**Performance Goals**: ask latency = responder's board replay (dogfood: a handful of
topics) + network; deadline default 2s, tests use 300–500 ms
**Constraints**: no registry/broker/queue-group (every responder answers; the asker
merges); responders never emit empty replies; silence resolves to empty results, not
errors
**Scale/Scope**: dogfood — a few responders, tens of topics

## Constitution Check

- **I. NATS-Native First** — PASS. The mechanism *is* core NATS request-reply with a
  reply inbox; the "service" is a subject convention, not a component. No queue
  groups (all responders answer by design), no persistence, no new server features.
- **II. Smallest Viable Implementation** — PASS. No ranking, no scoring, no caching,
  no pagination, no service announcements, no curator; matching is one pure
  substring function; the responder rebuilds the board per request rather than
  maintaining warm state.
- **III. ELI5 Documentation** — PASS. `docs/discovery.md` (currently board-only)
  gains the shout-across-the-workshop layer; `cli.md`/`mcp.md` updated in-story.

## Project Structure

### Documentation (this feature)

```text
specs/008-discover/
├── plan.md · research.md · data-model.md · quickstart.md
├── contracts/{library.md, wire.md}
└── tasks.md (Phase 2)
```

### Source Code (repository root)

```text
topic/
├── subjects.go          # SvcSubjectPrefix, SvcDiscoverSubject, ServiceDiscover;
│                        # canonicalBinding gains the SVC case (service-name binding)
├── vocab.go             # TypeDiscover, TypeDiscoverReply; DiscoverPayload,
│                        # DiscoverReplyPayload, DiscoverEntry
├── wire.go              # record build factored to take an explicit binding, so a
│                        # reply publishes to an inbox but signs over "DISCOVER"
├── discover.go          # NEW: Discover (ask+gather+merge), RespondDiscovery
│                        # (subscribe+match+reply), pure matchEntries + mergeReplies
└── discover_test.go     # pure matcher/merge tests + embedded round-trips

realm/connect.go         # (*Client).Conn() *nats.Conn accessor

internal/cli/            # discover command; respond command (long-running, like inbox)
internal/mcpserver/      # soulstream_discover tool (11th)
docs/                    # discovery.md second layer; cli.md; mcp.md; README.md index
```

**Structure Decision**: discovery is topic machinery (it serves the board's domain),
so it lives in `topic` beside the board. The pure matcher/merge keep the
serverless-testable convention. `realm.Client.Conn()` is the minimal new surface —
request-reply needs the raw connection, and realm already exposes `JetStream()` on
the same grounds.

## Complexity Tracking

No violations. Judgment call: every responder answers (no NATS queue group), because
the design wants *all* projections heard and merged — a queue group would pick one
responder and defeat the mechanism.
