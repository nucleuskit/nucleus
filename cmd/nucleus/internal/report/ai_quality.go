package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/decision"
)

type aiQualityReport struct {
	InputStatus             string                       `json:"input_status"`
	TasksDir                string                       `json:"tasks_dir"`
	TaskCount               int                          `json:"task_count"`
	ScenarioTaskCount       int                          `json:"scenario_task_count"`
	RealEvidenceTaskCount   int                          `json:"real_evidence_task_count"`
	SingleServiceTaskCount  int                          `json:"single_service_task_count"`
	FailedTaskCount         int                          `json:"failed_task_count"`
	RepairableTaskCount     int                          `json:"repairable_task_count"`
	SourceCoverageRate      float64                      `json:"source_coverage_rate"`
	FirstPassRate           float64                      `json:"first_pass_rate"`
	FailureLocatedRate      float64                      `json:"failure_located_rate"`
	RepairSuccessRate       float64                      `json:"repair_success_rate"`
	ManualInterventionRate  float64                      `json:"manual_intervention_rate"`
	RollbackRate            float64                      `json:"rollback_rate"`
	FirstPassCount          int                          `json:"first_pass_count"`
	FailureLocatedCount     int                          `json:"failure_located_count"`
	RepairSuccessCount      int                          `json:"repair_success_count"`
	ManualInterventionCount int                          `json:"manual_intervention_count"`
	RollbackCount           int                          `json:"rollback_count"`
	CapabilityEventCount    int                          `json:"capability_event_count"`
	CapabilityErrorCount    int                          `json:"capability_error_count"`
	CapabilitySummary       map[string]capabilitySummary `json:"capability_summary"`
	StrategySummary         strategyReport               `json:"strategy_summary"`
	DecisionQuality         decision.QualitySummary      `json:"decision_quality"`
	RecipeCandidateUsage    recipeCandidateUsageReport   `json:"recipe_candidate_usage"`
	WeeklyReport            string                       `json:"weekly_report"`
}

type aiTaskResult struct {
	ID                 string            `json:"id"`
	Kind               string            `json:"kind"`
	Source             string            `json:"source"`
	TaskType           string            `json:"task_type"`
	Labels             []string          `json:"labels"`
	PlanPass           bool              `json:"plan_pass"`
	ApplyPass          bool              `json:"apply_pass"`
	VerifyPass         bool              `json:"verify_pass"`
	RepairPass         bool              `json:"repair_pass"`
	FailureLocated     bool              `json:"failure_located"`
	FailureType        string            `json:"failure_type"`
	ManualAction       bool              `json:"manual_action"`
	ManualActionReason string            `json:"manual_action_reason"`
	RepairStrategy     string            `json:"repair_strategy"`
	RollbackPerformed  bool              `json:"rollback_performed"`
	Evidence           map[string]any    `json:"evidence"`
	CapabilityEvents   []capabilityEvent `json:"capability_events"`
}

type capabilityEvent struct {
	Capability string  `json:"capability"`
	Provider   string  `json:"provider,omitempty"`
	Operation  string  `json:"operation"`
	Status     string  `json:"status"`
	DurationMS float64 `json:"duration_ms,omitempty"`
	Resource   string  `json:"resource,omitempty"`
	Error      string  `json:"error,omitempty"`
}

type strategySummary struct {
	TaskCount            int            `json:"task_count"`
	SuccessCount         int            `json:"success_count"`
	SuccessRate          float64        `json:"success_rate"`
	FailureTypes         map[string]int `json:"failure_types,omitempty"`
	ManualActionReasons  map[string]int `json:"manual_action_reasons,omitempty"`
	RepairStrategyCounts map[string]int `json:"repair_strategy_counts,omitempty"`
}

type strategyReport struct {
	ByTaskType               map[string]strategySummary `json:"by_task_type"`
	ByLabel                  map[string]strategySummary `json:"by_label"`
	RepairStrategyCounts     map[string]int             `json:"repair_strategy_counts"`
	FailureTypeCounts        map[string]int             `json:"failure_type_counts"`
	ManualActionReasonCounts map[string]int             `json:"manual_action_reason_counts"`
}

