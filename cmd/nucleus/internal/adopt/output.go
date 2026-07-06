package adopt

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nucleuskit/contract/diagnostic"
)

type result struct {
	ResultKind              string                 `json:"result_kind"`
	SchemaVersion           string                 `json:"schema_version"`
	SchemaRef               string                 `json:"schema_ref"`
	OK                      bool                   `json:"ok"`
	Summary                 summary                `json:"summary"`
	DetectedModule          string                 `json:"detected_module"`
	PackageSummary          []string               `json:"package_summary"`
	DetectedContracts       []pathFact             `json:"detected_contracts"`
	DetectedTestCommands    []string               `json:"detected_test_commands"`
	CreatedFiles            []pathFact             `json:"created_files"`
	GeneratedFileCandidates []pathFact             `json:"generated_file_candidates"`
	SymbolIndexSummary      map[string]int         `json:"symbol_index_summary"`
	Diagnostics             diagnostic.Diagnostics `json:"diagnostics"`
}

type summary struct {
	CreatedFiles      int `json:"created_files"`
	DetectedContracts int `json:"detected_contracts"`
	DetectedCommands  int `json:"detected_commands"`
	Packages          int `json:"packages"`
	Symbols           int `json:"symbols"`
}

type pathFact struct {
	Path   string `json:"path"`
	Kind   string `json:"kind,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func renderJSON(writer io.Writer, output result, pretty bool) error {
	output = normalizeResult(output)
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(output)
}

func renderHuman(writer io.Writer, output result) {
	output = normalizeResult(output)
	if !output.OK {
		_, _ = fmt.Fprintln(writer, "FAILED")
	} else {
		_, _ = fmt.Fprintln(writer, "OK adopted")
	}
	_, _ = fmt.Fprintf(writer, "module: %s\n", output.DetectedModule)
	_, _ = fmt.Fprintf(writer, "created files: %d\n", len(output.CreatedFiles))
	for _, file := range output.CreatedFiles {
		_, _ = fmt.Fprintf(writer, "  - %s\n", file.Path)
	}
	if len(output.Diagnostics) > 0 {
		_, _ = fmt.Fprintln(writer, "diagnostics:")
		for _, item := range output.Diagnostics {
			_, _ = fmt.Fprintf(writer, "  - %s %s: %s\n", item.Severity, item.Code, item.Message)
		}
	}
}

func normalizeResult(output result) result {
	if output.PackageSummary == nil {
		output.PackageSummary = []string{}
	}
	if output.DetectedContracts == nil {
		output.DetectedContracts = []pathFact{}
	}
	if output.DetectedTestCommands == nil {
		output.DetectedTestCommands = []string{}
	}
	if output.CreatedFiles == nil {
		output.CreatedFiles = []pathFact{}
	}
	if output.GeneratedFileCandidates == nil {
		output.GeneratedFileCandidates = []pathFact{}
	}
	if output.SymbolIndexSummary == nil {
		output.SymbolIndexSummary = map[string]int{}
	}
	if output.Diagnostics == nil {
		output.Diagnostics = diagnostic.Diagnostics{}
	}
	output.Summary = summary{
		CreatedFiles:      len(output.CreatedFiles),
		DetectedContracts: len(output.DetectedContracts),
		DetectedCommands:  len(output.DetectedTestCommands),
		Packages:          len(output.PackageSummary),
		Symbols:           output.SymbolIndexSummary["symbols"],
	}
	return output
}
