# The `soulstream` command: your remote control

`soulstream` is the **remote control** for Soulstream — one word per action. You tell it
who you are and which realm once, then drive everything from the terminal.

## Point it at a realm

The remote needs to know three things: which server (a saved **NATS context**), which
**realm**, and which **persona** you're acting as. Set them once as environment
variables (or pass `--context/--realm/--persona` on any command):

```sh
export SOULSTREAM_CONTEXT=soulstream   # a context you made with: nats context add
export SOULSTREAM_REALM=acme
export SOULSTREAM_PERSONA=daan
```

Reading things (`board`, `show`, `get`) doesn't need a persona; *writing* things does.

## The buttons

```sh
soulstream provision                 # get the workshop ready (safe to re-run)
soulstream board                     # what topics exist?
soulstream start "Q2 VAT filing"     # begin a topic → prints its path
soulstream show   <path>             # what's happening in this topic?
soulstream post   <path> "hi @bookkeeper-agent, box 5?"   # say something (@ pings people)
soulstream comment <path> <op-id> "…"     # reply to a specific line
soulstream attach <path> ./file.csv  # clip a file on → prints its object key
soulstream get    <object> out.csv   # pull that file back out
soulstream close  <path>             # mark it finished
soulstream watch  <path>             # watch it update live (Ctrl-C to stop)
soulstream inbox                     # watch for @mentions of you (Ctrl-C to stop)
```

Every command says what it did and exits **0** when it worked; on trouble it prints a
clear message and exits non-zero. `board` and `show` take `--json` if you're scripting.

## Tips

- Flags can go before or after the positional arguments — `show <path> --json` and
  `show --json <path>` both work.
- `get` won't overwrite a file unless you pass `--force` — no accidental clobbering.
- Posting to a closed topic still works but prints a gentle "this is closed" warning.

## Related

- [The topic](./topic.md) · [Mentions](./mentions.md) · [Attachments](./attachments.md)
