# Scenario Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Make `nucleus scenario` a first-class CLI subcommand with contract-derived scenario planning, executable HTTP case drafts, runnable evidence, stable inputs, and verification behavior aligned with existing Nucleus commands.

**Architecture:** Keep Cobra command wiring, output rendering, HTTP execution, and evidence failure semantics in `cmd/nucleus/internal/scenario`. Reuse `github.com/nucleuskit/contract` for OpenAPI route metadata, error catalogs, and flow inspection without moving runtime execution into the contract module. Root command only wires `scenario.NewCommand`.

**Tech Stack:** Go 1.26, Cobra, standard-library HTTP testing/client APIs, existing `github.com/nucleuskit/contract/{errors,inspect,openapi}` packages.

---

### Task 1: Restore Module Correctness

**Files:**
- Modify: `cmd/nucleus/internal/scenario/scenario.go`
- Modify: `cmd/nucleus/internal/scenario/http_runner.go`

**Steps:**
1. Replace private import paths with `github.com/nucleuskit/contract/...`.
2. Run `rtk go test ./cmd/nucleus/internal/scenario`.
3. Run `rtk go test ./...` to confirm the root module compiles before deeper refactors.

### Task 2: Move Command Logic Out of Root

**Files:**
- Modify: `cmd/nucleus/internal/root/root.go`
- Create: `cmd/nucleus/internal/scenario/command.go`
- Create: `cmd/nucleus/internal/scenario/constants.go`
- Create: `cmd/nucleus/internal/scenario/run.go`
- Create: `cmd/nucleus/internal/scenario/output.go`
- Create: `cmd/nucleus/internal/scenario/doc.go`
- Create: `cmd/nucleus/internal/scenario/command_test.go`

**Steps:**
1. Add `Config{Dir *string}`, private `options`, `ErrScenarioFailed`, and `NewCommand`.
2. Support `--json`, `--pretty`, `--run-http`, `--base-url`, `--cases`, and `--draft-cases`.
3. Validate mutually exclusive modes and return helpful errors.
4. Render human summaries by default and JSON only with `--json`.
5. Return `ErrScenarioFailed` when runnable evidence has `pass:false`.
6. Add command-level tests for JSON plan, human output, flag validation, and failed evidence exit semantics.

### Task 3: Strengthen Scenario Generation

**Files:**
- Modify: `cmd/nucleus/internal/scenario/scenario.go`
- Modify: `cmd/nucleus/internal/scenario/http_runner.go`
- Modify: `cmd/nucleus/internal/scenario/cases.go`
- Modify: `cmd/nucleus/internal/scenario/scenario_test.go`
- Modify: `cmd/nucleus/internal/scenario/cases_test.go`

**Steps:**
1. Add Go doc for exported scenario types and functions.
2. Enrich route scenarios with sample values and invalid sample metadata.
3. Use contract schema examples/default/enum where available for request bodies.
4. Keep no hidden provider/runtime dependencies.
5. Ensure explicit cases support headers, JSON bodies, HTTP status assertions, JSON path assertions, and body contains assertions.

### Task 4: Integrate Evidence and Documentation

**Files:**
- Modify: `contract/schema/evidence.schema.json`
- Modify: `cmd/nucleus/internal/plan/constants.go`
- Modify: `cmd/nucleus/internal/plan/executable.go`
- Modify: `cmd/nucleus/internal/describe/constants.go`
- Modify: `cmd/nucleus/internal/describe/verification.go`
- Modify: `README.md`

**Steps:**
1. Allow scenario evidence as an accepted evidence kind without breaking verify evidence.
2. Add scenario command metadata to describe verification output where appropriate.
3. Document `nucleus scenario --json`, `--draft-cases`, and `--run-http`.
4. Add focused tests for policy/schema constants if existing tests require updates.

### Task 5: Verify

**Commands:**
- `rtk go test ./...`
- `rtk go test ./...` inside `contract`
- `rtk go run ./cmd/nucleus scenario --dir example/hello-http --json`
- `rtk go run ./cmd/nucleus scenario --dir example/hello-http --draft-cases --json`
- `rtk go run ./cmd/nucleus validate --dir .`
- `rtk go run ./cmd/nucleus lint --dir .`
- `rtk go run ./cmd/nucleus verify --dir . --json`

Record any skipped command with the concrete reason.
