package topic

import (
	"encoding/json"
	"testing"
)

func TestPayloadRoundTrip(t *testing.T) {
	t.Run("announce", func(t *testing.T) {
		in := AnnouncePayload{
			TopicID: "vat-x7m2", Name: "Q2 VAT", SubjectMatter: "filing",
			Expected: []string{"daan"}, Tags: []string{"finance"}, Parent: "",
		}
		var out AnnouncePayload
		remarshal(t, in, &out)
		if out.TopicID != in.TopicID || out.Name != in.Name || len(out.Tags) != 1 {
			t.Errorf("announce round-trip mismatch: %+v", out)
		}
	})

	t.Run("baseline", func(t *testing.T) {
		in := BaselinePayload{State: json.RawMessage(`{"k":1}`), Frontier: []string{"a", "b"}}
		var out BaselinePayload
		remarshal(t, in, &out)
		if string(out.State) != `{"k":1}` || len(out.Frontier) != 2 {
			t.Errorf("baseline round-trip mismatch: %+v", out)
		}
	})

	t.Run("turn", func(t *testing.T) {
		var out TurnPayload
		remarshal(t, TurnPayload{Body: "hello"}, &out)
		if out.Body != "hello" {
			t.Errorf("turn body = %q", out.Body)
		}
	})

	t.Run("comment", func(t *testing.T) {
		in := CommentPayload{Body: "noted", Anchor: Anchor{Kind: "op", OpID: "op-1"}}
		var out CommentPayload
		remarshal(t, in, &out)
		if out.Body != "noted" || out.Anchor.Kind != "op" || out.Anchor.OpID != "op-1" {
			t.Errorf("comment round-trip mismatch: %+v", out)
		}
	})

	t.Run("transition", func(t *testing.T) {
		var out TransitionPayload
		remarshal(t, TransitionPayload{To: Closed}, &out)
		if out.To != Closed {
			t.Errorf("transition To = %q", out.To)
		}
	})
}

func remarshal(t *testing.T, in, out any) {
	t.Helper()
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
