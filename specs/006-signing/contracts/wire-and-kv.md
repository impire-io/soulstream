# Contract: Wire & KV Shapes

## Soulstream-Sig header

- Value: standard base64 (`StdEncoding`) of the 64-byte Ed25519 signature.
- Signing input: `record.Record.Canonical(realm, binding)` computed while `Signature`
  is empty (the canonical `sig` key is `omitempty`, so these are the bytes-before-sig).
- Binding rule (the `topic` argument of `Canonical`):

| Subject | binding |
|---|---|
| `SOULSTREAM.TOPICS.OPS.<path>` | `<path>` |
| `SOULSTREAM.TOPICS.INFO.<path>` | `<path>` |
| `SOULSTREAM.PERSONA.NOTIFY.<persona>` | `<persona>` |

- Verification: parse wire record → take `Signature`, clear the field → recompute
  canonical with the binding derived from the subject the op was consumed from →
  `ed25519.Verify` against each key in the author's validated chain; any match ⇒
  `verified`.
- An op with no `Soulstream-Sig` is `unsigned` and valid forever. Nothing else on the
  wire changes; an unsigned publish is byte-identical to pre-feature output.

## KV bucket `soulstream-personas`

- Provisioned by `realm.Provision`, create-or-report; history 10; no TTL, no size cap
  beyond server defaults. Absence is tolerated by every reader.
- Key: persona slug. Value: profile JSON (UTF-8, no envelope):

```json
{
  "name":         "architect",
  "display_name": "The Architect",
  "kind":         "agent",
  "description":  "Reviews designs and asks hard questions.",
  "operated_by":  "daan",
  "created_at":   "2026-07-21T09:00:00Z",
  "signing_key":  { "ed25519": "<base64 32B>", "since": "2026-07-21T09:00:00Z" },
  "rotations": [
    { "from": "<base64 oldPub>", "to": "<base64 newPub>", "proof": "<base64 64B sig>" }
  ]
}
```

- Optional fields omitted when empty (`display_name`, `description`, `operated_by`,
  `signing_key`, `rotations`).
- Concurrency: first publish = KV `Create` (existing key ⇒ hard error, never
  overwrite); every later write = KV `Update` with the read revision (lost race ⇒
  retry-able error, never blind write).

## Rotation proof

```
proof = Ed25519-Sign(oldKey, "soulstream-key-rotation\n" + persona + "\n" + toB64)
```

- `persona` binds the proof to one directory entry (a stolen proof cannot rotate
  another persona that happens to use the same key).
- `since` is deliberately outside the proof — it is informational (clarification
  2026-07-21).
- Chain validity (see data-model.md): contiguous `from`/`to`, every proof verifies,
  final `to` equals current `signing_key.ed25519`. Any violation distrusts the persona.

## Pins file (client-side, not wire)

`<user-config-dir>/soulstream/pins/<realm>.json`, rewritten atomically:

```json
{
  "realm": "workshop",
  "personas": {
    "architect": ["<base64 rootPub>", "<base64 currentPub>"]
  }
}
```

- Semantics: the validated chain as first seen; only ever extended (published chain
  must have the pinned chain as a prefix). Divergence ⇒ persona distrusted + loud
  warning; the pin is kept as evidence, never overwritten by the diverged chain.

## Key seed file (client-side, not wire)

`<user-config-dir>/soulstream/keys/<realm>-<persona>.ed25519` — 32-byte seed, base64,
single line, file mode 0600. Overrides: `--key-file` / `SOULSTREAM_KEY_FILE`
(pins: `--pins-file` / `SOULSTREAM_PINS_FILE`).
