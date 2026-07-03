package root

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestServeCommandJSONRootWiring(t *testing.T) {
	dir := t.TempDir()
	writeRootServeFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - id: http
    kind: http
`)
	writeRootServeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
      responses:
        "200":
          description: ok
`)
	writeRootServeFile(t, dir, "api/errors.yaml", "errors: []\n")

	cmd := New()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", dir, "serve", "--check", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute serve: %v\nstderr=%s", err, stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode serve output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.serve_result")
	assertString(t, output, "schema_ref", "contract/schema/serve-result.v1.schema.json")
	assertString(t, output, "mode", "check")
	assertBool(t, output, "ok", true)
	summary := assertMap(t, output, "summary")
	assertString(t, summary, "service", "demo")
	assertNumber(t, summary, "endpoint_count", 1)
}

func writeRootServeFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
