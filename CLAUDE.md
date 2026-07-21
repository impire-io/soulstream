<!-- SPECKIT START -->
Active feature: **013-config** — config-file identity resolution + self-installing
plugin binary. Five who-acts-where fields (context, realm, persona, key_file,
pins_file) resolve PER-FIELD through: flag (only if actually passed — flag.Visit;
flags no longer default to os.Getenv) > SOULSTREAM_* env > nearest `.soulstream.json`
walking up from cwd (nearest file ONLY, no stacking) > `<UserConfigDir>/soulstream/
config.json` > unset. New `internal/config` (stdlib only, no NATS): File/Resolve/
Resolved with per-field Source provenance. Strict JSON decode (DisallowUnknownFields)
— malformed/unknown field = fail loud naming the file; absent file = skip. Relative
key/pins paths resolve against the config file's dir at load. Config files can NEVER
carry credentials — names/paths only, keys stay in the local keystore. New CLI
`soulstream config`: prints field/value/source, never connects, exit 0. Wired into
internal/cli/Run AND cmd/soulstream-mcp. Byte-for-byte old behaviour when no files
exist. Plugin wrapper self-installs: SOULSTREAM_MCP_BIN > PATH > cached
$DATA/bin/v<ver>/soulstream-mcp (sha256 recorded at install, RE-VERIFIED every start)
> download release matching plugin's own version (parsed from plugin.json) for
uname-detected os/arch, verify vs checksums.txt, temp-dir + atomic mv (failed =
nothing cached), exec. $DATA = CLAUDE_PLUGIN_DATA else XDG_DATA_HOME else
~/.local/share, /soulstream-plugin. Plugin + marketplace 0.2.0; tag v0.2.0 ships in
the same delivery. ELI5: docs/configuration.md NEW; cli/mcp docs, plugin README,
setup skill updated.

For details read: [specs/013-config/plan.md](specs/013-config/plan.md)
(spec: `specs/013-config/spec.md`, contract: `specs/013-config/contracts/library.md`,
model: `specs/013-config/data-model.md`).
Done: `001`–`005` (MVP), `006-signing`, `007-rollup`, `008-discover`, `009-curator`,
`010-work`, `011-vocab`, `012-distribution` (plugin marketplace + goreleaser
pipeline + module rename, v0.1.0 released) merged + pushed.

Project conventions:
- Go 1.26; module `github.com/impire-io/soulstream`.
- `record` and `identity` import NO NATS; `realm`, `topic`, `registry`, `curator` are NATS-touching.
- Keep pure logic (folds, chain validation, discovery match/merge, curator judgment) separate
  from NATS I/O so it unit-tests with no server.
- Connect via `github.com/synadia-io/orbit.go/natscontext`; modern `nats.go/jetstream` API.
- Quality gate before every commit: `make fmt && make test && make lint` — all green, none skipped.
- Push to origin after merging a completed feature to main.
<!-- SPECKIT END -->
