package verify

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	contractlint "github.com/nucleuskit/contract/lint"
)

type verifyResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Summary       verifySummary          `json:"summary"`
	Steps         []verifyStep           `json:"steps"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
	Findings      []contractlint.Finding `json:"findings"`
}

type verifySummary struct {
	Steps        int `json:"steps"`
	Passed       int `json:"passed"`
	Failed       int `json:"failed"`
	Errors       int `json:"errors"`
	Warnings     int `json:"warnings"`
	LintFindings int `json:"lint_findings"`
}

type verifyStep struct {
	ID                 string                       `json:"id"`
	Sequence           int                          `json:"sequence"`
	Phase              string                       `json:"phase"`
	Command            string                       `json:"command"`
	WorkingDir         string                       `json:"working_dir"`
	SchemaRef          string                       `json:"schema_ref"`
	Produces           string                       `json:"produces"`
	Status             string                       `json:"status"`
	OK                 bool                         `json:"ok"`
	ExitCode           int                          `json:"exit_code"`
	Output             string                       `json:"output,omitempty"`
	Error              string                       `json:"error,omitempty"`
	ChangedPaths       []string                     `json:"changed_paths,omitempty"`
	GeneratedFreshness []inspect.GeneratedFreshness `json:"generated_freshness,omitempty"`
}

func renderHuman(stdout io.Writer, stderr io.Writer, result verifyResult) {
	if !result.OK {
		_, _ = fmt.Fprintln(stderr, "FAILED")
	} else {
		_, _ = fmt.Fprintln(stdout, "OK")
	}
	for _, step := range result.Steps {
		status := "ok"
		if !step.OK {
			status = "failed"
		}
		_, _ = fmt.Fprintf(stdout, "%s: %s\n", step.Phase, status)
	}
	_, _ = fmt.Fprintf(stdout, "diagnostics: %d errors, %d warnings\n", result.Summary.Errors, result.Summary.Warnings)
	_, _ = fmt.Fprintf(stdout, "lint_findings: %d\n", result.Summary.LintFindings)
}

func renderJSON(writer io.Writer, result verifyResult, pretty bool) error {
	result.ResultKind = resultKindVerify
	if result.SchemaVersion == "" {
		result.SchemaVersion = schemaVersion
	}
	if result.SchemaRef == "" {
		result.SchemaRef = schemaRef
	}
	if result.Diagnostics == nil {
		result.Diagnostics = diagnostic.Diagnostics{}
	}
	if result.Findings == nil {
		result.Findings = []contractlint.Finding{}
	}
	if result.Steps == nil {
		result.Steps = []verifyStep{}
	}
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(result)
}
