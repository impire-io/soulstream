package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-core/topic"
	siclient "github.com/impire-io/soulstream-identity/client"
	inferclient "github.com/impire-io/soulstream-inference/client"
	"github.com/impire-io/soulstream-workloads/wrap"

	"github.com/impire-io/soulstream/ceremony"
)

// providerSentinel stands in for a real provider credential: it is
// written into the inference plane's own custody, resolved there by a
// configured anthropic instance at plane start, and then hunted for
// everywhere a served agent can read. A stand-in adapter holds no secret
// at all, so without this the custody arm would prove nothing.
const providerSentinel = "sk-ant-sentinel-the-agent-must-never-see-this"

// harnessRun is one scripted invocation as the rig saw it: whose it was,
// and the environment the plane constructed for it. The environment is
// the custody scan's most interesting exhibit — it is the one place a
// credential legitimately travels.
type harnessRun struct {
	persona string
	env     map[string]string
}

// scriptedHarness is the assistant this gate runs: it reads the lane the
// dispatcher plane gave it and thinks through whatever door that lane
// names, exactly as a real harness would with ANTHROPIC_BASE_URL set.
// With no such lane it answers on its own — the ambient case.
type scriptedHarness struct {
	mu   sync.Mutex
	runs []harnessRun
}

func (s *scriptedHarness) invoke(ctx context.Context, spec wrap.RunSpec) wrap.HarnessResult {
	env := map[string]string{}
	for k, v := range spec.Template.Env {
		env[k] = v
	}
	s.mu.Lock()
	s.runs = append(s.runs, harnessRun{persona: spec.Template.MCPEnv["SOULSTREAM_PERSONA"], env: env})
	s.mu.Unlock()

	base := env["ANTHROPIC_BASE_URL"]
	if base == "" {
		return wrap.HarnessResult{OK: true, Text: "answered on my own authentication"}
	}
	text, err := askDoor(ctx, base, env["ANTHROPIC_API_KEY"], env["ANTHROPIC_MODEL"], spec.Prompt)
	if err != nil {
		return wrap.HarnessResult{Detail: err.Error()}
	}
	return wrap.HarnessResult{OK: true, Text: text}
}

func (s *scriptedHarness) seen() []harnessRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]harnessRun(nil), s.runs...)
}

