package topic

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/impire-io/soulstream/identity"
	"github.com/impire-io/soulstream/realm"
	"github.com/impire-io/soulstream/record"
)

// rawMessages reads every message currently on subject (headers + payload + subject),
// bypassing all topic-layer machinery, so signature checks see exactly the wire form.
func rawMessages(t *testing.T, c *realm.Client, subject string) []struct {
	Subject string
	Rec     record.Record
} {
	t.Helper()
	ctx := context.Background()

	stream, err := c.JetStream().Stream(ctx, realm.StreamName)
	if err != nil {
		t.Fatalf("look up stream: %v", err)
	}
	if _, err := stream.GetLastMsgForSubject(ctx, subject); err != nil {
		if errors.Is(err, jetstream.ErrMsgNotFound) {
			return nil
		}
		t.Fatalf("probe %s: %v", subject, err)
	}
	cons, err := stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{subject},
		DeliverPolicy:  jetstream.DeliverAllPolicy,
	})
	if err != nil {
		t.Fatalf("consumer: %v", err)
	}
	it, err := cons.Messages()
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	defer it.Stop()

	var out []struct {
		Subject string
		Rec     record.Record
	}
	for {
		msg, err := it.Next()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		rec, perr := record.Parse(msg.Headers(), msg.Data())
		if perr != nil {
			t.Fatalf("parse %s: %v", msg.Subject(), perr)
		}
		out = append(out, struct {
			Subject string
			Rec     record.Record
		}{msg.Subject(), rec})
		md, err := msg.Metadata()
		if err != nil {
			t.Fatalf("metadata: %v", err)
		}
		if md.NumPending == 0 {
			return out
		}
	}
}

// verifyWire checks a wire record's signature offline: only the record, the subject it
// arrived on, and the public key — no server, no directory (SC-001, FR-014).
func verifyWire(rec record.Record, subject, pubKey string) bool {
	unsigned := rec
	unsigned.Signature = ""
	canonical, err := unsigned.Canonical("test-realm", canonicalBinding(subject))
	if err != nil {
		return false
	}
	return identity.VerifySignature(pubKey, canonical, rec.Signature)
}

// TestSignedPersonaSignsEveryOpFamily is US1's independent test: a key-configured
// persona publishes every op family and each wire record carries a verifying
// signature (SC-002), checked out-of-band with no registry anywhere.
func TestSignedPersonaSignsEveryOpFamily(t *testing.T) {
	ctx := context.Background()
	key, err := identity.GenerateSigningKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	url := testServer(t)
	c := connectClientSigned(t, url, "signer", key)
	if _, err := c.Provision(ctx); err != nil {
		t.Fatalf("provision: %v", err)
	}

	h, err := StartTopic(ctx, c, StartTopicInput{Name: "signed", SubjectMatter: "signing e2e"})
	if err != nil {
		t.Fatalf("start topic: %v", err)
	}
	turnID, err := h.PostTurn(ctx, "hello @reader")
	if err != nil {
		t.Fatalf("post turn: %v", err)
	}
	if _, err := h.AddComment(ctx, "a comment", turnID); err != nil {
		t.Fatalf("add comment: %v", err)
	}
	if _, err := h.Attach(ctx, "note.txt", "text/plain", []byte("attachment body"), turnID); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if _, err := h.Transition(ctx, Active); err != nil {
		t.Fatalf("transition: %v", err)
	}

	// Every op on every subject family must verify: ops, info (announce), notify
	// (the @reader mention fired one).
	families := []string{OpsSubject(h.Path()), InfoSubject(h.Path()), NotifySubject("reader")}
	total := 0
	for _, subject := range families {
		msgs := rawMessages(t, c, subject)
		if len(msgs) == 0 {
			t.Fatalf("no messages on %s", subject)
		}
		for _, m := range msgs {
			total++
			if m.Rec.Signature == "" {
				t.Errorf("%s %s: unsigned op from a key-configured persona", m.Subject, m.Rec.Type)
				continue
			}
			if !verifyWire(m.Rec, m.Subject, key.PublicKey()) {
				t.Errorf("%s %s: signature does not verify", m.Subject, m.Rec.Type)
			}
		}
	}
	// announce + baseline + turn + comment + attachment + transition + notify = 7
	if total != 7 {
		t.Errorf("op count = %d, want 7 (announce, baseline, turn, comment, attachment, transition, notify)", total)
	}
}