type capabilitySummary struct {
	EventCount  int            `json:"event_count"`
	ErrorCount  int            `json:"error_count"`
	Operations  map[string]int `json:"operations,omitempty"`
	Providers   map[string]int `json:"providers,omitempty"`
	StatusCount map[string]int `json:"status_count,omitempty"`
}

type recipeCandidateUsageReport struct {
	TaskCount              int      `json:"task_count"`
	CandidateTaskCount     int      `json:"candidate_task_count"`
	CandidateCount         int      `json:"candidate_count"`
	DecisionRequiredCount  int      `json:"decision_required_count"`
	UniqueCandidateIDs     []string `json:"unique_candidate_ids"`
	RecipeEvidenceCoverage float64  `json:"recipe_evidence_coverage"`
}

type aiQualityCountSet struct {
	scenario         int
	realEvidence     int
	singleService    int
	failed           int
	repairable       int
	firstPass        int
	failureLocated   int
	repairSuccess    int
	manualAction     int
	rollback         int
	capabilityEvents int
	capabilityErrors int
}

func buildAIQualityResult(dir string, tasksDir string, explicitTasksDir bool) reportResult {
	report, diagnostics := loadAIQualityReport(dir, tasksDir, explicitTasksDir)
	diagnostics = append(diagnostics, report.DecisionQuality.Diagnostics...)
	summary := reportSummaryFromAIQuality(report)
	return finalizeReportResult(reportResult{
		Mode:        reportModeAIQuality,
		Summary:     summary,
		Diagnostics: diagnostics,
		AIQuality:   &report,
	})
}

func loadAIQualityReport(dir string, tasksDir string, explicitTasksDir bool) (aiQualityReport, diagnostic.Diagnostics) {
	report := emptyAIQualityReport(dir, tasksDir, inputStatusLoaded)
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if !explicitTasksDir && errors.Is(err, os.ErrNotExist) {
			report.InputStatus = inputStatusMissing
			return report, diagnostic.Diagnostics{
				reportWarningDiagnostic(safeReportPath(tasksDir), reportDiagnosticAITasksMissing, "default AI task result directory does not exist"),
			}
		}
		report.InputStatus = inputStatusFailed
		return report, diagnostic.Diagnostics{
			reportErrorDiagnostic(safeReportPath(tasksDir), reportDiagnosticAITasksReadFailed, safeErrorMessage(err)),
		}
	}
	tasks := make([]aiTaskResult, 0, len(entries))
	var diagnostics diagnostic.Diagnostics
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(tasksDir, entry.Name())
		task, err := readAITaskResult(path)
		if err != nil {
			diagnostics = append(diagnostics, reportErrorDiagnostic(safeReportPath(path), reportDiagnosticAITaskParseFailed, safeErrorMessage(err)))
			continue
		}
		tasks = append(tasks, task)
	}
	return aiQualityReportFromTasks(dir, tasksDir, inputStatusLoaded, tasks), diagnostics
}

func emptyAIQualityReport(dir string, tasksDir string, status string) aiQualityReport {
	return aiQualityReportFromTasks(dir, tasksDir, status, nil)
}

func aiQualityReportFromTasks(dir string, tasksDir string, status string, tasks []aiTaskResult) aiQualityReport {
	counts := aiQualityCounts(tasks)
	decisionQuality := decision.QualityForDir(dir)
	recipeCandidateUsage := buildRecipeCandidateUsage(tasks)
	report := aiQualityReport{
		InputStatus:             status,
		TasksDir:                safeReportPath(tasksDir),
		TaskCount:               len(tasks),
		ScenarioTaskCount:       counts.scenario,
		RealEvidenceTaskCount:   counts.realEvidence,
		SingleServiceTaskCount:  counts.singleService,
		FailedTaskCount:         counts.failed,
		RepairableTaskCount:     counts.repairable,
		SourceCoverageRate:      rate(counts.sourcesCovered(), 2),
		FirstPassRate:           rate(counts.firstPass, counts.singleService),
		FailureLocatedRate:      rate(counts.failureLocated, counts.failed),
		RepairSuccessRate:       rate(counts.repairSuccess, counts.repairable),
		ManualInterventionRate:  rate(counts.manualAction, len(tasks)),
		RollbackRate:            rate(counts.rollback, len(tasks)),
		FirstPassCount:          counts.firstPass,
		FailureLocatedCount:     counts.failureLocated,
		RepairSuccessCount:      counts.repairSuccess,
		ManualInterventionCount: counts.manualAction,
		RollbackCount:           counts.rollback,
		CapabilityEventCount:    counts.capabilityEvents,
		CapabilityErrorCount:    counts.capabilityErrors,
		CapabilitySummary:       buildCapabilitySummary(tasks),
		StrategySummary:         buildStrategySummary(tasks),
		DecisionQuality:         decisionQuality,
		RecipeCandidateUsage:    recipeCandidateUsage,
	}
	report.WeeklyReport = aiQualityMarkdown(report)
	return report
}

