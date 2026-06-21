package repair

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildEvidenceRepairsMissingGeneratedEvidence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities:
  - http
`)
	writeFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n\nrequire github.com/nucleuskit/http v0.0.0\n\nrequire (\n\tgithub.com/nucleuskit/cap v0.1.0-alpha.2 // indirect\n\tgithub.com/nucleuskit/core v0.1.0-alpha.2 // indirect\n)\n\n"+localRuntimeReplace(t))
	writeFile(t, dir, "go.sum", `github.com/nucleuskit/cap v0.1.0-alpha.2 h1:UnBp5ezoi+UrgT92DwpTdQynrEnUNnsf3rQKB0pNfPw=
github.com/nucleuskit/cap v0.1.0-alpha.2/go.mod h1:DLSQmS/6irYQfpWGJjce9+STvqG33yZMCxkwvdLf7XM=
github.com/nucleuskit/core v0.1.0-alpha.2 h1:CT4RJvCYtVNo0+Gqyf5zx4T90dBiXq4JVhQxEKBXyJ8=
github.com/nucleuskit/core v0.1.0-alpha.2/go.mod h1:fwIlIS28wLh/VQ/jhR4TgrZcrN1nOgrAxSYYYWRdEYk=
`)
	writeFile(t, dir, "demo.go", "package demo\n")
	writeFile(t, dir, "internal/app/routes.go", `package app

import "net/http"

type Router struct{}

func (Router) Handle(method string, path string, handler func()) {}

func RegisterRoutes(router Router) {
	router.Handle(http.MethodGet, "/healthz", func() {})
}
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
      responses:
        "204":
          description: ok
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 1001
    message: ok
    http_status: 200
`)
	evidencePath := filepath.Join(dir, "missing-generated-evidence.json")
	writeFile(t, dir, "missing-generated-evidence.json", `{
  "kind": "nucleus.apply_evidence",
  "pass": false,
  "steps": [
    {
      "id": "missing_generated",
      "kind": "missing_generated",
      "pass": false,
      "path": "contract/gen/endpoints.go"
    }
  ]
}`)

	result, err := BuildEvidence(dir, evidencePath, 1)
	if err != nil {
		t.Fatalf("BuildEvidence returned error: %v", err)
	}
	if result["status"] != "repaired" {
		t.Fatalf("missing generated evidence should be repaired, got %#v", result)
	}
	if result["verification_pass"] != true {
		t.Fatalf("repair should verify after generation: %#v", result)
	}
	rounds := result["rounds"].([]map[string]any)
	if rounds[0]["strategy"] != "regenerate_missing_generated" {
		t.Fatalf("unexpected repair strategy: %#v", rounds[0])
	}
	if _, err := os.Stat(filepath.Join(dir, "contract", "gen", "endpoints.go")); err != nil {
		t.Fatalf("repair should generate endpoint metadata: %v", err)
	}
}

func TestBuildEvidenceDoesNotRepairVagueMissingGeneratedReason(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`)
	writeFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	writeFile(t, dir, "demo.go", "package demo\n")
	writeFile(t, dir, "api/openapi.yaml", "openapi: 3.0.3\npaths: {}\n")
	writeFile(t, dir, "api/errors.yaml", "errors: []\n")
	evidencePath := filepath.Join(dir, "vague-evidence.json")
	writeFile(t, dir, "vague-evidence.json", `{
  "kind": "nucleus.apply_evidence",
  "pass": false,
  "steps": [
    {
      "id": "custom_failure",
      "kind": "custom_failure",
      "pass": false,
      "reason": "missing generated text in README"
    }
  ]
}`)

	result, err := BuildEvidence(dir, evidencePath, 1)
	if err != nil {
		t.Fatalf("BuildEvidence returned error: %v", err)
	}
	if result["status"] != "needs_manual_action" {
		t.Fatalf("vague missing generated reason should not auto repair: %#v", result)
	}
}

