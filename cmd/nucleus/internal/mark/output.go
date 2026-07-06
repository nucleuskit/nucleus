package mark

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
	Action        string                 `json:"action"`
	ManifestPath  string                 `json:"manifest_path"`
	Changed       bool                   `json:"changed"`
	Entry         any                    `json:"entry,omitempty"`
	Symbols       []symbolMark           `json:"symbols,omitempty"`
	Candidates    []inspect.SymbolNode   `json:"candidates,omitempty"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
}

type symbolMark struct {
	Query  string `json:"query"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status"`
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
		_, _ = fmt.Fprintln(writer, "OK mark")
	} else {
		_, _ = fmt.Fprintln(writer, "FAILED mark")
	}
	_, _ = fmt.Fprintf(writer, "action: %s\n", output.Action)
	_, _ = fmt.Fprintf(writer, "changed: %t\n", output.Changed)
	for _, item := range output.Diagnostics {
		_, _ = fmt.Fprintf(writer, "  - %s %s %s: %s\n", item.Severity, item.Path, item.Code, item.Message)
	}
}

func normalizeResult(output result) result {
	if output.ManifestPath == "" {
		output.ManifestPath = manifestFileName
	}
	if output.Symbols == nil {
		output.Symbols = []symbolMark{}
	}
	if output.Candidates == nil {
		output.Candidates = []inspect.SymbolNode{}
	}
	if output.Diagnostics == nil {
		output.Diagnostics = diagnostic.Diagnostics{}
	}
	output.Diagnostics.Sort()
	output.ResultKind = resultKindMark
	output.SchemaVersion = schemaVersionMark
	output.SchemaRef = schemaRefMark
	output.OK = !output.Diagnostics.Failed()
	return output
}
