# Data Model: Scatter/Gather Topic Discovery

## Subjects & services

| Constant | Value |
|---|---|
| SvcSubjectPrefix | `SOULSTREAM.SVC.` |
| SvcDiscoverSubject | `SOULSTREAM.SVC.DISCOVER` |
| ServiceDiscover (canonical binding) | `DISCOVER` |

## DiscoverPayload (`topic.discover` request)

| Field | Type | Rules |
|---|---|---|
| query | string | case-insensitive substring; "" matches all |
| limit | int | per-answerer result cap; asker default 10, answerer clamps to [1, 50] |
| deadline | RFC 3339 | informational for answerers (skip stale work); enforcement is the asker's |

## DiscoverEntry (one topic, as one answerer knows it)

| Field | Type |
|---|---|
| path | string |
| name | string |
| subject_matter | string (omitempty) |
| tags | []string (omitempty) |
| lifecycle | Lifecycle (omitempty) |

## DiscoverReplyPayload (`topic.discover.reply`)

| Field | Type | Rules |
|---|---|---|
| matches | []DiscoverEntry | non-empty (a responder with no matches stays silent) |

## DiscoverAnswer / DiscoverResult (asker-side, merged)

```
DiscoverAnswer { persona string; sig SigStatus }
DiscoverResult { DiscoverEntry; answers []DiscoverAnswer }   // one per topic path
```

Merge rules: key = path; first-seen entry fields win (answers are testimony —
divergent metadata is not reconciled this cycle); one credit per (path, persona)
regardless of duplicate replies; results ordered by first arrival.

## Wire notes

- Request: record on `SOULSTREAM.SVC.DISCOVER` with `Reply` set to the asker's
  inbox; type `topic.discover`; signed over binding `DISCOVER` when keyed.
- Reply: record on the request's reply inbox; type `topic.discover.reply`; signed
  over binding `DISCOVER` (never the inbox subject); verified by the asker with the
  same keyring machinery as read paths.
- Nothing is stored; no stream, KV, or object-store change; provisioning untouched.

## Errors / outcomes

- Deadline with zero replies → empty result slice, nil error (silence is an answer).
- Malformed inbound reply → skipped by the asker (counted nowhere).
- Malformed inbound request → skipped by the responder, which keeps serving.
