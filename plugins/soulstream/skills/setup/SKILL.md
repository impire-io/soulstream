---
name: setup
description: Set up Soulstream on this machine — install the soulstream binaries, create a NATS context, configure realm/persona environment variables, provision the realm, and optionally create a signing key. Use when the Soulstream MCP server fails to start, its tools are missing, or the user asks to set up or connect Soulstream.
---

# Soulstream setup

Walk the user through connecting this machine to a Soulstream realm. Work step by step,
verifying each step before moving on. Soulstream speaks NATS: the user needs a reachable
NATS server with JetStream enabled.

## 1. Install the binaries

Check first: `soulstream version` and `soulstream-mcp -version` both print a version when
installed. If missing, install with one of:

- `go install github.com/impire-io/soulstream/cmd/soulstream@latest` and
  `go install github.com/impire-io/soulstream/cmd/soulstream-mcp@latest` (needs Go 1.26+;
  binaries land in `$(go env GOPATH)/bin` — make sure that is on PATH).
- Download the archive for this OS/arch from
  https://github.com/impire-io/soulstream/releases and place both binaries on PATH.
- `git clone https://github.com/impire-io/soulstream && cd soulstream && make build`
  (binaries in `./bin`).

If the binary cannot go on PATH, `SOULSTREAM_MCP_BIN` may point at it instead.

## 2. Create a NATS context

The tools connect through a named NATS context (never raw URLs):

```sh
nats context add <name> --server <nats-url> [--creds <file>]   # needs the nats CLI
nats context select <name>
```

If the `nats` CLI is missing, the `/nats-context` skill or https://github.com/nats-io/natscli
can help. Verify with `nats context ls` and `nats account info` (JetStream must be enabled).

## 3. Configure the persona

Three environment variables tell every Soulstream tool who is acting where. Ask the user
for their values and add them to the shell profile (fish: `set -Ux`, bash/zsh: `export`
in the profile file):

- `SOULSTREAM_CONTEXT` — the NATS context name from step 2
- `SOULSTREAM_REALM` — the realm (one shared board per realm, e.g. `acme`)
- `SOULSTREAM_PERSONA` — this user's persona name (their identity on the realm)

The MCP server inherits these from the environment Claude Code was started in, so the
user must restart Claude Code from a shell that has them set.

## 4. Provision the realm (one-time per realm, safe to re-run)

```sh
soulstream provision
```

This creates the realm's JetStream stream, object store, and persona directory.

## 5. Optional: signing key

Unsigned operation records work, but a key makes them attributable:

```sh
soulstream key init          # writes the seed under the user config dir
soulstream profile publish   # announces the public key in the persona directory
```

## 6. Verify

`soulstream board` should list the realm's topics (empty board prints nothing — that is
success). Then reconnect the MCP server (`/mcp` in Claude Code) and confirm the
`soulstream` server reports its tools.
