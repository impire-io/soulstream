// Package mcpserver exposes Soulstream operations as MCP tools so an AI persona can
// participate through tool calls. The server is bound to one persona for its lifetime;
// every tool acts as that persona over the realm + topic library.
//
// Tool logic lives in handler methods on a struct holding the session client, so it is
// testable directly against an in-process server without stdio or a live MCP client.
package mcpserver

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/impire/soulstream/identity"
	"github.com/impire/soulstream/internal/keystore"
	"github.com/impire/soulstream/realm"
	"github.com/impire/soulstream/registry"
)

type handlers struct {
	c *realm.Client
}

func newHandlers(c *realm.Client) *handlers { return &handlers{c: c} }

// keyring builds the session's reader keyring per call: pins + directory → keyring,
// persisting extended pins. Rebuilt per read (not cached) so a rotation published
// mid-session is honoured; the directory is one KV list at dogfood scale. Every
// failure degrades to nil — verification never blocks a read.
func (h *handlers) keyring(ctx context.Context) *identity.Keyring {
	pinsPath, err := keystore.ResolvePinsFile("", h.c.Realm())
	if err != nil {
		return nil
	}
	pins, err := keystore.LoadPins(pinsPath, h.c.Realm())
	if err != nil {
		return nil
	}
	profiles, err := registry.All(ctx, h.c)
	if err != nil {
		return nil
	}
	if len(profiles) == 0 && len(pins.Personas) == 0 {
		return nil
	}
	kr, newPins := registry.BuildKeyring(profiles, pins.Personas)
	pins.Personas = newPins
	_ = keystore.SavePins(pinsPath, pins)
	return kr
}

// distrusted lists the keyring's distrusted personas, sorted — the loud surface an
// AI persona can act on (empty means all clear).
func distrusted(kr *identity.Keyring) []string {
	if kr == nil {
		return nil
	}
	var names []string
	for name, bad := range kr.Distrusted {
		if bad {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

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
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_rollup_topic",
		Description: "Compact a topic: fold its history into a fresh baseline. The conversation reads identically afterwards; replay just gets cheap.",
	}, h.rollupTopic)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "soulstream_discover",
		Description: "Ask the realm whether topics about something already exist; whoever is answering discovery replies from their own view. An empty result just means nobody answered — the board tool always works.",
	}, h.discover)

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
