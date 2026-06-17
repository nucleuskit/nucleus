package describe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildOutputAddsDescribeMetadata(t *testing.T) {
	dir := newDescribeFixture(t)
	output, err := BuildOutput(OutputOptions{
		Dir:            dir,
		SchemaOverride: "9.9",
		IncludeFlow:    true,
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	if got := output["schema_version"]; got != "9.9" {
		t.Fatalf("schema_version = %v, want 9.9", got)
	}
	if output["flow_graph"] == nil {
		t.Fatal("flow_graph missing")
	}
	verification, ok := output["verification"].(map[string]any)
	if !ok {
		t.Fatalf("verification has type %T, want map[string]any", output["verification"])
	}
	if got := verification["result_kind"]; got != "nucleus.verify_result" {
		t.Fatalf("verification.result_kind = %v, want nucleus.verify_result", got)
	}
	if got := verification["evidence_schema"]; got != verificationEvidenceSchema {
		t.Fatalf("verification.evidence_schema = %v, want %s", got, verificationEvidenceSchema)
	}
	schemaPath := filepath.Join("..", "..", "..", "..", filepath.FromSlash(verificationEvidenceSchema))
	if _, err := os.Stat(schemaPath); err != nil {
		t.Fatalf("verification evidence schema %s is not readable: %v", schemaPath, err)
	}
	pipeline, ok := verification["pipeline"].([]map[string]any)
	if !ok {
		t.Fatalf("verification.pipeline has type %T, want []map[string]any", verification["pipeline"])
	}
	assertVerificationPipeline(t, pipeline)
	optional, ok := verification["optional_evidence"].([]map[string]any)
	if !ok {
		t.Fatalf("verification.optional_evidence has type %T, want []map[string]any", verification["optional_evidence"])
	}
	assertOptionalEvidence(t, optional)
}

func TestBuildOutputUsesDefaultSchemaVersion(t *testing.T) {
	dir := newDescribeFixture(t)
	output, err := BuildOutput(OutputOptions{Dir: dir})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	if got := output["schema_version"]; got != defaultSchemaVersion {
		t.Fatalf("schema_version = %v, want %s", got, defaultSchemaVersion)
	}
	if output["flow_graph"] != nil {
		t.Fatal("flow_graph present without IncludeFlow")
	}
}

func newDescribeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	manifest := []byte(`schema_version: "1.0"
service:
  name: fixture
  version: "0.1.0"
ai: {}
nucleus: {}
`)
	if err := os.WriteFile(filepath.Join(dir, "nucleus.yaml"), manifest, 0o600); err != nil {
		t.Fatalf("write nucleus.yaml: %v", err)
	}
	return dir
}

func assertVerificationPipeline(t *testing.T, pipeline []map[string]any) {
	t.Helper()
	want := []struct {
		phase   string
		command string
	}{
		{phaseValidate, commandValidate},
		{phaseLint, commandLintStrict},
		{phaseGeneratedFreshness, commandDescribeJSON},
		{phaseTidy, commandGoModTidy},
		{phaseImport, commandGoListAll},
		{phaseBuild, commandGoTestCompileOnly},
		{phaseTest, commandGoTestAll},
	}
	if len(pipeline) != len(want) {
		t.Fatalf("pipeline length = %d, want %d", len(pipeline), len(want))
	}
	for index, item := range pipeline {
		if got := item["id"]; got != want[index].phase {
			t.Fatalf("pipeline[%d].id = %v, want %s", index, got, want[index].phase)
		}
		if got := item["sequence"]; got != index+1 {
			t.Fatalf("pipeline[%d].sequence = %v, want %d", index, got, index+1)
		}
		if got := item["phase"]; got != want[index].phase {
			t.Fatalf("pipeline[%d].phase = %v, want %s", index, got, want[index].phase)
		}
		if got := item["command"]; got != want[index].command {
			t.Fatalf("pipeline[%d].command = %v, want %s", index, got, want[index].command)
		}
		if got := item["schema_ref"]; got != verificationEvidenceSchema {
			t.Fatalf("pipeline[%d].schema_ref = %v, want %s", index, got, verificationEvidenceSchema)
		}
		if got := item["produces"]; got != verificationResultKind {
			t.Fatalf("pipeline[%d].produces = %v, want %s", index, got, verificationResultKind)
		}
	}
}

func assertOptionalEvidence(t *testing.T, optional []map[string]any) {
	t.Helper()
	if len(optional) != 2 {
		t.Fatalf("optional evidence length = %d, want 2", len(optional))
	}
	if optional[0]["produces"] != "nucleus.scenario_plan_result" || optional[0]["required"] != false {
		t.Fatalf("unexpected scenario plan optional evidence: %#v", optional[0])
	}
	if optional[1]["produces"] != evidenceKindHTTPScenario || optional[1]["schema_ref"] != verificationEvidenceSchema || optional[1]["required"] != false {
		t.Fatalf("unexpected HTTP scenario optional evidence: %#v", optional[1])
	}
}
