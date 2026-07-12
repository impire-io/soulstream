# Quickstart: Foundation

*What you can do once this feature is built. Two things: set up a realm, and build/read an
operation record.*

## Prerequisites

- Go 1.26+.
- A NATS server (2.12+) with JetStream, and a named context that points at it:
  ```sh
  nats context add soulstream --server nats://127.0.0.1:4222
  # add --creds / --user / --password as your server requires
  ```

## 1. Provision a realm

```go
import (
    "context"
    "fmt"

    "github.com/impire/soulstream/realm"
)

func main() {
    ctx := context.Background()

    client, err := realm.Connect(ctx, realm.Config{
        ContextName: "soulstream", // the named nats context
        Realm:       "acme",       // your realm name — bound into every canonical record
        Persona:     "daan",       // optional; only lets this client publish as "daan"
    })
    if err != nil {
        panic(err) // missing context / unreachable server / no JetStream — all fail here, fast
    }
    defer client.Close()

    report, err := client.Provision(ctx)
    if err != nil {
        panic(err)
    }
    for _, r := range report.Results {
        fmt.Printf("%-13s %s %v\n", r.Artefact, r.Outcome, r.Nonconformities)
    }
    // First run:  stream created / object_store created
    // Second run: stream conformant / object_store conformant   (zero changes)
}
```

If someone previously created the stream with age-based expiry, you'll see:
`stream nonconformant [MaxAge is set (age-based expiry present)]` — and the library will **not**
silently "fix" it, because changing that setting could destroy history. Reconfiguring is your
explicit call.

## 2. Build an operation record and read it back

```go
import (
    "time"

    "github.com/impire/soulstream/record"
)

r := record.Record{
    ID:        record.NewID(),          // fresh UUIDv4
    Author:    "daan",
    Parents:   nil,                      // no parents -> no Soulstream-Parents header
    Type:      "turn.post",
    Timestamp: time.Now().UTC(),
    Payload:   []byte(`{"body":"hello soulstream"}`),
}

headers, payload, err := r.Build() // -> put on a NATS message: headers + payload, Nats-Msg-Id == r.ID
// ...publish/replay is a later feature; here we just round-trip...

got, err := record.Parse(headers, payload)
// got == r, field for field
```

Publishing the same `r.ID` twice inside the duplicate window is de-duplicated by the server — the
record's identity *is* its idempotency key.

## 3. Canonical form (for future signing / exhibits)

```go
canon, err := r.Canonical("acme", "vat-q2-2026-x7m2")
// canon is the deterministic RFC 8785 (JCS) byte sequence:
//   {"author":"daan","data":{"body":"hello soulstream"},"id":"...","parents":[],
//    "topic":"vat-q2-2026-x7m2","realm":"acme","ts":"...","type":"turn.post","v":1}
// (keys shown unsorted for readability; JCS emits them lexicographically sorted)
```

The same record with its Go fields supplied in any order produces byte-identical `canon`. That
stability is the whole point: whatever signs it later signs *these* bytes.

## Verify (the feature's definition of done)

```sh
make fmt    # gofmt/goimports clean
make test   # all tests pass, none skipped (record/canonical/identity need no server;
            # provisioning runs against an in-process JetStream server)
make lint   # golangci-lint clean
```