func reportSummaryFromAIQuality(report aiQualityReport) reportSummary {
	return reportSummary{
		TaskCount:               report.TaskCount,
		ScenarioTaskCount:       report.ScenarioTaskCount,
		RealEvidenceTaskCount:   report.RealEvidenceTaskCount,
		SingleServiceTaskCount:  report.SingleServiceTaskCount,
		FailedTaskCount:         report.FailedTaskCount,
		RepairableTaskCount:     report.RepairableTaskCount,
		FirstPassRate:           report.FirstPassRate,
		FailureLocatedRate:      report.FailureLocatedRate,
		RepairSuccessRate:       report.RepairSuccessRate,
		ManualInterventionRate:  report.ManualInterventionRate,
		RollbackRate:            report.RollbackRate,
		FirstPassCount:          report.FirstPassCount,
		FailureLocatedCount:     report.FailureLocatedCount,
		RepairSuccessCount:      report.RepairSuccessCount,
		ManualInterventionCount: report.ManualInterventionCount,
		RollbackCount:           report.RollbackCount,
		CapabilityEventCount:    report.CapabilityEventCount,
		CapabilityErrorCount:    report.CapabilityErrorCount,
		DecisionFileCount:       report.DecisionQuality.Files,
		DecisionValidCount:      report.DecisionQuality.Valid,
		DecisionErrorCount:      report.DecisionQuality.Errors,
		LockedDecisionCount:     report.DecisionQuality.AcceptedLocked,
		DecisionDriftCount:      report.DecisionQuality.Drift,
		RecipeCandidateTasks:    report.RecipeCandidateUsage.CandidateTaskCount,
		RecipeCandidateCount:    report.RecipeCandidateUsage.CandidateCount,
	}
}

func readAITaskResult(path string) (aiTaskResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return aiTaskResult{}, err
	}
	var task aiTaskResult
	if err := json.Unmarshal(data, &task); err != nil {
		return aiTaskResult{}, err
	}
	if strings.TrimSpace(task.ID) == "" {
		task.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	normalizeAITaskResult(&task)
	return task, nil
}

func normalizeAITaskResult(task *aiTaskResult) {
	if strings.TrimSpace(task.Source) == "" {
		task.Source = "scenario"
	}
	if strings.TrimSpace(task.TaskType) == "" {
		task.TaskType = "unspecified"
	}
	if task.Kind != "nucleus.evidence_replay" {
		return
	}
	task.Source = "real_evidence"
	if len(task.CapabilityEvents) == 0 {
		task.CapabilityEvents = replayCapabilityEvents(task.Evidence)
	}
	switch evidenceKind(task.Evidence) {
	case "nucleus.repair_evidence":
		normalizeRepairEvidenceTask(task)
	case "nucleus.verify_result":
		normalizeVerifyEvidenceTask(task)
	default:
		normalizeGenericEvidenceTask(task)
	}
}

func normalizeRepairEvidenceTask(task *aiTaskResult) {
	status := evidenceString(task.Evidence, "status", "")
	verificationPass := evidenceBool(task.Evidence, "verification_pass", evidenceBool(task.Evidence, "ok", false))
	task.FailureLocated = replayFailureLocated(task.Evidence)
	task.FailureType = evidenceString(task.Evidence, "failure_type", status)
	task.ManualActionReason = evidenceString(task.Evidence, "manual_action_reason", "")
	task.RepairStrategy = replayRepairStrategy(task.Evidence)
	task.PlanPass = true
	task.ApplyPass = false
	task.VerifyPass = verificationPass
	task.RepairPass = status == "repaired" && verificationPass
	if task.RepairPass && !hasString(task.Labels, "repairable") {
		task.Labels = append(task.Labels, "repairable")
	}
	task.ManualAction = status == "needs_manual_action"
}

