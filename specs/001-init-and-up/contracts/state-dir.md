# Contract — the state directory

The layout, modes, and verification rules in
[data-model.md](../data-model.md) are normative. Beyond them:

1. **The directory is the realm.** Copying it (node stopped) to another
   machine and running `soulnode up` there is a supported move — nothing
   in it is host-bound. (Day-2 tooling is a later horizon; the *format*
   commitment starts now.)
2. **Owner-only or refuse.** `init` fails rather than persisting secrets
   with modes weaker than `0700`/`0600` (e.g. a filesystem that cannot
   express them).
3. **Trust roots are never regenerated in place.** No command touches
   `keys/` after the founding run; repair of a damaged directory is
   restore-from-copy, never regenerate-in-place (regeneration would
   orphan every credential and the JetStream state).
4. **`sentinel.creds` is the completion marker** — written last by the
   founding run; its distribution to clients is deliberate (it is a
   deny-all bearer artifact; tokens are the secret).
5. **Additive evolution.** New planes add files/blocks (e.g. a
   `config.json` plane block at M1.2, `archive/` for the memory plane);
   existing paths and meanings do not change without a design amendment.
