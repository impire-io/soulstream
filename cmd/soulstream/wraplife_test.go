package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/impire-io/soulstream-core/presence"
	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-core/registry"

	"github.com/impire-io/soulstream/ceremony"
	"github.com/impire-io/soulstream/node"
)

// The live rig for spec 011: a founded realm, the wrap lane's own
// admission (sentinel + token — the persona scope exactly as minted),
// and the wrap's housekeeping measured end to end. The directory floor
// fills absence and never speaks over a richer profile; the lamp lands
// as `in` and farewells as `gone`; and the op-log never sees any of it.
func TestWrapLifeAgainstFoundedRealm(t *testing.T) {
	dir := t.TempDir()
	// Ephemeral ports like every sibling rig: the fixed plane defaults
	// (8080/8378/8500) collide across packages under `go test ./...`.
	// The fold's issuer must be named at config time: reserve its port
	// (the foldplane rig's pattern), and let the cleared issuer derive.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	foldAddr := ln.Addr().String()
	_ = ln.Close()

	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	st.MCPListen = "127.0.0.1:0"
	st.HelmListen = "127.0.0.1:0"
	st.SignInListen = foldAddr
	st.SignInIssuer = ""
	st.SignInAudience = ""
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	st, err = ceremony.Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	n, err := node.Start(node.Config{StateDir: dir, State: st, AuditWriter: io.Discard})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer n.Stop()
	token, err := node.Found(n, st, dir)
	if err != nil {
		t.Fatalf("found: %v", err)
	}

	// The wrap's whole path rides admission: sentinel + token, nothing
	// else — the scope this test exists to exercise for real.
	nc, err := nats.Connect(n.URL(),
		nats.UserCredentials(ceremony.SentinelPath(dir)), nats.Token(token),
		nats.RetryOnFailedConnect(false), nats.MaxReconnects(0))
	if err != nil {
		t.Fatalf("admission: %v", err)
	}
	ctx := context.Background()
	client, err := realm.NewClient(ctx, nc, realm.Config{
		Realm: st.Realm, Persona: ceremony.FoundingPersona,
	})
	if err != nil {
		t.Fatalf("realm client: %v", err)
	}
	defer func() { _ = client.Close() }()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	stream, err := client.JetStream().Stream(ctx, realm.StreamName)
	if err != nil {
		t.Fatalf("op-log: %v", err)
	}
	before, err := stream.Info(ctx)
	if err != nil {
		t.Fatalf("op-log census: %v", err)
	}

	// The directory floor: after the announce a profile exists; a richer
	// profile published by the agent's own hand is never spoken over.
	ensureProfile(ctx, client, log)
	p, found, err := registry.Lookup(ctx, client, ceremony.FoundingPersona)
	if err != nil || !found {
		t.Fatalf("no profile after announce: found=%v %v", found, err)
	}
	if p.Name != ceremony.FoundingPersona {
		t.Fatalf("profile came back %+v", p)
	}
	rich := p
	rich.DisplayName = "The Founder"
	if err := registry.Publish(ctx, client, rich); err != nil {
		t.Fatalf("enrich profile: %v", err)
	}
	ensureProfile(ctx, client, log)
	p, _, err = registry.Lookup(ctx, client, ceremony.FoundingPersona)
	if err != nil || p.DisplayName != "The Founder" {
		t.Fatalf("the floor spoke over a richer profile: %+v %v", p, err)
	}

	// The lamp: lit while held — the first write creating the bucket
	// through this very admission — and `gone` once the wrap lets go.
	holdCtx, cancel := context.WithCancel(ctx)
	wait := holdPresence(holdCtx, client, log)
	deadline := time.Now().Add(5 * time.Second)
	for {
		s, found, err := presence.Lookup(ctx, client, ceremony.FoundingPersona)
		if err != nil {
			t.Fatalf("read the face: %v", err)
		}
		if found {
			if s.Entry.Status != presence.StatusIn {
				t.Fatalf("the lamp lit as %q", s.Entry.Status)
			}
			if r := s.Read(time.Now()); r.Word != presence.WordPresent {
				t.Fatalf("a held lease read as %q", r.Word)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the lamp never lit")
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	wait()
	s, found, err := presence.Lookup(ctx, client, ceremony.FoundingPersona)
	if err != nil || !found {
		t.Fatalf("read after farewell: found=%v %v", found, err)
	}
	if s.Entry.Status != presence.StatusGone {
		t.Fatalf("the farewell was not written: %+v", s.Entry)
	}

	// And the op-log never saw any of it.
	after, err := stream.Info(ctx)
	if err != nil {
		t.Fatalf("op-log census: %v", err)
	}
	if after.State.Msgs != before.State.Msgs {
		t.Fatalf("presence leaked into the op-log: %d before, %d after",
			before.State.Msgs, after.State.Msgs)
	}
}
