package plan

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/decision"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/recipe"
)

func TestBuildOutputAddsPlanMetadata(t *testing.T) {
	dir := newPlanFixture(t, []string{"api/**", "internal/domain/**", "internal/adapter/http/**"})

	output, err := BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "新增 HTTP 接口",
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	if output["result_kind"] != resultKindPlan {
		t.Fatalf("result_kind = %v, want %s", output["result_kind"], resultKindPlan)
	}
	if output["schema_version"] != schemaVersionPlan {
		t.Fatalf("schema_version = %v, want %s", output["schema_version"], schemaVersionPlan)
	}
	if output["schema_ref"] != schemaRefPlan {
		t.Fatalf("schema_ref = %v, want %s", output["schema_ref"], schemaRefPlan)
	}
	if output["ok"] != true {
		t.Fatalf("ok = %v, want true", output["ok"])
	}
	if diagnostics, ok := output["diagnostics"].(diagnostic.Diagnostics); !ok || len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty diagnostic.Diagnostics", output["diagnostics"])
	}
	summary, ok := output["summary"].(planSummary)
	if !ok {
		t.Fatalf("summary has type %T, want planSummary", output["summary"])
	}
	if summary.TaskType != taskTypeHTTPEndpoint {
		t.Fatalf("summary.task_type = %q, want %q", summary.TaskType, taskTypeHTTPEndpoint)
	}
	if !summary.ContractFirst {
		t.Fatal("summary.contract_first = false, want true")
	}
	if !containsString(anyStringSlice(output["commands"]), commandValidate) {
		t.Fatalf("commands = %#v, want validate command", output["commands"])
	}
}

