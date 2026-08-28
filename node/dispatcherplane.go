package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"

	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-core/registry"
	"github.com/impire-io/soulstream-core/topic"
	siclient "github.com/impire-io/soulstream-identity/client"
	"github.com/impire-io/soulstream-workloads/declaration"
	"github.com/impire-io/soulstream-workloads/dispatcher"
	"github.com/impire-io/soulstream-workloads/fleet"
	"github.com/impire-io/soulstream-workloads/wrap"

	"github.com/impire-io/soulstream/ceremony"
)

// The dispatcher plane (soulstream-workloads design 0007, opt-in): the
// house runs a standing serve arm over its own realm, so a declared
// agent lives on the deployment rather than on somebody's laptop. The
// loop itself is upstream's; this file fills its two seams and nothing
// else.
//
// realmRole is the identity-vault entry the founding already imports
// (node.Found) holding the realm account's SCOPED persona signing key.
// Minting an ephemeral against it yields exactly the canonical persona
// scope, expanded by the server for the name the mint carries — design
// 0007 §5's measured shape, and the answer to its open question about
// which role key the founding installs: this one, already there, on
// every realm this binary ever founded.
const realmRole = "realm"

// agentCredTTL bounds a served agent's engine credential. Long enough
// that an ordinary serve never notices it, short enough to be a
// revocation bound. There is no renewal loop: an expiry ends the engine,
// the placement returns to the race, and the next serve mints fresh —
// design 0007 §5's TTL/renewal [O], taken as the honest simple end.
const agentCredTTL = 24 * time.Hour

// dispatcherPlane is the running plane: the node's own client, the
// resolved placement topic, and the per-persona material the two seams
// hand out.
type dispatcherPlane struct {
	client     *realm.Client // owns the plane's connection
	js         jetstream.JetStream
	ops        *siclient.Client // the founding lane, for the persona-scope mints
	audit      auditLog
	placements string

	url      string
	realm    string
	realmPub string
	exe      string
	scratch  string
	base     wrap.Config
	harness  string
	template wrap.Template
	fromFile bool

	// The inference lane, nil when this deployment runs no inference
	// plane. A declaration asking to think then refuses its placement
	// whole rather than serving an agent that cannot.
	keys    *doorKeys
	doorURL string

	mu    sync.Mutex
	creds map[string]mintedCreds

	d   *dispatcher.Dispatcher
	err chan error
}

// auditLog is the slice of the node's logger these planes use.
type auditLog interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// mintedCreds is one persona's engine credential on disk: the path the
// engine and its tool door both dial with, and when it stops being worth
// reusing.
type mintedCreds struct {
	path  string
	until time.Time
}

// Placements is the resolved path of the topic submissions land on ("" when
// the dispatcher plane is disabled).
func (n *Node) Placements() string {
	if n.dispatch == nil {
		return ""
	}
	return n.dispatch.placements
}