func TestBuildEvidenceAppliesSafeBusinessPatch(t *testing.T) {
	dir := t.TempDir()
	writePatchableService(t, dir, []string{"examples/hello-http/internal/adapter/http/**"})
	original := "package http\n\nfunc HealthMessage() string {\n\treturn \"broken\"\n}\n"
	writeFile(t, dir, "examples/hello-http/internal/adapter/http/handler.go", original)
	evidencePath := filepath.Join(dir, "evidence.json")
	writeFile(t, dir, "evidence.json", `{
  "kind": "nucleus.verify_result",
  "pass": false,
  "steps": [
    {
      "id": "go_test",
      "pass": false,
      "repair_suggestion": {
        "file": "examples/hello-http/internal/adapter/http/handler.go",
        "find": "return \"broken\"",
        "replace": "return \"ok\"",
        "reason": "health handler should return ok",
        "expected_hash": "`+sha256Hex(original)+`"
      }
    }
  ]
}`)

	result, err := BuildEvidence(dir, evidencePath, 1)
	if err != nil {
		t.Fatalf("BuildEvidence returned error: %v", err)
	}
	if result["status"] != "repaired" {
		t.Fatalf("safe business patch should be repaired: %#v", result)
	}
	if result["verification_pass"] != true {
		t.Fatalf("repair should verify after patch: %#v", result)
	}
	updated, err := os.ReadFile(filepath.Join(dir, "examples", "hello-http", "internal", "adapter", "http", "handler.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), `return "ok"`) {
		t.Fatalf("expected patched handler, got:\n%s", updated)
	}
	rounds := result["rounds"].([]map[string]any)
	if rounds[0]["strategy"] != "bounded_business_patch" {
		t.Fatalf("unexpected strategy: %#v", rounds[0])
	}
	rollback, ok := rounds[0]["rollback_point"].(map[string]any)
	if !ok || rollback["original_hash"] != sha256Hex(original) {
		t.Fatalf("repair should record rollback point with original hash: %#v", rounds[0])
	}
}

func TestBuildEvidenceRejectsBusinessPatchOutsideAllowedSurface(t *testing.T) {
	dir := t.TempDir()
	writePatchableService(t, dir, []string{"internal/adapter/http/**"})
	original := "secret: old\n"
	writeFile(t, dir, "configs/prod.yaml", original)
	evidencePath := filepath.Join(dir, "evidence.json")
	writeFile(t, dir, "evidence.json", `{
  "kind": "nucleus.verify_result",
  "pass": false,
  "failure": {
    "fix_candidate": {
      "file": "configs/prod.yaml",
      "find": "old",
      "replace": "new",
      "reason": "unsafe config edit",
      "expected_hash": "`+sha256Hex(original)+`"
    }
  }
}`)

	result, err := BuildEvidence(dir, evidencePath, 1)
	if err != nil {
		t.Fatalf("BuildEvidence returned error: %v", err)
	}
	if result["status"] != "needs_manual_action" {
		t.Fatalf("out-of-surface patch should require manual action: %#v", result)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "configs", "prod.yaml"))
	if string(data) != original {
		t.Fatalf("rejected patch must not write file, got:\n%s", data)
	}
}

func TestBuildEvidenceRejectsBusinessPatchWithMultipleFindMatches(t *testing.T) {
	dir := t.TempDir()
	writePatchableService(t, dir, []string{"internal/domain/**"})
	original := "package domain\n\nfunc A() string { return \"broken\" }\nfunc B() string { return \"broken\" }\n"
	writeFile(t, dir, "internal/domain/service.go", original)
	evidencePath := filepath.Join(dir, "evidence.json")
	writeFile(t, dir, "evidence.json", `{
  "kind": "nucleus.verify_result",
  "pass": false,
  "failure": {
    "fix_candidate": {
      "file": "internal/domain/service.go",
      "find": "return \"broken\"",
      "replace": "return \"ok\"",
      "reason": "ambiguous domain fix",
      "expected_hash": "`+sha256Hex(original)+`"
    }
  }
}`)

	result, err := BuildEvidence(dir, evidencePath, 1)
	if err != nil {
		t.Fatalf("BuildEvidence returned error: %v", err)
	}
	if result["status"] != "needs_manual_action" {
		t.Fatalf("ambiguous patch should require manual action: %#v", result)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "internal", "domain", "service.go"))
	if string(data) != original {
		t.Fatalf("ambiguous patch must not write file, got:\n%s", data)
	}
}

