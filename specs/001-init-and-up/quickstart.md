# Quickstart — your realm in two commands

```sh
soulnode init
```

Everything the realm stands on is generated into your state directory
(default: your OS config dir, `soulnode/`) — trust root, accounts,
admission machinery, sealing keys — and the command ends by printing your
access token **once**. Save it; only its fingerprint is kept.

```sh
soulnode up
```

The node is running: an operator-mode messaging server on
`127.0.0.1:4222` (loopback only) with the identity plane inside the same
process. Admission is exactly what a hosted deployment runs — sentinel
plus token through the callout:

```sh
nats --server nats://127.0.0.1:4222 \
     --creds "$STATE/sentinel.creds" \
     --token "sit_…" \
     req '$SYS.REQ.USER.INFO' ''
```

The reply names *you* — your persona, asserted by the server, confined to
your own prefix. A wrong token gets a refusal, and the refusal is in the
node's audit log (stderr).

Re-running `init` never regenerates anything and never mints another
token — it verifies and reports. `Ctrl-C` drains and exits; `soulnode up`
again resumes the same realm. Backing up the realm is copying the state
directory.

## Verifying the feature

```sh
make check    # ceremony unit suite + the init→up→admission e2e ride make test
```