// startDispatcher wires the plane: its own runtime-minted identity, the
// placement topic create-or-report, the base engine config, and the
// upstream loop running until the node stops.
func (n *Node) startDispatcher(ctx context.Context, cfg Config) error {
	st := cfg.State
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("node: dispatcher plane: finding my own executable for the tool door: %w", err)
	}
	p := &dispatcherPlane{
		audit: n.audit, url: n.url, realm: st.Realm, realmPub: st.RealmPub,
		exe: exe, scratch: filepath.Join(cfg.StateDir, "dispatcher"),
		harness: st.DispatcherHarness, creds: map[string]mintedCreds{},
	}
	n.dispatch = p

	// The template the house serves every persona with. A preset is
	// rebuilt per persona (its MCP lane is per-agent); a template file is
	// read once and its tool door stamped per persona, because the file
	// cannot know whose credential it will carry.
	switch {
	case st.DispatcherTemplate != "":
		if p.template, err = wrap.LoadTemplate(st.DispatcherTemplate); err != nil {
			return fmt.Errorf("node: dispatcher plane: %w", err)
		}
		p.fromFile = true
	default:
		if p.template, err = wrap.Preset(p.harness, wrap.Lane{}); err != nil {
			return fmt.Errorf("node: dispatcher plane: %w", err)
		}
	}

	nc, err := n.connectPlane(cfg, ceremony.DispatcherPersona)
	if err != nil {
		return err
	}
	if p.js, err = jetstream.New(nc); err != nil {
		nc.Close()
		return fmt.Errorf("node: dispatcher plane: jetstream: %w", err)
	}
	// The record substrate must exist before the placement topic does.
	// Create-or-report, exactly as the memory plane does it: a realm that
	// runs no memory plane still needs its streams.
	provCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if _, err := realm.ProvisionOn(realm.WithConn(provCtx, nc), p.js); err != nil {
		nc.Close()
		return fmt.Errorf("node: dispatcher plane: realm substrate: %w", err)
	}
	signer, err := siclient.New(nc, st.RealmPub, ceremony.DispatcherPersona).
		PersonaSigner(ceremony.DispatcherPersona)
	if err != nil {
		nc.Close()
		return fmt.Errorf("node: dispatcher plane: persona signer: %w", err)
	}
	client, err := realm.NewClient(ctx, nc, realm.Config{
		Realm: st.Realm, Persona: ceremony.DispatcherPersona, Signer: signer,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("node: dispatcher plane: realm client: %w", err)
	}
	p.client = client // owns nc from here
	if err := registry.EnsureSigningKey(ctx, client, signer); err != nil {
		n.audit.Warn("node: dispatcher plane: signing key not published to the directory (claims will read unknown-key)",
			"persona", ceremony.DispatcherPersona, "err", err)
	}
	p.ops = n.ops

	if p.placements, err = ensurePlacements(ctx, client, st.DispatcherPlacements); err != nil {
		return err
	}
	if n.inference != nil {
		p.keys, p.doorURL = n.inference.keys, n.inference.url
	}

	p.base = wrap.Config{Template: p.template, Scratch: p.scratch}
	d := &dispatcher.Dispatcher{
		Client:       client,
		Placements:   p.placements,
		ConnectAgent: p.connectAgent,
		EngineFor:    p.engineFor,
		Engine:       p.base,
		Invoke:       cfg.HarnessInvoke,
		Reclaim:      cfg.DispatcherReclaim,
		Log:          n.audit,
	}
	p.d = d
	p.err = make(chan error, 1)
	go func() { p.err <- d.Run(ctx) }()
	return nil
}

// drain runs the stop ceremony while the rest of the node is still up:
// every engine cancelled and waited on, so the agent's own self-report
// lands — signed, which needs the identity plane still answering. The
// placements stay claimed on the record, which is what makes the next
// start resume them.
func (p *dispatcherPlane) drain() {
	if p.d == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := p.d.Drain(ctx); err != nil {
		p.audit.Error("dispatcher plane did not drain", "err", err)
	}
}

// ensurePlacements resolves the configured placement topic NAME to a
// path, create-or-report: the realm's board answers whether the topic
// already exists, and only its absence starts one. The name is therefore
// stable configuration and the path is a fact of the record — a restart
// finds the same topic, which is what makes resume-from-the-log work.
func ensurePlacements(ctx context.Context, c *realm.Client, name string) (string, error) {
	entries, err := topic.Board(ctx, c)
	if err != nil {
		return "", fmt.Errorf("node: dispatcher plane: read the board: %w", err)
	}
	for _, e := range entries {
		if e.Announcement.Name == name {
			return e.Path, nil
		}
	}
	h, err := topic.StartTopic(ctx, c, topic.StartTopicInput{
		Name:          name,
		SubjectMatter: "where declared agents are placed on this deployment",
	})
	if err != nil {
		return "", fmt.Errorf("node: dispatcher plane: start the placement topic %q: %w", name, err)
	}
	return h.Path(), nil
}

