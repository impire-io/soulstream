# Wire Contract: Memory Convention (015)

## Subject & binding

| Aspect | Value |
|---|---|
| Subject | `SOULSTREAM.SVC.MEMORY` (requests); asker's ephemeral inbox (replies) |
| Canonical binding | `MEMORY` for requests AND replies — the service name, never the inbox (008 rule) |
| Transport | Core NATS request/reply. `SOULSTREAM.SVC.>` is captured by no stream (014): transient by construction |
| No-responders | Core NATS returns no-responders when nothing subscribes — treated as silence, exactly like discovery |
| Witness subscription | Plain subscribe (NO queue group — every witness hears every ask) + `Flush()` before assuming live |

## Op types

All are ordinary `record.Record` wire form: `Nats-Msg-Id` (op id, UUIDv4),
`Soulstream-Version: 1`, `Soulstream-Author`, `Soulstream-Type`, `Soulstream-Ts`,
optional `Soulstream-Sig` over `Canonical(realm, "MEMORY")`.

| Type | Direction | Payload |
|---|---|---|
| `memory.query` | asker → `SOULSTREAM.SVC.MEMORY` (Reply = inbox) | `{"query": string, "scope": {"topics": [string], "after": RFC3339}?, "deadline": RFC3339}` |
| `memory.answer` | witness → inbox | `{"answer": string, "citations": [{"topic": string, "op_id": string}]?, "coverage_from": RFC3339?}` |
| `memory.fetch` | asker → `SOULSTREAM.SVC.MEMORY` (Reply = inbox) | `{"topic": string, "op_id": string, "deadline": RFC3339}` |
| `memory.exhibit` | witness → inbox | `{"exhibit": ExhibitDocument}` |

## Rules

**Asker (query)**
- Deadline = now + timeout; timeout defaults 3s, clamped [100ms, 30s].
- Gathers until deadline or 100 answers; later arrivals discarded.
- Per answer: parse → type check → `VerifyRecord(rec, realm, "MEMORY", kr)`:
  `failed` ⇒ discard (malformed, like discovery); otherwise keep with status.
  Empty `answer` text ⇒ malformed, discard.
- Grades citations by live resolution only (materialise cited topic, `ContainsOp`):
  resolves ⇒ `fact`; doesn't ⇒ `unverifiable`. Fetching is never automatic.

**Witness**
- Silent on: nil capability, nothing relevant, malformed request, stale deadline.
- Answers/exhibits signed with binding `MEMORY` when a signer is configured.
- `coverage_from` included in answers when declared.

**Asker (fetch)**
- Live-first: scan the topic's own `OPS.<path>` / `INFO.<path>` subjects for
  `Nats-Msg-Id == op_id`; found ⇒ no scatter/gather at all.
- Otherwise publish `memory.fetch`; per reply: reply-op signature check as above, then the
  embedded exhibit judged by its OWN signature (`VerifyExhibit`): first `verified` exhibit
  wins immediately; `unsigned` held as fallback; `failed` discarded.
- Nothing by deadline ⇒ silence result (found = false). Silence is an answer.

## Exhibit document (version 1)

```json
{
  "version": 1,
  "realm": "soulstream",
  "binding": "vat-q2-2026-x7m2",
  "subject": "SOULSTREAM.TOPICS.OPS.vat-q2-2026-x7m2",
  "headers": {
    "Nats-Msg-Id": ["9f86d081-…"],
    "Soulstream-Version": ["1"],
    "Soulstream-Author": ["daan"],
    "Soulstream-Type": ["turn.post"],
    "Soulstream-Ts": ["2026-05-12T09:00:00Z"],
    "Soulstream-Sig": ["…base64…"]
  },
  "payload_b64": "…base64 of the op payload verbatim…"
}
```

- Headers and payload are VERBATIM captures — same bytes ⇒ signatures keep verifying
  (the 014-migration invariant).
- `binding` is the verification input; `subject` is provenance display. A tampered
  binding/realm/headers/payload flips verification to `failed` — the document cannot lie
  usefully.
- Strict decode: unknown fields rejected; `version != 1` rejected.
- Verification verdicts = the standing `SigStatus` set: `verified` / `failed` /
  `unsigned` / `unknown-key`; chain rule = any key in the author's validated chain;
  distrusted author ⇒ `failed`.
- Fits core-NATS payload limits by construction: the captured op already rode the stream
  (NGS 1MiB ceiling; manifest baselines stay small by design — their blobs are NOT
  bundled; an exhibit is evidence of the record, not a blob bundle).

## Retention

Nothing. No stream captures `SOULSTREAM.SVC.>`; no new streams, KV, or object-store use.
SC-006 (zero residue) holds by construction and is asserted by test.
