package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/impire-io/soulstream/internal/keystore"
	"github.com/impire-io/soulstream/realm"
	"github.com/impire-io/soulstream/registry"
)

// cmdProfile manages directory profiles: publish (create-or-metadata-update for the
// session persona) and show (any persona's profile, chain, and pin state).
func cmdProfile(ctx context.Context, connect Connector, cfg Config, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "soulstream: usage: profile publish|show")
		return 2
	}
	switch args[0] {
	case "publish":
		return profilePublish(ctx, connect, cfg, args[1:], stdout, stderr)
	case "show":
		return profileShow(ctx, connect, cfg, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "soulstream: unknown profile subcommand %q (want publish|show)\n", args[0])
		return 2
	}
}

func profilePublish(ctx context.Context, connect Connector, cfg Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("profile publish", flag.ContinueOnError)
	fs.SetOutput(stderr)
	displayName := fs.String("display-name", "", "presentation name")
	kind := fs.String("kind", registry.KindHuman, "human|agent|service (presentation only)")
	description := fs.String("description", "", "one-line description")
	operatedBy := fs.String("operated-by", "", "persona accountable for an agent")
	if err := parseInterspersed(fs, args); err != nil {
		return 2
	}

	return withClient(ctx, connect, cfg, true, stderr, func(c *realm.Client) error {
		p := registry.Profile{
			Name:        cfg.Persona,
			DisplayName: *displayName,
			Kind:        *kind,
			Description: *description,
			OperatedBy:  *operatedBy,
			CreatedAt:   time.Now().UTC(),
		}
		// Include the public key when this persona has one; the stored key material
		// stays authoritative on metadata updates.
		signer, err := loadSigner(cfg)
		if err != nil {
			return err
		}
		if signer != nil {
			p.SigningKey = &registry.SigningKeyInfo{Ed25519: signer.PublicKey(), Since: time.Now().UTC()}
		}
		if err := registry.Publish(ctx, c, p); err != nil {
			return err
		}
		if signer != nil {
			fmt.Fprintf(stdout, "published profile for %q (key %s)\n", cfg.Persona, signer.PublicKey())
		} else {
			fmt.Fprintf(stdout, "published profile for %q (no signing key)\n", cfg.Persona)
		}
		return nil
	})
}

func profileShow(ctx context.Context, connect Connector, cfg Config, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "usage: soulstream profile show <persona>")
		return 2
	}
	persona := args[0]

	return withClient(ctx, connect, cfg, false, stderr, func(c *realm.Client) error {
		p, ok, err := registry.Lookup(ctx, c, persona)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("no directory profile for %q", persona)
		}

		fmt.Fprintf(stdout, "name:         %s\n", p.Name)
		if p.DisplayName != "" {
			fmt.Fprintf(stdout, "display name: %s\n", p.DisplayName)
		}
		fmt.Fprintf(stdout, "kind:         %s\n", p.Kind)
		if p.Description != "" {
			fmt.Fprintf(stdout, "description:  %s\n", p.Description)
		}
		if p.OperatedBy != "" {
			fmt.Fprintf(stdout, "operated by:  %s\n", p.OperatedBy)
		}

		chain, chainErr := registry.Chain(p)
		switch {
		case chainErr != nil:
			fmt.Fprintf(stdout, "key chain:    INVALID — %v\n", chainErr)
		case len(chain) == 0:
			fmt.Fprintln(stdout, "key chain:    (no signing key published)")
		default:
			fmt.Fprintln(stdout, "key chain:")
			for i, k := range chain {
				marker := ""
				if i == len(chain)-1 {
					marker = "  (current)"
				}
				fmt.Fprintf(stdout, "  %d. %s%s\n", i+1, k, marker)
			}
		}

		fmt.Fprintf(stdout, "pin state:    %s\n", pinState(cfg, persona, chain, chainErr))
		return nil
	})
}

// pinState compares a persona's published chain with this client's pin.
func pinState(cfg Config, persona string, chain []string, chainErr error) string {
	pinsPath, err := keystore.ResolvePinsFile(cfg.PinsFile, cfg.Realm)
	if err != nil {
		return fmt.Sprintf("unknown (%v)", err)
	}
	pins, err := keystore.LoadPins(pinsPath, cfg.Realm)
	if err != nil {
		return fmt.Sprintf("unknown (%v)", err)
	}
	pinned := pins.Personas[persona]

	switch {
	case chainErr != nil:
		return "DISTRUSTED — published chain is invalid"
	case len(pinned) == 0:
		return "not pinned yet (will pin on next read)"
	case isChainPrefix(pinned, chain):
		if len(pinned) == len(chain) {
			return "pinned (matches)"
		}
		return "pinned (published chain extends the pin; will re-pin on next read)"
	default:
		return "DISTRUSTED — possible key substitution (published chain does not extend the pin)"
	}
}

// isChainPrefix mirrors the registry's pin rule for display.
func isChainPrefix(pin, chain []string) bool {
	if len(pin) > len(chain) {
		return false
	}
	for i := range pin {
		if pin[i] != chain[i] {
			return false
		}
	}
	return true
}
