package describe

import "testing"

func TestBuildOutputAddsDescribeMetadata(t *testing.T) {
	output, err := BuildOutput(OutputOptions{
		Dir:            ".",
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
}

func TestBuildOutputUsesDefaultSchemaVersion(t *testing.T) {
	output, err := BuildOutput(OutputOptions{Dir: "."})
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
