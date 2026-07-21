package mcpserver

import (
	"context"
	"strings"
	"testing"
)

// TestUpkeepTools: reply, resolve, and edit through the MCP surface — including
// the same-author pre-check an agent hits before wasting a void op.
func TestUpkeepTools(t *testing.T) {
	h, url := setup(t, "bookkeeper-agent")
	ctx := context.Background()
	rival := newHandlers(clientOn(t, url, "rival-agent"))

	res, _, err := h.startTopic(ctx, nil, startTopicInput{Name: "plan"})
	if err != nil {
		t.Fatal(err)
	}
	path := strings.TrimSpace(resultText(t, res))

	res, _, err = h.postTurn(ctx, nil, postTurnInput{Path: path, Body: "lets ship thursdy"})
	if err != nil {
		t.Fatal(err)
	}
	turnID := strings.TrimSpace(resultText(t, res))

	// Edit own words.
	if _, _, err := h.edit(ctx, nil, editInput{Path: path, OpID: turnID, Body: "let's ship Thursday"}); err != nil {
		t.Fatal(err)
	}
	res, _, _ = h.showTopic(ctx, nil, showTopicInput{Path: path})
	if !strings.Contains(resultText(t, res), "let's ship Thursday") || !strings.Contains(resultText(t, res), `"edits"`) {
		t.Errorf("show after edit = %q", resultText(t, res))
	}

	// A rival's edit is refused up front with the reason.
	if _, _, err := rival.edit(ctx, nil, editInput{Path: path, OpID: turnID, Body: "Friday"}); err == nil ||
		!strings.Contains(err.Error(), "only the author may edit") {
		t.Errorf("foreign edit pre-check = %v", err)
	}

	// Comment → reply → resolve.
	res, _, err = rival.addComment(ctx, nil, addCommentInput{Path: path, AnchorOpID: turnID, Body: "which Thursday?"})
	if err != nil {
		t.Fatal(err)
	}
	cmntID := strings.TrimSpace(resultText(t, res))
	if _, _, err := h.replyComment(ctx, nil, replyCommentInput{Path: path, AnchorOpID: cmntID, Body: "the 30th"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.resolveComment(ctx, nil, resolveCommentInput{Path: path, OpID: cmntID}); err != nil {
		t.Fatal(err)
	}
	res, _, _ = h.showTopic(ctx, nil, showTopicInput{Path: path})
	out := resultText(t, res)
	if !strings.Contains(out, `"comment.reply"`) || !strings.Contains(out, `"resolved_by": "bookkeeper-agent"`) {
		t.Errorf("show after reply/resolve = %q", out)
	}

	// Input validation.
	if _, _, err := h.edit(ctx, nil, editInput{Path: path, OpID: turnID}); err == nil {
		t.Error("edit accepted an empty body")
	}
	if _, _, err := h.replyComment(ctx, nil, replyCommentInput{Path: path, Body: "x"}); err == nil {
		t.Error("reply accepted a missing anchor")
	}
}