// transcript renders every environment the plane constructed, as text —
// the custody scan reads it the way an auditor would.
func (s *scriptedHarness) transcript() string {
	var b strings.Builder
	for _, run := range s.seen() {
		b.WriteString(run.persona)
		for k, v := range run.env {
			fmt.Fprintf(&b, " %s=%s", k, v)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// askDoor speaks the Messages API shape at the realm's own door, which is
// the whole point of the door: a harness needs no new protocol to think
// through the realm.
func askDoor(ctx context.Context, base, key, model, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": model, "max_tokens": 1024,
		"messages": []map[string]any{{"role": "user", "content": prompt}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", key)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("the door answered %d: %s", res.StatusCode, raw)
	}
	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Content) == 0 {
		return "", fmt.Errorf("the door answered with no content: %s", raw)
	}
	return out.Content[0].Text, nil
}

// listModels asks the door what it can think with — the enumeration a
// harness runs before it picks a model.
func listModels(ctx context.Context, t *testing.T, base, key string) []string {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/models", nil)
	if err != nil {
		t.Fatalf("models request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("models: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("models answered %d: %s", res.StatusCode, raw)
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("models decode: %v", err)
	}
	names := make([]string, 0, len(out.Data))
	for _, d := range out.Data {
		names = append(names, d.ID)
	}
	return names
}

// TestM15ThinkingHouse (specs/014 SC-001, SC-002): the whole composition
// as one run. A realm founds, its inference plane resolves a provider key
// from custody and serves a stand-in beside it, its dispatcher plane
// serves two declared agents — one that thinks through the realm, one on
// the ambient lane — and the record shows exactly one answer each, no
// provider material anywhere the agents can read, a door that refuses a
// key nobody issued without troubling the plane, and a restart that
// resumes the serve from the log.
func TestM15ThinkingHouse(t *testing.T) {
	dir := t.TempDir()
	// This gate stops and starts the same realm, so its listener is a
	// reserved real port rather than the usual ephemeral one: a restart
	// that moved address would be a different deployment, and resuming
	// from the log is the property under test.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listen := ln.Addr().String()
	_ = ln.Close()

	st, err := ceremony.Generate(listen, "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Port hygiene: only the planes this gate is about.
	st.MemoryEnabled, st.MCPEnabled, st.SignInEnabled, st.HelmEnabled = false, false, false, false
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	audit := &syncBuffer{}
	n, err := Start(Config{StateDir: dir, State: st, AuditWriter: audit})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	token, err := Found(n, st, dir)
	if err != nil {
		n.Stop()
		t.Fatalf("found: %v", err)
	}

	// The operator loads the provider key into the inference plane's own
	// custody — `soulstream provider set` in one act, on the same runtime
	// identity the plane will connect with.
	writeProviderSecret(t, n.URL(), st)
	n.Stop()

	// Now the planes are configured on. Both blocks are opt-in, which is
	// exactly why a realm founded a moment ago can grow them.
	st.DispatcherEnabled = true
	st.DispatcherPlacements = ceremony.DefaultPlacements
	st.DispatcherHarness = "claude"
	st.InferenceEnabled = true
	st.InferenceListen = "127.0.0.1:0"
	st.InferenceInstances = []ceremony.InferenceInstance{
		{Adapter: ceremony.AdapterStandin, Model: "standin-1", Capability: "chat"},
		{Adapter: ceremony.AdapterAnthropic, Model: "claude-x", Capability: "chat",
			Secret: "providers/anthropic"},
	}
	if err := st.Save(dir); err != nil {
		t.Fatalf("save with planes: %v", err)
	}
	st, err = ceremony.Verify(dir)
	if err != nil {
		t.Fatalf("verify with planes: %v", err)
	}

	harness := &scriptedHarness{}
	start := func() *Node {
		t.Helper()
		n, err := Start(Config{StateDir: dir, State: st, AuditWriter: audit,
			HarnessInvoke: harness.invoke, DispatcherReclaim: 2 * time.Second})
		if err != nil {
			t.Fatalf("start with the thinking planes: %v (audit: %s)", err, audit.String())
		}
		return n
	}
	n = start()
	stopped := false
	defer func() {
		if !stopped {
			n.Stop()
		}
	}()
	if n.InferenceURL() == "" || n.Placements() == "" {
		t.Fatalf("the planes did not announce themselves: door %q placements %q",
			n.InferenceURL(), n.Placements())
	}

	// The owner's own connection: the admission any person gets.
	ctx := context.Background()
	owner, ownerConn := ownerClient(ctx, t, n.URL(), dir, st, token)
	defer func() { _ = owner.Close() }()

	// A name for the thinking, pointed at the stand-in. The declaration
	// will carry this name and nothing else — no model, no provider.
	if err := CatalogueSet(ctx, owner.JetStream(), "realm-default", ModelEntry{
		Capability: "chat", ModelPin: "standin-1",
	}); err != nil {
		t.Fatalf("model set: %v", err)
	}

	h, err := topic.StartTopic(ctx, owner, topic.StartTopicInput{Name: "the-thinking-house"})
	if err != nil {
		t.Fatalf("start topic: %v", err)
	}

	// Every message the plane's subjects ever see, counted from outside
	// the queue groups — the keyless arm's whole measurement.
	var delivered atomic.Int64
	for _, subject := range []string{
		inferclient.AnycastSubject("chat"), inferclient.AnycastSubject("chat") + ".>",
	} {
		sub, err := ownerConn.Subscribe(subject, func(*nats.Msg) { delivered.Add(1) })
		if err != nil {
			t.Fatalf("watch %s: %v", subject, err)
		}
		defer func() { _ = sub.Unsubscribe() }()
	}
	if err := ownerConn.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	// Submit-and-forget, twice: one agent that thinks through the realm,
	// one that does not name inference at all.
	submit(t, dir, st, n.URL(), "thinker", h.Path(), `,"inference":{"model":"realm-default"}`)
	submit(t, dir, st, n.URL(), "ambient", h.Path(), "")

	// The submitters are gone; the mention is the only thing left.
	if _, err := h.PostTurnMentioning(ctx, "@thinker @ambient what is the realm for?",
		[]string{"thinker", "ambient"}); err != nil {
		t.Fatalf("mention: %v", err)
	}

	answers := waitForAnswers(ctx, t, owner, h.Path(), map[string]int{"thinker": 1, "ambient": 1})
	if !strings.Contains(answers["thinker"][0], "model=standin-1") {
		t.Fatalf("the thinker's answer did not come through the plane: %q", answers["thinker"][0])
	}
	if strings.Contains(answers["ambient"][0], "model=") {
		t.Fatalf("the ambient agent thought through the realm: %q", answers["ambient"][0])
	}

	// The ambient lane is untouched: no name from this feature reaches a
	// harness that did not ask for it.
	for _, run := range harness.seen() {
		if run.persona != "ambient" {
			continue
		}
		for k := range run.env {
			if strings.HasPrefix(k, "ANTHROPIC_") {
				t.Fatalf("the ambient agent's environment carried %s", k)
			}
		}
	}
	var thinkerKey string
	for _, run := range harness.seen() {
		if run.persona == "thinker" {
			thinkerKey = run.env["ANTHROPIC_API_KEY"]
			if run.env["ANTHROPIC_BASE_URL"] != n.InferenceURL() {
				t.Fatalf("the thinker was pointed at %q, not the house's door %q",
					run.env["ANTHROPIC_BASE_URL"], n.InferenceURL())
			}
			if run.env["ANTHROPIC_MODEL"] != "realm-default" {
				t.Fatalf("the thinker was given model %q, not the virtual name",
					run.env["ANTHROPIC_MODEL"])
			}
		}
	}
	if thinkerKey == "" {
		t.Fatal("the thinker was never given a door key")
	}

	// Custody: the provider credential exists in the sealed store and in
	// the serving instance, and in nothing a served agent can read.
	for name, exhibit := range map[string]string{
		"the wake topic (the declaration's home, the outcomes)": marshal(t, materialise(ctx, t, owner, h.Path())),
		"the placement topic (the declarations themselves)":     marshal(t, materialise(ctx, t, owner, n.Placements())),
		"the harness environments":                              harness.transcript(),
	} {
		if strings.Contains(exhibit, providerSentinel) {
			t.Fatalf("provider material found in %s", name)
		}
	}

	// What the door says it can think with is the catalogue itself: the
	// listing and the routing read the same names, so a harness that
	// enumerates is told the truth rather than an empty list.
	if names := listModels(ctx, t, n.InferenceURL(), thinkerKey); len(names) != 1 || names[0] != "realm-default" {
		t.Fatalf("the door advertises %v, want the catalogue's own names", names)
	}

	// A key nobody issued opens nothing, and the plane never hears about
	// it: the refusal is the door's alone.
	before := delivered.Load()
	for _, key := range []string{"", "rk-not-a-key-anyone-issued", thinkerKey + "x"} {
		if _, err := askDoor(ctx, n.InferenceURL(), key, "realm-default", "let me in"); err == nil {
			t.Fatalf("the door opened for key %q", key)
		} else if !strings.Contains(err.Error(), "401") {
			t.Fatalf("the door refused key %q with %v, want a 401", key, err)
		}
	}
	if after := delivered.Load(); after != before {
		t.Fatalf("a keyless request reached the plane: %d deliveries became %d", before, after)
	}

	// Restart: the record is the position. The placement stays claimed by
	// this node and the serve resumes with no second claim.
	claimsBefore := claimCount(ctx, t, owner, n.Placements(), "thinker")
	n.Stop()
	stopped = true
	n = start()
	stopped = false

	// A restart is a new server process on the same address: the person
	// watching reconnects, which is what anybody at a console does.
	reader, _ := ownerClient(ctx, t, n.URL(), dir, st, token)
	defer func() { _ = reader.Close() }()
	if got := claimCount(ctx, t, reader, n.Placements(), "thinker"); got != claimsBefore {
		t.Fatalf("the restart re-claimed: %d claims became %d", claimsBefore, got)
	}
	if _, err := topic.Open(reader, h.Path()).PostTurnMentioning(ctx,
		"@thinker and after a restart?", []string{"thinker"}); err != nil {
		t.Fatalf("second mention: %v", err)
	}
	answers = waitForAnswers(ctx, t, reader, h.Path(), map[string]int{"thinker": 2})
	if !strings.Contains(answers["thinker"][1], "model=standin-1") {
		t.Fatalf("the resumed serve did not think through the plane: %q", answers["thinker"][1])
	}
}

// TestThinkingHouseAbsent (specs/014 SC-003): a realm that declares
// neither plane creates nothing this feature added — the disabled arm, so
// the opt-in claim is measured rather than asserted.
func TestThinkingHouseAbsent(t *testing.T) {
	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "quiet")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	st.MemoryEnabled, st.MCPEnabled, st.SignInEnabled, st.HelmEnabled = false, false, false, false
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	// The config a founding writes names neither plane at all.
	cfg, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, key := range []string{`"dispatcher"`, `"inference"`} {
		if bytes.Contains(cfg, []byte(key)) {
			t.Fatalf("a founding wrote %s into config.json:\n%s", key, cfg)
		}
	}

	audit := &syncBuffer{}
	n, err := Start(Config{StateDir: dir, State: st, AuditWriter: audit})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer n.Stop()
	token, err := Found(n, st, dir)
	if err != nil {
		t.Fatalf("found: %v", err)
	}
	if n.InferenceURL() != "" || n.Placements() != "" {
		t.Fatalf("a plane ran unasked: door %q placements %q", n.InferenceURL(), n.Placements())
	}

	ctx := context.Background()
	owner, _ := ownerClient(ctx, t, n.URL(), dir, st, token)
	defer func() { _ = owner.Close() }()
	if _, err := owner.JetStream().KeyValue(ctx, CatalogueBucket); err == nil {
		t.Fatal("the catalogue bucket exists on a realm that named no models")
	}
	board, err := topic.Board(ctx, owner)
	if err != nil {
		t.Fatalf("board: %v", err)
	}
	for _, e := range board {
		if e.Announcement.Name == ceremony.DefaultPlacements {
			t.Fatalf("a placement topic exists on a realm with no dispatcher: %s", e.Path)
		}
	}
}

// TestThinkingConfigRefusals (specs/014 SC-004): every configuration the
// house cannot honour is refused by name at verify — before a listener
// binds, before an adapter is built, before anything serves.
func TestThinkingConfigRefusals(t *testing.T) {
	for _, tc := range []struct {
		name string
		mod  func(*ceremony.State)
		want string
	}{
		{"a dispatcher naming no assistant",
			func(s *ceremony.State) { s.DispatcherEnabled = true },
			"names no assistant"},
		{"a door off the loopback",
			func(s *ceremony.State) { s.InferenceEnabled, s.InferenceListen = true, "0.0.0.0:8600" },
			"is not loopback"},
		{"a door on another plane's address",
			func(s *ceremony.State) {
				s.MCPEnabled, s.MCPListen = true, "127.0.0.1:8600"
				s.InferenceEnabled, s.InferenceListen = true, "127.0.0.1:8600"
			},
			"separate addresses"},
		{"an instance with no model",
			func(s *ceremony.State) {
				s.InferenceEnabled, s.InferenceListen = true, "127.0.0.1:8600"
				s.InferenceInstances = []ceremony.InferenceInstance{{Adapter: ceremony.AdapterStandin}}
			},
			"names no model"},
		{"an adapter the house does not wire",
			func(s *ceremony.State) {
				s.InferenceEnabled, s.InferenceListen = true, "127.0.0.1:8600"
				s.InferenceInstances = []ceremony.InferenceInstance{{Adapter: "oracle", Model: "m"}}
			},
			"is not one the house wires"},
		{"an anthropic instance with no secret",
			func(s *ceremony.State) {
				s.InferenceEnabled, s.InferenceListen = true, "127.0.0.1:8600"
				s.InferenceInstances = []ceremony.InferenceInstance{{Adapter: ceremony.AdapterAnthropic, Model: "m"}}
			},
			"needs a secret"},
		{"a stand-in carrying a credential",
			func(s *ceremony.State) {
				s.InferenceEnabled, s.InferenceListen = true, "127.0.0.1:8600"
				s.InferenceInstances = []ceremony.InferenceInstance{
					{Adapter: ceremony.AdapterStandin, Model: "m", Secret: "providers/x"}}
			},
			"holds no provider credential"},
		{"two instances on one model",
			func(s *ceremony.State) {
				s.InferenceEnabled, s.InferenceListen = true, "127.0.0.1:8600"
				s.InferenceInstances = []ceremony.InferenceInstance{
					{Adapter: ceremony.AdapterStandin, Model: "m"},
					{Adapter: ceremony.AdapterStandin, Model: "m"}}
			},
			"twice"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			st, err := ceremony.Generate("127.0.0.1:0", "home")
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			st.MemoryEnabled, st.MCPEnabled, st.SignInEnabled, st.HelmEnabled = false, false, false, false
			st.DispatcherPlacements = ceremony.DefaultPlacements
			tc.mod(st)
			if err := st.Save(dir); err != nil {
				t.Fatalf("save: %v", err)
			}
			_, err = ceremony.Verify(dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Verify() = %v, want an error containing %q", err, tc.want)
			}
		})
	}
}

// TestUnresolvableProviderSecret (specs/014 FR-008): a configured
// provider whose key is not in custody fails the start whole. A plane
// that half-serves would answer some names and no-responders for others,
// and no caller could tell which.
func TestUnresolvableProviderSecret(t *testing.T) {
	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	st.MemoryEnabled, st.MCPEnabled, st.SignInEnabled, st.HelmEnabled = false, false, false, false
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	audit := &syncBuffer{}
	n, err := Start(Config{StateDir: dir, State: st, AuditWriter: audit})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := Found(n, st, dir); err != nil {
		n.Stop()
		t.Fatalf("found: %v", err)
	}
	n.Stop()

	st.InferenceEnabled, st.InferenceListen = true, "127.0.0.1:0"
	st.InferenceInstances = []ceremony.InferenceInstance{
		{Adapter: ceremony.AdapterAnthropic, Model: "claude-x", Capability: "chat",
			Secret: "providers/never-written"},
	}
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	n2, err := Start(Config{StateDir: dir, State: st, AuditWriter: audit})
	if n2 != nil {
		n2.Stop()
	}
	if err == nil || !strings.Contains(err.Error(), "does not resolve") {
		t.Fatalf("Start() = %v, want a refusal naming the unresolvable provider key", err)
	}
}

// --- the rig ---

// writeProviderSecret is `soulstream provider set` as one act: the house
// mints the inference plane's own identity and writes that principal's
// tree, which is the only hand that can — the store is caller-own.
func writeProviderSecret(t *testing.T, url string, st *ceremony.State) {
	t.Helper()
	token, seed, err := st.MintPlaneUser(ceremony.InferencePersona)
	if err != nil {
		t.Fatalf("mint the inference plane identity: %v", err)
	}
	nc, err := nats.Connect(url, nats.UserJWTAndSeed(token, string(seed)))
	if err != nil {
		t.Fatalf("inference plane admission: %v", err)
	}
	defer nc.Close()
	if _, err := siclient.New(nc, st.RealmPub, ceremony.InferencePersona).
		SecretPut("providers/anthropic", []byte(providerSentinel), 0); err != nil {
		t.Fatalf("write the provider key: %v", err)
	}
}

// ownerClient is the admission a person gets: the deployment's sentinel
// and a token, exchanged by the callout for the canonical persona scope.
func ownerClient(ctx context.Context, t *testing.T, url, dir string, st *ceremony.State, token string) (*realm.Client, *nats.Conn) {
	t.Helper()
	nc, err := nats.Connect(url,
		nats.UserCredentials(ceremony.SentinelPath(dir)), nats.Token(token),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	if err != nil {
		t.Fatalf("owner admission: %v", err)
	}
	signer, err := siclient.New(nc, st.RealmPub, ceremony.FoundingPersona).
		PersonaSigner(ceremony.FoundingPersona)
	if err != nil {
		t.Fatalf("owner signer: %v", err)
	}
	c, err := realm.NewClient(ctx, nc, realm.Config{
		Realm: st.Realm, Persona: ceremony.FoundingPersona, Signer: signer,
	})
	if err != nil {
		t.Fatalf("owner realm client: %v", err)
	}
	return c, nc
}

// submit writes a declaration and places it exactly as the verb does —
// through SubmitAgent, on its own connection, which closes before the
// agent is ever served. Submit-and-forget is the property under test.
func submit(t *testing.T, dir string, st *ceremony.State, url, persona, topicPath, extra string) {
	t.Helper()
	decl := fmt.Sprintf(`{"role":"agent","lifecycle":"service","persona":%q,"topic":%q,`+
		`"artifact":"file:///dev/null","wake":[{"kind":"mention"}]%s}`,
		persona, topicPath, extra)
	path := filepath.Join(t.TempDir(), persona+".json")
	if err := os.WriteFile(path, []byte(decl), 0o600); err != nil {
		t.Fatalf("write declaration: %v", err)
	}
	id, err := SubmitAgent(context.Background(),
		Config{StateDir: dir, State: st, AuditWriter: io.Discard}, url, path)
	if err != nil {
		t.Fatalf("submit %s: %v", persona, err)
	}
	if id == "" {
		t.Fatalf("submit %s returned no placement id", persona)
	}
}

// waitForAnswers polls the topic until every named persona has posted at
// least the wanted number of turns, then returns them — and fails if any
// persona posted MORE, because exactly-once is the claim.
func waitForAnswers(ctx context.Context, t *testing.T, c *realm.Client, path string, want map[string]int) map[string][]string {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for {
		got := map[string][]string{}
		mt := materialise(ctx, t, c, path)
		for _, con := range mt.Contributions {
			if con.Type == "turn.post" {
				got[con.Author] = append(got[con.Author], con.Body)
			}
		}
		done := true
		for persona, n := range want {
			switch {
			case len(got[persona]) > n:
				t.Fatalf("%s answered %d times, want %d: %q", persona, len(got[persona]), n, got[persona])
			case len(got[persona]) < n:
				done = false
			}
		}
		if done {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatalf("waited for %v, saw %v", want, got)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// claimCount counts the claim events on a persona's placement — the
// measurement behind "a restart resumes rather than re-claims".
func claimCount(ctx context.Context, t *testing.T, c *realm.Client, placements, persona string) int {
	t.Helper()
	mt := materialise(ctx, t, c, placements)
	for _, item := range mt.WorkItems {
		if !strings.HasSuffix(item.Title, "as "+persona) {
			continue
		}
		n := 0
		for _, ev := range item.Timeline {
			if ev.Kind == "claim" {
				n++
			}
		}
		if item.Owner != ceremony.DispatcherPersona {
			t.Fatalf("%s's placement is owned by %q, not the dispatcher plane", persona, item.Owner)
		}
		return n
	}
	t.Fatalf("no placement for %s in %+v", persona, mt.WorkItems)
	return 0
}

func materialise(ctx context.Context, t *testing.T, c *realm.Client, path string) *topic.MaterializedTopic {
	t.Helper()
	mt, err := topic.Open(c, path).Materialise(ctx)
	if err != nil {
		t.Fatalf("materialise %s: %v", path, err)
	}
	return mt
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal exhibit: %v", err)
	}
	return string(raw)
}
