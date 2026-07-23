# Library Contract Deltas: 014-persona-accountability

Only changed/new surfaces are listed. Everything else is byte-for-byte 013 behaviour.

## identity (pure, no NATS)

```go
// AttestationBytes is the domain-separated statement an operator signs to vouch for
// an operated persona. operatedKeyB64 may be "" (operated persona has no key yet).
func AttestationBytes(operator, operated, operatedKeyB64 string) []byte
```

## registry

```go
// REMOVED: KindHuman, KindAgent, KindService; Profile.Kind.

type OperatorAttestation struct {
    OperatedKey string `json:"operated_key,omitempty"`
    Sig         string `json:"sig"`
}

type Profile struct {
    // ... as before, minus Kind, plus:
    OperatorAttestation *OperatorAttestation `json:"operator_attestation,omitempty"`
}

// Validate additionally rejects: operated_by == name; attestation without operated_by;
// malformed attestation sig/key encodings.

// Strict decoding: every profile read path rejects unknown JSON fields, error names
// the persona and the field. Lookup/Publish/Rotate fail loudly; All skips + warns:
type ProfileWarning struct{ Persona string; Err error }
func All(ctx, c) ([]Profile, []ProfileWarning, error)      // was ([]Profile, error)

// Attestation token (transport form).
func NewAttestationToken(signer *identity.SigningKey, operator, operated, operatedKeyB64 string) (string, error)
type AttestationToken struct{ Operator, Operated, OperatedKey, Sig string }
func ParseAttestationToken(s string) (AttestationToken, error)

// Claim status. Statuses: "attested", "unverified", "failed"; "" when no claim.
func AttestationStatus(p Profile, operatorChain []string, operatorDistrusted bool, operatedChain []string) string

// Operator-chain walk (pure, cycle-guarded).
type ChainTerminal string // "principal" | "dangling" | "cycle"
func OperatorChain(profiles map[string]Profile, start string) ([]string, ChainTerminal)
```

## realm

```go
const (
    StreamSubject      = "SOULSTREAM.TOPICS.>"          // was "SOULSTREAM.>"
    NotifyStreamName   = "SOULSTREAM_NOTIFY"
    NotifyStreamSubject = "SOULSTREAM.PERSONA.NOTIFY.>"
    InboxWindow        = 100
    NotifyMaxBytes     = 64 << 20
)

// Provision: creates BOTH streams when missing; recognises the exact legacy shape
// (subjects ["SOULSTREAM.>"]) and converges it (narrow subjects → create notify
// stream → migrate newest ≤InboxWindow notifies per persona verbatim → purge
// PERSONA.>/SVC.> residue), reporting OutcomeUpdated. Any other drift: report only.
const OutcomeUpdated Outcome = "updated"
```

## topic

`FetchInbox` / `FollowInbox` read the `SOULSTREAM_NOTIFY` stream (subjects unchanged;
publishing unchanged; canonical binding unchanged). Discovery unchanged in code; its
traffic is simply no longer captured by any stream (the malformed-reply filter stays).

## CLI (cmd/soulstream)

- `profile publish`: `--kind` REMOVED. New `--attestation <token>` (requires
  `--operated-by`; token's operator/operated must match).
- `profile attest <operated-persona>`: NEW. Requires this persona's signing key.
  Prints the portable token (base64 JSON). Reads the operated persona's current key
  from the directory (absent profile/key → binds "").
- `profile show <persona>`: prints `operated by` with claim status
  (`attested`/`unverified`/`FAILED — …`), and the operator chain to its terminal
  (`principal`/`dangling`/`cycle`). No `kind` line. Warns on stderr for profiles
  skipped by strict decode wherever `All` is consulted.

## MCP (cmd/soulstream-mcp)

- `soulstream_publish_profile` input: `kind` REMOVED; `attestation` (token string)
  ADDED alongside existing `operated_by`. No new tools (count stays 21).

## Wire / storage compatibility

- Notify records: identical bytes, identical subjects; only the capturing stream
  changes. Signatures verify across the migration (binding derives from the subject).
- Profiles: NOT backwards compatible by design — old documents carrying `kind` are
  invalid until republished.
- `checksums`/release: ships as v0.3.0; plugin + marketplace bump to 0.3.0 in the
  same delivery (wrapper downloads by its own version).
