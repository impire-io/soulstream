# Contract: `realm` package

Connection + provisioning. **The only NATS-touching package.**

## Connection

```go
package realm

// Config is the required input to construct a Client.
type Config struct {
    ContextName string // named NATS context to connect from (FR-001)
    Realm       string // realm name; validated as a slug; bound into canonical records (FR-028)
    Persona     string // optional; when set, write-side attribution is enforced against it
}

// Client wraps a live NATS connection + JetStream handle for one realm.
type Client struct { /* nc *nats.Conn; js jetstream.JetStream; cfg Config */ }

// Connect validates cfg.Realm (and cfg.Persona if set), then connects via natscontext and builds a
// JetStream handle. It fails fast — before mutating anything — when the context is missing, the
// server is unreachable, or JetStream is unavailable (FR-002).
func Connect(ctx context.Context, cfg Config) (*Client, error)

func (c *Client) Close() error
func (c *Client) Realm() string
```

## Provisioning

```go
// Provision brings the realm to the mandated shape: create-or-report, never modify in place
// (FR-006/007/008). It creates only missing artefacts and reports conformance of existing ones.
// It succeeds even when an artefact is nonconformant (the report is informational); it fails only
// on connection/JetStream errors.
func (c *Client) Provision(ctx context.Context) (*ProvisionReport, error)

// ProvisionOn is the decoupled form used by tests: provision against any JetStream handle (e.g. one
// built from a direct in-process connection) without needing a configured context.
func ProvisionOn(ctx context.Context, js jetstream.JetStream) (*ProvisionReport, error)
```

## Report

```go
type Artefact string
const ( ArtefactStream Artefact = "stream"; ArtefactObjectStore Artefact = "object_store" )

type Outcome string
const (
    OutcomeCreated      Outcome = "created"      // was missing, now created
    OutcomeConformant   Outcome = "conformant"   // already present and correct
    OutcomeNonconformant Outcome = "nonconformant" // present but drifts (never mutated)
)

type ArtefactResult struct {
    Artefact        Artefact
    Outcome         Outcome
    Nonconformities []string // one per specific drift; present only when nonconformant
}

type ProvisionReport struct { Results []ArtefactResult }

func (r *ProvisionReport) Conformant() bool // true iff no result is nonconformant
```

## Mandated realm shape (`spec.go`, non-configurable)

- Stream `SOULSTREAM`: subjects `["SOULSTREAM.>"]`, `LimitsPolicy`, `MaxAge == 0`, `AllowRollup ==
  true`, `Duplicates >= 2*time.Minute`, `FileStorage`.
- Object store bucket `soulstream-objects`.

## Conformance checks (`conformance.go`)

Compare an existing stream's `CachedInfo().Config` to the mandated shape; each mismatch yields one
nonconformity string. History-risking drift is explicitly among them and **never** auto-fixed:
- `MaxAge != 0` → `"MaxAge is set (age-based expiry present)"`
- `!AllowRollup` → `"AllowRollup (allow_rollup_hdrs) is disabled"`
- `Retention != LimitsPolicy` → `"retention is not Limits"`
- `Storage != FileStorage` → `"storage is not File"`
- `Duplicates < 2*time.Minute` → `"duplicate_window is below 2m"`
- subjects not exactly `["SOULSTREAM.>"]` → `"subjects do not capture SOULSTREAM.>"`

## Contract guarantees (map to spec)

- **Fail fast, no partial mutation** (FR-002): `Connect` validates and connects before any write;
  missing context errors without server contact.
- **Idempotent, create-only** (FR-006/007): second run makes zero changes; missing parts are
  created, present parts untouched.
- **Never modify in place** (FR-008): no `UpdateStream`/`CreateOrUpdateStream`; drift is reported.
- **Structured report** (FR-009): per-artefact outcome + specific nonconformities.
- **Realm name from config** (FR-028): `Config.Realm` is validated and is the only source.