// connectAgent is the credential seam (design 0007 §5): one served
// agent's engine runs on a canonical persona-scope credential minted for
// its persona through the identity plane's own D28 lane. The narrow
// lanes are deliberately not used here — a workload scope and the agent
// capability scope both grant `$JS.API.INFO` alone, and an engine
// materialises its topic at every wake, so a narrow credential is
// refused at the transport before the first answer.
func (p *dispatcherPlane) connectAgent(ctx context.Context, persona string) (*realm.Client, error) {
	credsPath, err := p.credsFor(persona)
	if err != nil {
		return nil, err
	}
	nc, err := nats.Connect(p.url,
		nats.UserCredentials(credsPath),
		nats.Name("soulstream-agent-"+persona))
	if err != nil {
		return nil, fmt.Errorf("connect as %s: %w", persona, err)
	}
	signer, err := siclient.New(nc, p.realmPub, persona).PersonaSigner(persona)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("%s signer: %w", persona, err)
	}
	client, err := realm.NewClient(ctx, nc, realm.Config{
		Realm: p.realm, Persona: persona, Signer: signer,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("%s realm client: %w", persona, err)
	}
	if err := registry.EnsureSigningKey(ctx, client, signer); err != nil {
		p.audit.Warn("node: dispatcher plane: agent signing key not published to the directory (records will read unknown-key)",
			"persona", persona, "err", err)
	}
	return client, nil
}

// engineFor is the per-persona engine seam (design 0007 §5's finding 3):
// the harness template with a tool door carrying THIS agent's own
// credential — one dispatcher serving many personas cannot give them all
// one door — and, when the declaration asks to think, the lane to the
// house's own inference door.
//
// The seam carries the persona and not the declaration, so the plane
// re-reads the placement topic to find what this persona was declared
// with. Recorded as a finding: `EngineFor(ctx, declaration.Declaration)`
// would make the round trip unnecessary.
func (p *dispatcherPlane) engineFor(ctx context.Context, persona string) (*wrap.Config, error) {
	credsPath, err := p.credsFor(persona)
	if err != nil {
		return nil, err
	}
	lane := wrap.Lane{
		URL: p.url, CredsFile: credsPath, Realm: p.realm, Persona: persona,
		MCPCommandLoc: p.exe, MCPArgs: []string{"mcp"},
	}
	tpl := p.template
	if p.fromFile {
		tpl.MCPCommand, tpl.MCPArgs, tpl.MCPEnv = p.exe, lane.MCPArgs, doorEnv(lane)
	} else if tpl, err = wrap.Preset(p.harness, lane); err != nil {
		return nil, err
	}

	decl, found, err := p.declarationFor(ctx, persona)
	if err != nil {
		return nil, err
	}
	if found && decl.Inference != nil {
		env, err := p.thinkingLane(ctx, persona, decl.Inference.Model)
		if err != nil {
			return nil, err
		}
		tpl.Env = mergeEnv(tpl.Env, env)
	}

	cfg := p.base
	cfg.Template = tpl
	return &cfg, nil
}

// thinkingLane is what a declaration's `inference` block becomes: the
// house's own door, a key issued for this serve, and the virtual name
// itself — which travels as the harness's own model environment variable
// rather than a template variable, closing design 0007 §3's last
// question the way real harnesses already work.
//
// The name is checked against the catalogue here, so an unpointed name
// refuses the placement at serve time with the fix in the refusal — far
// better than an agent that starts and then cannot think.
func (p *dispatcherPlane) thinkingLane(ctx context.Context, persona, model string) (map[string]string, error) {
	if p.keys == nil {
		return nil, fmt.Errorf("the declaration asks to think through model %q but this deployment runs no inference plane — enable planes.inference in config.json, or take the inference block out of the declaration", model)
	}
	if _, found, err := CatalogueGet(ctx, p.js, model); err != nil {
		return nil, err
	} else if !found {
		return nil, fmt.Errorf("no model is named %q in this realm — point the name at something with `soulstream model set %s --pin <model>`", model, model)
	}
	key, err := p.keys.issue(persona)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"ANTHROPIC_BASE_URL": p.doorURL,
		"ANTHROPIC_API_KEY":  key,
		"ANTHROPIC_MODEL":    model,
	}, nil
}

