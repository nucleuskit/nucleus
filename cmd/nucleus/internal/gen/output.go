package gen

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
)

type genResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	SourceHash    string                 `json:"source_hash,omitempty"`
	Summary       genSummary             `json:"summary"`
	Files         []string               `json:"files"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
}

func renderHuman(stdout io.Writer, stderr io.Writer, result genResult) {
	for _, item := range result.Diagnostics {
		_, _ = fmt.Fprintf(stderr, "%s %s %s: %s\n", item.Severity, item.Path, item.Code, item.Message)
	}
	if !result.OK {
		_, _ = fmt.Fprintf(stderr, "FAILED\n")
		_, _ = fmt.Fprintf(stderr, "diagnostics: %d errors, %d warnings\n", result.Summary.Errors, result.Summary.Warnings)
		return
	}
	_, _ = fmt.Fprintln(stdout, "OK")
	_, _ = fmt.Fprintf(stdout, "generated: %d file(s)\n", result.Summary.Files)
	if result.SourceHash != "" {
		_, _ = fmt.Fprintf(stdout, "source_hash: %s\n", result.SourceHash)
	}
	if len(result.Summary.Targets) > 0 {
		_, _ = fmt.Fprintf(stdout, "targets: %s\n", strings.Join(result.Summary.Targets, ", "))
	}
	if len(result.Summary.ClientLanguages) > 0 {
		_, _ = fmt.Fprintf(stdout, "client_languages: %s\n", strings.Join(result.Summary.ClientLanguages, ", "))
	}
	for _, file := range result.Files {
		_, _ = fmt.Fprintln(stdout, file)
	}
	_, _ = fmt.Fprintf(stdout, "diagnostics: %d errors, %d warnings\n", result.Summary.Errors, result.Summary.Warnings)
}

func renderJSON(writer io.Writer, result genResult, pretty bool) error {
	result.ResultKind = resultKindGen
	result.SchemaVersion = schemaVersionGen
	result.SchemaRef = schemaRefGen
	if result.Files == nil {
		result.Files = []string{}
	}
	if result.Diagnostics == nil {
		result.Diagnostics = diagnostic.Diagnostics{}
	}
	if result.Summary.Targets == nil {
		result.Summary.Targets = []string{}
	}
	if result.Summary.ClientLanguages == nil {
		result.Summary.ClientLanguages = []string{}
	}
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(result)
}
