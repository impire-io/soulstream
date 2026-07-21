# Quickstart: 013-config

## Per-project identity

```sh
# machine-wide default: the context you always use
mkdir -p ~/Library/"Application Support"/soulstream   # (macOS; Linux: ~/.config/soulstream)
cat > ~/Library/"Application Support"/soulstream/config.json <<'EOF'
{ "context": "personal" }
EOF

# a project that talks to the impire realm as daan
cd ~/impire/soulstream
cat > .soulstream.json <<'EOF'
{ "realm": "soulstream", "persona": "daan" }
EOF

soulstream config      # shows: context ← user file, realm+persona ← project file
soulstream board       # connects with exactly that identity — no flags, no env

# a different project, a different identity — just cd
cd ~/work/client-a     # has its own .soulstream.json naming another realm/persona
soulstream config
```

Precedence check: `SOULSTREAM_REALM=elsewhere soulstream config` shows realm coming
from the environment; add `--realm other` and the flag wins.

Failure check: add `"presona": "x"` to a config file — every command fails
immediately, naming the file and the unknown field.

## Self-installing plugin

```sh
# fresh machine simulation: nothing on PATH, empty data dir
unset SOULSTREAM_MCP_BIN
CLAUDE_PLUGIN_DATA=$(mktemp -d) plugins/soulstream/scripts/soulstream-mcp.sh -version
# → downloads soulstream_0.2.0_<os>_<arch>.tar.gz, verifies via checksums.txt,
#   caches, prints 0.2.0

# second run: cache hit, no network
CLAUDE_PLUGIN_DATA=<same dir> plugins/soulstream/scripts/soulstream-mcp.sh -version

# tamper check: corrupt the cached binary → next start re-downloads
```

In Claude Code: install the plugin, put a `.soulstream.json` in the project, connect —
the MCP server picks up the project's identity from its working directory.
