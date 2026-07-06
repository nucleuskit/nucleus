package validate

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nucleuskit/contract/diagnostic"
)

func TestRenderHumanOutputIncludesDiagnostics(t *testing.T) {
	diagnostics := diagnostic.Diagnostics{{
		Severity: diagnostic.SeverityError,
		Code:     "manifest.service_name_required",
		Path:     "nucleus.yaml",
		Message:  "service.name is required",
	}}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderHuman(&stdout, &stderr, diagnostics, validateSummary{})
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "manifest.service_name_required") {
		t.Fatalf("stderr = %q, want diagnostic code", stderr.String())
	}
}

func TestRenderHumanOutputPrintsOK(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderHuman(&stdout, &stderr, nil, validateSummary{
		Checked:         []string{"nucleus.yaml", "api/openapi.yaml"},
		MissingOptional: []string{"api/proto"},
	})
	output := stdout.String()
	for _, want := range []string{
		"OK",
		"validated: nucleus.yaml, api/openapi.yaml",
		"missing optional: api/proto",
		"diagnostics: 0 errors, 0 warnings",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, want %q", output, want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRenderHumanOutputShowsWarningsAndPrintsOK(t *testing.T) {
	diagnostics := diagnostic.Diagnostics{{
		Severity: diagnostic.SeverityWarning,
		Code:     "manifest.ai_intent_missing",
		Path:     "nucleus.yaml",
		Message:  "ai.intent is recommended",
	}}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderHuman(&stdout, &stderr, diagnostics, validateSummary{Warnings: 1})
	if !strings.Contains(stdout.String(), "OK") {
		t.Fatalf("stdout = %q, want OK", stdout.String())
	}
	if !strings.Contains(stdout.String(), "diagnostics: 0 errors, 1 warnings") {
		t.Fatalf("stdout = %q, want warning count", stdout.String())
	}
	if !strings.Contains(stderr.String(), "manifest.ai_intent_missing") {
		t.Fatalf("stderr = %q, want warning code", stderr.String())
	}
}

func TestRenderJSONOutput(t *testing.T) {
	diagnostics := diagnostic.Diagnostics{{
		Severity: diagnostic.SeverityError,
		Code:     "manifest.service_name_required",
		Path:     "nucleus.yaml",
		Message:  "service.name is required",
	}}
	var stdout bytes.Buffer
	if err := renderJSON(&stdout, diagnostics, validateSummary{}, false); err != nil {
		t.Fatalf("renderJSON() error = %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if output["result_kind"] != resultKindValidate {
		t.Fatalf("result_kind = %v, want %s", output["result_kind"], resultKindValidate)
	}
	if output["schema_version"] != schemaVersionValidate {
		t.Fatalf("schema_version = %v, want %s", output["schema_version"], schemaVersionValidate)
	}
	if output["schema_ref"] != schemaRefValidate {
		t.Fatalf("schema_ref = %v, want %s", output["schema_ref"], schemaRefValidate)
	}
	if output["ok"] != false {
		t.Fatalf("ok = %v, want false", output["ok"])
	}
}

func TestRenderJSONOutputUsesEmptyDiagnosticsArray(t *testing.T) {
	var stdout bytes.Buffer
	if err := renderJSON(&stdout, nil, validateSummary{}, false); err != nil {
		t.Fatalf("renderJSON() error = %v", err)
	}
	var output struct {
		Diagnostics []any `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if output.Diagnostics == nil {
		t.Fatal("diagnostics = nil, want empty array")
	}
}

func TestRenderJSONOutputIncludesSummary(t *testing.T) {
	var stdout bytes.Buffer
	if err := renderJSON(&stdout, nil, validateSummary{
		Errors:          0,
		Warnings:        1,
		Checked:         []string{"nucleus.yaml"},
		MissingOptional: []string{"api/openapi.yaml"},
	}, false); err != nil {
		t.Fatalf("renderJSON() error = %v", err)
	}
	var output struct {
		Summary validateSummary `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if output.Summary.Warnings != 1 {
		t.Fatalf("summary.warnings = %d, want 1", output.Summary.Warnings)
	}
	if got := strings.Join(output.Summary.Checked, ","); got != "nucleus.yaml" {
		t.Fatalf("summary.checked = %q, want nucleus.yaml", got)
	}
}
