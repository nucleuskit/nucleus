package plan

import (
	"fmt"
	"strings"
)

// BuildExecutable converts the default plan payload into an executable plan contract.
func BuildExecutable(plan map[string]any) map[string]any {
	task, _ := plan["task"].(string)
	taskType, _ := plan["task_type"].(string)
	contractFirst, _ := plan["contract_first"].(bool)
	suggestedEdits := anyStringSlice(plan["suggested_edits"])
	blockedEdits := anyStringSlice(plan["blocked_edits"])
	commands := anyStringSlice(plan["commands"])
	risks := anyStringSlice(plan["risks"])

	return map[string]any{
		"result_kind":    resultKindExecutablePlan,
		"ok":             len(blockedEdits) == 0,
		"summary":        buildSummary(taskType, contractFirst, suggestedEdits, blockedEdits, commands, risks),
		"schema_version": schemaVersionExecutable,
		"kind":           executablePlanKind,
		"schema_ref":     schemaRefPlanExecutable,
		"schema_refs": map[string]any{
			"plan":     schemaRefPlanExecutable,
			"evidence": schemaRefEvidence,
		},
		"task":      task,
		"task_type": taskType,
		"intent": map[string]any{
			"goal":        task,
			"constraints": intentConstraints(taskType, contractFirst),
			"non_goals":   []string{"do not execute apply/verify/repair in plan mode"},
			"risk_level":  riskLevel(risks),
			"acceptance":  []string{"planned edits, commands, assertions, and rollback are machine-readable"},
		},
		"edits":           executableEdits(suggestedEdits),
		"blocked_edits":   executableBlockedEdits(blockedEdits),
		"commands":        executableCommands(commands),
		"assertions":      executableAssertions(commands, blockedEdits),
		"rollback":        executableRollback(suggestedEdits),
		"risks":           risks,
		"context":         plan["context"],
		"evidence_policy": evidencePolicy(),
	}
}

func intentConstraints(taskType string, contractFirst bool) []string {
	constraints := []string{
		"respect describe edit surfaces",
		"do not write readonly or forbidden paths",
	}
	if contractFirst {
		constraints = append(constraints, "update contract before implementation for "+taskType)
	}
	return constraints
}

func riskLevel(risks []string) string {
	if len(risks) == 0 {
		return "low"
	}
	if len(risks) <= 2 {
		return "medium"
	}
	return "high"
}

func executableEdits(paths []string) []map[string]any {
	edits := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		edits = append(edits, map[string]any{
			"path":     path,
			"action":   "modify",
			"surface":  "allowed",
			"required": true,
			"reason":   "planned edit surface",
		})
	}
	return edits
}

func executableCommands(commands []string) []map[string]any {
	items := make([]map[string]any, 0, len(commands))
	for index, command := range commands {
		items = append(items, map[string]any{
			"id":              fmt.Sprintf("cmd-%d", index+1),
			"command":         command,
			"phase":           commandPhase(command),
			"working_dir":     ".",
			"timeout_seconds": 120,
			"allowed":         true,
			"produces":        commandProduces(command),
			"schema_ref":      commandSchemaRef(command),
		})
	}
	return items
}

func executableBlockedEdits(paths []string) []map[string]any {
	edits := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		edits = append(edits, map[string]any{
			"path":     path,
			"action":   "modify",
			"surface":  "manual",
			"required": true,
			"reason":   "not covered by describe edit_surfaces.allowed",
		})
	}
	return edits
}

func executableAssertions(commands []string, blockedEdits []string) []map[string]any {
	assertions := []map[string]any{
		{
			"id":       "assert-edit-surfaces",
			"type":     "edit_surface",
			"target":   "edits",
			"expected": "all edits stay within allowed surfaces",
		},
	}
	if len(blockedEdits) > 0 {
		assertions = append(assertions, map[string]any{
			"id":       "assert-blocked-edits",
			"type":     "manual_boundary",
			"target":   "blocked_edits",
			"expected": "blocked edits require explicit edit surface expansion before apply",
		})
	}
	for index, command := range commands {
		assertions = append(assertions, map[string]any{
			"id":       fmt.Sprintf("assert-cmd-%d", index+1),
			"type":     "command_exit",
			"target":   command,
			"expected": "exit_code == 0",
		})
	}
	return assertions
}

func executableRollback(paths []string) []map[string]any {
	rollback := make([]map[string]any, 0, len(paths))
	for index, path := range paths {
		rollback = append(rollback, map[string]any{
			"id":       fmt.Sprintf("rollback-%d", index+1),
			"target":   path,
			"strategy": "restore from pre-apply diff",
			"required": true,
		})
	}
	return rollback
}

func evidencePolicy() map[string]any {
	return map[string]any{
		"record_diff":      true,
		"record_exit_code": true,
		"redact":           []string{"token", "secret", "password", "dsn"},
		"schema_ref":       schemaRefEvidence,
		"accepted_kinds":   []string{evidenceKindApply, evidenceKindExecutor, evidenceKindVerify},
	}
}

func commandPhase(command string) string {
	switch {
	case strings.Contains(command, " capability add "):
		return commandPhaseScaffold
	case strings.Contains(command, " gen "):
		return commandPhaseGenerate
	case strings.Contains(command, " lint "):
		return commandPhaseLint
	case strings.Contains(command, " test "):
		return commandPhaseTest
	case strings.Contains(command, " validate "):
		return commandPhaseValidate
	case strings.Contains(command, " verify "):
		return commandPhaseVerify
	default:
		return commandPhaseVerify
	}
}

func commandProduces(command string) string {
	if strings.Contains(command, " verify ") {
		return evidenceKindVerify
	}
	if strings.Contains(command, " execute ") {
		return evidenceKindExecutor
	}
	return commandProducesExit
}

func commandSchemaRef(command string) string {
	if strings.Contains(command, " verify ") || strings.Contains(command, " execute ") {
		return schemaRefEvidence
	}
	return schemaRefPlanExecutable
}