func normalizeVerifyEvidenceTask(task *aiTaskResult) {
	ok := evidenceBool(task.Evidence, "ok", evidenceBool(task.Evidence, "verification_pass", false))
	task.PlanPass = true
	task.ApplyPass = true
	task.VerifyPass = ok
	task.FailureLocated = replayFailureLocated(task.Evidence) || verifyEvidenceHasFailedStep(task.Evidence)
	task.FailureType = evidenceString(task.Evidence, "failure_type", verifyFailureType(task.Evidence))
	task.ManualActionReason = evidenceString(task.Evidence, "manual_action_reason", "")
	task.ManualAction = !ok && task.ManualActionReason != ""
}

func normalizeGenericEvidenceTask(task *aiTaskResult) {
	pass := evidenceBool(task.Evidence, "ok", evidenceBool(task.Evidence, "verification_pass", false))
	status := evidenceString(task.Evidence, "status", "")
	task.PlanPass = true
	task.ApplyPass = pass
	task.VerifyPass = pass
	task.FailureLocated = replayFailureLocated(task.Evidence)
	task.FailureType = evidenceString(task.Evidence, "failure_type", status)
	task.ManualActionReason = evidenceString(task.Evidence, "manual_action_reason", "")
	task.ManualAction = status == "needs_manual_action"
}

func evidenceKind(evidence map[string]any) string {
	if evidence == nil {
		return ""
	}
	return evidenceString(evidence, "result_kind", "")
}

func replayCapabilityEvents(evidence map[string]any) []capabilityEvent {
	rawEvents, ok := evidence["capability_events"].([]any)
	if !ok {
		return nil
	}
	events := make([]capabilityEvent, 0, len(rawEvents))
	for _, raw := range rawEvents {
		eventMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		event := capabilityEvent{
			Capability: evidenceString(eventMap, "capability", ""),
			Provider:   evidenceString(eventMap, "provider", ""),
			Operation:  evidenceString(eventMap, "operation", ""),
			Status:     evidenceString(eventMap, "status", ""),
			Resource:   evidenceString(eventMap, "resource", ""),
			Error:      evidenceString(eventMap, "error", ""),
		}
		if duration, ok := eventMap["duration_ms"].(float64); ok {
			event.DurationMS = duration
		}
		if strings.TrimSpace(event.Capability) != "" && strings.TrimSpace(event.Operation) != "" {
			events = append(events, event)
		}
	}
	return events
}

