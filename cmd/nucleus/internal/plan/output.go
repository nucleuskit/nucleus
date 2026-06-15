package plan

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/nucleuskit/contract/inspect"
)

// OutputOptions controls the plan JSON payload.
type OutputOptions struct {
	Dir        string
	Task       string
	Executable bool
}

type planSummary struct {
	TaskType       string `json:"task_type"`
	ContractFirst  bool   `json:"contract_first"`
	SuggestedEdits int    `json:"suggested_edits"`
	BlockedEdits   int    `json:"blocked_edits"`
	Commands       int    `json:"commands"`
	Risks          int    `json:"risks"`
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
	if len(blockedEdits) > 0 {
		risks = append(risks, "部分建议改动不在 describe edit surfaces allowed 范围内")
	}
	risks = uniqueStrings(risks)
	contractFirst := contractFirst(taskType)

	return map[string]any{
		"result_kind":       resultKindPlan,
		"ok":                len(blockedEdits) == 0,
		"summary":           buildSummary(taskType, contractFirst, suggestedEdits, blockedEdits, commands, risks),
		"schema_version":    schemaVersionPlan,
		"kind":              planKind,
		"schema_ref":        schemaRefPlanExecutable,
		"evidence_schema":   schemaRefEvidence,
		"task":              task,
		"task_type":         taskType,
		"contract_first":    contractFirst,
		"suggested_edits":   suggestedEdits,
		"generated_outputs": generatedOutputs(taskType),
		"blocked_edits":     blockedEdits,
		"forbidden_edits":   description.EditSurfaces.Forbidden,
		"readonly_edits":    description.EditSurfaces.Readonly,
		"allowed_edits":     description.EditSurfaces.Allowed,
		"commands":          commands,
		"risks":             risks,
		"context": map[string]any{
			"service":                description.Service,
			"existing_endpoints":     description.Endpoints,
			"grpc_services":          description.GRPCServices,
			"config_keys":            configKeys(description),
			"generated_freshness":    description.GeneratedFreshness,
			"edit_surfaces":          description.EditSurfaces,
			"declared_capabilities":  description.Capabilities,
			"requested_capabilities": capabilityContext(task, description, requestedCapabilities),
		},
	}, nil
}

func buildSummary(taskType string, contractFirst bool, suggestedEdits []string, blockedEdits []string, commands []string, risks []string) planSummary {
	return planSummary{
		TaskType:       taskType,
		ContractFirst:  contractFirst,
		SuggestedEdits: len(suggestedEdits),
		BlockedEdits:   len(blockedEdits),
		Commands:       len(commands),
		Risks:          len(risks),
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

func renderHuman(stdout io.Writer, stderr io.Writer, output map[string]any) {
	summary, _ := output["summary"].(planSummary)
	blockedEdits := anyStringSlice(output["blocked_edits"])
	if len(blockedEdits) > 0 {
		for _, path := range blockedEdits {
			_, _ = fmt.Fprintf(stderr, "blocked %s: not covered by describe edit_surfaces.allowed\n", path)
		}
		return
	}

	_, _ = fmt.Fprintln(stdout, "OK")
	_, _ = fmt.Fprintf(stdout, "planned: %s\n", stringField(output, "task_type"))
	_, _ = fmt.Fprintf(stdout, "contract first: %t\n", summary.ContractFirst)
	_, _ = fmt.Fprintf(stdout, "edits: %d allowed, %d blocked\n", summary.SuggestedEdits, summary.BlockedEdits)
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
