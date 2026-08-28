# Feature Specification: The thinking house

**Feature Branch**: `014-the-thinking-house`
**Created**: 2026-08-28
**Status**: Draft
**Input**: soul-hq designs [`0007-agents-as-infrastructure.md`](../../../soul-hq/02-DESIGN/soulstream-workloads/0007-agents-as-infrastructure.md) (§2 the serve seam, §3 the `inference` block, §5 the served agent's credentials) and [`0001-the-inference-plane.md`](../../../soul-hq/02-DESIGN/soulstream-inference/0001-the-inference-plane.md) (§5 the catalogue's home, §6 the door and its keys). Both designs left their last open questions to "the product wiring". This is that wiring: the house grows two opt-in planes — a **dispatcher** that serves declared agents as infrastructure, and an **inference** plane (door + in-process instances) that serves their thinking — plus the catalogue that gives a virtual model name a home.

## The two questions this spec answers

1. **The catalogue's home** (inference 0001 §5 [O]): a **realm KV bucket**, `soulstream-inference-catalogue`, created create-or-report by whichever plane starts, keyed by virtual name, values `{capability, model_pin, tags, default_params}`. Resolvers read it fresh per resolution — the dispatcher when it injects a lane, the door when it routes a request. Watching is a named [O], not a silent cache.
2. **The founding's role key for engine mints** (workloads 0007 §5 [O]): the founding already installs it. The realm signing key — the **canonical persona scope** — is imported into the identity vault as `realm` by `node.Found`, and `mint.ephemeral` against that role name is the served agent's engine credential. No new founding artifact, and every realm ever founded by this binary already carries it.

## The load-bearing finding (recorded openly)

The brief for this build said to mint the served agent's credential "exactly as `workload start` does today" — the plain workload lane. **That credential cannot run a wake engine.** Both narrow lanes (`minter`'s realm-semantic scope and the identity plane's agent scope) grant `$JS.API.INFO` and nothing else on the JetStream API, while the engine's every wake materialises the topic (catch-up, the exactly-once outcome check) and reads the persona's inbox — all of which are JetStream reads. On an operator-mode server the narrow credential is refused at the transport before the first wake completes.

Design 0007 §5 already recorded the measured shape and this build follows it: the engine's connection is a **D28 `mint.ephemeral` against the persona-scope role**. The narrow lanes keep their jobs — the plain lane still mints for `workload start`, the agent scope still narrows a declared agent's *tools* — but neither is an engine credential. The same finding decides the harness's tool door: it, too, reads topics, so it carries the same persona-scope credential (design 0007 §5's finding 3, filled).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - An agent is submitted, served, and thinks through the realm (Priority: P1)

An operator declares an agent with a wake set and an `inference` block naming a virtual model, points the catalogue at a served instance, and submits the declaration. The submitting process exits. When somebody mentions the agent, the house's dispatcher — already serving the placement — runs the harness with `ANTHROPIC_BASE_URL` and `ANTHROPIC_API_KEY` pointing at the house's own door, and the answer lands in the topic exactly once. The harness never sees a provider credential.

**Independent Test**: `TestM15ThinkingHouse` — found a realm, `up` with both planes and a stand-in instance, submit a declaration carrying `wake:[mention]` and `inference:{model:"realm-default"}`, close the submitter, post a mention, and read the topic.

**Acceptance Scenarios**:

1. **Given** a submitted declaration and a closed submitter, **When** a mention is posted, **Then** the agent answers exactly once and the answer carries the stand-in's report (the thinking really travelled door → plane → instance).
2. **Given** the served agent, **When** everything it can read is scanned — its declaration on the record, the environment its harness was constructed with, the outcome it posted — **Then** no provider material appears anywhere, while the plane's own instance holds the key it resolved from custody.
3. **Given** a revoked or never-issued door key, **When** a request presents it, **Then** the door answers 401 and **zero** messages reach the plane's subjects.
4. **Given** a served placement, **When** the node stops and starts again, **Then** the dispatcher resumes the serve from the log — no second claim, no duplicate answer.

---

### User Story 2 - The ambient lane is untouched (Priority: P1)

A declaration with no `inference` block serves exactly as it did before this feature: the harness authenticates the way it always has, and nothing in its environment mentions the door.

**Acceptance Scenarios**:

1. **Given** a declaration without `inference`, **When** it is served, **Then** the harness environment carries no `ANTHROPIC_BASE_URL` and no `ANTHROPIC_API_KEY`, and the agent answers.
2. **Given** a realm whose config declares neither plane, **When** it runs, **Then** nothing changes anywhere — no topic, no bucket, no listener, no connection.

---

### User Story 3 - The operator names models and providers (Priority: P2)

`soulstream model set realm-default --pin claude-sonnet-5` points a virtual name at a model; `soulstream model ls` shows the catalogue; re-pointing the name moves traffic with zero change to any declaration. `soulstream provider set anthropic` loads the provider key into the inference plane's own custody, where no agent scope can reach it.

