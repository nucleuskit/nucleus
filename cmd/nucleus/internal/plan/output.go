package plan

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/decision"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/recipe"
)

// OutputOptions controls the plan JSON payload.
type OutputOptions struct {
	Dir        string
	Task       string
	Executable bool
}

type planSummary struct {
	TaskType             string `json:"task_type"`
	ContractFirst        bool   `json:"contract_first"`
	SuggestedEdits       int    `json:"suggested_edits"`
	BlockedEdits         int    `json:"blocked_edits"`
	Commands             int    `json:"commands"`
	Risks                int    `json:"risks"`
	BlockedDecisions     int    `json:"blocked_decisions"`
	AffectedSymbols      int    `json:"affected_symbols"`
	AffectedRoutes       int    `json:"affected_routes"`
	AffectedTests        int    `json:"affected_tests"`
	AffectedCapabilities int    `json:"affected_capabilities"`
}

// BuildOutput builds either the default plan payload or executable plan payload.
func BuildOutput(opts OutputOptions) (map[string]any, error) {
	output, err := Build(opts.Dir, opts.Task)
	if err != nil {
		return nil, err
	}
	if opts.Executable {
		return BuildExecutable(output), nil
	}
	return output, nil
}

// Build describes a service directory and plans bounded edits for a task.
func Build(dir string, task string) (map[string]any, error) {
	description, err := inspect.Describe(dir)
	if err != nil {
		return nil, err
	}
	requestedCapabilities := requestedCapabilities(task)
	taskType := taskType(task)
	if taskType == taskTypeGeneral && len(requestedCapabilities) > 0 {
		taskType = taskTypeCapability
	}
	suggestedEdits, blockedEdits := suggestedEdits(taskType, description, requestedCapabilities)
	commands := commands(taskType, task, requestedCapabilities)
	risks := risks(taskType, description, requestedCapabilities)
	recipeCandidates := recipe.Candidates(dir, recipe.CandidateQuery{Task: task, Kinds: requestedCapabilities, Limit: 20})
	if recipeCandidates.Diagnostics.Failed() {
		risks = append(risks, "存在无效 recipe，已从 plan candidates 中忽略")
	}
	decisionState := decision.PlanStateForDir(dir)
	manifestCapabilities := loadPlanCapabilities(dir)
	blockedDecisions := lockedDecisionBlocks(task, requestedCapabilities, manifestCapabilities, decisionState)
	if len(blockedDecisions) > 0 {
		risks = append(risks, "locked decision 阻止 provider/library/driver 静默替换")
	}
	if len(blockedEdits) > 0 {
		risks = append(risks, "部分建议改动不在 describe edit surfaces allowed 范围内")
	}
	risks = uniqueStrings(risks)
	contractFirst := contractFirst(taskType)
	impact := buildImpactSummary(dir, task, taskType, description, requestedCapabilities, suggestedEdits, commands)
	diagnostics := planDiagnostics(description.Diagnostics, decisionState.Diagnostics, recipeCandidates.Diagnostics)
	ok := len(blockedEdits) == 0 && len(blockedDecisions) == 0 && !diagnostics.Failed()

	return map[string]any{
		"result_kind":       resultKindPlan,
		"ok":                ok,
		"summary":           buildSummary(taskType, contractFirst, suggestedEdits, blockedEdits, blockedDecisions, commands, risks, impact),
		"schema_version":    schemaVersionPlan,
		"kind":              planKind,
		"schema_ref":        schemaRefPlan,
		"evidence_schema":   schemaRefEvidence,
		"diagnostics":       diagnostics,
		"task":              task,
		"task_type":         taskType,
		"contract_first":    contractFirst,
		"suggested_edits":   suggestedEdits,
		"generated_outputs": generatedOutputs(taskType),
		"blocked_edits":     blockedEdits,
		"blocked_decisions": blockedDecisions,
		"forbidden_edits":   description.EditSurfaces.Forbidden,
		"readonly_edits":    description.EditSurfaces.Readonly,
		"allowed_edits":     description.EditSurfaces.Allowed,
		"commands":          commands,
		"risks":             risks,
		"impact_summary":    impact,
		"context": map[string]any{
			"service":                description.Service,
			"existing_endpoints":     description.Endpoints,
			"grpc_services":          description.GRPCServices,
			"config_keys":            configKeys(description),
			"generated_freshness":    description.GeneratedFreshness,
			"edit_surfaces":          description.EditSurfaces,
			"declared_capabilities":  description.Capabilities,
			"requested_capabilities": capabilityContext(task, description, requestedCapabilities),
			"locked_decisions":       decisionState.Locked,
			"decision_diagnostics":   decisionState.Diagnostics,
			"recipe_candidates":      recipeCandidates.Candidates,
			"recipe_diagnostics":     recipeCandidates.Diagnostics,
			"recipe_policy":          recipePolicy(),
		},
	}, nil
}