func TestBuildEvidenceRejectsBusinessPatchWithHashMismatch(t *testing.T) {
	dir := t.TempDir()
	writePatchableService(t, dir, []string{"internal/domain/**"})
	original := "package domain\n\nfunc Message() string { return \"broken\" }\n"
	writeFile(t, dir, "internal/domain/service.go", original)
	evidencePath := filepath.Join(dir, "evidence.json")
	writeFile(t, dir, "evidence.json", `{
  "kind": "nucleus.verify_result",
  "pass": false,
  "failure": {
    "fix_candidate": {
      "file": "internal/domain/service.go",
      "find": "return \"broken\"",
      "replace": "return \"ok\"",
      "reason": "stale evidence",
      "expected_hash": "deadbeef"
    }
  }
}`)

	result, err := BuildEvidence(dir, evidencePath, 1)
	if err != nil {
		t.Fatalf("BuildEvidence returned error: %v", err)
	}
	if result["status"] != "needs_manual_action" {
		t.Fatalf("hash mismatch should require manual action: %#v", result)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "internal", "domain", "service.go"))
	if string(data) != original {
		t.Fatalf("hash mismatch must not write file, got:\n%s", data)
	}
}

func TestBuildEvidenceRejectsBusinessPatchSymlinkPath(t *testing.T) {
	dir := t.TempDir()
	writePatchableService(t, dir, []string{"internal/domain/**"})
	original := "package domain\n\nfunc Message() string { return \"broken\" }\n"
	outsidePath := filepath.Join(dir, "outside.go")
	if err := os.WriteFile(outsidePath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(dir, "internal", "domain", "service.go")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	evidencePath := filepath.Join(dir, "evidence.json")
	writeFile(t, dir, "evidence.json", `{
  "kind": "nucleus.verify_result",
  "pass": false,
  "failure": {
    "fix_candidate": {
      "file": "internal/domain/service.go",
      "find": "return \"broken\"",
      "replace": "return \"ok\"",
      "reason": "symlink target must not be patched",
      "expected_hash": "`+sha256Hex(original)+`"
    }
  }
}`)

	result, err := BuildEvidence(dir, evidencePath, 1)
	if err != nil {
		t.Fatalf("BuildEvidence returned error: %v", err)
	}
	if result["status"] != "needs_manual_action" {
		t.Fatalf("symlink patch should require manual action: %#v", result)
	}
	data, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("symlink target changed:\n%s", data)
	}
}

func writeFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePatchableService(t *testing.T, dir string, allowed []string) {
	t.Helper()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
ai:
  intent: test
  allowed_changes:
`+yamlList(allowed)+`
`)
	writeFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	writeFile(t, dir, "demo.go", "package demo\n")
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
      responses:
        "204":
          description: ok
`)
	writeFile(t, dir, "api/errors.yaml", "errors: []\n")
}

func yamlList(values []string) string {
	var builder strings.Builder
	for _, value := range values {
		builder.WriteString("    - ")
		builder.WriteString(value)
		builder.WriteString("\n")
	}
	return builder.String()
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func runtimeHTTPReplace(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../../../../runtime/http")
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(path)
}

func localRuntimeReplace(t *testing.T) string {
	t.Helper()
	return "replace github.com/nucleuskit/http => " + runtimeHTTPReplace(t) + "\n"
}
