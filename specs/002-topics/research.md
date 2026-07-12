# Research: Topics — JetStream Consumer & Publish API

**Feature**: 002-topics | **Date**: 2026-07-12
**Method**: Verified against `github.com/nats-io/nats.go@v1.52.0/jetstream` source (module cache).

## 1. Publishing an op with headers

- **Decision**: `js.PublishMsg(ctx, *nats.Msg) (*PubAck, error)`; build the `*nats.Msg` from
  `record.Build()` output (`msg.Header = nats.Header(headers)`, `msg.Subject`, `msg.Data`).
- **PubAck** carries `Sequence uint64` (the op's stream sequence) and `Duplicate bool`.
- **Rationale**: one call, gives the stream sequence immediately (useful for the handle to know
  where its op landed). Dedup by `Nats-Msg-Id` is inherited from 001.

## 2. Ordered consumer — replay then live, one path

- **Decision**: `stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{FilterSubjects:
  []string{exactOpsSubject}, DeliverPolicy: jetstream.DeliverAllPolicy})`, consumed via the
  **iterator**: `it, _ := cons.Messages(); msg, err := it.Next()`.
- **Rationale**: an ordered consumer with `DeliverAllPolicy` delivers the full backlog in stream
  order and then continues live on the *same* subscription — no replay/live switchover, satisfying
  FR-017's "no seam" structurally. The iterator (vs. `Consume` callback) gives a synchronous,
  back-pressured loop where "drain then optionally keep following" is inline and cancellable.
- **Exact subject filter**: a `FilterSubjects` with one non-wildcard element materialises exactly
  one topic (a parent never absorbs a child's ops).
- **Cancellation**: `it.Stop()` unblocks `Next()` with `jetstream.ErrMsgIteratorClosed`; Follow runs
  a goroutine that calls `it.Stop()` on `ctx.Done()`.

## 3. Per-message metadata

- **Decision**: `msg.Headers() nats.Header`, `msg.Subject() string`, `msg.Data() []byte`,
  `md, _ := msg.Metadata()` → `md.Sequence.Stream uint64` (the ordering key) and `md.NumPending
  uint64` (backlog remaining).
- **Rationale**: `Sequence.Stream` is the total order the whole engine sorts/appends by (FR-012).

## 4. Bounded cold materialise ("caught up")

- **Decision**: stop draining when the delivered message reports `md.NumPending == 0` — that message
  is the last of the backlog for the filtered subject. Guard the **empty** case first with
  `stream.GetLastMsgForSubject(ctx, opsSubject)`: `ErrMsgNotFound` ⇒ the topic has no ops (return an
  empty/malformed view without creating a consumer, avoiding a blocking `Next()`).
- **Rationale**: `NumPending` is server-computed per the consumer's own filter, so it is race-safe
  and needs no extra round trip or timeout. The empty guard is the one edge (`Next()` blocks with
  nothing to deliver); a normal topic always has ≥1 message (its baseline), so this only bites
  malformed/nonexistent paths.
- **Alternatives rejected**: stream `LastSeq` (spans all of `SOULSTREAM.>`, wrong number); receive
  timeout (heuristic, slow on every cold replay).

## 5. Discovery board — latest per info subject

- **Decision**: `stream.Info(ctx, jetstream.WithSubjectFilter("SOULSTREAM.TOPICS.INFO.>"))` populates
  `StreamInfo.State.Subjects map[string]uint64` (one key per info subject with a message). For each
  subject, `stream.GetLastMsgForSubject(ctx, subject) (*RawStreamMsg, error)` fetches the latest
  announcement directly (`RawStreamMsg{Subject, Sequence, Header, Data, Time}`).
- **Rationale**: O(topics) direct fetches from stream storage, no full-history scan. Empty realm ⇒
  `State.Subjects` empty ⇒ empty board, not an error (FR-027). (If a *live* board were needed, an
  ordered `INFO.>` consumer reducing to `map[subject]latest` would be used instead — deferred, MVP
  board is point-in-time.)

## 6. Rollup / expected-last-subject-sequence (confirm only; unused this cycle)

- Rollup: plain header `msg.Header.Set(jetstream.MsgRollup, jetstream.MsgRollupSubject)` ("Nats-Rollup":"sub"),
  requires `AllowRollup` (already provisioned).
- Optimistic concurrency: `jetstream.WithExpectLastSequenceForSubject(seq, subject)` publish option
  (or the raw `Nats-Expected-Last-Subject-Sequence` header).
- **Rationale for noting**: these are the day-2 rollup primitives; recording the exact API here so
  the re-baselining cycle needs no re-research. **Not used in 002.**

## Sentinels used

`jetstream.ErrMsgIteratorClosed`, `jetstream.ErrMsgNotFound`, `jetstream.ErrStreamNotFound`
(errors.Is). No new third-party dependency is introduced by this cycle.
