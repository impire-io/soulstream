// Package mcpserver exposes Soulstream operations as MCP tools so an AI persona can
// participate through tool calls. The server is bound to one persona for its lifetime;
// every tool acts as that persona over the realm + topic library.
//
// Tool logic lives in handler methods on a struct holding the session client, so it is
// testable directly against an in-process server without stdio or a live MCP client.
package mcpserver

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire/soulstream/realm"
)

type handlers struct {
	c *realm.Client
}

func newHandlers(c *realm.Client) *handlers { return &handlers{c: c} }

// NewServer builds an MCP server exposing the Soulstream tools, all acting as c's persona.
func NewServer(c *realm.Client) *mcp.Server {
	h := newHandlers(c)
	s := mcp.NewServer(&mcp.Implementation{Name: "soulstream", Version: "0.1.0"}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_board",
		Description: "List every topic on the realm's board (path, name, lifecycle).",
	}, h.board)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_show_topic",
		Description: "Read a topic: its announcement, contributions (with mentions), attachments, and lifecycle.",
	}, h.showTopic)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_start_topic",
		Description: "Start a new topic and return its path.",
	}, h.startTopic)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_post_turn",
		Description: "Post a turn to a topic. Use @name in the body to mention a persona.",
	}, h.postTurn)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_add_comment",
		Description: "Post a comment anchored to an operation.",
	}, h.addComment)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_close_topic",
		Description: "Close a topic (record a close transition).",
	}, h.closeTopic)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_attach_text",
		Description: "Attach text content to a topic; returns the attachment's object key.",
	}, h.attachText)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_check_inbox",
		Description: "Return your mention notifications (topic, op-id, author), newest first.",
	}, h.checkInbox)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_publish_profile",
		Description: "Publish or update your persona's directory profile (display metadata; includes your public signing key when this session holds one).",
	}, h.publishProfile)

	return s
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, nil, nil
}

func textResult(s string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}, nil, nil
}
