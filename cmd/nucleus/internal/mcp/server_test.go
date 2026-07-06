package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestToolsIncludeAgentNativeSet(t *testing.T) {
	server := NewServer(t.TempDir())
	names := map[string]bool{}
	for _, tool := range server.Tools() {
		names[tool.Name] = true
	}
	for _, want := range []string{
		toolGetServiceDescription,
		toolGetEditSurfaces,
		toolGetContracts,
		toolGetCapabilities,
		toolTraceSymbol,
		toolTraceRoute,
		toolTraceCapability,
		toolImpactSymbol,
		toolImpactFile,
		toolImpactContract,
		toolFindSymbol,
		toolListCallers,
		toolListCallees,
		toolValidateDecision,
		toolListDecisions,
		toolGetReport,
		toolBuildPlan,
		toolListRecipes,
		toolGetRecipe,
	} {
		if !names[want] {
			t.Fatalf("tool %q missing from %#v", want, names)
		}
	}
}

func TestCallToolBuildPlanAndTraceSymbol(t *testing.T) {
	dir := writeMCPFixture(t)
	server := NewServer(dir)

	planRaw, err := server.CallTool(toolBuildPlan, map[string]any{"task": "change ListOrders", "executable": true})
	if err != nil {
		t.Fatalf("build_plan: %v", err)
	}
	planOutput := asMap(t, planRaw)
	if planOutput["result_kind"] != "nucleus.executable_plan_result" {
		t.Fatalf("plan result_kind = %#v", planOutput["result_kind"])
	}
	impactSummary := asMap(t, planOutput["impact_summary"])
	if len(asSlice(t, impactSummary["affected_symbols"])) == 0 {
		t.Fatalf("impact_summary missing symbols: %#v", impactSummary)
	}

	traceRaw, err := server.CallTool(toolTraceSymbol, map[string]any{"symbol": "ListOrders"})
	if err != nil {
		t.Fatalf("trace_symbol: %v", err)
	}
	traceOutput := asMap(t, traceRaw)
	if traceOutput["result_kind"] != "nucleus.trace_result" || traceOutput["ok"] != true {
		t.Fatalf("unexpected trace output: %#v", traceOutput)
	}
	if len(asSlice(t, traceOutput["callers"])) == 0 {
		t.Fatalf("trace callers missing: %#v", traceOutput)
	}
}

func TestCallToolOutputsCarryAgentEnvelope(t *testing.T) {
	dir := writeMCPFixture(t)
	server := NewServer(dir)
	cases := []struct {
		name          string
		args          map[string]any
		schemaVersion string
		schemaRef     string
	}{
		{toolGetServiceDescription, nil, "describe-result.v1", "contract/schema/describe-result.v1.schema.json"},
		{toolGetEditSurfaces, nil, "mcp-result.v1", "contract/schema/mcp-result.v1.schema.json"},
		{toolGetContracts, nil, "mcp-result.v1", "contract/schema/mcp-result.v1.schema.json"},
		{toolGetCapabilities, nil, "mcp-result.v1", "contract/schema/mcp-result.v1.schema.json"},
		{toolFindSymbol, map[string]any{"query": "ListOrders"}, "mcp-result.v1", "contract/schema/mcp-result.v1.schema.json"},
		{toolListCallers, map[string]any{"symbol": "ListOrders"}, "mcp-result.v1", "contract/schema/mcp-result.v1.schema.json"},
		{toolListCallees, map[string]any{"symbol": "ListOrders"}, "mcp-result.v1", "contract/schema/mcp-result.v1.schema.json"},
		{toolListRecipes, nil, "recipe-result.v1", "contract/schema/recipe-result.v1.schema.json"},
		{toolGetRecipe, map[string]any{"id": "sql-port-boundary"}, "recipe-result.v1", "contract/schema/recipe-result.v1.schema.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := server.CallTool(tc.name, tc.args)
			if err != nil {
				t.Fatalf("CallTool(%s): %v", tc.name, err)
			}
			output := asMap(t, raw)
			assertEnvelope(t, output, tc.schemaVersion, tc.schemaRef)
		})
	}
}

