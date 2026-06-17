package scenario

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildScenarioPlanGeneratesSuccessAndInvalidArgument(t *testing.T) {
	dir := t.TempDir()
	writeScenarioFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    get:
      operationId: getUser
      parameters:
        - name: verbose
          in: query
          required: false
          schema:
            type: boolean
`)
	writeScenarioFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
  - code: 2
    message: invalid argument
    http_status: 400
`)

	plan, err := BuildScenarioPlan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if plan["kind"] != "nucleus.scenario_plan" {
		t.Fatalf("unexpected kind: %#v", plan["kind"])
	}
	if plan["result_kind"] != resultKindScenarioPlan || plan["ok"] != true {
		t.Fatalf("unexpected result metadata: %#v", plan)
	}
	if plan["schema_version"] != scenarioSchemaVersion {
		t.Fatalf("unexpected schema version: %#v", plan["schema_version"])
	}
	summary, ok := plan["summary"].(map[string]any)
	if !ok || summary["scenarios"] != 3 {
		t.Fatalf("unexpected summary: %#v", plan["summary"])
	}
	scenarios, ok := plan["scenarios"].([]map[string]any)
	if !ok {
		t.Fatalf("unexpected scenarios shape: %#v", plan["scenarios"])
	}
	if len(scenarios) != 3 {
		t.Fatalf("expected 3 scenarios, got %#v", scenarios)
	}

	success := scenarios[0]
	if success["kind"] != "success" || success["method"] != "GET" || success["path"] != "/users/{id}" {
		t.Fatalf("unexpected success scenario: %#v", success)
	}
	if success["operation_id"] != "getUser" {
		t.Fatalf("unexpected operation id: %#v", success)
	}
	if success["suggestion"] == "" {
		t.Fatalf("success scenario should include a suggestion: %#v", success)
	}

	invalidArgument := scenarios[1]
	if invalidArgument["kind"] != "invalid_argument" {
		t.Fatalf("unexpected validation scenario: %#v", invalidArgument)
	}
	if invalidArgument["error_code"] != 2 || invalidArgument["http_status"] != 400 {
		t.Fatalf("validation scenario should carry invalid argument error metadata: %#v", invalidArgument)
	}
	params, ok := invalidArgument["parameters"].([]map[string]any)
	if !ok || len(params) != 1 {
		t.Fatalf("expected one required parameter in validation scenario: %#v", invalidArgument["parameters"])
	}
	if params[0]["name"] != "id" || params[0]["in"] != "path" || params[0]["schema_type"] != "string" {
		t.Fatalf("unexpected required parameter metadata: %#v", params[0])
	}

	errorHint := scenarios[2]
	if errorHint["kind"] != "error_assertion_hint" {
		t.Fatalf("unexpected error assertion scenario: %#v", errorHint)
	}
	if errorHint["code"] != 2 || errorHint["message"] != "invalid argument" {
		t.Fatalf("unexpected error assertion metadata: %#v", errorHint)
	}
}

func TestBuildScenarioPlanRequiresOpenAPIContract(t *testing.T) {
	_, err := BuildScenarioPlan(t.TempDir())
	if err == nil {
		t.Fatal("expected missing contract error")
	}
	if !strings.Contains(err.Error(), "api/openapi.yaml") {
		t.Fatalf("expected error to mention api/openapi.yaml, got %v", err)
	}
}

func TestBuildHTTPCaseDraftsFromScenarioPlan(t *testing.T) {
	dir := t.TempDir()
	writeScenarioFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
`)
	writeScenarioFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
  - code: 2
    message: invalid argument
    http_status: 400
`)

	cases, err := BuildHTTPCaseDrafts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 2 {
		t.Fatalf("expected success and invalid argument drafts, got %#v", cases)
	}
	if cases[0].ID != "getUser.success" || cases[0].Path != "/users/sample-string" {
		t.Fatalf("unexpected success draft: %#v", cases[0])
	}
	if cases[1].ID != "getUser.invalid-argument" || cases[1].Path != "/users/invalid-sample" {
		t.Fatalf("unexpected invalid draft: %#v", cases[1])
	}
	if len(cases[0].Assertions) != 2 || cases[0].Assertions[1].Path != "code" {
		t.Fatalf("draft should include executable assertions: %#v", cases[0])
	}
}

func TestDraftedHTTPCaseCanRunAsEvidence(t *testing.T) {
	dir := t.TempDir()
	writeScenarioFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeScenarioFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)
	cases, err := BuildHTTPCaseDrafts(dir)
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"code": 0, "message": "ok"})
	})
	evidence, err := RunHTTPCases(HTTPRunnerOptions{Handler: handler}, cases)
	if err != nil {
		t.Fatal(err)
	}
	if evidence["kind"] != httpEvidenceKind || evidence["pass"] != true {
		t.Fatalf("unexpected evidence: %#v", evidence)
	}
	if evidence["schema_ref"] != evidenceSchemaRef {
		t.Fatalf("schema_ref = %#v, want %s", evidence["schema_ref"], evidenceSchemaRef)
	}
}

