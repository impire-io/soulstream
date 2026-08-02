# Contract — `config.json` (as of 002)

```json
{
  "listen": "127.0.0.1:4222",
  "realm": "home",
  "planes": {
    "memory": { "enabled": true }
  }
}
```

- `listen` — the loopback listener (001's contract, unchanged).
- `realm` — the realm name, set at founding (`init --realm`), immutable
  thereafter through this file's own rule: the file is the
  configuration; a re-`init --realm` that disagrees is an error.
- `planes.<name>.enabled` — design §2's block, first instance. Absent
  block or absent field means enabled (the bundle is the default).
  `enabled: false` for `memory` runs the node without the archivist —
  no `archive/` directory is created, admission unaffected.
- Growth (design §2): each block gains `url` and `creds` when a plane
  can actually be pointed elsewhere; absent fields keep the loopback +
  state-dir defaults. Additive only.

## Data model deltas (state directory)

- `users/archivist.creds` (0600) joins the ceremony inventory — the
  memory plane's transport credential.
- `archive/` (0700, created at first enabled `up`) — the exhibit store;
  deliberately *not* part of `init`'s inventory (it is the plane's
  working data, not founding material) and never required by `Verify`.
- The archivist's *persona* key is nowhere on disk — vault-held
  (research R3, SC-003).
