package hellohttp_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNucleusDescribeExample(t *testing.T) {
	exampleDir := exampleRoot(t)
	repoRoot := filepath.Clean(filepath.Join(exampleDir, "..", ".."))
	workspace := writeWorkspace(t, repoRoot, exampleDir)

	cmd := exec.Command("go", "run", filepath.Join(repoRoot, "cmd", "nucleus"), "describe", "--dir", exampleDir, "--json", "--flow", "--schema", "example-test")
	cmd.Dir = exampleDir
	cmd.Env = append(os.Environ(), "GOWORK="+workspace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("nucleus describe failed: %v\n%s", err, output)
	}

	var description map[string]any
	if err := json.Unmarshal(output, &description); err != nil {
		t.Fatalf("decode describe output: %v\n%s", err, output)
	}

	service := requireMap(t, description, "service")
	if service["name"] != "hello-http" {
		t.Fatalf("service.name = %v, want hello-http", service["name"])
	}
	if description["schema_version"] != "example-test" {
		t.Fatalf("schema_version = %v, want example-test", description["schema_version"])
	}

	endpoints := requireSlice(t, description, "endpoints")
	if len(endpoints) != 1 {
		t.Fatalf("len(endpoints) = %d, want 1", len(endpoints))
	}
	endpoint, ok := endpoints[0].(map[string]any)
	if !ok {
		t.Fatalf("endpoint has type %T, want map[string]any", endpoints[0])
	}
	if endpoint["operation_id"] != "get_hello" {
		t.Fatalf("endpoint.operation_id = %v, want get_hello", endpoint["operation_id"])
	}

	flowGraph := requireMap(t, description, "flow_graph")
	if len(requireSlice(t, flowGraph, "nodes")) == 0 {
		t.Fatal("flow_graph.nodes is empty")
	}
	errorPaths := requireSlice(t, flowGraph, "error_paths")
	if len(errorPaths) != 2 {
		t.Fatalf("len(flow_graph.error_paths) = %d, want 2", len(errorPaths))
	}
}

func exampleRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(filename)
}

func writeWorkspace(t *testing.T, repoRoot string, exampleDir string) string {
	t.Helper()
	workspace := filepath.Join(t.TempDir(), "go.work")
	content := "go 1.26.3\n\nuse (\n" +
		"\t" + filepath.ToSlash(repoRoot) + "\n" +
		"\t" + filepath.ToSlash(exampleDir) + "\n" +
		"\t" + filepath.ToSlash(filepath.Join(repoRoot, "bridge")) + "\n" +
		"\t" + filepath.ToSlash(filepath.Join(repoRoot, "cap")) + "\n" +
		"\t" + filepath.ToSlash(filepath.Join(repoRoot, "contract")) + "\n" +
		"\t" + filepath.ToSlash(filepath.Join(repoRoot, "core")) + "\n" +
		"\t" + filepath.ToSlash(filepath.Join(repoRoot, "runtime", "grpc")) + "\n" +
		"\t" + filepath.ToSlash(filepath.Join(repoRoot, "runtime", "http")) + "\n" +
		"\t" + filepath.ToSlash(filepath.Join(repoRoot, "runtime", "worker")) + "\n" +
		")\n"
	if err := os.WriteFile(workspace, []byte(content), 0o600); err != nil {
		t.Fatalf("write temporary go.work: %v", err)
	}
	return workspace
}

func requireMap(t *testing.T, value map[string]any, key string) map[string]any {
	t.Helper()
	item, ok := value[key].(map[string]any)
	if !ok {
		t.Fatalf("%s has type %T, want map[string]any", key, value[key])
	}
	return item
}

func requireSlice(t *testing.T, value map[string]any, key string) []any {
	t.Helper()
	item, ok := value[key].([]any)
	if !ok {
		t.Fatalf("%s has type %T, want []any", key, value[key])
	}
	return item
}
