package realm

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/synadia-io/orbit.go/natscontext"

	"github.com/impire/soulstream/identity"
)

// Config is the required input to construct a [Client].
type Config struct {
	// ContextName is the named NATS context to connect from (empty uses the selected
	// context).
	ContextName string
	// Realm is the realm name; validated as a slug and bound into canonical records.
	Realm string
	// Persona is optional; when set, write-side attribution is enforced against it.
	Persona string
}

// Client wraps a live NATS connection and JetStream handle for one realm.
type Client struct {
	nc  *nats.Conn
	js  jetstream.JetStream
	cfg Config
}

// Connect validates cfg, connects via the named NATS context, and builds a JetStream
// handle. It fails fast — before touching any realm artefact — when a name is
// invalid, the context is missing, the server is unreachable, or JetStream is
// unavailable.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	if err := identity.CheckName(cfg.Realm); err != nil {
		return nil, fmt.Errorf("realm: invalid realm name: %w", err)
	}
	if cfg.Persona != "" {
		if err := identity.CheckName(cfg.Persona); err != nil {
			return nil, fmt.Errorf("realm: invalid persona name: %w", err)
		}
	}

	nc, _, err := natscontext.Connect(cfg.ContextName, nats.Name("soulstream/"+cfg.Realm))
	if err != nil {
		return nil, fmt.Errorf("realm: connect via context %q: %w", cfg.ContextName, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("realm: initialise jetstream: %w", err)
	}

	// Fail fast if JetStream is not actually available on the server.
	if _, err := js.AccountInfo(ctx); err != nil {
		nc.Close()
		return nil, fmt.Errorf("realm: jetstream unavailable: %w", err)
	}

	return &Client{nc: nc, js: js, cfg: cfg}, nil
}

// Provision brings this client's realm to the mandated shape (see [ProvisionOn]).
func (c *Client) Provision(ctx context.Context) (*ProvisionReport, error) {
	return ProvisionOn(ctx, c.js)
}

// EnforceAuthor is the write-side attribution guard. When the client is persona-bound
// (Config.Persona set), it returns an error unless author is that persona, so the
// client can only ever publish as itself. A read-only client (no persona) permits any
// author — enforcing attribution is not its job. Publish paths in later features call
// this before sending.
func (c *Client) EnforceAuthor(author string) error {
	if c.cfg.Persona == "" {
		return nil
	}
	return identity.EnforceAuthor(c.cfg.Persona, author)
}

// JetStream returns the client's JetStream handle, so higher-level engines (such as the
// topic package) can build on the realm without re-connecting.
func (c *Client) JetStream() jetstream.JetStream { return c.js }

// Realm returns the client's realm name.
func (c *Client) Realm() string { return c.cfg.Realm }

// Persona returns the client's configured persona (empty if read-only).
func (c *Client) Persona() string { return c.cfg.Persona }

// Close releases the underlying NATS connection.
func (c *Client) Close() error {
	c.nc.Close()
	return nil
}
