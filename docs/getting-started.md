# Getting started with Soulstream

Soulstream puts a whole realm — the shared record, identity, memory, a
passkey sign-in, a console for people, and an MCP endpoint for your AI tools — in
one binary on a machine you own. This guide takes you from nothing to a
working realm with your own assistant answering mentions, in about five
minutes. Nothing here requires understanding NATS, key ceremonies, or the
ecosystem's internals; that is the point of the product.

## 1. Get the binary

With [Homebrew](https://brew.sh) (macOS and Linux):

```sh
brew install impire-io/tap/soulstream
```

Or download the archive for your platform from the
[releases page](https://github.com/impire-io/soulstream/releases)
(current pre-release: `v0.13.0-rc.3`; macOS and Linux, amd64 and arm64),
unpack it, and put `soulstream` on your PATH. Or build from the repo
(Go 1.26+):

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
`soulstream/`), boots the node once to perform the founding acts, and
ends with the two things you must keep:

```
soulstream: realm "home" founded at /Users/you/Library/Application Support/soulstream
your access token (shown once, never stored):

    sit_4f2a…

your passkey enrollment invite (single use, shown once):

    http://localhost:8378/enroll?invite=sfi_…
```

**Save the token in your password manager now** — it is shown exactly
once; the node keeps only its fingerprint. The invite is how you enroll
your passkey in step 4.

Defaults you can change at founding time (written into the state
directory's `config.json`, which is the configuration from then on):
`--realm` (default `home`), `--listen` (NATS, default `127.0.0.1:4222`),
`--mcp-listen` (the MCP endpoint, default `127.0.0.1:8080`),
`--signin-listen` (sign-in), `--shell-listen` (the console), `--state`
(where the realm lives).

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
soulstream: MCP (assistants) http://127.0.0.1:8080
soulstream: sign-in          http://localhost:8378/login/
soulstream: shell console    http://127.0.0.1:8500
```

Everything binds loopback only — nothing is reachable from outside the
machine unless you deliberately front it (step 8). `Ctrl-C` drains and
exits; `soulstream up` again resumes the same realm. The audit log
(admissions, refusals) goes to stderr.

## 4. Sign in with a passkey

Open the enrollment invite from step 2, register a passkey, and you land
in the **shell** — the console where people read topics, post turns, and
manage the realm. No password exists anywhere in the system: not as a
fallback, not behind a flag. (Lose all passkeys and the admin console is
where a signed-in administrator re-invites; the last administrator
cannot be locked out by design.)

## 5. Connect your assistant

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

## 6. Give your assistant a seat of its own

Sessions through your token act as *you*. To let an assistant work as a
persona of its own — its name on its work, revocable the moment you say
so — open **Agents** in the shell and add one: a handle, a display name,
and your own signature vouching for it. The screen answers with one
paste block, shown once.

## 7. Let it answer mentions — paste the block

The step that makes an agent *addressable*. On the machine where your
assistant is already signed in (your logins, your configuration —
nothing to hand over), paste the block from step 6 into a terminal.
That's the whole step: the block saves the agent's credentials file and
runs

```sh
soulstream wrap --harness claude    # or codex, or --template file.json
```

with the agent's identity in its environment. The `soulstream` binary
you downloaded in step 1 is everything the block needs — the wrapper
and the MCP server it launches (`soulstream mcp`) live inside it, so
a different machine only needs the same one-file download. Prefer the
hard way, or an assistant nobody wrote a preset for? The screen's
collapsed sections carry the raw MCP configuration too.

Now a mention of the agent in any topic becomes an answer — including
mentions posted while the wrapper was off; they are caught up from the
agent's inbox when it starts. Every wake ends in exactly one outcome:
the reply, or the agent's own note that it could not answer. Stop the
wrapper any time — nothing is lost; take the credential away in the
shell and it stops for good.

## 8. Run a declared workload

Beyond wrapping your own assistant, the runtime launches declared
agents and tools as workloads — each minted a tight, short-lived
credential of its own; they never see your keys. Create `echo.json`
(use a topic path that exists — `soulstream_start_topic` returns it):

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

## 9. Reaching it from elsewhere (optional)

The MCP endpoint is plain HTTP on loopback. To reach it from your other devices,
front it with something you already trust — the simplest being a
tailnet:

```sh
tailscale serve 8080
```

Your token remains the only secret: the sentinel file is deliberately
public (it is a deny-all routing artifact), and the endpoint itself holds no
keys and no per-user state.

## 10. Where everything lives

The state directory **is** the realm: keys, configuration, the message
store, the archive. Back it up by copying it (node stopped); move to a
new machine by copying it there and running `soulstream up`. Secrets are
owner-only on disk (`0700`/`0600`) — `init` refuses filesystems that
cannot hold that.

## Bring your own NATS server

If you already run a NATS server, the realm can live on it instead of
the embedded one (design 0003). Two flavours, and operator mode is the
requirement in both — conf-file auth servers are refused by name:

- **Self-hosted** (you speak `nsc` and control the server's config):

      soulstream init --byo self-hosted --url nats://your-host:4222 --realm home

  That prints **the kit** — the exact `nsc` commands and config
  fragments your server needs (public keys only; no secret ever crosses
  in either direction). Apply it, `nsc push`, then re-run `init` with
  the two account public keys the kit's last command prints. soulstream
  verifies what you applied, refuses by name anything missing, and
  finishes the founding over the wire.

- **Synadia Cloud BYON** (their control plane drives the account half):

      SOULSTREAM_SYNADIA_TOKEN=uat_… soulstream init --byo synadia-cloud \
        --url nats://your-byon-host:4222 --synadia-system your-system

  One command; the token is used for the setup calls and never stored.

Everything else — `up`, the planes, the shell, agents — works the same;
the state directory simply holds no server.

## What's deliberately not here yet

- **Public HTTPS mode with real OAuth end-to-end** — the passkey sign-in
  and OIDC lane run locally today; the fronted public-mode story ships
  behind its own pass. Today's answer is fronting (step 9).
- **Remote planes** — the configuration shape is built for them; the
  split ships behind its own pass.
- **Agents woken as isolated workloads** — the wrapper runs your
  assistant where you are; running wraps behind the isolation backends
  waits on a distributable-harness story, recorded in the design.

The honest, complete history of what works and how it was proven lives
in [`../soul-hq/04-JOURNEY/`](../soul-hq/04-JOURNEY/README.md).