func planDiagnostics(parts ...diagnostic.Diagnostics) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	for _, part := range parts {
		diagnostics = append(diagnostics, part...)
	}
	if diagnostics == nil {
		return diagnostic.Diagnostics{}
	}
	diagnostics.Sort()
	return diagnostics
}

func buildSummary(taskType string, contractFirst bool, suggestedEdits []string, blockedEdits []string, blockedDecisions []lockedDecisionBlock, commands []string, risks []string, impact impactSummary) planSummary {
	return planSummary{
		TaskType:             taskType,
		ContractFirst:        contractFirst,
		SuggestedEdits:       len(suggestedEdits),
		BlockedEdits:         len(blockedEdits),
		Commands:             len(commands),
		Risks:                len(risks),
		BlockedDecisions:     len(blockedDecisions),
		AffectedSymbols:      len(impact.AffectedSymbols),
		AffectedRoutes:       len(impact.AffectedRoutes),
		AffectedTests:        len(impact.AffectedTests),
		AffectedCapabilities: len(impact.AffectedCapabilities),
	}
}

func configKeys(description any) any {
	value := reflect.ValueOf(description)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}
	field := value.FieldByName("ConfigKeys")
	if !field.IsValid() || !field.CanInterface() {
		return nil
	}
	return field.Interface()
}

func recipePolicy() map[string]any {
	return map[string]any{
		"schema_ref":        recipeSchemaRef,
		"selection":         "candidate_only",
		"decision_required": true,
		"may_write_files":   false,
		"may_execute":       false,
		"may_accept":        false,
	}
}

func renderHuman(stdout io.Writer, stderr io.Writer, output map[string]any) {
	summary, _ := output["summary"].(planSummary)
	blockedEdits := anyStringSlice(output["blocked_edits"])
	if len(blockedEdits) > 0 || blockedDecisionCount(output) > 0 {
		for _, path := range blockedEdits {
			_, _ = fmt.Fprintf(stderr, "blocked %s: not covered by describe edit_surfaces.allowed\n", path)
		}
		if blockedDecisionCount(output) > 0 {
			_, _ = fmt.Fprintf(stderr, "blocked decisions: %d locked decision conflict(s)\n", blockedDecisionCount(output))
		}
		return
	}

	_, _ = fmt.Fprintln(stdout, "OK")
	_, _ = fmt.Fprintf(stdout, "planned: %s\n", stringField(output, "task_type"))
	_, _ = fmt.Fprintf(stdout, "contract first: %t\n", summary.ContractFirst)
	_, _ = fmt.Fprintf(stdout, "edits: %d allowed, %d blocked\n", summary.SuggestedEdits, summary.BlockedEdits)
	_, _ = fmt.Fprintf(stdout, "impact: %d symbols, %d routes, %d tests, %d capabilities\n", summary.AffectedSymbols, summary.AffectedRoutes, summary.AffectedTests, summary.AffectedCapabilities)
	if generatedOutputs := anyStringSlice(output["generated_outputs"]); len(generatedOutputs) > 0 {
		_, _ = fmt.Fprintf(stdout, "generated: %s\n", strings.Join(generatedOutputs, ", "))
	}
	if commands := anyStringSlice(output["commands"]); len(commands) > 0 {
		_, _ = fmt.Fprintf(stdout, "commands: %s\n", strings.Join(commands, " -> "))
	}
	if risks := anyStringSlice(output["risks"]); len(risks) > 0 {
		_, _ = fmt.Fprintf(stdout, "risks: %s\n", strings.Join(risks, "; "))
	} else {
		_, _ = fmt.Fprintln(stdout, "risks: none")
	}
}

func renderJSON(writer io.Writer, output map[string]any, pretty bool) error {
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(output)
}

func stringField(output map[string]any, key string) string {
	value, _ := output[key].(string)
	return value
}
