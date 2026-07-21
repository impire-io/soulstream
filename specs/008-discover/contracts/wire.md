# Contract: Wire Shapes

## The request

```
Subject: SOULSTREAM.SVC.DISCOVER
Reply:   _INBOX.<asker's inbox>
Headers: ordinary record headers (Nats-Msg-Id, Soulstream-*, Soulstream-Sig when keyed)
         Soulstream-Type: topic.discover
Payload: { "query": "vat", "limit": 10, "deadline": "2026-07-21T12:00:02Z" }
```

- Canonical binding for the signature: `DISCOVER` (the service name).
- The deadline is advisory to answerers; the asker enforces it by ceasing to listen.

## The reply

```
Subject: _INBOX.<asker's inbox>   (the request's Reply)
Headers: ordinary record headers
         Soulstream-Type: topic.discover.reply
Payload: { "matches": [ { "path": "vat-q2-x7m2", "name": "Q2 VAT filing",
                          "subject_matter": "filing", "tags": ["finance"],
                          "lifecycle": "active" } ] }
```

- Canonical binding for the signature: `DISCOVER` — **never** the inbox subject
  (ephemeral inboxes would make verification meaningless outside the exchange).
- One reply per responder per request; a responder with no matches sends nothing.
- `Soulstream-Author` is the answering persona; the asker verifies against its
  pinned chain and labels the answer unsigned/verified/failed/unknown-key.

## Non-negotiables

- Nothing on SVC subjects is stored: no stream capture (`SOULSTREAM.SVC.*` is
  outside... note: the stream captures `SOULSTREAM.>` — see plan note below), no KV,
  no obligation survives the exchange.
- No queue groups: every responder hears every request; the asker's merge is the
  only aggregation point.
- Malformed traffic is skipped silently on both sides.

### Observed in implementation: the stream's pub-ack

Because the stream captures the request subject, JetStream delivers a publish ack
(`{"stream":"SOULSTREAM","seq":N}`) to the request's reply inbox. It carries no
record headers, fails `record.Parse`, and is skipped by the asker's malformed-reply
rule — one more reason that rule exists. Tests asserting "no reply on the wire" must
filter for actual `topic.discover.reply` records.

### Plan note: stream capture of SVC subjects

The realm stream captures `SOULSTREAM.>`, which includes `SOULSTREAM.SVC.DISCOVER`
(requests — replies travel on `_INBOX.*` and are never captured). Requests landing in
the stream are harmless (typed records; topic folds ignore unknown subjects) but they
are clutter with no reader. The design treats SVC as request-reply, so requests are
published as **core NATS publishes** (not JetStream publishes): the subject overlap
still stores them in the stream. This cycle accepts that: stream retention is
limits-based, the volume is trivial, and excluding `SVC.>` from the stream would be
an in-place stream mutation (a one-way-door reconfiguration provisioning refuses).
Revisit if SVC volume ever matters.
