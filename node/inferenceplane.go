package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	siclient "github.com/impire-io/soulstream-identity/client"
	"github.com/impire-io/soulstream-inference/adapter/anthropic"
	"github.com/impire-io/soulstream-inference/adapter/standin"
	inferclient "github.com/impire-io/soulstream-inference/client"
	"github.com/impire-io/soulstream-inference/door"
	"github.com/impire-io/soulstream-inference/instance"

	"github.com/impire-io/soulstream/ceremony"
)

// The inference plane (soulstream-inference design 0001, opt-in): the door
// harnesses think through, and the instances that answer, in this
// process on one ordinary connection. The plane custodies exactly one
// kind of secret — an instance's provider credential, resolved from this
// plane principal's own D36 tree at start — and hands out exactly one
// kind of key: a realm key a served agent's harness presents at the door.
// Neither ever meets the other: the door holds no provider material, and
// the instance holds no door key.
//
// resolveWindow bounds the discovery scatter a pinned name costs. It is
// paid per resolution because the catalogue is read fresh per resolution;
// caching (or watching) both is the named [O].
const resolveWindow = 150 * time.Millisecond

// doorKeys is the door's whole authorization: keys the dispatcher plane
// issued for serves that are running right now. Issuing for a persona
// replaces — and so revokes — that persona's previous key, and stopping
// the plane revokes every one. A key nobody issued opens nothing, which
// is what makes the keyless arm measurable rather than argued.
//
// The granularity is per SERVE, not per wake. A per-wake key needs the
// mint inside the engine's admission path, and no workloads seam offers
// one — recorded as this feature's [O] rather than faked here.
type doorKeys struct {
	mu        sync.Mutex
	byPersona map[string]string
	holder    map[string]string // key → the persona it was issued for
}

func newDoorKeys() *doorKeys {
	return &doorKeys{byPersona: map[string]string{}, holder: map[string]string{}}
}

// issue mints a fresh key for one persona's serve, revoking whatever that
// persona held before.
func (k *doorKeys) issue(persona string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("node: inference plane: door key: %w", err)
	}
	key := "rk-" + hex.EncodeToString(raw)
	k.mu.Lock()
	defer k.mu.Unlock()
	if old, ok := k.byPersona[persona]; ok {
		delete(k.holder, old)
	}
	k.byPersona[persona] = key
	k.holder[key] = persona
	return key, nil
}

// allows is the door's Authorize check.
func (k *doorKeys) allows(key string) bool {
	if key == "" {
		return false
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	_, ok := k.holder[key]
	return ok
}

// revokeAll ends every issued key at once — the stop ceremony's half.
func (k *doorKeys) revokeAll() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.byPersona = map[string]string{}
	k.holder = map[string]string{}
}

// inferencePlane is the running plane.
type inferencePlane struct {
	nc        *nats.Conn
	js        jetstream.JetStream
	keys      *doorKeys
	instances []*instance.Instance
	srv       *http.Server
	err       chan error
	url       string
}

// InferenceURL is the inference door's HTTP endpoint ("" when disabled).
func (n *Node) InferenceURL() string {
	if n.inference == nil {
		return ""
	}
	return n.inference.url
}

