package cli

import (
	"fmt"
	"io"

	"github.com/impire/soulstream/identity"
	"github.com/impire/soulstream/internal/keystore"
)

// keyFilePath resolves the seed-file location for cfg: --key-file flag, then
// SOULSTREAM_KEY_FILE, then the per-realm default under the user config dir.
func keyFilePath(cfg Config) (string, error) {
	return keystore.ResolveKeyFile(cfg.KeyFile, cfg.Realm, cfg.Persona)
}

// loadSigner loads the persona's signing key for cfg, or nil when no key file
// exists (publish unsigned). Both the production connector and tests use it, so
// signing behaves identically everywhere.
func loadSigner(cfg Config) (*identity.SigningKey, error) {
	path, err := keyFilePath(cfg)
	if err != nil {
		return nil, err
	}
	return keystore.LoadKey(path)
}

// cmdKey manages the persona's signing identity. It never connects: keys are
// client-side state.
func cmdKey(cfg Config, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "soulstream: usage: key init|show")
		return 2
	}
	if cfg.Persona == "" {
		fmt.Fprintln(stderr, "soulstream: key commands require a persona (--persona or SOULSTREAM_PERSONA)")
		return 2
	}
	if cfg.Realm == "" {
		fmt.Fprintln(stderr, "soulstream: key commands require a realm (--realm or SOULSTREAM_REALM)")
		return 2
	}

	switch args[0] {
	case "init":
		return keyInit(cfg, stdout, stderr)
	case "show":
		return keyShow(cfg, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "soulstream: unknown key subcommand %q (want init|show)\n", args[0])
		return 2
	}
}

func keyInit(cfg Config, stdout, stderr io.Writer) int {
	path, err := keyFilePath(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "soulstream: %v\n", err)
		return 2
	}
	key, err := identity.GenerateSigningKey()
	if err != nil {
		fmt.Fprintf(stderr, "soulstream: %v\n", err)
		return 2
	}
	if err := keystore.SaveKey(path, key); err != nil {
		fmt.Fprintf(stderr, "soulstream: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "generated signing key for persona %q\n", cfg.Persona)
	fmt.Fprintf(stdout, "public key: %s (ed25519)\n", key.PublicKey())
	fmt.Fprintf(stdout, "seed file:  %s\n", path)
	return 0
}

func keyShow(cfg Config, stdout, stderr io.Writer) int {
	path, err := keyFilePath(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "soulstream: %v\n", err)
		return 2
	}
	key, err := keystore.LoadKey(path)
	if err != nil {
		fmt.Fprintf(stderr, "soulstream: %v\n", err)
		return 2
	}
	if key == nil {
		fmt.Fprintf(stderr, "soulstream: no signing key for %q (looked at %s); run: key init\n", cfg.Persona, path)
		return 2
	}
	fmt.Fprintf(stdout, "public key: %s (ed25519)\n", key.PublicKey())
	return 0
}
