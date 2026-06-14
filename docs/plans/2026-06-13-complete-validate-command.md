# Complete Validate Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build `nucleus validate` into a complete legality gate for service manifests and contract source files while preserving the existing CLI subcommand structure.

**Architecture:** Keep orchestration in `cmd/nucleus/internal/validate`, and put reusable parsing and validation rules in the `contract` module. The CLI should collect diagnostics from manifest, OpenAPI, proto, and error catalog validation, then render human-readable output by default and JSON evidence when requested.

**Tech Stack:** Go 1.26, Cobra, `go.yaml.in/yaml/v3`, existing `github.com/nucleuskit/contract/{manifest,openapi,proto,errors,inspect}` packages.

---

### Task 1: Define Contract Diagnostics

**Files:**
- Create: `contract/validation/diagnostic.go`
- Create: `contract/validation/diagnostic_test.go`

**Step 1: Write the failing tests**

Add tests for deterministic diagnostic ordering, severity counting, and failed status.

```go
package validation

import "testing"

func TestDiagnosticsFailedWhenErrorsExist(t *testing.T) {
	diagnostics := Diagnostics{
		{Severity: SeverityWarning, Code: "manifest.owner_missing"},
		{Severity: SeverityError, Code: "manifest.service_name_required"},
	}
	if !diagnostics.Failed() {
		t.Fatal("Failed() = false, want true")
	}
}

func TestDiagnosticsSortByPathThenCode(t *testing.T) {
	diagnostics := Diagnostics{
		{Path: "nucleus.yaml", Code: "b"},
		{Path: "api/openapi.yaml", Code: "z"},
		{Path: "nucleus.yaml", Code: "a"},
	}
	diagnostics.Sort()
	got := []string{diagnostics[0].Path + ":" + diagnostics[0].Code, diagnostics[1].Path + ":" + diagnostics[1].Code, diagnostics[2].Path + ":" + diagnostics[2].Code}
	want := []string{"api/openapi.yaml:z", "nucleus.yaml:a", "nucleus.yaml:b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
```

**Step 2: Run the tests to verify they fail**

Run: `rtk go test ./contract/validation`

Expected: FAIL because the package does not exist.

**Step 3: Implement the diagnostic model**

Create a small public package with English Go doc. Include fields that are stable enough for CLI JSON:

```go
// Package validation defines reusable diagnostics for contract and manifest validation.
package validation

import "sort"

// Severity classifies whether a diagnostic should fail validation.
type Severity string

const (
	// SeverityError marks a validation failure.
	SeverityError Severity = "error"
	// SeverityWarning marks a non-fatal validation concern.
	SeverityWarning Severity = "warning"
)

// Diagnostic describes one validation finding.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
}

// Diagnostics is a sortable collection of validation findings.
type Diagnostics []Diagnostic

// Failed reports whether any diagnostic is fatal.
func (diagnostics Diagnostics) Failed() bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Sort orders diagnostics deterministically for CLI output and tests.
func (diagnostics Diagnostics) Sort() {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path == diagnostics[j].Path {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Path < diagnostics[j].Path
	})
}
```

**Step 4: Run the tests**

Run: `rtk go test ./contract/validation`

Expected: PASS.

**Step 5: Commit**

```bash
git add contract/validation
git commit -m "feat: add validation diagnostics"
```

---

### Task 2: Move Manifest Validation To Diagnostics

**Files:**
- Modify: `contract/manifest/manifest.go`
- Create: `contract/manifest/manifest_test.go`

**Step 1: Write the failing tests**

Cover required fields and warnings for project-relevant metadata.

```go
func TestValidateReportsRequiredFields(t *testing.T) {
	diagnostics := Validate(Manifest{})
	assertDiagnostic(t, diagnostics, "manifest.schema_version_required")
	assertDiagnostic(t, diagnostics, "manifest.service_name_required")
	assertDiagnostic(t, diagnostics, "manifest.service_version_required")
}

func TestValidateWarnsForMissingAISurface(t *testing.T) {
	diagnostics := Validate(Manifest{
		SchemaVersion: "1.0",
		Service: Service{Name: "demo", Version: "0.1.0"},
	})
	assertDiagnostic(t, diagnostics, "manifest.ai_intent_missing")
}
```

**Step 2: Run the tests**

Run: `rtk go test ./contract/manifest`

Expected: FAIL until `Validate` returns structured diagnostics.

**Step 3: Update `Validate`**

Change `Validate` from `[]error` to `validation.Diagnostics`. Minimum rules:

- `schema_version` is required.
- `service.name` is required.
- `service.version` is required.
- `ai.intent` warning when empty.
- `ai.allowed_changes`, `ai.readonly`, `ai.forbidden`, and `ai.generated` entries must not be absolute paths or contain `..`.
- dependency entries require `name` and `contract`.
- duplicate capabilities warn or fail consistently.

