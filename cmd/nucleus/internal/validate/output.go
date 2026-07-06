package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
)

type validateResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Summary       validateSummary        `json:"summary"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
}

type validateSummary struct {
	Errors          int      `json:"errors"`
	Warnings        int      `json:"warnings"`
	Checked         []string `json:"checked"`
	MissingOptional []string `json:"missing_optional"`
}

func renderHuman(stdout io.Writer, stderr io.Writer, diagnostics diagnostic.Diagnostics, summary validateSummary) {
	for _, item := range diagnostics {
		_, _ = fmt.Fprintf(stderr, "%s %s %s: %s\n", item.Severity, item.Path, item.Code, item.Message)
	}
	if !diagnostics.Failed() {
		_, _ = fmt.Fprintln(stdout, "OK")
		if len(summary.Checked) > 0 {
			_, _ = fmt.Fprintf(stdout, "validated: %s\n", strings.Join(summary.Checked, ", "))
		}
		if len(summary.MissingOptional) > 0 {
			_, _ = fmt.Fprintf(stdout, "missing optional: %s\n", strings.Join(summary.MissingOptional, ", "))
		}
		_, _ = fmt.Fprintf(stdout, "diagnostics: %d errors, %d warnings\n", summary.Errors, summary.Warnings)
	}
}

func renderJSON(writer io.Writer, diagnostics diagnostic.Diagnostics, summary validateSummary, pretty bool) error {
	if diagnostics == nil {
		diagnostics = diagnostic.Diagnostics{}
	}
	result := validateResult{
		ResultKind:    resultKindValidate,
		SchemaVersion: schemaVersionValidate,
		SchemaRef:     schemaRefValidate,
		OK:            !diagnostics.Failed(),
		Summary:       summary,
		Diagnostics:   diagnostics,
	}
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(result)
}
