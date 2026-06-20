package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandlerExposesMetadataEndpoints(t *testing.T) {
	dir := writeServiceFixture(t)
	result := buildResult(Config{Dir: &dir}, &options{addr: "127.0.0.1:9090", mode: modeServe})
	if !result.OK {
		t.Fatalf("result should be OK: %#v", result.Diagnostics)
	}
	handler := newHandler(result.Description)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("/healthz status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "OK\n" {
		t.Fatalf("/healthz body = %q, want OK newline", recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("/readyz status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var ready map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &ready); err != nil {
		t.Fatalf("decode readyz: %v\n%s", err, recorder.Body.String())
	}
	if ready["status"] != "ready" || ready["service"] != "demo" || ready["endpoints"] != float64(1) {
		t.Fatalf("unexpected readyz payload: %#v", ready)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/.well-known/nucleus.json", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("well-known status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var wellKnown map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &wellKnown); err != nil {
		t.Fatalf("decode well-known: %v\n%s", err, recorder.Body.String())
	}
	service := requireMap(t, wellKnown, "service")
	if service["name"] != "demo" {
		t.Fatalf("well-known service.name = %v, want demo", service["name"])
	}
}

func TestHandlerRejectsNonGETMetadataRequests(t *testing.T) {
	dir := writeServiceFixture(t)
	result := buildResult(Config{Dir: &dir}, &options{addr: defaultAddr, mode: modeServe})
	if !result.OK {
		t.Fatalf("result should be OK: %#v", result.Diagnostics)
	}
	handler := newHandler(result.Description)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/readyz", nil))

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /readyz status = %d, want 405", recorder.Code)
	}
}

func writeServiceFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.1.0
info:
  title: Demo
  version: "0.1.0"
paths:
  /hello:
    get:
      operationId: get_hello
      responses:
        "200":
          description: OK
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 4001
    message: invalid
    http_status: 400
`)
	writeFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	return dir
}

func writeFile(t *testing.T, dir string, name string, contents string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
