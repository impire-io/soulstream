package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/nats-io/nats.go"

	siclient "github.com/impire-io/soulstream-identity/client"

	"github.com/impire-io/soulstream/ceremony"
)

// providerKeyEnv is where the value arrives. Never a flag: a flag is
// visible to every process on the machine, and this is the one secret in
// the whole deployment a provider would charge somebody for.
const providerKeyEnv = "SOULSTREAM_PROVIDER_KEY"

// cmdProvider loads a provider credential into the inference plane's own
// custody (identity D36). The store is caller-own by construction —
// every secret op reaches only the calling principal's tree — so writing
// the plane's tree means BEING the plane: this verb mints the same
// runtime identity the plane connects with, which the house can do
// because it holds the realm's minting key and nobody outside it does.
//
// That is the custody story in one act: the value exists in the sealed
// store and in the serving instance's memory, and in no third place. No
// agent scope reaches an identity-plane subject at all, so no declared
// agent can read it however it is worded.
func cmdProvider(args []string, out, errw io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("provider needs a subcommand: set <name>")
	}
	if args[0] != "set" {
		return fmt.Errorf("unknown provider subcommand %q: set", args[0])
	}
	fs := flag.NewFlagSet("provider set", flag.ContinueOnError)
	fs.SetOutput(errw)
	stateFlag := fs.String("state", "", "state directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("provider set needs exactly one provider name")
	}
	name := fs.Arg(0)
	value := os.Getenv(providerKeyEnv)
	if value == "" {
		return fmt.Errorf("no key in the environment — set %s to the provider credential (a flag would show it to every process on this machine)", providerKeyEnv)
	}

	_, st, url, err := realmState(*stateFlag)
	if err != nil {
		return err
	}
	token, seed, err := st.MintPlaneUser(ceremony.InferencePersona)
	if err != nil {
		return err
	}
	nc, err := nats.Connect(url,
		nats.UserJWTAndSeed(token, string(seed)),
		nats.Name("soulstream-provider"))
	if err != nil {
		return fmt.Errorf("connect to the running realm at %s (is `soulstream up` running?): %w", url, err)
	}
	defer nc.Close()

	ident := siclient.New(nc, st.RealmPub, ceremony.InferencePersona)
	path := "providers/" + name
	// Conditional on the revision the store holds, so two operators
	// racing on one path is a refusal rather than a lost write.
	var rev uint64
	if held, err := ident.SecretGet(path); err == nil {
		rev = held.Rev
	}
	if _, err := ident.SecretPut(path, []byte(value), rev); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Fprintf(out, "provider %q loaded into the inference plane's custody at %s\n", name, path)
	fmt.Fprintf(out, "point an instance at it: planes.inference.instances[].secret = %q\n", path)
	return nil
}
