package gen

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandJSONSuccess(t *testing.T) {
	dir := t.TempDir()
	writeService(t, dir)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json", "--http", "--errors", "--docs", "--typescript", "--clients", "--client-language", "typescript"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute gen: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output struct {
		ResultKind string     `json:"result_kind"`
		OK         bool       `json:"ok"`
		SourceHash string     `json:"source_hash"`
		Summary    genSummary `json:"summary"`
		Files      []string   `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.ResultKind != resultKindGen {
		t.Fatalf("result_kind = %q, want %q", output.ResultKind, resultKindGen)
	}
	if !output.OK {
		t.Fatal("ok = false, want true")
	}
	if output.SourceHash == "" {
		t.Fatal("source_hash is empty")
	}
	if output.Summary.Files != len(output.Files) {
		t.Fatalf("summary.files = %d, want %d", output.Summary.Files, len(output.Files))
	}
	for _, want := range []string{
		"contract/gen/endpoints.go",
		"contract/gen/errors.go",
		"contract/gen/contract_source.go",
		"contract/gen/contract.md",
		"contract/gen/types.ts",
		"internal/adapter/http/gen/routes.gen.go",
		"sdk/typescript/client.ts",
		"sdk/typescript/.nucleus-source.sha256",
	} {
		if !containsString(output.Files, want) {
			t.Fatalf("files = %#v, want %s", output.Files, want)
		}
	}
}

func TestCommandJSONValidationFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  version: "0.1.0"
ai:
  intent: test
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrGenFailed) {
		t.Fatalf("execute gen error = %v, want ErrGenFailed", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON output", stderr.String())
	}

	var output struct {
		OK          bool `json:"ok"`
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.OK {
		t.Fatal("ok = true, want false")
	}
	found := false
	for _, item := range output.Diagnostics {
		if item.Code == "manifest.service_name_required" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v, want manifest.service_name_required", output.Diagnostics)
	}
}

func TestCommandHumanSuccessOutput(t *testing.T) {
	dir := t.TempDir()
	writeService(t, dir)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--http", "--errors"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute gen: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"OK",
		"generated:",
		"source_hash:",
		"contract/gen/endpoints.go",
		"internal/adapter/http/gen/routes.gen.go",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, want %q", output, want)
		}
	}
}

func TestCommandJSONUnsupportedClientLanguage(t *testing.T) {
	dir := t.TempDir()
	writeService(t, dir)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json", "--clients", "--client-language", "ruby"})

	err := cmd.Execute()
	if !errors.Is(err, ErrGenFailed) {
		t.Fatalf("execute gen error = %v, want ErrGenFailed", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON output", stderr.String())
	}

	var output struct {
		OK          bool `json:"ok"`
		Diagnostics []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.OK {
		t.Fatal("ok = true, want false")
	}
	if len(output.Diagnostics) == 0 || output.Diagnostics[0].Code != "gen.failed" {
		t.Fatalf("diagnostics = %#v, want gen.failed", output.Diagnostics)
	}
	if !strings.Contains(output.Diagnostics[0].Message, "unsupported client language") {
		t.Fatalf("message = %q, want unsupported language", output.Diagnostics[0].Message)
	}
}

func writeService(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
  generated:
    - contract/gen
    - internal/adapter/http/gen
    - sdk/typescript
capabilities:
  - id: http
    kind: http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /hello/{name}:
    get:
      operationId: getHello
      parameters:
        - name: name
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
components:
  schemas:
    Greeting:
      type: object
      required: [message]
      properties:
        message:
          type: string
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 1001
    message: greeting failed
    http_status: 500
`)
}

func writeFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
