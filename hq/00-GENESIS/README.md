# 00-GENESIS — why Soulstream exists and how it decides

This folder is the fixed point every decision is held against. It changes
rarely, deliberately, and always with a journey episode recording why.

| File | Role |
|---|---|
| [`vision.md`](vision.md) | What Soulstream is, who it's for, where it's pointed, and what it refuses to become |
| [`constitution.md`](constitution.md) | The testable articles: principles no work is allowed to violate, plus the anti-drift working agreement. Canonical copy — spec-kit's Constitution Check reads it through the `.specify/memory/constitution.md` symlink |
| [`how-we-work.md`](how-we-work.md) | The process: pipeline state machine, research lifecycle, quality gates, documentation duties, and the anti-drift working agreement in daily terms |
| [`rationale.md`](rationale.md) | How we got here — the reasons behind every non-obvious call. Not normative; it exists so future changes argue against the real reasons, not guesses |

## The decision test

When a choice comes up — a new direction, a shortcut, a scope change — run it
through, in order:

1. **Vision**: does it serve what [`vision.md`](vision.md) says Soulstream is
   for? If it serves something else (a bigger platform, a convenient coordinator,
   a special door for one client), say so out loud.
2. **Constitution**: does it violate an article? Articles don't bend for product
   work. The load-bearing question is usually the first two: does this stay
   NATS-native (I), and is it the smallest thing that satisfies the spec (II)?
   If one genuinely must change, that's an amendment with a version bump and a
   journey episode, never a quiet exception.
3. **Working agreement**: if the decision is load-bearing, it does not get
   recorded until it survives teach-back, carries its evidence class, names its
   reversal condition, and (for changes to the protocol's shape or a core
   boundary) has had the other side argued at full strength. See
   [`how-we-work.md`](how-we-work.md).

If the test doesn't produce a clear answer, the decision waits for the human.