// declarationFor finds what a persona was declared with, preferring the
// placement this node owns. One node serves one placement per persona
// (upstream refuses the second), so the preference is the whole
// disambiguation.
func (p *dispatcherPlane) declarationFor(ctx context.Context, persona string) (declaration.Declaration, bool, error) {
	mt, err := topic.Open(p.client, p.placements).Materialise(ctx)
	if err != nil {
		return declaration.Declaration{}, false, fmt.Errorf("read the placements of %s: %w", persona, err)
	}
	var found declaration.Declaration
	var matched bool
	for _, item := range mt.WorkItems {
		decl, ok := fleet.DeclarationOf(item)
		if !ok || decl.Persona != persona {
			continue
		}
		found, matched = decl, true
		if item.Owner == ceremony.DispatcherPersona && item.Status == topic.WorkClaimed {
			return decl, true, nil
		}
	}
	return found, matched, nil
}

// credsFor mints (and reuses) one persona's engine credential: a
// caller-generated key, a JWT minted against the persona-scope role, and
// a creds file under the plane's scratch — the one place the pair rests,
// because the harness's tool door is a separate process and dials with a
// path. The seed never leaves this machine and the JWT expires on its
// own.
func (p *dispatcherPlane) credsFor(persona string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if held, ok := p.creds[persona]; ok && time.Now().Before(held.until) {
		return held.path, nil
	}
	ukp, err := nkeys.CreateUser()
	if err != nil {
		return "", fmt.Errorf("%s user key: %w", persona, err)
	}
	pub, err := ukp.PublicKey()
	if err != nil {
		return "", fmt.Errorf("%s user key: %w", persona, err)
	}
	seed, err := ukp.Seed()
	if err != nil {
		return "", fmt.Errorf("%s user key: %w", persona, err)
	}
	token, err := p.ops.MintEphemeral(realmRole, persona, pub, agentCredTTL, nil)
	if err != nil {
		return "", fmt.Errorf("mint the persona-scope credential for %s (role %q): %w", persona, realmRole, err)
	}
	creds, err := jwt.FormatUserConfig(token, seed)
	if err != nil {
		return "", fmt.Errorf("%s creds: %w", persona, err)
	}
	dir := filepath.Join(p.scratch, "personas")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("%s creds directory: %w", persona, err)
	}
	path := filepath.Join(dir, persona+".creds")
	if err := os.WriteFile(path, creds, 0o600); err != nil {
		return "", fmt.Errorf("%s creds: %w", persona, err)
	}
	// Re-mint at half life: a credential that expires mid-serve would end
	// the engine, and the record would carry the gap.
	p.creds[persona] = mintedCreds{path: path, until: time.Now().Add(agentCredTTL / 2)}
	return path, nil
}

// stop waits for the upstream loop's own stop ceremony — it drains, so
// an in-flight harness's failure lands as the agent's own self-report
// before the process ends.
func (p *dispatcherPlane) stop() {
	if p.err == nil {
		return
	}
	select {
	case err := <-p.err:
		if err != nil && !errors.Is(err, context.Canceled) {
			p.audit.Error("dispatcher plane exited", "err", err)
		}
	case <-time.After(90 * time.Second):
		p.audit.Error("dispatcher plane did not drain in time")
	}
	if p.client != nil {
		_ = p.client.Close()
	}
}

// doorEnv is the tool door's lane for a template file, which cannot
// carry a persona's credential because it is written before any persona
// is served. The preset builds the same five names from the same Lane.
func doorEnv(lane wrap.Lane) map[string]string {
	env := map[string]string{}
	for k, v := range map[string]string{
		"SOULSTREAM_URL":     lane.URL,
		"SOULSTREAM_CREDS":   lane.CredsFile,
		"SOULSTREAM_REALM":   lane.Realm,
		"SOULSTREAM_PERSONA": lane.Persona,
	} {
		if v != "" {
			env[k] = v
		}
	}
	return env
}

// mergeEnv layers the house's own names over a template's, leaving the
// template's copy untouched — the value returned by Preset or LoadTemplate
// is shared between personas.
func mergeEnv(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
