# Getting started with SoulNode

SoulNode puts a whole realm — record, identity, memory, and a door for
your AI tools — in one binary on a machine you own. This guide takes you
from nothing to a working realm your Claude can talk to, in about five
minutes. Nothing here requires understanding NATS, key ceremonies, or
the ecosystem's internals; that is the point of the product.

## 1. Get the binary

There is no packaged release yet — build from the repo (Go 1.26+):

```sh
git clone git@github.com:impire-io/soulstream.git
cd soulstream && make build
./bin/soulstream version
```

## 2. Found your realm

```sh
soulstream init
```

One command, no questions asked. It generates everything your realm
stands on — the trust root, the accounts, the admission machinery, the
sealing keys — into a state directory (default: your OS config dir,
`soulstream/`), boots the node once to perform the founding acts, and ends
with the only secret you will ever be shown:

```
soulstream: realm "home" founded at /Users/you/Library/Application Support/soulstream
your access token (shown once, never stored):

    sit_4f2a…

point an MCP client at http://127.0.0.1:8080 with that token as its bearer,
or a NATS client at nats://127.0.0.1:4222 with sentinel …/sentinel.creds
```

**Save the token in your password manager now.** It is shown exactly
once; the node keeps only its fingerprint. (Lose it and, for today, you
found a fresh realm — a `soulstream token` command for minting more is an
obvious next feature, not yet built.)

Defaults you can change at founding time (they are written into the
state directory's `config.json`, which is the configuration from then
on): `--realm` (the realm's name, default `home`), `--listen` (the NATS
listener, default `127.0.0.1:4222`), `--door-listen` (the MCP door,
default `127.0.0.1:8080`), `--state` (where the realm lives).

Running `init` again is always safe: it verifies and reports, never
regenerates, never mints a second token.

## 3. Run it

```sh
soulstream up
```

```
soulstream: state /Users/you/…/soulstream (realm "home")
soulstream: listening on nats://127.0.0.1:4222 (loopback)
soulstream: identity plane serving
soulstream: memory plane serving
soulstream: front door serving http://127.0.0.1:8080
```

Everything binds loopback only — nothing is reachable from outside the
machine unless you deliberately front it (step 6). `Ctrl-C` drains and
exits; `soulstream up` again resumes the same realm. The audit log
(admissions, refusals) goes to stderr.

## 4. Connect Claude

```sh
claude mcp add --transport http soulstream http://127.0.0.1:8080 \
  --header "Authorization: Bearer sit_4f2a…"
```

Any MCP client that speaks streamable HTTP with a static header works
the same way. Then, in a session, try:

- `soulstream_whoami` — who the *realm* says you are. The answer comes
  from the server after admission, never from anything the client
  claims.
- `soulstream_start_topic` — open your first topic; then post turns,
  attach text, open work.
- `soulstream_memory_query` — ask the archivist. Everything that happens
  in the realm is kept, verbatim, and answered with citations.

A wrong or revoked token never gets a session, and you will see the
refusal in the node's log.

## 5. Run an agent

Workloads are declared, then run — the runtime mints them a tight,
short-lived credential of their own; they never see your keys. Create
`echo.json` (use a topic path that exists — `soulstream_start_topic`
returns it):

```json
{
  "role": "agent",
  "lifecycle": "service",
  "persona": "echo",
  "topic": "<your-topic-path>",
  "artifact": "file:///path/to/agent-binary"
}
```

```sh
soulstream workload start echo.json
```

The agent's turn appears on the topic attributed to *its* persona, its
lifecycle (opened, claimed, done) is realm activity like everything
else, and the archivist remembers all of it.

## 6. Reaching it from elsewhere (optional)

The door is plain HTTP on loopback. To reach it from your other devices,
front it with something you already trust — the simplest being a
tailnet:

```sh
tailscale serve 8080
```

Your token remains the only secret: the sentinel file is deliberately
public (it is a deny-all routing artifact), and the door itself holds no
keys and no per-user state.

## 7. Where everything lives

The state directory **is** the realm: keys, configuration, the message
store, the archive. Back it up by copying it (node stopped); move to a
new machine by copying it there and running `soulstream up`. Secrets are
owner-only on disk (`0700`/`0600`) — `init` refuses filesystems that
cannot hold that.

## What's deliberately not here yet

- **Minting more tokens / personas for other people** — the identity
  plane supports it; the CLI surface doesn't yet.
- **Public HTTPS mode with real OAuth** — waits on the ecosystem's
  authorization server (soulstream-idp); today the answer is fronting (step 6).
- **Bring-your-own NATS and remote planes** — the configuration shape is
  built for it; the ceremony split ships behind its own design pass.

The honest, complete history of what works and how it was proven lives
in [`../soul-hq/04-JOURNEY/`](../soul-hq/04-JOURNEY/README.md).
