# Tasks: One console, one vocabulary

## Format: `[ID] [P?] Description`

- [x] T001 Go renames across ceremony/node/cmd (+tests): Door*→MCP*,
  Fold*→SignIn*, DoorURL/FoldURL→MCPURL/SignInURL.
- [x] T002 ceremony: dual-key config read (signin/mcp first, fold/door
  forever), new founds write signin.creds + user signin; refusal
  wording functional.
- [x] T003 node: creds fallback, sign-in plane console-off, log/label
  wording.
- [x] T004 cmd: --signin-listen/--mcp-listen with old spellings
  accepted; usage + printEndpoints functional; admin-console line gone.
- [x] T005 [P] Migration fixture test: legacy keys + fold.creds boots.
- [x] T006 Docs sweep (README, getting-started); bump idp v0.5.0 +
  shell v0.6.0; gate green; merge.
