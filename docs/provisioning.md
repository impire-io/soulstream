# What is provisioning?

**Provisioning** is setting up the empty workshop before anyone starts working: making
sure the notebook and the supply cupboard exist, and that they're the *right kind*.

You run it once, and you can safely run it again any time — on every restart, in a
setup script, whenever. It follows one simple rule:

> **If something is missing, make it. If it already exists, look at it — don't
> rearrange it.**

So:

- **Empty workshop?** Provisioning creates the notebook and the cupboard, set up
  exactly the way Soulstream needs. It tells you "created, created".
- **Already set up correctly?** Provisioning changes nothing and tells you
  "conformant, conformant". Running it ten times in a row does the same thing as
  running it once.
- **Half set up?** (Say the cupboard is there but the notebook isn't.) Provisioning
  makes only the missing part and leaves the rest alone.

## Why won't it "fix" a notebook that's set up wrong?

Say someone earlier made a notebook that **throws away old pages after a month**.
That's not how a Soulstream notebook should work — but if provisioning quietly changed
that setting, it could **delete history that's already there**. That's a door you can't
un-open.

So provisioning refuses to touch it. Instead it tells you plainly:

> `stream nonconformant [MaxAge is set to 720h0m0s (age-based expiry present)]`

Now *you* decide what to do — because you're the only one who knows whether that
history matters. Looking is safe; silently rearranging someone's notebook is not.

## Related

- [The realm](./realm.md) — the workshop that gets provisioned.