func evidenceString(evidence map[string]any, key string, fallback string) string {
	value, _ := evidence[key].(string)
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func evidenceBool(evidence map[string]any, key string, fallback bool) bool {
	value, ok := evidence[key].(bool)
	if ok {
		return value
	}
	return fallback
}

func replayFailureLocated(evidence map[string]any) bool {
	if located, ok := evidence["failure_located"].(bool); ok {
		return located
	}
	rounds, ok := evidence["rounds"].([]any)
	if !ok {
		return false
	}
	for _, roundValue := range rounds {
		round, ok := roundValue.(map[string]any)
		if !ok {
			continue
		}
		strategy, _ := round["strategy"].(string)
		if strings.TrimSpace(strategy) != "" {
			return true
		}
	}
	return false
}

func replayRepairStrategy(evidence map[string]any) string {
	rounds, ok := evidence["rounds"].([]any)
	if !ok {
		return ""
	}
	for _, roundValue := range rounds {
		round, ok := roundValue.(map[string]any)
		if !ok {
			continue
		}
		strategy, _ := round["strategy"].(string)
		if strings.TrimSpace(strategy) != "" {
			return strategy
		}
	}
	return ""
}

func verifyEvidenceHasFailedStep(evidence map[string]any) bool {
	for _, step := range evidenceSteps(evidence) {
		if ok, exists := step["ok"].(bool); exists && !ok {
			return true
		}
		status, _ := step["status"].(string)
		if status == "failed" {
			return true
		}
	}
	return false
}

func verifyFailureType(evidence map[string]any) string {
	for _, step := range evidenceSteps(evidence) {
		ok, hasOK := step["ok"].(bool)
		status, _ := step["status"].(string)
		if (hasOK && ok) || status == "passed" {
			continue
		}
		if phase := evidenceString(step, "phase", ""); phase != "" {
			return phase
		}
		if id := evidenceString(step, "id", ""); id != "" {
			return id
		}
	}
	return ""
}

func evidenceSteps(evidence map[string]any) []map[string]any {
	rawSteps, ok := evidence["steps"].([]any)
	if !ok {
		return nil
	}
	steps := make([]map[string]any, 0, len(rawSteps))
	for _, raw := range rawSteps {
		step, ok := raw.(map[string]any)
		if ok {
			steps = append(steps, step)
		}
	}
	return steps
}

func aiQualityCounts(tasks []aiTaskResult) aiQualityCountSet {
	var counts aiQualityCountSet
	for _, task := range tasks {
		switch task.Source {
		case "real_evidence":
			counts.realEvidence++
		default:
			counts.scenario++
		}
		if hasString(task.Labels, "single_service") {
			counts.singleService++
			if task.PlanPass && task.ApplyPass && task.VerifyPass {
				counts.firstPass++
			}
		}
		failed := !task.PlanPass || !task.ApplyPass || !task.VerifyPass
		if failed {
			counts.failed++
			if task.FailureLocated {
				counts.failureLocated++
			}
		}
		if hasString(task.Labels, "repairable") {
			counts.repairable++
			if task.RepairPass {
				counts.repairSuccess++
			}
		}
		if task.ManualAction {
			counts.manualAction++
		}
		if task.RollbackPerformed {
			counts.rollback++
		}
		for _, event := range task.CapabilityEvents {
			counts.capabilityEvents++
			if capabilityEventFailed(event) {
				counts.capabilityErrors++
			}
		}
	}
	return counts
}

func (counts aiQualityCountSet) sourcesCovered() int {
	covered := 0
	if counts.scenario > 0 {
		covered++
	}
	if counts.realEvidence > 0 {
		covered++
	}
	return covered
}

func rate(numerator int, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func buildStrategySummary(tasks []aiTaskResult) strategyReport {
	byTaskType := map[string]strategySummary{}
	byLabel := map[string]strategySummary{}
	repairStrategies := map[string]int{}
	failureTypes := map[string]int{}
	manualReasons := map[string]int{}
	for _, task := range tasks {
		taskType := chooseNonEmpty(task.TaskType, "unspecified")
		addStrategySummary(byTaskType, taskType, task)
		for _, label := range task.Labels {
			addStrategySummary(byLabel, label, task)
		}
		if strings.TrimSpace(task.RepairStrategy) != "" {
			repairStrategies[task.RepairStrategy]++
		}
		if strings.TrimSpace(task.FailureType) != "" && !taskSucceeded(task) {
			failureTypes[task.FailureType]++
		}
		if strings.TrimSpace(task.ManualActionReason) != "" && task.ManualAction {
			manualReasons[task.ManualActionReason]++
		}
	}
	finalizeStrategySummaries(byTaskType)
	finalizeStrategySummaries(byLabel)
	return strategyReport{
		ByTaskType:               byTaskType,
		ByLabel:                  byLabel,
		RepairStrategyCounts:     repairStrategies,
		FailureTypeCounts:        failureTypes,
		ManualActionReasonCounts: manualReasons,
	}
}

func buildCapabilitySummary(tasks []aiTaskResult) map[string]capabilitySummary {
	summaries := map[string]capabilitySummary{}
	for _, task := range tasks {
		for _, event := range task.CapabilityEvents {
			if strings.TrimSpace(event.Capability) == "" {
				continue
			}
			summary := summaries[event.Capability]
			if summary.Operations == nil {
				summary.Operations = map[string]int{}
			}
			if summary.Providers == nil {
				summary.Providers = map[string]int{}
			}
			if summary.StatusCount == nil {
				summary.StatusCount = map[string]int{}
			}
			summary.EventCount++
			if capabilityEventFailed(event) {
				summary.ErrorCount++
			}
			if strings.TrimSpace(event.Operation) != "" {
				summary.Operations[event.Operation]++
			}
			if strings.TrimSpace(event.Provider) != "" {
				summary.Providers[event.Provider]++
			}
			if strings.TrimSpace(event.Status) != "" {
				summary.StatusCount[event.Status]++
			}
			summaries[event.Capability] = summary
		}
	}
	return summaries
}

func buildRecipeCandidateUsage(tasks []aiTaskResult) recipeCandidateUsageReport {
	usage := recipeCandidateUsageReport{TaskCount: len(tasks)}
	ids := map[string]bool{}
	for _, task := range tasks {
		before := usage.CandidateCount
		collectRecipeCandidates(task.Evidence, &usage, ids)
		if usage.CandidateCount > before {
			usage.CandidateTaskCount++
		}
	}
	usage.UniqueCandidateIDs = sortedStringKeys(ids)
	usage.RecipeEvidenceCoverage = rate(usage.CandidateTaskCount, usage.TaskCount)
	return usage
}

func collectRecipeCandidates(value any, usage *recipeCandidateUsageReport, ids map[string]bool) {
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			if key == "recipe_candidates" {
				countRecipeCandidateArray(child, usage, ids)
				continue
			}
			if key == "candidates" && evidenceKind(item) == "nucleus.recipe_candidate_result" {
				countRecipeCandidateArray(child, usage, ids)
				continue
			}
			collectRecipeCandidates(child, usage, ids)
		}
	case []any:
		for _, child := range item {
			collectRecipeCandidates(child, usage, ids)
		}
	}
}

