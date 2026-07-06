package root

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenCommandJSONRootWiring(t *testing.T) {
	dir := t.TempDir()
	writeRootGenService(t, dir)

	cmd := New()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", dir, "gen", "--json", "--http", "--errors"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute gen: %v\nstderr=%s", err, stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode gen output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.gen_result")
	assertBool(t, output, "ok", true)
	files := assertSlice(t, output, "files")
	assertContainsString(t, files, "contract/gen/endpoints.go")
	assertContainsString(t, files, "internal/adapter/http/gen/routes.gen.go")
	if _, err := filepath.Abs(dir); err != nil {
		t.Fatalf("abs dir: %v", err)
	}
}

func writeRootGenService(t *testing.T, dir string) {
	t.Helper()
	writeRootGenFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
  generated:
    - contract/gen
    - internal/adapter/http/gen
capabilities:
  - id: http
    kind: http
`)
	writeRootGenFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
      responses:
        "200":
          description: ok
`)
	writeRootGenFile(t, dir, "api/errors.yaml", `errors:
  - code: 1001
    message: health check failed
    http_status: 500
`)
}

func writeRootGenFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
