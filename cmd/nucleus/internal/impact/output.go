package impact

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
)

type result struct {
	ResultKind      string                 `json:"result_kind"`
	SchemaVersion   string                 `json:"schema_version"`
	SchemaRef       string                 `json:"schema_ref"`
	OK              bool                   `json:"ok"`
	Query           query                  `json:"query"`
	Summary         summary                `json:"summary"`
	Target          *inspect.SymbolNode    `json:"target,omitempty"`
	AffectedSymbols []inspect.SymbolNode   `json:"affected_symbols"`
	AffectedFiles   []string               `json:"affected_files"`
	AffectedTests   []inspect.SymbolNode   `json:"affected_tests"`
	AffectedRoutes  []inspect.FlowNode     `json:"affected_routes"`
	Edges           []inspect.SymbolEdge   `json:"edges"`
	FlowEdges       []inspect.FlowEdge     `json:"flow_edges"`
	Candidates      []inspect.SymbolNode   `json:"candidates,omitempty"`
	Diagnostics     diagnostic.Diagnostics `json:"diagnostics"`
}

type query struct {
	Kind       string `json:"kind"`
	Value      string `json:"value"`
	ResolvedID string `json:"resolved_id,omitempty"`
}

type summary struct {
	Symbols int `json:"symbols"`
	Files   int `json:"files"`
	Tests   int `json:"tests"`
	Routes  int `json:"routes"`
	Edges   int `json:"edges"`
}

func renderJSON(writer io.Writer, output result, pretty bool) error {
	output = normalizeResult(output)
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(output)
}

func renderHuman(writer io.Writer, output result) {
	output = normalizeResult(output)
	if output.OK {
		_, _ = fmt.Fprintln(writer, "OK impact")
	} else {
		_, _ = fmt.Fprintln(writer, "FAILED impact")
	}
	_, _ = fmt.Fprintf(writer, "query: %s %s\n", output.Query.Kind, output.Query.Value)
	_, _ = fmt.Fprintf(writer, "symbols: %d\n", len(output.AffectedSymbols))
	_, _ = fmt.Fprintf(writer, "files: %d\n", len(output.AffectedFiles))
	_, _ = fmt.Fprintf(writer, "tests: %d\n", len(output.AffectedTests))
	_, _ = fmt.Fprintf(writer, "routes: %d\n", len(output.AffectedRoutes))
	for _, item := range output.Diagnostics {
		_, _ = fmt.Fprintf(writer, "  - %s %s: %s\n", item.Severity, item.Code, item.Message)
	}
}

func normalizeResult(output result) result {
	if output.AffectedSymbols == nil {
		output.AffectedSymbols = []inspect.SymbolNode{}
	}
	if output.AffectedFiles == nil {
		output.AffectedFiles = []string{}
	}
	if output.AffectedTests == nil {
		output.AffectedTests = []inspect.SymbolNode{}
	}
	if output.AffectedRoutes == nil {
		output.AffectedRoutes = []inspect.FlowNode{}
	}
	if output.Edges == nil {
		output.Edges = []inspect.SymbolEdge{}
	}
	if output.FlowEdges == nil {
		output.FlowEdges = []inspect.FlowEdge{}
	}
	if output.Candidates == nil {
		output.Candidates = []inspect.SymbolNode{}
	}
	if output.Diagnostics == nil {
		output.Diagnostics = diagnostic.Diagnostics{}
	}
	output.ResultKind = resultKindImpact
	output.SchemaVersion = schemaVersionImpact
	output.SchemaRef = schemaRefImpact
	output.OK = !output.Diagnostics.Failed()
	output.Summary = summary{
		Symbols: len(output.AffectedSymbols),
		Files:   len(output.AffectedFiles),
		Tests:   len(output.AffectedTests),
		Routes:  len(output.AffectedRoutes),
		Edges:   len(output.Edges) + len(output.FlowEdges),
	}
	return output
}