// startInference wires the plane: the plane's own connection, the
// catalogue bucket, the configured instances (each with its credential
// resolved from custody), and the door on the node's own listener so a
// bind conflict is a named refusal and tests get real ports.
//
// A configured instance the house cannot construct — an unresolvable
// provider secret, most of all — fails the start whole. A plane that
// half-serves would answer some names and no-responders for others, and
// the caller could not tell which.
func (n *Node) startInference(cfg Config) error {
	st := cfg.State
	p := &inferencePlane{keys: newDoorKeys()}

	nc, err := n.connectPlane(cfg, ceremony.InferencePersona)
	if err != nil {
		return err
	}
	p.nc = nc
	n.inference = p // owns nc from here: Stop closes it even on a later failure

	if p.js, err = jetstream.New(nc); err != nil {
		return fmt.Errorf("node: inference plane: jetstream: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := EnsureCatalogue(ctx, p.js); err != nil {
		return fmt.Errorf("node: inference plane: %w", err)
	}

	ident := siclient.New(nc, st.RealmPub, ceremony.InferencePersona)
	for i, in := range st.InferenceInstances {
		adapter, err := buildAdapter(ident, in)
		if err != nil {
			return fmt.Errorf("node: inference plane: planes.inference.instances[%d]: %w", i, err)
		}
		served, err := instance.Serve(nc, instance.Config{
			Capability: in.Capability, Tags: in.Tags,
			Formats: "text/plain,application/json",
		}, adapter)
		if err != nil {
			return fmt.Errorf("node: inference plane: serve %s: %w", in.Model, err)
		}
		p.instances = append(p.instances, served)
		n.audit.Info("inference instance serving",
			"instance", served.ID(), "adapter", in.Adapter, "model", in.Model,
			"capability", in.Capability)
	}

	l, err := net.Listen("tcp", st.InferenceListen)
	if err != nil {
		return fmt.Errorf("node: the inference door cannot listen on %s (change planes.inference.listen in %s): %w",
			st.InferenceListen, filepath.Join(cfg.StateDir, "config.json"), err)
	}
	d := &door.Door{Conn: nc, Authorize: p.keys.allows, Route: p.route}
	h, err := d.Handler()
	if err != nil {
		_ = l.Close()
		return fmt.Errorf("node: inference plane: door: %w", err)
	}
	p.url = "http://" + l.Addr().String()
	p.srv = &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}
	p.err = make(chan error, 1)
	go func() { p.err <- p.srv.Serve(l) }()
	return nil
}

// buildAdapter constructs one configured instance's adapter. The
// stand-in needs nothing; a real provider's credential is resolved here
// and nowhere else — the adapter takes it at construction, and the value
// exists in this process and in the sealed store, in no third place.
func buildAdapter(ident *siclient.Client, in ceremony.InferenceInstance) (instance.Adapter, error) {
	switch in.Adapter {
	case ceremony.AdapterStandin:
		return standin.New(in.Model), nil
	case ceremony.AdapterAnthropic:
		secret, err := ident.SecretGet(in.Secret)
		if err != nil {
			return nil, fmt.Errorf("the provider key at %q does not resolve — write it with `soulstream provider set anthropic`, or take the instance out of the config; a plane never half-serves: %w", in.Secret, err)
		}
		if len(secret.Value) == 0 {
			return nil, fmt.Errorf("the provider key at %q is empty", in.Secret)
		}
		return anthropic.New(anthropic.Config{APIKey: string(secret.Value), Model: in.Model})
	default:
		return nil, fmt.Errorf("adapter %q is not one the house wires", in.Adapter)
	}
}

// route is the door's resolver: the requested model name read through the
// catalogue, pinned where the descriptor pins it. A name nobody has
// pointed anywhere falls back to the chat capability's anycast — which
// answers no-responders when nobody serves it, the routing layer telling
// the truth for free rather than the door inventing a refusal.
func (p *inferencePlane) route(model string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	entry, found, err := CatalogueGet(ctx, p.js, model)
	if err != nil {
		return "", err
	}
	if !found {
		return inferclient.AnycastSubject("chat"), nil
	}
	return entry.Descriptor().Route(p.nc, resolveWindow)
}

// stop ends the plane: the door first (no new requests while the
// instances are still there to answer the ones in flight), then every
// key, then the instances. The connection is closed by Node.Stop with
// the others.
func (p *inferencePlane) stop(audit interface{ Error(string, ...any) }) {
	if p.srv != nil {
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = p.srv.Shutdown(shCtx)
		cancel()
		select {
		case err := <-p.err:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				audit.Error("inference door exited", "err", err)
			}
		case <-time.After(5 * time.Second):
		}
	}
	p.keys.revokeAll()
	for _, in := range p.instances {
		if err := in.Stop(); err != nil {
			audit.Error("inference instance did not stop", "instance", in.ID(), "err", err)
		}
	}
}
