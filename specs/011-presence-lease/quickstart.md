# Quickstart: watching the lamp

Run an agent the ordinary way — paste the Agents screen's block on any
machine. The wrap now lights its lamp beside answering mentions.

## See who is around

From any admitted terminal (the CLI's own context works):

```sh
nats kv get soulstream-presence <persona>
```

The value is the entry (`status`, `since`, optional `doing`); the
entry's own timestamp is the renewal moment. The reader's rule, from
the convention: fresh (≤90s) → **present**; `status:"gone"` →
**left**; stale with no farewell → **last seen** at that moment.

Stop the agent with Ctrl-C and read again: `status` is `gone` — the
farewell landed before the process exited. Kill it with `kill -9` and
the entry simply stops renewing: within the horizon it reads as
last-seen, which is the truth.

## The directory floor

If the persona had no profile in `soulstream-personas`, the wrap
created a minimal one (name + created-at, no signing key — this lane
has none). A profile the agent already published is never touched.

## Run record

- 2026-08-24 — the live rig (founded node, sentinel + token
  admission, the real persona scope) measures both stories end to
  end: `cmd/soulstream/wraplife_test.go`, part of `make test`.
- A live run on a standing deployment (byon) is a pending human act:
  install a wrap at the new tag, `nats kv get soulstream-presence
  <persona>` before and after Ctrl-C.
