# Contract — `config.json` (as of 004)

```json
{
  "listen": "127.0.0.1:4222",
  "realm": "home",
  "planes": {
    "memory": { "enabled": true },
    "door":   { "enabled": true, "listen": "127.0.0.1:8080" }
  }
}
```

- `planes.door.enabled` — the MCP door plane; absent means enabled (the
  bundle default). Disabled: no HTTP listener exists at all.
- `planes.door.listen` — the door's loopback HTTP bind; same
  fixed-at-founding, refuse-on-conflict rules as the NATS listener.
- Public mode (`public_url`, `auth_issuer` — upstream's OAuth story) is
  deliberately absent until soulfold exists; it will join this block
  additively.
- Everything else: as of 002, unchanged. The state-dir inventory is
  untouched by this feature (the door custodies nothing).
