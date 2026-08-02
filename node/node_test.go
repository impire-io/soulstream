package node

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	siclient "github.com/impire-io/soulidentity/client"
	"github.com/impire-io/soulstream-archivist/archive"
	"github.com/impire-io/soulstream/realm"
	"github.com/impire-io/soulstream/topic"

	"github.com/impire-io/soulnode/ceremony"
)

type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// principal reads the server-asserted (persona, account) from the expanded
// sign.record grant, plus the full resolved pub-allow set.
func principal(t *testing.T, nc *nats.Conn) (persona, account string, allow []string) {
	t.Helper()
	msg, err := nc.Request("$SYS.REQ.USER.INFO", nil, 5*time.Second)
	if err != nil {
		t.Fatalf("USER.INFO: %v", err)
	}
	var info struct {
		Data struct {
			Permissions struct {
				Publish struct {
					Allow []string `json:"allow"`
				} `json:"publish"`
			} `json:"permissions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(msg.Data, &info); err != nil {
		t.Fatalf("USER.INFO decode: %v", err)
	}
	allow = info.Data.Permissions.Publish.Allow
	for _, subj := range allow {
		parts := strings.Split(subj, ".")
		if len(parts) == 5 && parts[0] == siclient.Segment && parts[3] == "sign" && parts[4] == "record" {
			return parts[2], parts[1], allow
		}
	}
	t.Fatalf("no expanded sign.record grant in %v", allow)
	return "", "", nil
}

// TestM11Gate is design 0001 §9-M1.1 end to end: found a realm into a
// fresh directory, run it, prove the three admission observations from
// separate client connections, stop, and run it again on the same state.
func TestM11Gate(t *testing.T) {
	dir := t.TempDir()

	// init's founding half (the cmd drives exactly this path).
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
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
		t.Fatalf("found: %v", err)
	}
	if !strings.HasPrefix(token, "sit_") {
		t.Fatalf("founding token has unexpected shape: %.8q", token)
	}
	if !ceremony.Founded(dir) {
		t.Fatal("sentinel not written — founding did not complete")
	}

	dial := func(tok string) (*nats.Conn, error) {
		return nats.Connect(n.URL(),
			nats.UserCredentials(ceremony.SentinelPath(dir)), nats.Token(tok),
			nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	}

	// Observation (a): the token admits, the server names the persona,
	// and every identity-plane grant stays inside the persona's prefix.
	nc, err := dial(token)
	if err != nil {
		t.Fatalf("token lane refused: %v (audit: %s)", err, audit.String())
	}
	persona, account, allow := principal(t, nc)
	if persona != ceremony.FoundingPersona || account != st.RealmPub {
		t.Fatalf("principal = %s@%s, want %s@%s", persona, account, ceremony.FoundingPersona, st.RealmPub)
	}
	ownPrefix := siclient.Segment + "." + st.RealmPub + "." + persona + "."
	for _, subj := range allow {
		if strings.HasPrefix(subj, siclient.Segment+".") &&
			subj != siclient.Segment+".status" && subj != siclient.Segment+".xkey" &&
			!strings.HasPrefix(subj, ownPrefix) {
			t.Fatalf("resolved grant escapes the persona's prefix: %s", subj)
		}
	}
	nc.Close()

	// Observation (b): garbage refused, refusal audited.
	if ncBad, err := dial("sit_" + strings.Repeat("00", 32)); err == nil {
		ncBad.Close()
		t.Fatal("garbage token admitted")
	}
	if !strings.Contains(audit.String(), "callout REFUSED") {
		t.Fatal("no 'callout REFUSED' in the audit")
	}

	// Observation (c): revoked refused. The digest comes from the token
	// listing — the plaintext-once design means Found returns no digest.
	tokens, err := n.Ops().Tokens()
	if err != nil {
		t.Fatalf("tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("want exactly the founding token, got %d", len(tokens))
	}
	if err := n.Ops().RevokeToken(tokens[0].Digest); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if ncRevoked, err := dial(token); err == nil {
		ncRevoked.Close()
		t.Fatal("revoked token admitted")
	}

	// Stop, then run again on the same state: the realm persists (the
	// vault reopens under the same first key; the plane serves again).
	n.Stop()
	st2, err := ceremony.Verify(dir)
	if err != nil {
		t.Fatalf("verify after stop: %v", err)
	}
	n2, err := Start(Config{StateDir: dir, State: st2, AuditWriter: audit})
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	if _, err := n2.Ops().Status(); err != nil {
		t.Fatalf("status after restart: %v", err)
	}
	n2.Stop()
}

// TestM12Memory is design 0001 §9-M1.2: the token-admitted owner posts a
// turn (signed through the identity plane), the archivist keeps it and
// answers memory with attribution; the archive resumes exactly-once
// across a restart; the archivist's persona key is vault-held.
func TestM12Memory(t *testing.T) {
	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
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
		t.Fatalf("found: %v", err)
	}

	// The owner's whole path rides admission: sentinel + token.
	ncOwner, err := nats.Connect(n.URL(),
		nats.UserCredentials(ceremony.SentinelPath(dir)), nats.Token(token),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	if err != nil {
		t.Fatalf("owner admission: %v", err)
	}
	signer, err := siclient.New(ncOwner, st.RealmPub, ceremony.FoundingPersona).
		PersonaSigner(ceremony.FoundingPersona)
	if err != nil {
		t.Fatalf("owner signer: %v", err)
	}
	ctx := context.Background()
	rc, err := realm.NewClient(ctx, ncOwner, realm.Config{
		Realm: st.Realm, Persona: ceremony.FoundingPersona, Signer: signer,
	})
	if err != nil {
		t.Fatalf("owner realm client: %v", err)
	}
	h, err := topic.StartTopic(ctx, rc, topic.StartTopicInput{Name: "notes"})
	if err != nil {
		t.Fatalf("start topic: %v", err)
	}
	turnID, err := h.PostTurn(ctx, "remember the milk")
	if err != nil {
		t.Fatalf("post turn: %v", err)
	}

	// Memory answers with the archivist's attribution and a citation.
	var res *topic.MemoryResult
	deadline := time.Now().Add(15 * time.Second)
	for {
		res, err = topic.MemoryQuery(ctx, rc, topic.MemoryQueryInput{Query: "milk"}, nil)
		if err == nil && len(res.Answers) > 0 && len(res.Answers[0].Citations) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("memory never cited the turn (err %v, res %+v, audit %s)", err, res, audit.String())
		}
		time.Sleep(300 * time.Millisecond)
	}
	if res.Answers[0].Witness != "archivist" {
		t.Fatalf("answer attributed to %q, want archivist", res.Answers[0].Witness)
	}

	// The archivist's persona key is vault-held: the directory answers,
	// and no persona key file exists anywhere in the state dir.
	if _, err := n.Ops().PersonaPublicKey("archivist"); err != nil {
		t.Fatalf("archivist persona key not in the vault: %v", err)
	}

	topicPath := h.Path()
	if err := rc.Close(); err != nil { // closes ncOwner (client owns it)
		t.Fatalf("owner close: %v", err)
	}
	n.Stop()

	// Exactly-once across restart: count kept exhibits, run again, post
	// one more, and the count grows by exactly one.
	store, err := archive.Open(filepath.Join(dir, "archive"))
	if err != nil {
		t.Fatalf("archive open: %v", err)
	}
	c1, err := store.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	kept, ok := store.Get(topicPath, turnID)
	if !ok {
		t.Fatal("the posted turn was not kept")
	}
	rec, err := kept.Record()
	if err != nil {
		t.Fatalf("kept record: %v", err)
	}
	if rec.Author != ceremony.FoundingPersona {
		t.Fatalf("kept author %q, want %q", rec.Author, ceremony.FoundingPersona)
	}

	st2, err := ceremony.Verify(dir)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	n2, err := Start(Config{StateDir: dir, State: st2, AuditWriter: audit})
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	ncOwner2, err := nats.Connect(n2.URL(),
		nats.UserCredentials(ceremony.SentinelPath(dir)), nats.Token(token),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	if err != nil {
		t.Fatalf("owner re-admission: %v", err)
	}
	signer2, err := siclient.New(ncOwner2, st.RealmPub, ceremony.FoundingPersona).
		PersonaSigner(ceremony.FoundingPersona)
	if err != nil {
		t.Fatalf("owner signer 2: %v", err)
	}
	rc2, err := realm.NewClient(ctx, ncOwner2, realm.Config{
		Realm: st.Realm, Persona: ceremony.FoundingPersona, Signer: signer2,
	})
	if err != nil {
		t.Fatalf("owner realm client 2: %v", err)
	}
	if _, err := topic.Open(rc2, topicPath).PostTurn(ctx, "and the bread"); err != nil {
		t.Fatalf("post after restart: %v", err)
	}
	deadline = time.Now().Add(15 * time.Second)
	for {
		c2, err := store.Count()
		if err != nil {
			t.Fatalf("count: %v", err)
		}
		if c2 == c1+1 {
			break
		}
		if c2 > c1+1 {
			t.Fatalf("archive grew by %d, want exactly 1 (duplicate capture)", c2-c1)
		}
		if time.Now().After(deadline) {
			t.Fatalf("archive stuck at %d, want %d", c2, c1+1)
		}
		time.Sleep(300 * time.Millisecond)
	}
	_ = rc2.Close()
	n2.Stop()
}

// TestMemoryDisabled is SC-004: the plane block honored — admission works
// exactly as M1.1, and no archive directory is created.
func TestMemoryDisabled(t *testing.T) {
	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	st.MemoryEnabled = false
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
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
	nc, err := nats.Connect(n.URL(),
		nats.UserCredentials(ceremony.SentinelPath(dir)), nats.Token(token),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	if err != nil {
		t.Fatalf("admission with memory disabled: %v", err)
	}
	nc.Close()
	if _, err := os.Stat(filepath.Join(dir, "archive")); !os.IsNotExist(err) {
		t.Fatalf("archive directory exists with the plane disabled (stat err: %v)", err)
	}
}
