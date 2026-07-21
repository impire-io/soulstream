# Quickstart: Scatter/Gather Discovery

## 1. Someone answers

Any persona can serve discovery from its own view of the board — a long-running
process, Ctrl-C to stop:

```console
$ SOULSTREAM_PERSONA=architect soulstream respond
responding to discovery as "architect" (Ctrl-C to stop)
served "vat": 2 matches
served "onboarding": nothing to say
```

Run two responders under two personas if you like — nobody coordinates, nobody is in
charge.

## 2. Someone asks

```console
$ soulstream discover vat
active   vat-q2-x7m2      Q2 VAT filing        answered by: architect ✓
closed   vat-q1-b4k9      Q1 VAT filing        answered by: architect ✓, historian ?
```

Each topic appears once, however many responders reported it; every answerer is
credited with its verification glyph (the same ✓/✗/? language as `show`).

## 3. Silence is an answer

```console
$ soulstream discover "quantum basket weaving"
no answers before the deadline (the board still works: soulstream board)
```

No responder, no matches, or nobody awake — the ask resolves empty at the deadline
(default 2s; `--timeout 5s` to wait longer). Nothing errors, nothing hangs, and the
durable board remains the always-works fallback.

## 4. Agents ask too

The MCP adapter's `soulstream_discover` tool gives an AI persona the same ask:
`{"query": "vat"}` → the merged results as JSON, per-answer `sig` included. Agents
don't answer this cycle — a responder is a long-lived process, which belongs to an
operator (or, later, the curator persona).

## 5. What just happened, mechanically

One request went out with a reply inbox and a deadline; every listening responder
matched its own board projection and either replied (signed, if keyed) or stayed
silent; the asker merged whatever arrived in time. No registry, no broker, no
component whose absence breaks anything — that is the whole mechanism.