Keep the rules in `contract/manifest`; do not put manifest semantics in the CLI.

**Step 4: Run tests**

Run: `rtk go test ./contract/manifest`

Expected: PASS.

**Step 5: Commit**

```bash
git add contract/manifest
git commit -m "feat: validate manifest diagnostics"
```

---

### Task 3: Add Error Catalog Validation

**Files:**
- Modify: `contract/errors/catalog.go`
- Create: `contract/errors/catalog_test.go`

**Step 1: Write the failing tests**

Use temp directories with `api/errors.yaml`.

```go
func TestValidateReportsMalformedErrorsYAML(t *testing.T) {
	dir := writeErrorsCatalog(t, "errors: [")
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "errors.parse_failed")
}

func TestValidateReportsDuplicateErrorCodes(t *testing.T) {
	dir := writeErrorsCatalog(t, "errors:\n  - code: 4001\n    message: invalid\n    http_status: 400\n  - code: 4001\n    message: duplicate\n    http_status: 400\n")
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "errors.code_duplicate")
}
```

**Step 2: Run tests**

Run: `rtk go test ./contract/errors`

Expected: FAIL until validation exists.

**Step 3: Implement `ValidateDir`**

Rules:

- Missing `api/errors.yaml` is OK unless manifest references it later.
- Malformed YAML is an error.
- Each error needs positive `code`, non-empty `message`, and valid `http_status` in `100..599`.
- Duplicate codes are errors.
- Duplicate messages are warnings or errors; choose error if messages are intended as stable identifiers.

**Step 4: Run tests**

Run: `rtk go test ./contract/errors`

Expected: PASS.

**Step 5: Commit**

```bash
git add contract/errors
git commit -m "feat: validate error catalog"
```

---

### Task 4: Add OpenAPI Contract Validation

**Files:**
- Modify: `contract/openapi/route.go`
- Create: `contract/openapi/validation_test.go`

**Step 1: Write failing tests**

Cover malformed YAML, duplicate operation IDs, path parameter mismatches, and missing responses.

```go
func TestValidateReportsMalformedOpenAPI(t *testing.T) {
	dir := writeOpenAPI(t, "openapi: [")
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "openapi.parse_failed")
}

func TestValidateReportsPathParameterWithoutDefinition(t *testing.T) {
	dir := writeOpenAPI(t, "openapi: 3.1.0\npaths:\n  /widgets/{id}:\n    get:\n      operationId: get_widget\n      responses:\n        \"200\": {description: ok}\n")
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "openapi.path_parameter_missing")
}
```

**Step 2: Run tests**

Run: `rtk go test ./contract/openapi`

Expected: FAIL until validation exists.

**Step 3: Implement `ValidateDir`**

Rules:

- Missing `api/openapi.yaml` is OK for non-HTTP services.
- Malformed YAML is an error.
- `paths` must be a map when file exists.
- Each operation should have `operationId`.
- `operationId` values must be unique.
- Every `{name}` in path must have a required `in: path` parameter with the same name.
- Every operation should define at least one response.
- HTTP methods remain restricted to existing known method fields.

**Step 4: Run tests**

Run: `rtk go test ./contract/openapi`

Expected: PASS.

**Step 5: Commit**

```bash
git add contract/openapi
git commit -m "feat: validate openapi contracts"
```

---

### Task 5: Add Proto Contract Validation

**Files:**
- Modify: `contract/proto/proto.go`
- Create: `contract/proto/proto_test.go`

**Step 1: Write failing tests**

Cover unreadable/malformed proto shape enough for the current lightweight parser.

```go
func TestValidateReportsUnbalancedServiceBlock(t *testing.T) {
	dir := writeProto(t, "greeting.proto", "syntax = \"proto3\";\nservice Greeting {\n  rpc SayHello (HelloRequest) returns (HelloResponse);\n")
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "proto.service_block_unclosed")
}

func TestValidateReportsMethodWithoutRequestOrResponse(t *testing.T) {
	dir := writeProto(t, "greeting.proto", "syntax = \"proto3\";\nservice Greeting {\n  rpc SayHello;\n}\n")
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "proto.rpc_invalid")
}
```

**Step 2: Run tests**

Run: `rtk go test ./contract/proto`

Expected: FAIL until validation exists.

**Step 3: Implement `ValidateDir`**

Rules:

- Missing `api/proto` is OK for non-gRPC services.
- Invalid read errors become diagnostics.
- Unbalanced service blocks are errors.
- `service` names should be unique per package.
- RPC declarations inside a service must parse into request and response names.
- `google.api.http` options with no HTTP verb become warnings or errors.

Do not add a heavy proto parser dependency without ADR; keep this lightweight for now.

**Step 4: Run tests**

