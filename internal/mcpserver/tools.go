package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire/soulstream/topic"
)

// openTopic opens and materialises a handle so posts parent onto the current tip and
// closed-topic warnings fire.
func (h *handlers) openTopic(ctx context.Context, path string) (*topic.Handle, error) {
	th := topic.Open(h.c, path)
	if _, err := th.Materialise(ctx); err != nil {
		return nil, err
	}
	return th, nil
}

type boardInput struct{}

func (h *handlers) board(ctx context.Context, _ *mcp.CallToolRequest, _ boardInput) (*mcp.CallToolResult, any, error) {
	entries, err := topic.Board(ctx, h.c)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(entries)
}

type showTopicInput struct {
	Path string `json:"path" jsonschema:"the topic path to read"`
}

func (h *handlers) showTopic(ctx context.Context, _ *mcp.CallToolRequest, in showTopicInput) (*mcp.CallToolResult, any, error) {
	if in.Path == "" {
		return nil, nil, fmt.Errorf("path is required")
	}
	v, err := topic.Open(h.c, in.Path).Materialise(ctx)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(v)
}

type startTopicInput struct {
	Name    string   `json:"name" jsonschema:"the display name of the topic"`
	Subject string   `json:"subject,omitempty" jsonschema:"what the topic is about"`
	Tags    []string `json:"tags,omitempty" jsonschema:"tags for the topic"`
	Parent  string   `json:"parent,omitempty" jsonschema:"parent topic path, for a sub-topic"`
}

func (h *handlers) startTopic(ctx context.Context, _ *mcp.CallToolRequest, in startTopicInput) (*mcp.CallToolResult, any, error) {
	if in.Name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}
	th, err := topic.StartTopic(ctx, h.c, topic.StartTopicInput{
		Name: in.Name, SubjectMatter: in.Subject, Tags: in.Tags, Parent: in.Parent,
	})
	if err != nil {
		return nil, nil, err
	}
	return textResult(th.Path())
}

type postTurnInput struct {
	Path string `json:"path" jsonschema:"the topic path"`
	Body string `json:"body" jsonschema:"the message text; use @name to mention a persona"`
}

func (h *handlers) postTurn(ctx context.Context, _ *mcp.CallToolRequest, in postTurnInput) (*mcp.CallToolResult, any, error) {
	if in.Path == "" || in.Body == "" {
		return nil, nil, fmt.Errorf("path and body are required")
	}
	th, err := h.openTopic(ctx, in.Path)
	if err != nil {
		return nil, nil, err
	}
	id, err := th.PostTurn(ctx, in.Body)
	if err != nil {
		return nil, nil, err
	}
	return textResult(id)
}

type addCommentInput struct {
	Path       string `json:"path" jsonschema:"the topic path"`
	AnchorOpID string `json:"anchor_op_id" jsonschema:"the op-id this comment is anchored to"`
	Body       string `json:"body" jsonschema:"the comment text"`
}

func (h *handlers) addComment(ctx context.Context, _ *mcp.CallToolRequest, in addCommentInput) (*mcp.CallToolResult, any, error) {
	if in.Path == "" || in.AnchorOpID == "" || in.Body == "" {
		return nil, nil, fmt.Errorf("path, anchor_op_id and body are required")
	}
	th, err := h.openTopic(ctx, in.Path)
	if err != nil {
		return nil, nil, err
	}
	id, err := th.AddComment(ctx, in.Body, in.AnchorOpID)
	if err != nil {
		return nil, nil, err
	}
	return textResult(id)
}

type closeTopicInput struct {
	Path string `json:"path" jsonschema:"the topic path"`
}

func (h *handlers) closeTopic(ctx context.Context, _ *mcp.CallToolRequest, in closeTopicInput) (*mcp.CallToolResult, any, error) {
	if in.Path == "" {
		return nil, nil, fmt.Errorf("path is required")
	}
	th, err := h.openTopic(ctx, in.Path)
	if err != nil {
		return nil, nil, err
	}
	if _, err := th.Transition(ctx, topic.Closed); err != nil {
		return nil, nil, err
	}
	return textResult("closed " + in.Path)
}

type attachTextInput struct {
	Path        string `json:"path" jsonschema:"the topic path"`
	Name        string `json:"name" jsonschema:"the attachment's file name"`
	ContentType string `json:"content_type,omitempty" jsonschema:"the content type (default text/plain)"`
	Body        string `json:"body" jsonschema:"the text content to attach"`
}

func (h *handlers) attachText(ctx context.Context, _ *mcp.CallToolRequest, in attachTextInput) (*mcp.CallToolResult, any, error) {
	if in.Path == "" || in.Name == "" {
		return nil, nil, fmt.Errorf("path and name are required")
	}
	ct := in.ContentType
	if ct == "" {
		ct = "text/plain"
	}
	th, err := h.openTopic(ctx, in.Path)
	if err != nil {
		return nil, nil, err
	}
	opID, err := th.Attach(ctx, in.Name, ct, []byte(in.Body), "")
	if err != nil {
		return nil, nil, err
	}
	// Return the object key (looked up from the fresh view) for later retrieval.
	if v, err := th.Materialise(ctx); err == nil {
		for _, a := range v.Attachments {
			if a.OpID == opID {
				return textResult(a.Object)
			}
		}
	}
	return textResult(opID)
}

type checkInboxInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum notifications to return (default 50)"`
}

func (h *handlers) checkInbox(ctx context.Context, _ *mcp.CallToolRequest, in checkInboxInput) (*mcp.CallToolResult, any, error) {
	notes, err := topic.FetchInbox(ctx, h.c, h.c.Persona(), in.Limit)
	if err != nil {
		return nil, nil, err
	}
	if notes == nil {
		notes = []topic.Notification{}
	}
	return jsonResult(notes)
}
