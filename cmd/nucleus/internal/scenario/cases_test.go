package scenario

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndRunHTTPCasesWithAssertionDSL(t *testing.T) {
	dir := t.TempDir()
	casesPath := filepath.Join(dir, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[
  {
    "id": "invalid-user",
    "method": "GET",
    "path": "/users/bad",
    "assertions": [
      {"type": "http_status", "equals": 400},
      {"type": "json_path_equals", "path": "code", "equals": 2},
      {"type": "body_contains", "contains": "invalid argument"}
    ]
  }
]`), 0o644); err != nil {
		t.Fatal(err)
	}

	cases, err := LoadHTTPCases(casesPath)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := RunHTTPCases(HTTPRunnerOptions{
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"code":    2,
				"message": "invalid argument",
			})
		}),
	}, cases)
	if err != nil {
		t.Fatal(err)
	}

	if evidence["result_kind"] != httpEvidenceKind || evidence["schema_ref"] != evidenceSchemaRef || evidence["ok"] != true {
		t.Fatalf("expected case evidence to pass: %#v", evidence)
	}
	results := evidence["assertion_results"].([]map[string]any)
	if len(results) != 3 {
		t.Fatalf("expected 3 assertion results, got %#v", results)
	}
	for _, result := range results {
		if result["pass"] != true {
			t.Fatalf("expected assertion to pass: %#v", result)
		}
	}
}

func TestLoadHTTPCasesAcceptsWrappedCases(t *testing.T) {
	dir := t.TempDir()
	casesPath := filepath.Join(dir, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`{
  "cases": [
    {
      "id": "healthz",
      "path": "/healthz",
      "assertions": [
        {"type": "http_status", "equals": 200}
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cases, err := LoadHTTPCases(casesPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].Method != http.MethodGet || cases[0].ID != "healthz" {
		t.Fatalf("unexpected cases: %#v", cases)
	}
}

func TestRunHTTPCasesWithBaseURLPreservesQueryString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/users/sample" {
			t.Fatalf("path = %s, want /users/sample", request.URL.Path)
		}
		if request.URL.Query().Get("verbose") != "true" {
			t.Fatalf("query = %s, want verbose=true", request.URL.RawQuery)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"code": 0, "message": "ok"})
	}))
	defer server.Close()

	evidence, err := RunHTTPCases(HTTPRunnerOptions{BaseURL: server.URL}, []HTTPCase{
		{
			ID:     "query-case",
			Method: http.MethodGet,
			Path:   "/users/sample?verbose=true",
			Assertions: []Assertion{
				{Type: "http_status", Equals: 200},
				{Type: "json_path_equals", Path: "code", Equals: 0},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if evidence["ok"] != true {
		t.Fatalf("unexpected evidence: %#v", evidence)
	}
}
