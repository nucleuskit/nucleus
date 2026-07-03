package root

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRootExampleService(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeRootFixtureFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: hello-http
  version: "0.1.0"
  owner: nucleus-maintainers
  tier: example
  description: Contract-first HTTP service used by root command tests.
capabilities:
  - id: http
    kind: http
  - id: log
    kind: log
dependencies:
  - name: greeting-profile
    contract: api/openapi.yaml#/paths/~1hello~1{name}
    required: false
ai:
  intent: Exercise root command wiring against a stable service fixture.
  allowed_changes:
    - api/**
    - configs/**
    - internal/**
  readonly:
    - internal/adapter/http/gen/**
  generated:
    - contract/gen
    - internal/adapter/http/gen
  forbidden:
    - configs/*.local.yaml
`)
	writeRootFixtureFile(t, dir, "api/openapi.yaml", `openapi: 3.1.0
info:
  title: Hello HTTP Example
  version: 0.1.0
paths:
  /hello/{name}:
    parameters:
      - name: name
        in: path
        required: true
        schema:
          type: string
    get:
      operationId: get_hello
      x-nucleus-priority: 10
      parameters:
        - name: trace_id
          in: header
          required: false
          schema:
            type: string
      responses:
        "200":
          description: Greeting response.
        "404":
          description: Greeting target was not found.
`)
	writeRootFixtureFile(t, dir, "api/errors.yaml", `errors:
  - code: 4001
    message: invalid_name
    http_status: 400
  - code: 4041
    message: greeting_not_found
    http_status: 404
`)
	writeRootFixtureFile(t, dir, "configs/app.yaml", `http:
  address: "${HTTP_ADDR:-127.0.0.1:8080}"
greeting:
  prefix: hello
`)
	writeRootFixtureFile(t, dir, "go.mod", "module example.com/hello-http\n\ngo 1.26.3\n")
	writeRootFixtureFile(t, dir, "internal/app/routes.go", `package app

import "net/http"

type Router struct{}

func (Router) Handle(method string, path string, handler func()) {}

type Logger struct{}

func (Logger) Info() {}

var log Logger

func RegisterRoutes(router Router) {
	router.Handle(http.MethodGet, "/hello/{name}", func() {
		log.Info()
	})
}
`)
	return dir
}

func writeRootFixtureFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
