# Complete Gen Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `nucleus gen` a complete, testable CLI subcommand that follows the existing `describe`/`validate`/`lint` structure and delegates reusable generation to `github.com/nucleuskit/contract/gen`.

**Architecture:** Keep root command wiring in `cmd/nucleus/internal/root`, move command flags/output/orchestration to `cmd/nucleus/internal/gen`, and keep reusable contract artifact rendering in `contract/gen`. Generated artifacts must use public module paths, produce freshness markers, and be covered by CLI and contract-level tests.

**Tech Stack:** Go 1.26, Cobra, multi-module Go workspace, `github.com/nucleuskit/contract/*`.

---

### Task 1: CLI Package Shape

**Files:**
- Create: `cmd/nucleus/internal/gen/command.go`
- Create: `cmd/nucleus/internal/gen/constants.go`
- Create: `cmd/nucleus/internal/gen/output.go`
- Create: `cmd/nucleus/internal/gen/summary.go`
- Create: `cmd/nucleus/internal/gen/paths.go`
- Modify: `cmd/nucleus/internal/root/root.go`
- Test: `cmd/nucleus/internal/gen/output_test.go`
- Test: `cmd/nucleus/internal/gen/command_test.go`
- Test: `cmd/nucleus/internal/root/gen_example_test.go`

- [ ] Add failing tests for JSON, human output, validation failure, and root wiring.
- [ ] Move `gen` command construction out of `root.go`.
- [ ] Add `--json`, `--pretty`, `--http`, `--grpc`, `--errors`, `--clients`, `--client-language`, `--docs`, and `--typescript`.
- [ ] Emit `result_kind: "nucleus.gen_result"`, `ok`, `source_hash`, `files`, `summary`, and validation diagnostics.

### Task 2: Contract Generator Boundary

**Files:**
- Modify: `contract/gen/generate.go`
- Modify: `contract/gen/export.go`
- Delete: `contract/gen/errors.go`
- Delete: `contract/gen/endpoints.go`
- Delete: `contract/gen/contract_source.go`
- Test: `contract/gen/generate_test.go`
- Test: `contract/gen/export_test.go`

- [ ] Fix all module paths to `github.com/nucleuskit/*`.
- [x] Obsolete: generated HTTP binders must not import a Nucleus runtime package; they expose project-owned registrar interfaces instead.
- [ ] Keep reusable rendering/export logic in `contract/gen`; do not duplicate it in CLI root.
- [ ] Add Go doc for exported APIs.
- [ ] Ensure client export and optional generated directories can receive freshness markers from the CLI.

### Task 3: OpenAPI Schema Reliability

**Files:**
- Modify: `contract/openapi/schema.go`
- Test: `contract/openapi/schema_test.go`

- [ ] Add tests for local `$ref` resolution, circular references, examples/defaults/enums, and invalid refs.
- [ ] Keep schema parsing contract-focused and runtime-free.

### Task 4: Documentation And Example Alignment

**Files:**
- Modify: `README.md`
- Modify: `example/hello-http/nucleus.yaml`

- [ ] Document `nucleus gen` evidence output.
- [ ] Add generated targets that match actual generated directories.
- [ ] Keep public documentation in English.

### Task 5: Verification

- [ ] Run `rtk go test ./...` from the repository root.
- [ ] Run `rtk go test ./...` from `contract`.
- [ ] Run `rtk go run ./cmd/nucleus validate --dir example/hello-http`.
- [ ] Run `rtk go run ./cmd/nucleus gen --dir example/hello-http --json`.
- [ ] Run `rtk go run ./cmd/nucleus lint --dir example/hello-http --strict`.
