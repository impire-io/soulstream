# Contract — the `soulnode` CLI (M1.1 surface)

```
soulnode init [--state DIR] [--listen ADDR]
soulnode up   [--state DIR]
soulnode version
```

State dir resolution (both commands): `--state` flag, else
`SOULNODE_STATE`, else `<os user config dir>/soulnode`.

## `init`

- Fresh directory (absent or empty): runs the full ceremony (design 0001
  §4), persists (`0700`/`0600`), boots the composition transiently,
  performs the founding acts through the public client, prints exactly
  one token block:

  ```
  soulnode: realm founded at <state>
  your access token (shown once, never stored):

      sit_…

  point a client at nats://<listen> with sentinel <state>/sentinel.creds
  ```

  Exit 0.
- Complete directory: verifies, reports (`soulnode: state at <dir>
  verified — <n> artifacts, listener <addr>`), prints **no** token.
  Exit 0.
- Incomplete directory (keys without sentinel): refuses, names the
  condition, documents the recovery (delete the never-booted directory).
  Exit 1.
- Damaged directory: refuses, names the first damaged artifact. Exit 1.
- `--listen` is only honored on the founding run (it is written into
  `config.json`); on a verify run a differing `--listen` is an error
  (the file is the configuration).

## `up`

- Complete directory: starts server + identity plane; logs the state
  dir, the listener, and `identity plane serving`; runs until
  SIGINT/SIGTERM; drains and exits 0. The identity plane's audit log
  (including `callout REFUSED` lines) goes to stderr.
- Uninitialized/incomplete/damaged directory: refuses with the reason
  and a pointer to `init`. Exit 1.
- Bind conflict: refuses naming the address and `config.json`. Exit 1.

## `version`

Prints the module version (dev builds: `dev`). Exit 0.

## Compatibility

Flags are append-only; changing a default or an output line that the
contract above names is a breaking change to this contract file.
