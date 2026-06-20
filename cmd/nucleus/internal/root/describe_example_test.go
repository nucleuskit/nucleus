package root

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDescribeCommandWithHelloHTTPExample(t *testing.T) {
	exampleDir := writeRootExampleService(t)

	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--dir", exampleDir, "--schema", "9.9", "describe", "--json", "--flow"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute describe: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode describe output: %v\n%s", err, stdout.String())
	}

	assertString(t, output, "schema_version", "9.9")
	service := assertMap(t, output, "service")
	assertString(t, service, "name", "hello-http")

	endpoints := assertSlice(t, output, "endpoints")
	if len(endpoints) != 1 {
		t.Fatalf("len(endpoints) = %d, want 1", len(endpoints))
	}
	endpoint, ok := endpoints[0].(map[string]any)
	if !ok {
		t.Fatalf("endpoint has type %T, want map[string]any", endpoints[0])
	}
	assertString(t, endpoint, "method", "GET")
	assertString(t, endpoint, "path", "/hello/{name}")
	assertString(t, endpoint, "operation_id", "get_hello")

	configKeys := assertSlice(t, output, "config_keys")
	assertContainsFact(t, configKeys, "key", "http.address")
	assertContainsFact(t, configKeys, "env", "HTTP_ADDR")

	flowGraph := assertMap(t, output, "flow_graph")
	assertContainsFact(t, assertSlice(t, flowGraph, "nodes"), "kind", "response")
	assertContainsFact(t, assertSlice(t, flowGraph, "nodes"), "name", "log")
	assertContainsFact(t, assertSlice(t, flowGraph, "params"), "name", "path.name")
	assertContainsFact(t, assertSlice(t, flowGraph, "error_paths"), "name", "greeting_not_found")

	verification := assertMap(t, output, "verification")
	assertString(t, verification, "result_kind", "nucleus.verify_result")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "cmd", "nucleus")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}

func assertMap(t *testing.T, value map[string]any, key string) map[string]any {
	t.Helper()
	item, ok := value[key].(map[string]any)
	if !ok {
		t.Fatalf("%s has type %T, want map[string]any", key, value[key])
	}
	return item
}

func assertSlice(t *testing.T, value map[string]any, key string) []any {
	t.Helper()
	item, ok := value[key].([]any)
	if !ok {
		t.Fatalf("%s has type %T, want []any", key, value[key])
	}
	return item
}

func assertString(t *testing.T, value map[string]any, key string, want string) {
	t.Helper()
	got, ok := value[key].(string)
	if !ok {
		t.Fatalf("%s has type %T, want string", key, value[key])
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func assertContainsFact(t *testing.T, values []any, key string, want string) {
	t.Helper()
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if item[key] == want {
			return
		}
	}
	t.Fatalf("no item with %s = %q in %#v", key, want, values)
}