func TestRecipeToolsAreReadOnlyStructuredKnowledge(t *testing.T) {
	dir := t.TempDir()
	writeMCPFile(t, dir, ".nucleus/recipes/gorm.yaml", `schema_version: "recipe.v1"
id: gorm-relational-store
kind: relational_store
provider: gorm
language: go
detect:
  imports:
    - gorm.io/gorm
suggest:
  interfaces:
    - keep ORM behind interface
  verification:
    - go test ./...
risks:
  - transaction boundaries must be explicit
`)
	writeMCPFile(t, dir, ".nucleus/recipes/unsafe.yaml", `schema_version: "recipe.v1"
id: unsafe
kind: relational_store
language: go
commands:
  - rm -rf .
`)

	server := NewServer(dir)
	listRaw, err := server.CallTool(toolListRecipes, map[string]any{"kind": "relational_store"})
	if err != nil {
		t.Fatalf("list_recipes: %v", err)
	}
	listOutput := asMap(t, listRaw)
	recipes := asSlice(t, listOutput["recipes"])
	if len(recipes) != 1 {
		t.Fatalf("recipes = %#v, want only safe recipe", recipes)
	}
	diagnostics := asSlice(t, listOutput["diagnostics"])
	if len(diagnostics) == 0 {
		t.Fatalf("diagnostics should include unsafe recipe parse failure: %#v", listOutput)
	}

	recipeRaw, err := server.CallTool(toolGetRecipe, map[string]any{"id": "gorm-relational-store"})
	if err != nil {
		t.Fatalf("get_recipe: %v", err)
	}
	recipeOutput := asMap(t, recipeRaw)
	if recipeOutput["ok"] != true {
		t.Fatalf("recipe output = %#v", recipeOutput)
	}
}

func TestServeHandlesInitializeAndToolCallFrames(t *testing.T) {
	dir := writeMCPFixture(t)
	server := NewServer(dir)
	initialize := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	listTools := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	input := bytes.NewReader(append(frame(initialize), frame(listTools)...))
	var output bytes.Buffer

	if err := server.Serve(input, &output); err != nil {
		t.Fatalf("serve: %v", err)
	}
	first, err := readFrame(bufio.NewReader(bytes.NewReader(output.Bytes())))
	if err != nil {
		t.Fatalf("read first frame: %v", err)
	}
	var response map[string]any
	if err := json.Unmarshal(first, &response); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if response["id"].(float64) != 1 {
		t.Fatalf("first response = %#v", response)
	}
}

func writeMCPFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeMCPFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: orders
  version: "0.1.0"
capabilities:
  - id: order_store
    kind: relational_store
    symbols:
      - id: go://example.com/orders/order#OrderStore
        status: resolved
ai:
  intent: mcp test
  allowed_changes:
    - order/**
    - api/**
`)
	writeMCPFile(t, dir, "go.mod", "module example.com/orders\n\ngo 1.26.3\n")
	writeMCPFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /orders:
    get:
      operationId: listOrders
      responses:
        "200":
          description: ok
`)
	writeMCPFile(t, dir, "order/service.go", `package order

type OrderStore interface { List() error }

func ListOrders(store OrderStore) { loadOrders() }
func loadOrders() {}
func Handler(store OrderStore) { ListOrders(store) }
`)
	return dir
}

func writeMCPFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("decode map: %v\n%s", err, string(data))
	}
	return output
}

func asSlice(t *testing.T, value any) []any {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("value has type %T, want []any", value)
	}
	return items
}

func assertEnvelope(t *testing.T, output map[string]any, schemaVersion string, schemaRef string) {
	t.Helper()
	if output["result_kind"] == "" {
		t.Fatalf("missing result_kind: %#v", output)
	}
	if output["schema_version"] != schemaVersion {
		t.Fatalf("schema_version = %#v, want %s", output["schema_version"], schemaVersion)
	}
	if output["schema_ref"] != schemaRef {
		t.Fatalf("schema_ref = %#v, want %s", output["schema_ref"], schemaRef)
	}
	if _, ok := output["ok"].(bool); !ok {
		t.Fatalf("ok has type %T, want bool", output["ok"])
	}
	if _, ok := output["diagnostics"].([]any); !ok {
		t.Fatalf("diagnostics has type %T, want []any", output["diagnostics"])
	}
}
