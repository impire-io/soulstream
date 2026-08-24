# Tasks: The presence lease (011)

- [x] T001 go.mod: soulstream-core v0.12.1 → v0.13.0 (the `presence`
      package arrives)
- [x] T002 cmd: wraplife.go — `ensureProfile` (lookup-first,
      warn-not-fatal, the honest no-signer floor) and `holdPresence`
      (goroutine around `presence.Hold`, returning the wait)
- [x] T003 cmd: wrap.go — cmdWrap wires announce + lease after
      Connect; waits for the farewell after Run returns, before the
      deferred Close
- [x] T004 cmd tests: wraplife_test.go — the live rig (founded node,
      sentinel + token, real persona scope): profile created when
      missing, untouched when rich; lease lands as `in`, farewells as
      `gone`; op-log census unchanged (US1 + US2 + US3.2) — PASS,
      0.3s
- [x] T005 quickstart.md: how to watch the lamp from a terminal; run
      record
- [x] T006 gate: make fmt && make test && make lint green; signed
      commits; hq journey episode + roadmap in the same working
      session
