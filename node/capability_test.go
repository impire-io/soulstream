package node

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"

	"github.com/impire-io/soulstream-core/realm"
	"github.com/impire-io/soulstream-core/topic"
	siclient "github.com/impire-io/soulstream-identity/client"

	"github.com/impire-io/soulstream/ceremony"
)

// TestM14CapabilityAgent (specs/013 SC-001): the full authority chain —
// founding writes the agent scoped signer, the runtime's scoped lane mints
// a permission-less tagged user, and the SERVER's template expansion is the
// entire policy. The probe is upstream's scope-probe: exit 0 (work done)
// means its granted subject passed AND an out-of-scope publish was denied;
// the second arm grants a different tool, so the probe's own subject is
// denied and the run ends abandoned — proof the narrowing bites.
func TestM14CapabilityAgent(t *testing.T) {
	probePath := filepath.Join(t.TempDir(), "scope-probe")
	build := exec.Command("go", "build", "-o", probePath,
		"github.com/impire-io/soulstream-workloads/cmd/scope-probe")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build scope-probe: %v\n%s", err, out)
	}

	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "home")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	st.MCPListen = "127.0.0.1:0"
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
	defer func() { _ = rc.Close() }()
	h, err := topic.StartTopic(ctx, rc, topic.StartTopicInput{Name: "capability-jobs"})
	if err != nil {
		t.Fatalf("start topic: %v", err)
	}

	run := func(persona, tool string) {
		t.Helper()
		decl := fmt.Sprintf(`{"role":"agent","lifecycle":"service","persona":%q,"topic":%q,"artifact":%q,"capabilities":{"role":%q,"tools":[%q]}}`,
			persona, h.Path(), "file://"+probePath, ceremony.AgentRole, tool)
		declPath := filepath.Join(t.TempDir(), persona+".json")
		if err := os.WriteFile(declPath, []byte(decl), 0o600); err != nil {
			t.Fatalf("write declaration: %v", err)
		}
		if err := RunWorkload(ctx, Config{StateDir: dir, State: st, AuditWriter: audit},
			n.URL(), declPath); err != nil {
			t.Fatalf("run workload %s: %v (audit: %s)", persona, err, audit.String())
		}
	}
	itemOf := func(persona string) topic.WorkItem {
		t.Helper()
		mt, err := h.Materialise(ctx)
		if err != nil {
			t.Fatalf("materialise: %v", err)
		}
		for _, w := range mt.WorkItems {
			if strings.HasSuffix(w.Title, "as "+persona) {
				return w
			}
		}
		t.Fatalf("no work item for %s; items: %+v", persona, mt.WorkItems)
		return topic.WorkItem{}
	}

	// Arm 1: the probe's own subject is granted → the probe verifies both
	// directions itself and exits 0 → the run completes done.
	run("prober", "probe-ping")
	if got := itemOf("prober").Status; string(got) != "done" {
		t.Fatalf("granted arm: work item status %q, want done", got)
	}

	// Arm 2: a different tool is granted → the probe's own subject is
	// DENIED at the server (the narrowing bites) → exit 2 → the runner
	// abandons. An abandon REOPENS the item (work.md semantics — fleet's
	// reclaim rides exactly this), so the proof is the timeline: an
	// abandon event by the runner, no done, owner cleared. Under the
	// plain wildcard lane this arm would have completed done.
	run("prober-narrow", "other-tool")
	item := itemOf("prober-narrow")
	if string(item.Status) == "done" {
		t.Fatal("narrowed arm completed done — the credential was not narrowed")
	}
	var abandoned bool
	for _, ev := range item.Timeline {
		if ev.Kind == "done" {
			t.Fatalf("narrowed arm carries a done event: %+v", item.Timeline)
		}
		if ev.Kind == "abandon" && ev.Author == "runner" {
			abandoned = true
		}
	}
	if !abandoned || item.Owner != "" {
		t.Fatalf("narrowed arm: want a runner abandon and a cleared owner, got timeline %+v owner %q", item.Timeline, item.Owner)
	}
}

// TestCapabilityPreflightRefusals (specs/013 SC-002): both refusals are
// named and fire before any connection exists — no server runs in this
// test at all.
func TestCapabilityPreflightRefusals(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "noop")
	if err := os.WriteFile(artifact, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("artifact: %v", err)
	}
	writeDecl := func(role string) string {
		decl := fmt.Sprintf(`{"role":"agent","lifecycle":"service","persona":"cap","topic":"t-ab12","artifact":%q,"capabilities":{"role":%q,"tools":["x"]}}`,
			"file://"+artifact, role)
		p := filepath.Join(t.TempDir(), "cap.json")
		if err := os.WriteFile(p, []byte(decl), 0o600); err != nil {
			t.Fatalf("write declaration: %v", err)
		}
		return p
	}

	// A realm founded before capability-minting: the agent key is absent.
	dir := t.TempDir()
	st, err := ceremony.Generate("127.0.0.1:0", "old")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if err := st.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, ceremony.AgentSigningFile)); err != nil {
		t.Fatalf("remove agent key: %v", err)
	}
	legacy, err := ceremony.Verify(dir)
	if err != nil {
		t.Fatalf("a realm without the agent key must still load: %v", err)
	}
	err = RunWorkload(context.Background(), Config{StateDir: dir, State: legacy},
		"nats://127.0.0.1:1", writeDecl(ceremony.AgentRole))
	if err == nil || !strings.Contains(err.Error(), "no agent capability key") {
		t.Fatalf("legacy realm refusal missing, got: %v", err)
	}

	// A foreign role name on a capability-bearing realm.
	err = RunWorkload(context.Background(), Config{StateDir: dir, State: st},
		"nats://127.0.0.1:1", writeDecl("ghost"))
	if err == nil || !strings.Contains(err.Error(), "one capability role") {
		t.Fatalf("foreign-role refusal missing, got: %v", err)
	}
}
