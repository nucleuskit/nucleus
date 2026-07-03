package trace

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
)

type result struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Query         query                  `json:"query"`
	Target        *inspect.SymbolNode    `json:"target,omitempty"`
	Nodes         []inspect.SymbolNode   `json:"nodes"`
	Edges         []inspect.SymbolEdge   `json:"edges"`
	Callers       []traceHop             `json:"callers,omitempty"`
	Callees       []traceHop             `json:"callees,omitempty"`
	FlowNodes     []inspect.FlowNode     `json:"flow_nodes,omitempty"`
	FlowEdges     []inspect.FlowEdge     `json:"flow_edges,omitempty"`
	Candidates    []inspect.SymbolNode   `json:"candidates,omitempty"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
}

type query struct {
	Kind       string `json:"kind"`
	Value      string `json:"value"`
	ResolvedID string `json:"resolved_id,omitempty"`
}

type traceHop struct {
	Node inspect.SymbolNode `json:"node"`
	Edge inspect.SymbolEdge `json:"edge"`
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
		_, _ = fmt.Fprintln(writer, "OK trace")
	} else {
		_, _ = fmt.Fprintln(writer, "FAILED trace")
	}
	_, _ = fmt.Fprintf(writer, "query: %s %s\n", output.Query.Kind, output.Query.Value)
	_, _ = fmt.Fprintf(writer, "nodes: %d\n", len(output.Nodes)+len(output.FlowNodes))
	_, _ = fmt.Fprintf(writer, "edges: %d\n", len(output.Edges)+len(output.FlowEdges))
	for _, item := range output.Diagnostics {
		_, _ = fmt.Fprintf(writer, "  - %s %s: %s\n", item.Severity, item.Code, item.Message)
	}
}

func normalizeResult(output result) result {
	if output.Nodes == nil {
		output.Nodes = []inspect.SymbolNode{}
	}
	if output.Edges == nil {
		output.Edges = []inspect.SymbolEdge{}
	}
	if output.Callers == nil {
		output.Callers = []traceHop{}
	}
	if output.Callees == nil {
		output.Callees = []traceHop{}
	}
	if output.FlowNodes == nil {
		output.FlowNodes = []inspect.FlowNode{}
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
	output.ResultKind = resultKindTrace
	output.SchemaVersion = schemaVersionTrace
	output.SchemaRef = schemaRefTrace
	output.OK = !output.Diagnostics.Failed()
	return output
}