func TestBuildHTTPCaseDraftsUseRequestBodySchemaExample(t *testing.T) {
	dir := t.TempDir()
	writeScenarioFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users:
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/CreateUser"
      responses:
        "200":
          description: ok
components:
  schemas:
    CreateUser:
      type: object
      properties:
        name:
          type: string
          example: Ada
`)
	writeScenarioFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)

	cases, err := BuildHTTPCaseDrafts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 {
		t.Fatalf("expected one success case, got %#v", cases)
	}
	if string(cases[0].Body) != `{"name":"Ada"}` {
		t.Fatalf("body = %s, want schema example", string(cases[0].Body))
	}
}

func TestRunHTTPScenariosAppliesHeaderAndBodyContentType(t *testing.T) {
	dir := t.TempDir()
	writeScenarioFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users:
    post:
      operationId: createUser
      parameters:
        - name: X-Tenant
          in: header
          required: true
          schema:
            type: string
            enum: ["acme"]
      requestBody:
        required: true
        content:
          application/merge-patch+json:
            schema:
              type: object
              properties:
                name:
                  type: string
                  example: Ada
      responses:
        "200":
          description: ok
`)
	writeScenarioFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Tenant") != "acme" {
			t.Fatalf("X-Tenant = %q, want acme", request.Header.Get("X-Tenant"))
		}
		if request.Header.Get("Content-Type") != "application/merge-patch+json" {
			t.Fatalf("Content-Type = %q, want application/merge-patch+json", request.Header.Get("Content-Type"))
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["name"] != "Ada" {
			t.Fatalf("body = %#v, want schema example", body)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"code": 0, "message": "ok"})
	})
	evidence, err := RunHTTPScenarios(dir, HTTPRunnerOptions{Handler: handler})
	if err != nil {
		t.Fatal(err)
	}
	if evidence["pass"] != true {
		t.Fatalf("unexpected evidence: %#v", evidence)
	}
}

func TestNewHTTPRunnerUsesDefaultTimeout(t *testing.T) {
	runner, err := newHTTPRunner(HTTPRunnerOptions{BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	if runner.client.Timeout != defaultHTTPTimeout {
		t.Fatalf("timeout = %v, want %v", runner.client.Timeout, defaultHTTPTimeout)
	}
}

func TestRunHTTPScenariosWithHandlerCapturesSamplesAndEnvelope(t *testing.T) {
	dir := t.TempDir()
	writeScenarioFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    get:
      operationId: getUser
      parameters:
        - name: verbose
          in: query
          required: true
          schema:
            type: boolean
`)
	writeScenarioFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)

	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/users/sample-string" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("verbose") != "true" {
			t.Fatalf("expected generated query sample, got %s", request.URL.RawQuery)
		}
		if request.Header.Get("X-Scenario-Hook") != "seen" {
			t.Fatalf("request hook header was not applied")
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"code":     0,
			"message":  "ok",
			"data":     map[string]string{"id": "sample-string"},
			"trace_id": request.Header.Get("X-Request-Id"),
		})
	})

	evidence, err := RunHTTPScenarios(dir, HTTPRunnerOptions{
		Handler: handler,
		RequestHook: func(request *http.Request, scenario map[string]any) error {
			if scenario["operation_id"] != "getUser" {
				t.Fatalf("unexpected scenario metadata: %#v", scenario)
			}
			request.Header.Set("X-Scenario-Hook", "seen")
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if evidence["kind"] != httpEvidenceKind || evidence["pass"] != true || evidence["status"] != "passed" {
		t.Fatalf("unexpected evidence: %#v", evidence)
	}
	steps := evidence["steps"].([]map[string]any)
	if len(steps) != 1 || steps[0]["pass"] != true {
		t.Fatalf("unexpected steps: %#v", steps)
	}
	if steps[0]["envelope_code"] != 0 || steps[0]["envelope_message"] != "ok" {
		t.Fatalf("expected unwrapped envelope metadata: %#v", steps[0])
	}
	samples := evidence["http_samples"].([]map[string]any)
	if len(samples) != 1 {
		t.Fatalf("expected one HTTP sample: %#v", samples)
	}
	requestSample := samples[0]["request"].(map[string]any)
	responseSample := samples[0]["response"].(map[string]any)
	if requestSample["path"] != "/users/sample-string" || !strings.Contains(requestSample["url"].(string), "verbose=true") {
		t.Fatalf("unexpected request sample: %#v", requestSample)
	}
	if responseSample["status_code"] != 200 || responseSample["envelope_code"] != 0 {
		t.Fatalf("unexpected response sample: %#v", responseSample)
	}
}

func TestRunHTTPScenariosWithBaseURLReportsEnvelopeFailure(t *testing.T) {
	dir := t.TempDir()
	writeScenarioFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeScenarioFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"code":    1,
			"message": "upstream unavailable",
		})
	}))
	defer server.Close()

	evidence, err := RunHTTPScenarios(dir, HTTPRunnerOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if evidence["pass"] != false || evidence["status"] != "failed" {
		t.Fatalf("unexpected evidence: %#v", evidence)
	}
	steps := evidence["steps"].([]map[string]any)
	if steps[0]["http_status"] != 502 || steps[0]["envelope_code"] != 1 {
		t.Fatalf("expected failed HTTP/envelope metadata: %#v", steps[0])
	}
	assertions := evidence["assertion_results"].([]map[string]any)
	if len(assertions) == 0 || assertions[0]["pass"] != false {
		t.Fatalf("expected failing assertion: %#v", assertions)
	}
}

func writeScenarioFile(t *testing.T, dir, name, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
