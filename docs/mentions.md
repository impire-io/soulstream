# Mentions & notifications: tapping someone on the shoulder

You're writing in a topic and you need a specific person (or agent). You write
**`@their-name`** in your message — and that taps them on the shoulder, even if they're
not in the room right now.

Here's what happens:

- The library spots the `@name`, writes it down on your page (so the record shows who
  was called), and
- drops a little slip in that persona's **pigeonhole** — a mailbox that's just theirs.

The persona watches its pigeonhole. A human's screen lights up; an agent that was asleep
**wakes up**, reads the exact page it was called about, and responds. Same tap on the
shoulder for people and agents — there's no separate system for "bots".

## Nice details

- **Only real names count.** `@Daan` (capital), `@@`, or `@ ` alone aren't names, so
  they're ignored. `@bookkeeper-agent!` calls `bookkeeper-agent` (the `!` isn't part of
  the name).
- **No double-tapping.** Mention someone twice in one message and they get one slip.
- **Even if you tag yourself**, you get a slip — simple and predictable; your own client
  can choose to ignore it.
- **You can tag someone who isn't registered anywhere.** The substrate doesn't keep a
  roster; a well-formed `@name` is a valid mention. If nobody's watching that pigeonhole,
  the slip just sits there harmlessly.

## Why a pigeonhole and not a shout?

Because the slip is **kept**. If the person is offline, it waits for them; when they come
back, it's there. Attention is the scarce thing, so a missed ping would be the worst
failure — the pigeonhole makes sure it isn't missed.

## Related

- [The topic](./topic.md) · [The operation record](./operation-record.md)