func TestBuildOutputExecutableMarksBlockedEditsRequired(t *testing.T) {
	dir := newPlanFixture(t, []string{"docs/**"})

	output, err := BuildOutput(OutputOptions{
		Dir:        dir,
		Task:       "新增 HTTP 接口",
		Executable: true,
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	if output["result_kind"] != resultKindExecutablePlan {
		t.Fatalf("result_kind = %v, want %s", output["result_kind"], resultKindExecutablePlan)
	}
	if output["schema_version"] != schemaVersionExecutable {
		t.Fatalf("schema_version = %v, want %s", output["schema_version"], schemaVersionExecutable)
	}
	if output["schema_ref"] != schemaRefPlanExecutable {
		t.Fatalf("schema_ref = %v, want %s", output["schema_ref"], schemaRefPlanExecutable)
	}
	if output["ok"] != false {
		t.Fatalf("ok = %v, want false", output["ok"])
	}
	if diagnostics, ok := output["diagnostics"].(diagnostic.Diagnostics); !ok || len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty diagnostic.Diagnostics", output["diagnostics"])
	}
	blocked, ok := output["blocked_edits"].([]map[string]any)
	if !ok {
		t.Fatalf("blocked_edits has type %T, want []map[string]any", output["blocked_edits"])
	}
	if len(blocked) == 0 {
		t.Fatal("blocked_edits = empty, want blocked edit")
	}
	if blocked[0]["required"] != true {
		t.Fatalf("blocked_edits[0].required = %v, want true", blocked[0]["required"])
	}
}

func TestExecutablePlanAcceptsHTTPScenarioEvidence(t *testing.T) {
	dir := newPlanFixture(t, []string{"api/**", "internal/domain/**", "internal/adapter/http/**"})

	output, err := BuildOutput(OutputOptions{
		Dir:        dir,
		Task:       "新增 HTTP 接口",
		Executable: true,
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	policy, ok := output["evidence_policy"].(map[string]any)
	if !ok {
		t.Fatalf("evidence_policy has type %T, want map[string]any", output["evidence_policy"])
	}
	kinds := anyStringSlice(policy["accepted_kinds"])
	if !containsString(kinds, evidenceKindHTTPScenario) {
		t.Fatalf("accepted_kinds = %#v, want %s", kinds, evidenceKindHTTPScenario)
	}
	if got := commandProduces("nucleus scenario --dir . --json"); got != commandProducesExit {
		t.Fatalf("plain scenario produces = %q, want %q", got, commandProducesExit)
	}
	if got := commandProduces("nucleus scenario --dir . --run-http --base-url http://127.0.0.1 --json"); got != evidenceKindHTTPScenario {
		t.Fatalf("run-http scenario produces = %q, want %q", got, evidenceKindHTTPScenario)
	}
	if got := commandSchemaRef("nucleus scenario --dir . --cases cases.json --base-url http://127.0.0.1 --json"); got != schemaRefEvidence {
		t.Fatalf("cases scenario schema_ref = %q, want %q", got, schemaRefEvidence)
	}
}

func TestCapabilityPlanRequiresDecisionEvidenceInsteadOfProviderScaffold(t *testing.T) {
	dir := newPlanFixture(t, []string{"nucleus.yaml", ".nucleus/**", "docs/**"})

	output, err := BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "add mysql capability",
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	for _, command := range anyStringSlice(output["commands"]) {
		if strings.Contains(command, "capability add") || strings.Contains(command, "--provider") {
			t.Fatalf("capability plan leaked provider scaffold command %q", command)
		}
	}
	for _, path := range anyStringSlice(output["generated_outputs"]) {
		if path == "go.mod" || path == "go.sum" || strings.HasPrefix(path, "internal/") {
			t.Fatalf("capability plan leaked implementation output %q", path)
		}
	}
	context, ok := output["context"].(map[string]any)
	if !ok {
		t.Fatalf("context has type %T, want map[string]any", output["context"])
	}
	rawCapabilities, ok := context["requested_capabilities"].([]map[string]any)
	if !ok || len(rawCapabilities) != 1 {
		t.Fatalf("requested_capabilities = %#v, want one structured capability", context["requested_capabilities"])
	}
	capability := rawCapabilities[0]
	if _, exists := capability["provider_hint"]; exists {
		t.Fatalf("capability context leaked provider_hint: %#v", capability)
	}
	if _, exists := capability["module"]; exists {
		t.Fatalf("capability context leaked fixed module mapping: %#v", capability)
	}
	if capability["decision_schema_ref"] != decisionSchemaRef || capability["provider_selection"] != "decision_only" {
		t.Fatalf("capability context missing decision evidence contract: %#v", capability)
	}
}

func TestCapabilityPlanUsesVocabularyWithoutProviderCatalogFalsePositive(t *testing.T) {
	dir := newPlanFixture(t, []string{"nucleus.yaml", ".nucleus/**", "docs/**"})

	output, err := BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "move capability kind suggestions from Go catalog to vocab data",
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	context, ok := output["context"].(map[string]any)
	if !ok {
		t.Fatalf("context has type %T, want map[string]any", output["context"])
	}
	rawCapabilities, ok := context["requested_capabilities"].([]map[string]any)
	if !ok {
		t.Fatalf("requested_capabilities has type %T", context["requested_capabilities"])
	}
	if len(rawCapabilities) != 0 {
		t.Fatalf("catalog should not false-match log capability: %#v", rawCapabilities)
	}
}

func TestCapabilityPlanMatchesVocabularyAliases(t *testing.T) {
	dir := newPlanFixture(t, []string{"nucleus.yaml", ".nucleus/**", "docs/**"})

	output, err := BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "接入消息队列并增加指标",
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	context, ok := output["context"].(map[string]any)
	if !ok {
		t.Fatalf("context has type %T, want map[string]any", output["context"])
	}
	rawCapabilities, ok := context["requested_capabilities"].([]map[string]any)
	if !ok {
		t.Fatalf("requested_capabilities has type %T", context["requested_capabilities"])
	}
	names := map[string]bool{}
	for _, capability := range rawCapabilities {
		name, _ := capability["name"].(string)
		names[name] = true
	}
	if !names["mq"] || !names["metric"] {
		t.Fatalf("requested capability names = %#v, want mq and metric", names)
	}
}

func TestPlanIncludesRecipeCandidatesWithoutSelectingProvider(t *testing.T) {
	dir := newPlanFixture(t, []string{"nucleus.yaml", ".nucleus/**", "docs/**"})
	writePlanFile(t, dir, ".nucleus/recipes/gorm.yaml", `schema_version: "recipe.v1"
id: gorm-sql
kind: sql
provider: gorm
language: go
suggest:
  interfaces:
    - keep ORM behind a project-owned interface
  verification:
    - go test ./storage
risks:
  - transaction boundary must be explicit
`)

	output, err := BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "add mysql capability",
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	context := output["context"].(map[string]any)
	candidates, ok := context["recipe_candidates"].([]recipe.Candidate)
	if !ok || len(candidates) != 2 {
		t.Fatalf("recipe_candidates = %#v, want project and built-in candidates", context["recipe_candidates"])
	}
	projectCandidate := findRecipeCandidate(t, candidates, "gorm-sql")
	if projectCandidate.Selection != "candidate_only" || !projectCandidate.DecisionRequired {
		t.Fatalf("recipe candidate should not be selected automatically: %#v", projectCandidate)
	}
	if projectCandidate.Source != "project" {
		t.Fatalf("project candidate source = %q, want project", projectCandidate.Source)
	}
	builtinCandidate := findRecipeCandidate(t, candidates, "sql-port-boundary")
	if builtinCandidate.Provider != "" || builtinCandidate.Source != "builtin" {
		t.Fatalf("builtin candidate should be provider-neutral: %#v", builtinCandidate)
	}
	policy, ok := context["recipe_policy"].(map[string]any)
	if !ok {
		t.Fatalf("recipe_policy has type %T", context["recipe_policy"])
	}
	if policy["may_write_files"] != false || policy["may_execute"] != false || policy["may_accept"] != false {
		t.Fatalf("recipe policy is not read-only: %#v", policy)
	}
	for _, command := range anyStringSlice(output["commands"]) {
		if strings.Contains(command, "go test ./storage") || strings.Contains(command, "gorm") {
			t.Fatalf("recipe suggestion leaked into executable commands: %q", command)
		}
	}
	for _, generated := range anyStringSlice(output["generated_outputs"]) {
		if generated == "go.mod" || generated == "go.sum" {
			t.Fatalf("recipe candidate leaked dependency output: %q", generated)
		}
	}
}

func TestPlanIgnoresUnsafeRecipeAndSurfacesDiagnostics(t *testing.T) {
	dir := newPlanFixture(t, []string{"nucleus.yaml", ".nucleus/**", "docs/**"})
	writePlanFile(t, dir, ".nucleus/recipes/unsafe.yaml", `schema_version: "recipe.v1"
id: unsafe
kind: sql
language: go
commands:
  - go get gorm.io/gorm
`)

	output, err := BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "add mysql capability",
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	context := output["context"].(map[string]any)
	candidates, ok := context["recipe_candidates"].([]recipe.Candidate)
	if !ok {
		t.Fatalf("recipe_candidates has type %T", context["recipe_candidates"])
	}
	if containsRecipeCandidate(candidates, "unsafe") {
		t.Fatalf("unsafe recipe became a candidate: %#v", candidates)
	}
	if !containsRecipeCandidate(candidates, "sql-port-boundary") {
		t.Fatalf("built-in safe recipe missing: %#v", candidates)
	}
	diagnostics, ok := context["recipe_diagnostics"].(diagnostic.Diagnostics)
	if ok && !diagnostics.Failed() {
		t.Fatalf("recipe diagnostics did not fail: %#v", diagnostics)
	}
	if !ok {
		t.Fatalf("recipe_diagnostics has type %T", context["recipe_diagnostics"])
	}
	if !containsString(anyStringSlice(output["risks"]), "存在无效 recipe，已从 plan candidates 中忽略") {
		t.Fatalf("risks = %#v, want invalid recipe risk", output["risks"])
	}
}

func TestPlanBlocksLockedDecisionProviderChangeUntilSupersedeExists(t *testing.T) {
	dir := newDecisionPlanFixture(t)
	acceptPlanDecision(t, dir, ".nucleus/decisions/order-store.yaml")

	output, err := BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "switch order_store provider to xorm",
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	if output["ok"] != false {
		t.Fatalf("ok = %v, want false for locked decision conflict", output["ok"])
	}
	blocks, ok := output["blocked_decisions"].([]lockedDecisionBlock)
	if !ok || len(blocks) != 1 {
		t.Fatalf("blocked_decisions = %#v, want one block", output["blocked_decisions"])
	}
	if blocks[0].DecisionID != "order-store-provider" || blocks[0].RequiredAction == "" {
		t.Fatalf("unexpected locked decision block: %#v", blocks[0])
	}
	summary := output["summary"].(planSummary)
	if summary.BlockedDecisions != 1 {
		t.Fatalf("summary = %#v, want one blocked decision", summary)
	}

	writePlanFile(t, dir, ".nucleus/decisions/order-store-xorm.yaml", `schema_version: "decision.v1"
id: order-store-provider-v2
capability: order_store
supersedes: order-store-provider
decision:
  provider: xorm
  library: xorm.io/xorm
  status: proposed
  locked: false
reason:
  - replace locked provider through explicit supersede evidence
impact:
  files:
    - internal/order/store.go
verification:
  commands:
    - go test ./internal/order
`)
	supersedePlanDecision(t, dir, ".nucleus/decisions/order-store-xorm.yaml")

	output, err = BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "switch order_store provider to xorm",
	})
	if err != nil {
		t.Fatalf("BuildOutput() after supersede error = %v", err)
	}
	if output["ok"] != true {
		t.Fatalf("ok = %v, want true after supersede; blocked=%#v risks=%#v", output["ok"], output["blocked_decisions"], output["risks"])
	}
	if blocks := output["blocked_decisions"].([]lockedDecisionBlock); len(blocks) != 0 {
		t.Fatalf("blocked decisions after supersede = %#v", blocks)
	}
}

func TestPlanEmbedsImpactSummaryFromGraphFacts(t *testing.T) {
	dir := newPlanImpactFixture(t)

	output, err := BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "add order status filter to ListOrders",
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	impact, ok := output["impact_summary"].(impactSummary)
	if !ok {
		t.Fatalf("impact_summary has type %T", output["impact_summary"])
	}
	if impact.Mode != "best_effort" {
		t.Fatalf("impact mode = %q", impact.Mode)
	}
	if !containsPlanSymbol(impact.AffectedSymbols, "ListOrders") {
		t.Fatalf("affected symbols = %#v, want ListOrders", impact.AffectedSymbols)
	}
	if !containsPlanSymbol(impact.AffectedTests, "TestListOrders") {
		t.Fatalf("affected tests = %#v, want TestListOrders", impact.AffectedTests)
	}
	if !containsString(impact.AffectedContracts, "api/openapi.yaml") {
		t.Fatalf("affected contracts = %#v, want openapi", impact.AffectedContracts)
	}
	if !containsString(impact.AffectedCapabilities, "order_store") {
		t.Fatalf("affected capabilities = %#v, want order_store", impact.AffectedCapabilities)
	}
	if !containsPlanRoute(impact.AffectedRoutes, "GET", "/orders") {
		t.Fatalf("affected routes = %#v, want GET /orders", impact.AffectedRoutes)
	}
	if len(impact.GraphEdges) == 0 {
		t.Fatalf("graph_edges = empty, want impact evidence")
	}
	if !containsString(impact.SuggestedVerification, commandVerifyJSON) {
		t.Fatalf("suggested verification = %#v, want verify command", impact.SuggestedVerification)
	}
	summary := output["summary"].(planSummary)
	if summary.AffectedSymbols == 0 || summary.AffectedRoutes == 0 || summary.AffectedTests == 0 || summary.AffectedCapabilities == 0 {
		t.Fatalf("summary missing impact counts: %#v", summary)
	}
}

func TestExecutablePlanIncludesImpactSummary(t *testing.T) {
	dir := newPlanImpactFixture(t)

	output, err := BuildOutput(OutputOptions{
		Dir:        dir,
		Task:       "change ListOrders",
		Executable: true,
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	impact, ok := output["impact_summary"].(impactSummary)
	if !ok {
		t.Fatalf("impact_summary has type %T", output["impact_summary"])
	}
	if !containsPlanSymbol(impact.AffectedSymbols, "ListOrders") {
		t.Fatalf("affected symbols = %#v, want ListOrders", impact.AffectedSymbols)
	}
}

func TestCommandJSONBlockedReturnsSentinel(t *testing.T) {
	dir := newPlanFixture(t, []string{"docs/**"})
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--task", "新增 HTTP 接口", "--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrPlanBlocked) {
		t.Fatalf("execute plan error = %v, want ErrPlanBlocked", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON output", stderr.String())
	}

	var output struct {
		ResultKind   string      `json:"result_kind"`
		OK           bool        `json:"ok"`
		Summary      planSummary `json:"summary"`
		BlockedEdits []string    `json:"blocked_edits"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.ResultKind != resultKindPlan {
		t.Fatalf("result_kind = %q, want %q", output.ResultKind, resultKindPlan)
	}
	if output.OK {
		t.Fatal("ok = true, want false")
	}
	if output.Summary.BlockedEdits == 0 {
		t.Fatalf("summary.blocked_edits = 0, want blocked count; output=%s", stdout.String())
	}
	if len(output.BlockedEdits) == 0 {
		t.Fatalf("blocked_edits = empty, want blocked paths; output=%s", stdout.String())
	}
}

func TestCommandHumanSuccessOutput(t *testing.T) {
	dir := newPlanFixture(t, []string{"api/**", "internal/domain/**", "internal/adapter/http/**"})
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--task", "新增 HTTP 接口"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute plan: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"OK",
		"planned: http_endpoint",
		"contract first: true",
		"edits:",
		"commands:",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestCommandPrettyJSONOutput(t *testing.T) {
	dir := newPlanFixture(t, []string{"api/**", "internal/domain/**", "internal/adapter/http/**"})
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--task", "新增 HTTP 接口", "--json", "--pretty"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute plan: %v", err)
	}
	if !strings.Contains(stdout.String(), "\n  \"result_kind\"") {
		t.Fatalf("stdout = %q, want indented JSON", stdout.String())
	}
}

func newPlanImpactFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writePlanFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: orders
  version: "0.1.0"
contracts:
  - id: http
    kind: openapi
    path: api/openapi.yaml
capabilities:
  - id: order_store
    kind: relational_store
    symbols:
      - id: go://example.com/orders/order#OrderStore
        status: resolved
ai:
  intent: test
  allowed_changes:
    - api/**
    - order/**
    - nucleus.yaml
verify:
  commands:
    - go test ./order
`)
	writePlanFile(t, dir, "go.mod", "module example.com/orders\n\ngo 1.26.3\n")
	writePlanFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /orders:
    get:
      operationId: listOrders
      responses:
        "200":
          description: ok
`)
	writePlanFile(t, dir, "order/service.go", `package order

type OrderStore interface { List() error }

func ListOrders(store OrderStore) { loadOrders() }
func loadOrders() {}
func Handler(store OrderStore) { ListOrders(store) }
`)
	writePlanFile(t, dir, "order/service_test.go", `package order

import "testing"

func TestListOrders(t *testing.T) { ListOrders(nil) }
`)
	return dir
}

func writePlanFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsPlanSymbol(nodes []inspect.SymbolNode, name string) bool {
	for _, node := range nodes {
		if node.Name == name {
			return true
		}
	}
	return false
}

func containsPlanRoute(routes []routeImpact, method string, path string) bool {
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}

func newPlanFixture(t *testing.T, allowedChanges []string) string {
	t.Helper()
	dir := t.TempDir()
	manifest := `schema_version: "2.0"
service:
  name: fixture
  version: "0.1.0"
ai:
  intent: test
  allowed_changes:
`
	for _, path := range allowedChanges {
		manifest += "    - " + quoteYAMLString(path) + "\n"
	}
	manifest += `capabilities: []
`
	if err := os.WriteFile(filepath.Join(dir, "nucleus.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write nucleus.yaml: %v", err)
	}
	return dir
}

func newDecisionPlanFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writePlanFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: orders
  version: "0.1.0"
capabilities:
  - id: order_store
    kind: sql
ai:
  intent: test
  allowed_changes:
    - nucleus.yaml
    - .nucleus/**
    - docs/**
    - internal/**
`)
	writePlanFile(t, dir, "go.mod", "module example.com/orders\n\ngo 1.26.3\n")
	writePlanFile(t, dir, "internal/order/store.go", "package order\n")
	writePlanFile(t, dir, ".nucleus/decisions/order-store.yaml", `schema_version: "decision.v1"
id: order-store-provider
capability: order_store
decision:
  provider: gorm
  library: gorm.io/gorm
  status: proposed
  locked: false
reason:
  - project already uses gorm in storage package
impact:
  files:
    - internal/order/store.go
verification:
  commands:
    - go test ./internal/order
`)
	return dir
}

func acceptPlanDecision(t *testing.T, dir string, path string) {
	t.Helper()
	cmd := decision.NewCommand(decision.Config{Dir: &dir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"accept", path, "--by", "human", "--accepted-at", "2026-07-03T00:00:00Z", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("accept decision %s: %v", path, err)
	}
}

func supersedePlanDecision(t *testing.T, dir string, path string) {
	t.Helper()
	cmd := decision.NewCommand(decision.Config{Dir: &dir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"supersede", path, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("supersede decision %s: %v", path, err)
	}
}

func quoteYAMLString(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func findRecipeCandidate(t *testing.T, values []recipe.Candidate, id string) recipe.Candidate {
	t.Helper()
	for _, value := range values {
		if value.ID == id {
			return value
		}
	}
	t.Fatalf("recipe candidate %q not found in %#v", id, values)
	return recipe.Candidate{}
}

func containsRecipeCandidate(values []recipe.Candidate, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}
