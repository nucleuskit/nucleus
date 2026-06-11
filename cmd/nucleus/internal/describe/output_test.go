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
