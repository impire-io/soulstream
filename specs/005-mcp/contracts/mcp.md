# Contract: `soulstream-mcp` tools

Server: `mcp.NewServer(&mcp.Implementation{Name:"soulstream", Version:...})`, run over stdio, bound
to one configured persona. All tools map to `realm`/`topic`.

| Tool | Input (struct fields) | Result |
|---|---|---|
| `soulstream_board` | *(none)* | JSON: `[]BoardEntry` (path, name, lifecycle, parent). |
| `soulstream_show_topic` | `path` | JSON: the materialised view (announcement, contributions+mentions, attachments, lifecycle, malformed). |
| `soulstream_start_topic` | `name`, `subject?`, `tags?[]`, `parent?` | text: the new topic path. |
| `soulstream_post_turn` | `path`, `body` | text: the op-id (mentions parsed + notified by the library). |
| `soulstream_add_comment` | `path`, `anchor_op_id`, `body` | text: the op-id. |
| `soulstream_close_topic` | `path` | text: `closed <path>`. |
| `soulstream_attach_text` | `path`, `name`, `content_type?`, `body` | text: the attachment object key. |
| `soulstream_check_inbox` | `limit?` (default 50) | JSON: `[]Notification` (topic, op_id, author), newest-first. |

## Go surface

```go
package mcpserver

// NewServer builds an MCP server exposing the tools, all acting as c's persona.
func NewServer(c *realm.Client) *mcp.Server

// (internal) handlers hold the session client; each tool is a method:
//   func (h *handlers) board(ctx, *mcp.CallToolRequest, boardInput) (*mcp.CallToolResult, any, error)
//   ...tested directly against an embedded server.
```

New library helper:

```go
package topic
// FetchInbox returns the persona's mention notifications, newest-first, capped at limit
// (default 50 when limit <= 0); empty (no error) when there are none.
func FetchInbox(ctx context.Context, c *realm.Client, persona string, limit int) ([]Notification, error)
```

## Guarantees (→ spec)

- stdio server, typed tools, structured results (FR-001/013); one persona, every write attributed to
  it (FR-002/003); fail-fast startup (FR-004); the eight tools (FR-005…012); tool-level errors, no
  crash (FR-014); handlers testable without stdio (FR-015); pure library reuse (FR-016).
