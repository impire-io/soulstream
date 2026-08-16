package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire-io/soulstream/ceremony"
)

// ProbeSubstrate verifies a BYO substrate carries the account half —
// behaviourally, refusing by name, never repairing (design 0003 §4).
// Each probe's failure names the kit item that was not applied; the
// probes read, connect, and disconnect, and mutate nothing.
func ProbeSubstrate(st *ceremony.State, stateDir string) error {
	kit := ceremony.KitPath(stateDir)
	dial := func(opts ...nats.Option) (*nats.Conn, error) {
		return nats.Connect(st.BYOURL, append(opts, nats.Timeout(5*time.Second))...)
	}

	// Operator mode rejects anonymous connections; a server that admits
	// one is running conf-file auth, which cannot carry the scoped
	// signing keys the permission model rides on. Refused by name — no
	// degraded lane (design 0003 §1).
	nc, err := dial()
	if err == nil {
		nc.Close()
		return fmt.Errorf("node: the server at %s admitted an anonymous connection — it is not running operator mode, and soulstream's admission model (scoped signing keys, auth callout) exists only there. Convert it (the kit's §2 fragments, %s) or point byo.url at an operator-mode server", st.BYOURL, kit)
	}
	if !errors.Is(err, nats.ErrAuthorization) {
		return fmt.Errorf("node: cannot reach the server at %s — is it running, and is byo.url right? (%w)", st.BYOURL, err)
	}

	// The realm account and its PLAIN workload signing key: the ops user
	// is signed by that key, so admission proves both landed.
	ncOps, err := dial(nats.UserCredentials(ceremony.UserCredsPath(stateDir, "ops")),
		nats.Name("soulstream-byo-probe"))
	if err != nil {
		return fmt.Errorf("node: the realm account refused its own service user — the realm account or its plain workload signing key is not on the server (kit §1, %s): %w", kit, err)
	}
	defer ncOps.Close()

	// The server-asserted whoami everything downstream leans on.
	if _, err := ncOps.Request("$SYS.REQ.USER.INFO", nil, 3*time.Second); err != nil {
		return fmt.Errorf("node: $SYS.REQ.USER.INFO did not answer on %s — the server is not exposing the system whoami this composition leans on: %w", st.BYOURL, err)
	}

	// JetStream on the realm account (kit §1's limits).
	js, err := jetstream.New(ncOps)
	if err != nil {
		return fmt.Errorf("node: jetstream: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := js.AccountInfo(ctx); err != nil {
		return fmt.Errorf("node: JetStream is not enabled for the realm account — kit §1's JetStream limits were not applied (%s): %w", kit, err)
	}

	// The AUTH account, its signing key, and the issuer user's seat in
	// auth_users: the issuer's own admission proves all three.
	ncIss, err := dial(nats.UserCredentials(ceremony.UserCredsPath(stateDir, "issuer")),
		nats.Name("soulstream-byo-probe-issuer"))
	if err != nil {
		return fmt.Errorf("node: the AUTH account refused the callout issuer — the AUTH account, its signing key, or the issuer user's auth_users seat is not on the server (kit §1, %s): %w", kit, err)
	}
	ncIss.Close()
	return nil
}

// SmokeAdmission is the one callout round the founding run owes the
// operator (design 0003 §4): sentinel + the founding token must admit,
// and a garbage token must refuse. Run after Found, against the live
// planes.
func SmokeAdmission(url, sentinelPath, token string) error {
	nc, err := nats.Connect(url, nats.UserCredentials(sentinelPath),
		nats.Token(token), nats.Timeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("node: callout smoke: sentinel + the founding token were refused — the callout lane is not admitting (%w)", err)
	}
	_, err = nc.Request("$SYS.REQ.USER.INFO", nil, 3*time.Second)
	nc.Close()
	if err != nil {
		return fmt.Errorf("node: callout smoke: admitted but the scoped whoami did not answer: %w", err)
	}
	if nc, err := nats.Connect(url, nats.UserCredentials(sentinelPath),
		nats.Token("sit_garbage"), nats.Timeout(10*time.Second)); err == nil {
		nc.Close()
		return errors.New("node: callout smoke: a garbage token was admitted — the callout is not the verifier on this server; refusing to call this realm founded")
	}
	return nil
}
