package node

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	siclient "github.com/impire-io/soulidentity/client"

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
	st, err := ceremony.Generate("127.0.0.1:0")
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