func countRecipeCandidateArray(value any, usage *recipeCandidateUsageReport, ids map[string]bool) {
	items, ok := value.([]any)
	if !ok {
		return
	}
	for _, raw := range items {
		candidate, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		usage.CandidateCount++
		if id := evidenceString(candidate, "id", ""); id != "" {
			ids[id] = true
		}
		if required, ok := candidate["decision_required"].(bool); ok && required {
			usage.DecisionRequiredCount++
		}
	}
}

func sortedStringKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func capabilityEventFailed(event capabilityEvent) bool {
	switch event.Status {
	case "error", "failed", "timeout", "rejected":
		return true
	default:
		return strings.TrimSpace(event.Error) != ""
	}
}

func addStrategySummary(summaries map[string]strategySummary, key string, task aiTaskResult) {
	if strings.TrimSpace(key) == "" {
		return
	}
	summary := summaries[key]
	if summary.FailureTypes == nil {
		summary.FailureTypes = map[string]int{}
	}
	if summary.ManualActionReasons == nil {
		summary.ManualActionReasons = map[string]int{}
	}
	if summary.RepairStrategyCounts == nil {
		summary.RepairStrategyCounts = map[string]int{}
	}
	summary.TaskCount++
	if taskSucceeded(task) {
		summary.SuccessCount++
	} else if strings.TrimSpace(task.FailureType) != "" {
		summary.FailureTypes[task.FailureType]++
	}
	if task.ManualAction && strings.TrimSpace(task.ManualActionReason) != "" {
		summary.ManualActionReasons[task.ManualActionReason]++
	}
	if strings.TrimSpace(task.RepairStrategy) != "" {
		summary.RepairStrategyCounts[task.RepairStrategy]++
	}
	summaries[key] = summary
}

func finalizeStrategySummaries(summaries map[string]strategySummary) {
	for key, summary := range summaries {
		summary.SuccessRate = rate(summary.SuccessCount, summary.TaskCount)
		summaries[key] = summary
	}
}

func taskSucceeded(task aiTaskResult) bool {
	return task.PlanPass && task.ApplyPass && task.VerifyPass
}

func chooseNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func aiQualityMarkdown(report aiQualityReport) string {
	return fmt.Sprintf(
		"AI quality report: tasks=%d scenario=%d real_evidence=%d source_coverage=%.2f first_pass=%.2f failure_located=%.2f repair_success=%.2f manual_intervention=%.2f rollback=%.2f decisions=%d locked=%d decision_drift=%d recipe_candidates=%d",
		report.TaskCount,
		report.ScenarioTaskCount,
		report.RealEvidenceTaskCount,
		report.SourceCoverageRate,
		report.FirstPassRate,
		report.FailureLocatedRate,
		report.RepairSuccessRate,
		report.ManualInterventionRate,
		report.RollbackRate,
		report.DecisionQuality.Files,
		report.DecisionQuality.AcceptedLocked,
		report.DecisionQuality.Drift,
		report.RecipeCandidateUsage.CandidateCount,
	)
}

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
