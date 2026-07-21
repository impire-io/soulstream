package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/impire/soulstream/identity"
	"github.com/impire/soulstream/internal/keystore"
	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/registry"
	"github.com/impire/soulstream/topic"
)

// Connector builds a realm client from config. It is injectable so tests can supply a
// client bound to an in-process server instead of a named NATS context.
type Connector func(ctx context.Context, cfg Config) (*realm.Client, error)

// realmConnect is the production connector: it dials the named NATS context, and
// signs published ops when the persona's key file exists.
func realmConnect(ctx context.Context, cfg Config) (*realm.Client, error) {
	signer, err := loadSigner(cfg)
	if err != nil {
		return nil, err
	}
	return realm.Connect(ctx, realm.Config{
		ContextName: cfg.Context,
		Realm:       cfg.Realm,
		Persona:     cfg.Persona,
		Signer:      signer,
	})
}

// withClient connects, enforces a persona for write commands, runs fn, and maps the
// outcome to an exit code: 0 on success, 2 on error (with a message on stderr).
func withClient(ctx context.Context, connect Connector, cfg Config, requirePersona bool, stderr io.Writer, fn func(*realm.Client) error) int {
	if requirePersona && cfg.Persona == "" {
		fmt.Fprintln(stderr, "soulstream: this command requires a persona (--persona or SOULSTREAM_PERSONA)")
		return 2
	}
	c, err := connect(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "soulstream: %v\n", err)
		return 2
	}
	defer func() { _ = c.Close() }()

	if err := fn(c); err != nil {
		fmt.Fprintf(stderr, "soulstream: %v\n", err)
		return 2
	}
	return 0
}

// openAndMaterialise opens a handle and materialises it so posts parent onto the current
// tip and closed-topic warnings fire.
func openAndMaterialise(ctx context.Context, c *realm.Client, path string) (*topic.Handle, error) {
	h := topic.Open(c, path)
	if _, err := h.Materialise(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

// realmKeyring builds the reader's keyring: pins + directory → keyring, persisting
// extended pins. A realm without a directory and without pins yields nil (signed ops
// degrade to unknown-key). Errors also degrade to nil — verification state must never
// break reading.
func realmKeyring(ctx context.Context, c *realm.Client, cfg Config) *identity.Keyring {
	pinsPath, err := keystore.ResolvePinsFile(cfg.PinsFile, cfg.Realm)
	if err != nil {
		return nil
	}
	pins, err := keystore.LoadPins(pinsPath, cfg.Realm)
	if err != nil {
		return nil
	}
	profiles, err := registry.All(ctx, c)
	if err != nil {
		return nil
	}
	if len(profiles) == 0 && len(pins.Personas) == 0 {
		return nil
	}

	kr, newPins := registry.BuildKeyring(profiles, pins.Personas)
	pins.Personas = newPins
	_ = keystore.SavePins(pinsPath, pins) // best effort; reading must not fail on pin I/O
	return kr
}
