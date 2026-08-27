# Quickstart: accounts in a running realm

With a founded realm and `soulstream up` running:

```sh
# Create an isolated account for a second tenant.
soulstream account create acme

# See what exists.
soulstream account list
soulstream account show acme

# Admission: mint a token for a person in the new account through the
# identity plane (the account key is in `account show`'s output), hand
# them the sentinel + token — they land in acme, scoped to it, and see
# nothing of any other account.

# Pause and restore a tenant (data untouched either way):
soulstream account suspend acme
soulstream account resume acme
```

Notes:

- Tenants survive restarts: they live in `<state>/resolver`, beside
  the JetStream data.
- On a BYO realm the command reports the service's own refusal — the
  tenancy authority needs the operator key, which a BYO deployment
  deliberately does not hold (self-hosted: run the authority where
  your operator key lives; Synadia Cloud: the provider arm).
