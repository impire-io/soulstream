module github.com/impire-io/soulstream/node

go 1.26.2

// The node always builds from a repo checkout (it is not `go install @latest`-able):
// it must see the soulstream packages of the SAME change-set — the public mcpserver
// surface lands together with the node that first embeds it.
replace github.com/impire-io/soulstream => ../

require (
	github.com/impire-io/soulidentity v0.0.0-20260802073850-5eaf52cf9c8d
	github.com/impire-io/soulstream v0.0.0-00010101000000-000000000000
	github.com/modelcontextprotocol/go-sdk v1.6.1
	github.com/nats-io/nats.go v1.52.0
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gowebpki/jcs v1.0.1 // indirect
	github.com/klauspost/compress v1.19.1 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/synadia-io/orbit.go/natscontext v0.1.3 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)
