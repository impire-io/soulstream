# Implementation Plan: The presence lease (011)

## Summary

Upstream ask #3 of hq shell design 0008, wired: the wrap announces the
agent (profile-ensure, never clobbering) and holds the presence lease
(`presence.Hold`) for as long as it answers mentions, with the
farewell written after the run loop returns and before the connection
closes. All convention logic lives in soulstream-core v0.13.0's
`presence` package; this repo adds composition only.

## Constitution Check

- **I — Composition, not invention**: PASS. The lease, its cadence,
  its farewell, and the reader semantics are `presence.Hold` /
  `presence.Lookup` upstream; the profile floor is `registry.Publish`
  upstream. This repo contributes a lookup-first guard, a goroutine,
  and a wait — wiring, no wire vocabulary.
- **II — Same shape as any deployment**: PASS. Nothing
  deployment-specific: the wrap writes through its own admitted
  connection, and the bucket is the convention's own
  created-on-first-write.
- **III — One process, planes by configuration**: untouched — no new
  plane, no new listener.
- **V — No manual key step**: PASS. The announce publishes no key at
  all: the wrap lane holds no signer, the profile floor says so
  honestly, and a richer profile stays the agent's own act.

## Project Structure

```
cmd/soulstream/
  wraplife.go        # ensureProfile (lookup-first, warn-not-fatal) +
                     # holdPresence (goroutine around presence.Hold,
                     # returning the wait that lets the farewell land)
  wrap.go            # cmdWrap: announce + lease after Connect, wait
                     # after Run returns, before the deferred Close
  wraplife_test.go   # the live rig: founded node, sentinel+token
                     # admission, both stories measured end to end
go.mod               # soulstream-core v0.12.1 → v0.13.0
```

## Decisions carried from the designs

- Farewell context: by the time the farewell is written the wrap's
  ctx is cancelled, so `presence.Hold` writes it on a fresh
  short-lived context of its own; cmdWrap's only duty is to WAIT for
  the hold goroutine before the deferred `client.Close()` runs.
- The announce is lookup-first because `registry.Publish` REPLACES
  display metadata on an existing entry — publishing
  unconditionally would let a minimal floor overwrite a rich profile.
- Lease failures never stop the wrap (advisory, courtesy-never-
  correctness): errors are log lines.
