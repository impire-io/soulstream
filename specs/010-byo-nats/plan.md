# Implementation Plan: BYO NATS (010)

## Summary

Founding against a server soulstream does not run, per design 0003. The
ceremony's steps regroup: **local material** (signing keys, issuer-user key,
curve keys — generated here, seeds never crossing the boundary on
self-hosted), the **account half** (the kit applied by the operator, or the
Synadia control-plane driver), and the **wire half** (unchanged: provision,
vault imports, first token, sentinel last). `node.Start` grows one branch:
embedded off, `n.url` = the substrate URL — the single seam every plane
already reads.

## Constitution Check

- **I — Composition, not invention**: PASS. No new wire vocabulary; upstream
  surfaces consumed unchanged (`embed.Run`, `realm.ProvisionOn`, the client).
  The Synadia driver is founding tooling graduated from `soulstream-mcp`'s
  `byon-setup` (spec 018 Q2 named it best-effort; design 0003 §5 graduates
  it), talking to an external API, not a plane.
- **II — Same shape as any deployment**: PASS by construction — BYO *is* the
  hosted deployment shape; admission stays operator mode + callout,
  byte-identical (design 0003 §1).
- **III — One process, planes by configuration**: PASS. BYO is configuration:
  `byo` block present ⇒ embedded server off, every plane's ordinary NATS
  connection dials the substrate URL. No in-process transport anywhere.
- **V — No manual key step in the ceremony**: PASS with the design's reading:
  the account half on a self-hosted substrate is the substrate operator's own
  domain (the DBA job, design 0003 §7); soulstream's half stays zero-manual.

## Project Structure

```
ceremony/
  byo.go            # GenerateBYO (local material per flavour), MintBYOUsers
                    # (signing-key-signed creds, IssuerAccount set), flavour consts
  kit.go            # Kit(*State) string — the generated nsc commands, config
                    # fragments, push, and hand-back; exact values only
  state.go          # byo config block; BYO branches in files()/Save/Load/Verify;
                    # ArtifactCount becomes a State method; issuer-user seed path
  byo_test.go       # generate/save/load/verify round-trip, kit content, refusals
node/
  node.go           # Start: BYO branch (no bind pre-flight, no server,
                    # n.url = substrate URL); Stop guards nil srv
  byo.go            # ProbeSubstrate (anonymous-connect refusal, ops/issuer
                    # probes, whoami, JetStream) + SmokeAdmission (sentinel+token
                    # admits, garbage refuses) — each failure names its kit item
  byo_test.go       # the rig plays the operator: external operator-mode server
                    # from a config file, kit publics → account JWTs, full
                    # founding, M1.1 semantics, custody audit, named refusals
internal/synadia/
  synadia.go        # the control-plane driver: ensure accounts, sk groups
                    # (programmatic + on-demand), callout, issuer user; ported
                    # from soulstream-mcp cmd/byon-setup; idempotent by name
  synadia_test.go   # httptest stub asserting sequence, idempotence, seed capture
cmd/soulstream/
  main.go           # init: --byo/--url/--auth-account/--realm-account/
                    # --synadia-system; the two-phase self-hosted flow; up and
                    # workload read the substrate URL; BYO-aware prose
  main_test.go      # phase-1 emission, idempotent re-emission, flag refusals
```

## Decisions taken here (propagated back to design 0003 on landing)

- The config block gains `url` (the design sketch omitted the substrate URL);
  `listen` and `byo` are mutually exclusive, refused by name together.
- The issuer-user seed is a persisted BYO artifact (`keys/issuer-user.nk`):
  its public key must be in the kit before its creds can exist, so it
  outlives phase 1 on self-hosted. Synadia's issuer creds are downloaded from
  the platform instead (on-demand group).
- Callout sealing on Synadia: `CalloutKey` stays empty unless the platform
  yields the xkey seed — unsealed callout, said out loud at founding
  (embed.Options already treats the key as optional).
