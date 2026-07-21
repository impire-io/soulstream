#!/usr/bin/env bash
# Launches the soulstream-mcp stdio server for the Claude Code plugin.
# The binary is platform-specific and therefore not bundled with the plugin;
# it must be installed separately (see the error text below).
set -euo pipefail

bin="${SOULSTREAM_MCP_BIN:-}"
if [[ -z "$bin" ]]; then
  if command -v soulstream-mcp >/dev/null 2>&1; then
    bin="soulstream-mcp"
  else
    cat >&2 <<'EOF'
soulstream-mcp not found on PATH (and SOULSTREAM_MCP_BIN is not set).

Install it one of these ways, then reconnect the MCP server:
  go install github.com/impire-io/soulstream/cmd/soulstream-mcp@latest
  # or download a binary from https://github.com/impire-io/soulstream/releases
  # or clone the repo and run `make build`

Run /soulstream:setup in Claude Code for a guided setup.
EOF
    exit 1
  fi
fi

exec "$bin" "$@"
