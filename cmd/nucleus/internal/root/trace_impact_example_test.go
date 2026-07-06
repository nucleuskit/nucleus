package root

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestTraceAndImpactFromRoot(t *testing.T) {
	dir := t.TempDir()
	writeRootFixtureFile(t, dir, "go.mod", "module example.com/orders\n\ngo 1.26.3\n")
	writeRootFixtureFile(t, dir, "order/service.go", `package order

func CreateOrder() { validateOrder() }
func validateOrder() {}
func Caller() { CreateOrder() }
`)
	writeRootFixtureFile(t, dir, "order/service_test.go", `package order

func TestCreateOrder(t *testing.T) {}
`)

	traceOutput := executeRootJSON(t, dir, "trace", "symbol", "CreateOrder", "--json")
	assertString(t, traceOutput, "result_kind", "nucleus.trace_result")
	assertBool(t, traceOutput, "ok", true)
	callers := assertSlice(t, traceOutput, "callers")
	if len(callers) != 1 {
		t.Fatalf("trace callers = %#v", callers)
	}

	impactOutput := executeRootJSON(t, dir, "impact", "symbol", "CreateOrder", "--json")
	assertString(t, impactOutput, "result_kind", "nucleus.impact_result")
	assertBool(t, impactOutput, "ok", true)
	tests := assertSlice(t, impactOutput, "affected_tests")
	if len(tests) != 1 {
		t.Fatalf("impact tests = %#v", tests)
	}
}

func executeRootJSON(t *testing.T, dir string, args ...string) map[string]any {
	t.Helper()
	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(append([]string{"--dir", dir}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v\n%s", args, err, stdout.String())
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	return output
}