Run: `rtk go test ./contract/proto`

Expected: PASS.

**Step 5: Commit**

```bash
git add contract/proto
git commit -m "feat: validate proto contracts"
```

---

### Task 6: Validate Cross-File References

**Files:**
- Create: `contract/validation/service.go`
- Create: `contract/validation/service_test.go`

**Step 1: Write failing tests**

Cover manifest dependencies that reference missing local contract files.

```go
func TestValidateServiceReportsMissingDependencyContractFile(t *testing.T) {
	dir := writeManifest(t, "schema_version: \"1.0\"\nservice:\n  name: demo\n  version: \"0.1.0\"\ndependencies:\n  - name: upstream\n    contract: api/missing.yaml#/paths/~1widgets\n    required: true\n")
	diagnostics := ValidateService(dir)
	assertDiagnostic(t, diagnostics, "dependency.contract_missing")
}
```

**Step 2: Run tests**

Run: `rtk go test ./contract/validation`

Expected: FAIL until orchestration exists.

**Step 3: Implement `ValidateService`**

Orchestrate:

- `manifest.Load` and `manifest.Validate`
- `openapi.ValidateDir`
- `errors.ValidateDir`
- `proto.ValidateDir`
- cross-reference checks based on manifest:
  - local `dependencies[].contract` path before `#` exists when non-empty
  - `capabilities` containing `http` should have OpenAPI or proto HTTP rules
  - `capabilities` containing `grpc` should have proto services
  - generated target paths in `ai.generated` should be relative and not forbidden

Return sorted `validation.Diagnostics`.

**Step 4: Run tests**

Run: `rtk go test ./contract/validation`

Expected: PASS.

**Step 5: Commit**

```bash
git add contract/validation
git commit -m "feat: validate service contracts"
```

---

### Task 7: Update The Validate CLI Renderer

**Files:**
- Modify: `cmd/nucleus/internal/validate/command.go`
- Modify: `cmd/nucleus/internal/validate/constants.go`
- Create: `cmd/nucleus/internal/validate/output.go`
- Create: `cmd/nucleus/internal/validate/output_test.go`

**Step 1: Write failing tests**

Test default output and JSON output without using the root command.

```go
func TestRenderHumanOutputIncludesDiagnostics(t *testing.T) {
	diagnostics := validation.Diagnostics{{Severity: validation.SeverityError, Code: "manifest.service_name_required", Path: "nucleus.yaml", Message: "service.name is required"}}
	var buffer bytes.Buffer
	err := renderHuman(&buffer, diagnostics)
	if err != nil {
		t.Fatalf("renderHuman() error = %v", err)
	}
	if !strings.Contains(buffer.String(), "manifest.service_name_required") {
		t.Fatalf("output = %q, want diagnostic code", buffer.String())
	}
}
```

**Step 2: Run tests**

Run: `rtk go test ./cmd/nucleus/internal/validate`

Expected: FAIL until renderer exists.

**Step 3: Implement CLI options and output**

Rules:

- Keep `cmd/nucleus/internal/validate` as the subcommand package.
- Rename copied constants: `commandUseValidate`, not `commandUseDescribe`.
- Remove unused `options` fields.
- Add `--json` and `--pretty` flags.
- Default success output: `OK`.
- Default failure output: one diagnostic per line, then return non-zero.
- JSON output shape:

```json
{
  "result_kind": "nucleus.validate_result",
  "ok": false,
  "diagnostics": []
}
```

**Step 4: Run tests**

Run: `rtk go test ./cmd/nucleus/internal/validate`

Expected: PASS.

**Step 5: Commit**

```bash
git add cmd/nucleus/internal/validate
git commit -m "feat: render validate diagnostics"
```

---

### Task 8: Add Root-Level Validate Integration Tests

**Files:**
- Modify: `cmd/nucleus/internal/root/describe_example_test.go`
- Or create: `cmd/nucleus/internal/root/validate_example_test.go`

**Step 1: Write failing tests**

Cover example success and bad fixture failure.

```go
func TestValidateCommandWithHelloHTTPExample(t *testing.T) {
	repoRoot := repositoryRoot(t)
	exampleDir := filepath.Join(repoRoot, "example", "hello-http")
	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--dir", exampleDir, "validate", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute validate: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode validate output: %v\n%s", err, stdout.String())
	}
	assertBool(t, output, "ok", true)
}
```

**Step 2: Run tests**

Run: `rtk go test ./cmd/nucleus/internal/root`

Expected: FAIL until CLI wiring and output are complete.

**Step 3: Implement any missing wiring**

Keep root structure:

- persistent `--dir` remains in root.
- validate-specific flags live only in `cmd/nucleus/internal/validate`.
- do not let validate import describe internals.

**Step 4: Run tests**

