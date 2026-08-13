# 007 — the helm plane

**Status:** Shipped (soulhelm v0.1.0 composed by tag)

The human cockpit joins the bundle: `planes.helm` runs soulhelm through
its public embed seam — observe the realm (board, topics with earned
signature verdicts, storage, plane health) and act in it, with every
session signing in against the deployment's AS and admitting through
the identity plane's OIDC callout lane as itself. Composition, not
invention: the node hands the helm its ops read lane, the public
sentinel, and the resolved sign-in issuer; the helm founds and owns
nothing.

## Config

```json
"planes": {
  "helm": { "enabled": true, "listen": "127.0.0.1:8500" }
}
```

- On by default at founding, beside the fold. **Absent block means
  disabled** — state dirs founded before the plane existed do not
  sprout one on upgrade (the fold's rule).
- `--helm-listen` on `init` (founding-only, like every listener).
- Verify refuses: a non-loopback listen; a listener collision with the
  door or the fold (except `:0`); the helm with no sign-in issuer
  (enable `planes.fold` or set `planes.door.auth_issuer`).

## The sign-in issuer (decision of record)

`State.SessionIssuer()` resolves the OIDC AS for both the identity
plane's callout lane and the helm: an explicit external AS
(`planes.door.auth_issuer`, public door mode) wins; otherwise the
bundled fold when the helm is enabled. **Enabling the helm therefore
switches the OIDC admission lane on in local mode** — before this
feature the lane existed only with public door mode. The door's own
public-mode contract (all-three-or-none) is untouched. An ephemeral
fold listener (`:0`) yields no resolvable issuer: the lane stays off
and the helm skips its surface with an audit warning, never silently.

## Acceptance

- `TestHelmPlane` (node): the plane serves; the surface is closed until
  sign-in (front page carries no session, `/live` refuses 401).
- `TestHelmDisabled` (node): the disabled arm answers no URL.
- `TestHelmWiring` (ceremony): founding defaults, absent-block rule,
  collision and issuer refusals by name.
- The full human ceremony — passkey enrolment, DCR + PKCE sign-in, the
  session's own admission, an act attributed to the fold principal,
  sign-out, the custody scan with a fired positive control, the
  offline-render gate — is soulhelm's own consumer-position e2e
  (`soulhelm.invalid/e2e`), which boots this node at its published tag.

## Named, not built

- A dedicated scoped `helm` read-lane user in the ceremony (today the
  plane hands the ops lane — operator standing, more than observe
  needs). Hardening follow-up; touches the artifact count.
- Configure classes (b)/(c) surfaces in the helm UI (soulhelm's next
  milestone); the grant lane for standalone helms rides the tenancy
  topic.
