# Quickstart: 011-vocab — conversation upkeep, withdrawal, napping topics

Personas: `daan` and `scribe`. Topic: `gadget-plan`.

## 1. Second thoughts, same conversation

```sh
soulstream --persona daan post gadget-plan "lets ship thursdy"
# → op 1111-...
soulstream --persona daan edit gadget-plan 1111-... "let's ship Thursday"
# → readers now render the corrected text, marked (edited)

soulstream --persona scribe comment gadget-plan 1111-... "which Thursday?"
# → op 2222-...
soulstream --persona daan reply gadget-plan 2222-... "the 30th"
soulstream --persona daan resolve gadget-plan 2222-...

soulstream show gadget-plan
#   [1111]✓ daan (turn, edited by daan): let's ship Thursday
#   [2222]✓ scribe (comment -> 1111, resolved by daan): which Thursday?
#   [3333]✓ daan (reply -> 2222): the 30th

# scribe tries to rewrite daan's words:
soulstream --persona scribe edit gadget-plan 1111-... "let's ship Friday"
# → the op lands, the view warns, nothing changes — your words are yours
```

## 2. Withdrawing a file

```sh
soulstream --persona daan attach gadget-plan ./specs.pdf
soulstream --persona daan revise gadget-plan ./specs-v2.pdf --of specs.pdf
soulstream --persona daan detach gadget-plan <v2-op-id>
# → artefact tip falls back to v1; v2 shows "removed by daan", bytes still fetchable

soulstream archive gadget-plan
# → final baseline lands, THEN the withdrawn blob is deleted; v1's bytes remain
```

## 3. Napping topics, lapsed claims

```sh
# any persona may apply the idle rule by hand:
soulstream mark-dormant quiet-topic --idle 336h
# → "marked dormant" — or "not idle (newest op 3d ago, window 14d)" and no op

soulstream post quiet-topic "picking this back up"
# → active again, no ceremony

# or let a curator do the housekeeping (both sweeps are opt-in):
soulstream curate --mark-dormant --reclaim 168h
# → marks eligible topics dormant; abandons claims idle past 7 days
#   (the abandoned item reopens; the next claim wins fresh, 010 rules)
```

## MCP mirror

`soulstream_reply_comment`, `soulstream_resolve_comment`, `soulstream_edit`
(own contributions only) — plus `soulstream_show_topic`, whose output now carries
`edits`, `resolved`/`resolved_by`, `removed`/`removed_by`, and the `dormant`
lifecycle.
