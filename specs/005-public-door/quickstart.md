# Quickstart: opening the public door

The door plane serves loopback HTTP; public mode adds the OAuth story
for hosted clients. HTTPS and the public name come from deployment
fronting — `tailscale serve` is the measured default until Phase 3's
tsnet gate says otherwise.

```jsonc
// <state-dir>/config.json — the door block, public mode:
{
  "planes": {
    "door": {
      "listen": "127.0.0.1:8080",
      "public_url": "https://mynode.tail1234.ts.net",   // the advertised name
      "auth_issuer": "https://auth.example.com",         // your OIDC AS (soulfold)
      "auth_audience": "soulnode-mynode"                 // the fixed token audience
    }
  }
}
```

Then:

```sh
tailscale serve --bg --https=443 http://127.0.0.1:8080
soulnode up
```

Requirements on the AS (any conforming OIDC provider; soulfold is the
ecosystem's default): RS256 JWT access tokens carrying `oid` (subject),
`roles` (values naming roles your node declares — the founding declares
`realm`), and the deployment's fixed audience; dynamic client
registration for hosted MCP clients.

The person's roles decide what they may touch: a token naming `realm`
gets the persona scope — the same template every realm member holds.
Revocation propagates within the callout TTL after the AS stops
issuing.