Run: `rtk go test ./cmd/nucleus/internal/root`

Expected: PASS.

**Step 5: Commit**

```bash
git add cmd/nucleus/internal/root
git commit -m "test: cover validate command integration"
```

---

### Task 9: Fix CLI Error Printing Behavior

**Files:**
- Modify: `cmd/nucleus/main.go`
- Create or modify: `cmd/nucleus/internal/root/root_test.go`

**Step 1: Write failing test or command check**

Use a malformed OpenAPI fixture and confirm the command emits diagnostics even though Cobra has `SilenceErrors`.

Run:

```bash
tmp=$(mktemp -d)
printf 'schema_version: "1.0"\nservice:\n  name: demo\n  version: "0.1.0"\n' > "$tmp/nucleus.yaml"
mkdir -p "$tmp/api"
printf 'openapi: [\n' > "$tmp/api/openapi.yaml"
rtk go run ./cmd/nucleus validate --dir "$tmp"
```

Expected: non-zero exit and visible diagnostic output.

**Step 2: Implement minimal behavior**

Prefer keeping `SilenceErrors` on root, and make `validate` render diagnostics before returning its failure sentinel. For unexpected internal errors from other commands, decide whether `main` should print to stderr. If changed, keep the behavior consistent for all commands and add a root-level test.

**Step 3: Run command check**

Run the same malformed fixture command.

Expected: diagnostic line includes `openapi.parse_failed`.

**Step 4: Commit**

```bash
git add cmd/nucleus
git commit -m "fix: print validate diagnostics"
```

---

### Task 10: Update Describe Verification Contract If Needed

**Files:**
- Modify: `cmd/nucleus/internal/describe/constants.go`
- Modify: `cmd/nucleus/internal/describe/verification.go`
- Modify: `cmd/nucleus/internal/describe/output_test.go`

**Step 1: Review expected pipeline**

Decide whether describe should advertise `nucleus validate --dir .` or `nucleus validate --dir . --json`.

Recommended:

- Keep human command in `commands`: `nucleus validate --dir .`
- Use JSON in machine-oriented pipeline if `verify` will consume evidence later: `nucleus validate --dir . --json`

**Step 2: Write/update tests**

Assert the verification pipeline includes validate and produces `nucleus.validate_result` if schema differentiates phase output.

**Step 3: Run tests**

Run: `rtk go test ./cmd/nucleus/internal/describe`

Expected: PASS.

**Step 4: Commit**

```bash
git add cmd/nucleus/internal/describe
git commit -m "docs: align describe validation contract"
```

---

### Task 11: Update Documentation

**Files:**
- Modify: `README.md`
- Modify: `CONTRIBUTING.md`
- Optionally modify: `docs/concepts/ai-first-microservice-kernel.md`

**Step 1: Add concise validate semantics**

Document the command boundary:

- `describe` emits service facts.
- `validate` checks manifest and contract legality.
- `lint` checks project conventions and risk rules.
- `verify` executes validation and test evidence.

**Step 2: Add sample commands**

Include:

```bash
go run ./cmd/nucleus validate --dir example/hello-http
go run ./cmd/nucleus validate --dir example/hello-http --json
```

**Step 3: Run documentation sanity checks**

Run: `rtk rg -n "nucleus validate|validate --dir|lint|verify" README.md CONTRIBUTING.md docs`

Expected: validate semantics are consistent.

**Step 4: Commit**

```bash
git add README.md CONTRIBUTING.md docs
git commit -m "docs: describe validate command scope"
```

---

### Task 12: Final Verification

**Files:**
- No new files expected.

**Step 1: Run focused tests**

```bash
rtk go test ./cmd/nucleus/internal/validate ./cmd/nucleus/internal/root
rtk go test ./contract/validation ./contract/manifest ./contract/openapi ./contract/errors ./contract/proto
```

Expected: PASS.

**Step 2: Run full tests**

```bash
rtk go test ./...
```

Expected: PASS.

**Step 3: Run CLI smoke checks**

```bash
rtk go run ./cmd/nucleus validate --dir example/hello-http
rtk go run ./cmd/nucleus validate --dir example/hello-http --json
rtk go run ./cmd/nucleus describe --dir example/hello-http --json --flow
```

Expected: validate succeeds; describe still emits metadata and verification contract.

**Step 4: Run project-required commands where available**

```bash
rtk go test ./... -race -count=1
rtk go run ./cmd/nucleus validate --dir .
rtk go run ./cmd/nucleus lint --dir .
rtk go run ./cmd/nucleus verify --dir . --json
```

Expected: race test passes. If `lint` or `verify` are still unimplemented, record that explicitly in the final PR notes.

**Step 5: Final commit or PR**

```bash
git status --short
git log --oneline -n 12
```

Expected: only intended validate-related changes remain.

