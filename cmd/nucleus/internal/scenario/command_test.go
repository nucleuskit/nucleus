package scenario

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandJSONPlanOutput(t *testing.T) {
	dir := t.TempDir()
	writeScenarioService(t, dir)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute scenario: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output["kind"] != planKind {
		t.Fatalf("kind = %#v, want %s", output["kind"], planKind)
	}
	if output["result_kind"] != resultKindScenarioPlan || output["ok"] != true {
		t.Fatalf("unexpected result metadata: %#v", output)
	}
	if len(output["scenarios"].([]any)) == 0 {
		t.Fatalf("expected scenarios in output: %#v", output)
	}
}

func TestCommandJSONDraftCasesOutput(t *testing.T) {
	dir := t.TempDir()
	writeScenarioService(t, dir)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--draft-cases", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute scenario: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	var output struct {
		ResultKind string     `json:"result_kind"`
		OK         bool       `json:"ok"`
		Kind       string     `json:"kind"`
		Cases      []HTTPCase `json:"cases"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.ResultKind != resultKindHTTPCaseDrafts || !output.OK || output.Kind != httpCaseDraftsKind {
		t.Fatalf("unexpected draft output: %#v", output)
	}
	if len(output.Cases) == 0 {
		t.Fatalf("expected cases in output: %#v", output)
	}
}

func TestCommandPrettyJSONOutput(t *testing.T) {
	dir := t.TempDir()
	writeScenarioService(t, dir)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "--pretty"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute scenario: %v", err)
	}
	if !strings.Contains(stdout.String(), "\n  \"result_kind\"") {
		t.Fatalf("stdout = %q, want indented JSON", stdout.String())
	}
}

func TestCommandHumanDraftCasesOutput(t *testing.T) {
	dir := t.TempDir()
	writeScenarioService(t, dir)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--draft-cases"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute scenario: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{"OK", "cases:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestCommandRunHTTPFailureReturnsSentinel(t *testing.T) {
	dir := t.TempDir()
	writeScenarioService(t, dir)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = writer.Write([]byte(`{"code":1,"message":"bad gateway"}`))
	}))
	defer server.Close()

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json", "--run-http", "--base-url", server.URL})

	err := cmd.Execute()
	if !errors.Is(err, ErrScenarioFailed) {
		t.Fatalf("execute scenario error = %v, want ErrScenarioFailed", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON output", stderr.String())
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output["kind"] != httpEvidenceKind || output["pass"] != false || output["status"] != "failed" {
		t.Fatalf("unexpected evidence: %#v", output)
	}
}

func TestCommandRejectsAmbiguousModes(t *testing.T) {
	dir := t.TempDir()
	cmd := NewCommand(Config{Dir: &dir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--draft-cases", "--run-http", "--base-url", "http://127.0.0.1"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("execute scenario error = %v, want mutually exclusive modes", err)
	}
}

func TestCommandCasesRequireBaseURL(t *testing.T) {
	dir := t.TempDir()
	casesPath := filepath.Join(dir, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"path":"/healthz"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := NewCommand(Config{Dir: &dir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--cases", casesPath})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--cases requires --base-url") {
		t.Fatalf("execute scenario error = %v, want --cases base-url requirement", err)
	}
}

func writeScenarioService(t *testing.T, dir string) {
	t.Helper()
	writeScenarioFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
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
`)
	writeScenarioFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
  - code: 2
    message: invalid argument
    http_status: 400
`)
}
