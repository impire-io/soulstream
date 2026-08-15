# Tasks: Wrap in the house

## Format: `[ID] [P?] Description`

- [x] T001 `cmd/soulstream/wrap.go`: the five-name env lane; `cmdWrap`
  (flags mirroring soulstream-wrap; door = own executable + ["mcp"];
  loud connect refusal) and `cmdMCP` (env-first flags, stdio server).
- [x] T002 `cmd/soulstream/main.go`: dispatch + usage lines for both
  verbs; `go.mod` bumps workloads to v0.4.0.
- [x] T003 [P] Refusal tests in `main_test.go` (missing variables,
  unknown preset, usage carries the verbs).
- [x] T004 `docs/getting-started.md`: steps 6–7 become download-and-
  paste; no `go install` anywhere in the file.
- [ ] T005 Gate green (`make check`); live SC-002 recorded; merge; hq
  episode/roadmap ride the arc's landing.