// TestTamperingBreaksTheSignature is the SC-003 matrix: altering any canonical field
// of a signed op makes verification fail, including cross-topic and cross-realm
// splicing (US1 scenarios 4–5).
func TestTamperingBreaksTheSignature(t *testing.T) {
	key, err := identity.GenerateSigningKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	rec := record.Record{
		ID:        record.NewID(),
		Author:    "signer",
		Parents:   []string{record.NewID()},
		Type:      TypeTurnPost,
		Timestamp: time.Now().UTC().Truncate(time.Second),
		Payload:   []byte(`{"body":"the truth"}`),
	}
	canonical, err := rec.Canonical("test-realm", "a-topic")
	if err != nil {
		t.Fatalf("canonical: %v", err)
	}
	rec.Signature = key.Sign(canonical)

	verify := func(r record.Record, realmName, binding string) bool {
		unsigned := r
		unsigned.Signature = ""
		c, err := unsigned.Canonical(realmName, binding)
		if err != nil {
			return false
		}
		return identity.VerifySignature(key.PublicKey(), c, r.Signature)
	}

	if !verify(rec, "test-realm", "a-topic") {
		t.Fatal("untampered record must verify")
	}

	tampers := map[string]func(r *record.Record) (realmName, binding string){
		"author": func(r *record.Record) (string, string) { r.Author = "mallory"; return "test-realm", "a-topic" },
		"payload": func(r *record.Record) (string, string) {
			r.Payload = []byte(`{"body":"a lie"}`)
			return "test-realm", "a-topic"
		},
		"parents": func(r *record.Record) (string, string) { r.Parents = nil; return "test-realm", "a-topic" },
		"ts": func(r *record.Record) (string, string) {
			r.Timestamp = r.Timestamp.Add(time.Hour)
			return "test-realm", "a-topic"
		},
		"type":        func(r *record.Record) (string, string) { r.Type = TypeCommentAdd; return "test-realm", "a-topic" },
		"id":          func(r *record.Record) (string, string) { r.ID = record.NewID(); return "test-realm", "a-topic" },
		"cross-topic": func(_ *record.Record) (string, string) { return "test-realm", "other-topic" },
		"cross-realm": func(_ *record.Record) (string, string) { return "other-realm", "a-topic" },
	}
	for name, tamper := range tampers {
		tampered := rec
		realmName, binding := tamper(&tampered)
		if verify(tampered, realmName, binding) {
			t.Errorf("tampering %s: still verifies", name)
		}
	}
}

// TestNoSignerPublishesUnsigned is US1 scenario 3: a persona with no key publishes
// exactly as before — no Soulstream-Sig header, no error.
func TestNoSignerPublishesUnsigned(t *testing.T) {
	ctx := context.Background()
	c := provisionedClient(t, "plain")

	h, err := StartTopic(ctx, c, StartTopicInput{Name: "unsigned", SubjectMatter: "no key"})
	if err != nil {
		t.Fatalf("start topic: %v", err)
	}
	if _, err := h.PostTurn(ctx, "no signature here"); err != nil {
		t.Fatalf("post turn: %v", err)
	}

	for _, subject := range []string{OpsSubject(h.Path()), InfoSubject(h.Path())} {
		for _, m := range rawMessages(t, c, subject) {
			if m.Rec.Signature != "" {
				t.Errorf("%s %s: signed op from a key-less persona", m.Subject, m.Rec.Type)
			}
		}
	}
}

// TestCanonicalBinding pins the binding rule readers rely on to recompute signing
// input from the subject alone.
func TestCanonicalBinding(t *testing.T) {
	cases := []struct{ subject, want string }{
		{"SOULSTREAM.TOPICS.OPS.vat-q2-x7m2", "vat-q2-x7m2"},
		{"SOULSTREAM.TOPICS.OPS.parent-a1b2.child-c3d4", "parent-a1b2.child-c3d4"},
		{"SOULSTREAM.TOPICS.INFO.vat-q2-x7m2", "vat-q2-x7m2"},
		{"SOULSTREAM.PERSONA.NOTIFY.architect", "architect"},
		{"SOULSTREAM.SVC.DISCOVER", "DISCOVER"},
		{"SOMETHING.ELSE", ""},
	}
	for _, c := range cases {
		if got := canonicalBinding(c.subject); got != c.want {
			t.Errorf("canonicalBinding(%q) = %q, want %q", c.subject, got, c.want)
		}
	}
}

// TestSignedTurnVerifiesAgainstPublicKey is US1 scenario 1 in its smallest form,
// and a marshalling sanity check that the payload round-trips into canonical form.
func TestSignedTurnVerifiesAgainstPublicKey(t *testing.T) {
	ctx := context.Background()
	key, err := identity.GenerateSigningKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	url := testServer(t)
	c := connectClientSigned(t, url, "signer", key)
	if _, err := c.Provision(ctx); err != nil {
		t.Fatalf("provision: %v", err)
	}
	h, err := StartTopic(ctx, c, StartTopicInput{Name: "one turn", SubjectMatter: "s"})
	if err != nil {
		t.Fatalf("start topic: %v", err)
	}
	if _, err := h.PostTurn(ctx, "signed statement"); err != nil {
		t.Fatalf("post turn: %v", err)
	}

	for _, m := range rawMessages(t, c, OpsSubject(h.Path())) {
		if m.Rec.Type != TypeTurnPost {
			continue
		}
		var tp TurnPayload
		if err := json.Unmarshal(m.Rec.Payload, &tp); err != nil {
			t.Fatalf("unmarshal turn payload: %v", err)
		}
		if tp.Body != "signed statement" {
			t.Errorf("body = %q", tp.Body)
		}
		if !verifyWire(m.Rec, m.Subject, key.PublicKey()) {
			t.Error("turn signature does not verify against the persona's public key")
		}
		return
	}
	t.Fatal("no turn.post found")
}