**Acceptance Scenarios**:

1. **Given** a catalogue entry, **When** it is re-pointed to another instance's model, **Then** the next request routes to the newly named instance without any declaration changing.
2. **Given** `planes.inference` configured with an anthropic instance whose secret does not resolve, **When** `soulstream up` runs, **Then** it refuses by name and serves nothing — never half.

---

### Edge Cases

- A declaration carrying `inference` on a deployment with no inference plane: the placement is **refused whole** and handed back (never half-served), naming the missing plane.
- A virtual name absent from the catalogue: the dispatcher refuses the placement by name at serve time; at the door an unknown name falls back to the capability's anycast, which answers no-responders when nobody serves it — the routing layer telling the truth.
- The dispatcher plane on a realm founded before this feature: it needs no new artifact — the plane's own identity is minted at runtime from the workload signing key (`connectSys`'s posture) and the engine credential from the `realm` vault role the founding already imports.
- BYO realms: both planes are configuration; the minting material they use (the plain workload signing key, the `realm` vault role) exists on both flavours.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `config.json` MUST grow two **opt-in** plane blocks. `planes.dispatcher`: `enabled`, `placements` (topic name, default `placements`), `harness` (preset name) or `template` (file). `planes.inference`: `enabled`, `listen` (default `127.0.0.1:8600`), `instances[]` of `{adapter: standin|anthropic, model, capability, tags, secret}`. An absent block MUST change nothing anywhere.
- **FR-002**: Verify MUST refuse, by name: a non-loopback inference listener; a listener shared with another plane; a dispatcher with neither `harness` nor `template`; an instance with no adapter or model; an unknown adapter; an anthropic instance with no `secret`.
- **FR-003**: The dispatcher plane MUST run on a **runtime-minted** identity named `dispatcher` (signed by the plain workload signing key, `IssuerAccount` the realm account) — no new founding artifact, nothing on disk.
- **FR-004**: The plane MUST ensure the placement topic exists create-or-report, resolving the configured name against the realm's board and starting the topic only in its absence.
- **FR-005**: `ConnectAgent` MUST yield a client on a **persona-scope** credential minted through the identity plane's `mint.ephemeral` against the `realm` role, named for the served persona.
- **FR-006**: `EngineFor` MUST build one persona's engine config: the configured harness template with the tool door pointed at this binary's `mcp` verb carrying **that persona's** credential; and, when the served declaration carries `inference`, `Template.Env` MUST gain `ANTHROPIC_BASE_URL` (the house's door), `ANTHROPIC_API_KEY` (a key issued for this serve), and `ANTHROPIC_MODEL` (the virtual name).
- **FR-007**: The inference plane MUST serve `door.Door` on the configured listener with an in-process key registry as `Authorize` — a key exists only while a serve holds it — and a `Route` that resolves the requested model name through the catalogue, pinning when the descriptor pins and falling back to the capability's anycast for an unknown name.
- **FR-008**: The inference plane MUST start its configured instances in-process, resolving an anthropic instance's key from the plane principal's own D36 secret tree at plane start, and MUST fail `up` loudly when it cannot.
- **FR-009**: The catalogue MUST live in the realm KV bucket `soulstream-inference-catalogue`, created create-or-report, values JSON `{capability, model_pin, tags, default_params}` keyed by virtual name, read fresh per resolution.
- **FR-010**: `soulstream agent submit <declaration.json>` MUST submit through `fleet.Submit` on the caller's ordinary connection and print the placement id. `soulstream model set|ls` MUST write and read the catalogue. `soulstream provider set <name>` MUST write the plane principal's secret tree, reading the value from the environment, never a flag.
- **FR-011**: Node stop MUST drain the dispatcher (the engines' self-reports land) and revoke every issued door key.

## Success Criteria *(mandatory)*

- **SC-001**: `TestM15ThinkingHouse` — submit-and-forget with the submitter closed, exactly one answer carrying the stand-in's report, a custody scan clean of provider material, the keyless door refused with zero plane deliveries, and a restart that resumes the serve [measured in `make test`].
- **SC-002**: The no-`inference` arm serves with an environment free of every `ANTHROPIC_*` name [measured].
- **SC-003**: The disabled arm — a realm with neither block — creates no topic, no bucket, and no listener [measured].
- **SC-004**: The configuration refusals of FR-002 are named errors from `ceremony.Verify`, before anything starts [measured].
- **SC-005**: All existing gates green — `make check`; the new tests `-race -count=3`.

## Assumptions

- go.mod pins ride the mains of soulstream-workloads and soulstream-inference (the episode 0089 standing exception); they move to tags at the next rc.
- Door keys are **per serve**, not per wake: the per-wake mint needs the mint inside the engine's admission path, a workloads seam that does not exist yet. Recorded as an [O], not built.
- The engine credential's TTL is 24h with no renewal loop; an expiry ends the engine, the placement returns to the race, and the next serve mints fresh. Renewal stays design 0007 §5's [O].
</content>
</invoke>
