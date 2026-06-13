package openapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEndpointsUsesRouteRegistry(t *testing.T) {
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o700); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	openapiYAML := []byte(`openapi: 3.1.0
paths:
  /widgets/{id}:
    get:
      operationId: get_widget
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
`)
	if err := os.WriteFile(filepath.Join(apiDir, "openapi.yaml"), openapiYAML, 0o600); err != nil {
		t.Fatalf("write openapi.yaml: %v", err)
	}

	endpoints, err := LoadEndpoints(dir)
	if err != nil {
		t.Fatalf("LoadEndpoints() error = %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("len(endpoints) = %d, want 1", len(endpoints))
	}
	if got := endpoints[0].Method; got != "GET" {
		t.Fatalf("Method = %q, want GET", got)
	}
	if got := endpoints[0].Path; got != "/widgets/{id}" {
		t.Fatalf("Path = %q, want /widgets/{id}", got)
	}
	if got := endpoints[0].OperationID; got != "get_widget" {
		t.Fatalf("OperationID = %q, want get_widget", got)
	}
}
