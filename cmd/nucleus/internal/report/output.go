package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nucleuskit/contract/diagnostic"
)

type reportResult struct {
	ResultKind        string                   `json:"result_kind"`
	SchemaVersion     string                   `json:"schema_version"`
	SchemaRef         string                   `json:"schema_ref"`
	OK                bool                     `json:"ok"`
	Mode              string                   `json:"mode"`
	Summary           reportSummary            `json:"summary"`
	Diagnostics       diagnostic.Diagnostics   `json:"diagnostics"`
	AIQuality         *aiQualityReport         `json:"ai_quality,omitempty"`
	PlatformReadiness *platformReadinessReport `json:"platform_readiness,omitempty"`
}

type reportSummary struct {
	Errors                  int     `json:"errors"`
	Warnings                int     `json:"warnings"`
	TaskCount               int     `json:"task_count"`
	ScenarioTaskCount       int     `json:"scenario_task_count"`
	RealEvidenceTaskCount   int     `json:"real_evidence_task_count"`
	SingleServiceTaskCount  int     `json:"single_service_task_count"`
	FailedTaskCount         int     `json:"failed_task_count"`
	RepairableTaskCount     int     `json:"repairable_task_count"`
	FirstPassRate           float64 `json:"first_pass_rate"`
	FailureLocatedRate      float64 `json:"failure_located_rate"`
	RepairSuccessRate       float64 `json:"repair_success_rate"`
	ManualInterventionRate  float64 `json:"manual_intervention_rate"`
	RollbackRate            float64 `json:"rollback_rate"`
	FirstPassCount          int     `json:"first_pass_count"`
	FailureLocatedCount     int     `json:"failure_located_count"`
	RepairSuccessCount      int     `json:"repair_success_count"`
	ManualInterventionCount int     `json:"manual_intervention_count"`
	RollbackCount           int     `json:"rollback_count"`
	CapabilityEventCount    int     `json:"capability_event_count"`
	CapabilityErrorCount    int     `json:"capability_error_count"`
	EndpointCount           int     `json:"endpoint_count"`
	GRPCServiceCount        int     `json:"grpc_service_count"`
	CapabilityCount         int     `json:"capability_count"`
	GeneratedFresh          bool    `json:"generated_fresh"`
	ReadinessGateCount      int     `json:"readiness_gate_count"`
	ReadinessGatePassed     int     `json:"readiness_gate_passed"`
	ReadinessGateFailed     int     `json:"readiness_gate_failed"`
	RiskGateCount           int     `json:"risk_gate_count"`
	RiskGatePassed          int     `json:"risk_gate_passed"`
	RiskGateFailed          int     `json:"risk_gate_failed"`
}

func renderHuman(stdout io.Writer, stderr io.Writer, result reportResult) {
	for _, item := range result.Diagnostics {
		_, _ = fmt.Fprintf(stderr, "%s %s %s: %s\n", item.Severity, item.Path, item.Code, item.Message)
	}
	if result.OK {
		_, _ = fmt.Fprintln(stdout, "OK")
	} else {
		_, _ = fmt.Fprintln(stderr, "FAILED")
	}
	_, _ = fmt.Fprintf(stdout, "mode: %s\n", result.Mode)
	switch result.Mode {
	case reportModePlatformReadiness:
		renderPlatformHuman(stdout, result)
	default:
		renderAIQualityHuman(stdout, result)
	}
	_, _ = fmt.Fprintf(stdout, "diagnostics: %d errors, %d warnings\n", result.Summary.Errors, result.Summary.Warnings)
}

func renderAIQualityHuman(stdout io.Writer, result reportResult) {
	_, _ = fmt.Fprintf(stdout, "tasks: %d\n", result.Summary.TaskCount)
	_, _ = fmt.Fprintf(stdout, "sources: scenario=%d real_evidence=%d\n", result.Summary.ScenarioTaskCount, result.Summary.RealEvidenceTaskCount)
	_, _ = fmt.Fprintf(stdout, "first_pass_rate: %.2f\n", result.Summary.FirstPassRate)
	_, _ = fmt.Fprintf(stdout, "repair_success_rate: %.2f\n", result.Summary.RepairSuccessRate)
	_, _ = fmt.Fprintf(stdout, "manual_intervention_rate: %.2f\n", result.Summary.ManualInterventionRate)
	if result.AIQuality != nil && result.AIQuality.TasksDir != "" {
		_, _ = fmt.Fprintf(stdout, "ai_tasks: %s\n", result.AIQuality.TasksDir)
	}
}

func renderPlatformHuman(stdout io.Writer, result reportResult) {
	if result.PlatformReadiness == nil {
		return
	}
	platform := result.PlatformReadiness
	_, _ = fmt.Fprintf(stdout, "service: %s\n", platform.Service)
	_, _ = fmt.Fprintf(stdout, "version: %s\n", platform.Version)
	_, _ = fmt.Fprintf(stdout, "endpoints: %d\n", result.Summary.EndpointCount)
	_, _ = fmt.Fprintf(stdout, "capabilities: %d\n", result.Summary.CapabilityCount)
	_, _ = fmt.Fprintf(stdout, "generated_fresh: %t\n", result.Summary.GeneratedFresh)
	_, _ = fmt.Fprintf(stdout, "readiness_gates: %d passed, %d failed\n", result.Summary.ReadinessGatePassed, result.Summary.ReadinessGateFailed)
}

func renderJSON(writer io.Writer, result reportResult, pretty bool) error {
	result.ResultKind = resultKindReport
	result.SchemaVersion = schemaVersionReport
	result.SchemaRef = schemaRefReport
	if result.Diagnostics == nil {
		result.Diagnostics = diagnostic.Diagnostics{}
	}
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(result)
}

func finalizeReportResult(result reportResult) reportResult {
	result.ResultKind = resultKindReport
	result.SchemaVersion = schemaVersionReport
	result.SchemaRef = schemaRefReport
	if result.Diagnostics == nil {
		result.Diagnostics = diagnostic.Diagnostics{}
	}
	result.Diagnostics.Sort()
	result.OK = !result.Diagnostics.Failed()
	result.Summary.Errors = result.Diagnostics.Count(diagnostic.SeverityError)
	result.Summary.Warnings = result.Diagnostics.Count(diagnostic.SeverityWarning)
	return result
}

func reportErrorDiagnostic(path string, code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}

func reportWarningDiagnostic(path string, code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityWarning,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}
