package mcpserver

import (
	"context"
	"strings"
	"testing"
)

func TestRollupTopicTool(t *testing.T) {
	h, _ := setup(t, "bookkeeper-agent")
	ctx := context.Background()

	res, _, err := h.startTopic(ctx, nil, startTopicInput{Name: "Compact Me"})
	if err != nil {
		t.Fatal(err)
	}
	path := strings.TrimSpace(resultText(t, res))
	for _, body := range []string{"a", "b", "c"} {
		if _, _, err := h.postTurn(ctx, nil, postTurnInput{Path: path, Body: body}); err != nil {
			t.Fatal(err)
		}
	}

	res, _, err = h.rollupTopic(ctx, nil, rollupTopicInput{Path: path})
	if err != nil {
		t.Fatalf("rollupTopic: %v", err)
	}
	if strings.TrimSpace(resultText(t, res)) == "" {
		t.Error("rollup returned no baseline op-id")
	}

	// The view survives compaction.
	res, _, err = h.showTopic(ctx, nil, showTopicInput{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	out := resultText(t, res)
	for _, body := range []string{"a", "b", "c"} {
		if !strings.Contains(out, `"body": "`+body+`"`) {
			t.Errorf("post-rollup show missing %q", body)
		}
	}

	// Already compact: friendly no-op.
	res, _, err = h.rollupTopic(ctx, nil, rollupTopicInput{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resultText(t, res), "nothing to compact") {
		t.Errorf("second rollup: %q", resultText(t, res))
	}

	if _, _, err := h.rollupTopic(ctx, nil, rollupTopicInput{}); err == nil {
		t.Error("rollup with empty path should error")
	}
}
