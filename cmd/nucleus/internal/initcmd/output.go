package initcmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nucleuskit/contract/diagnostic"
)

type initResult struct {
	ResultKind  string                 `json:"result_kind"`
	OK          bool                   `json:"ok"`
	Template    string                 `json:"template,omitempty"`
	ServiceName string                 `json:"service_name,omitempty"`
	Module      string                 `json:"module,omitempty"`
	Summary     initSummary            `json:"summary"`
	Files       []string               `json:"files"`
	Generated   []string               `json:"generated,omitempty"`
	Diagnostics diagnostic.Diagnostics `json:"diagnostics"`
}

type initSummary struct {
	Files     int `json:"files"`
	Generated int `json:"generated"`
	Errors    int `json:"errors"`
	Warnings  int `json:"warnings"`
}

func renderJSON(writer io.Writer, result initResult, pretty bool) error {
	result = normalizeResult(result)
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(result)
}

func normalizeResult(result initResult) initResult {
	if result.Files == nil {
		result.Files = []string{}
	}
	if result.Diagnostics == nil {
		result.Diagnostics = diagnostic.Diagnostics{}
	}
	return result
}

func renderHuman(writer io.Writer, result initResult) {
	_, _ = fmt.Fprintf(writer, "OK initialized %s %s\n", result.Template, result.ServiceName)
	_, _ = fmt.Fprintf(writer, "module: %s\n", result.Module)
	_, _ = fmt.Fprintln(writer, "files:")
	for _, file := range result.Files {
		_, _ = fmt.Fprintf(writer, "  - %s\n", file)
	}
	if len(result.Generated) == 0 {
		return
	}
	_, _ = fmt.Fprintln(writer, "generated:")
	for _, target := range result.Generated {
		_, _ = fmt.Fprintf(writer, "  - %s\n", target)
	}
}

func renderError(writer io.Writer, err error) {
	_, _ = fmt.Fprintln(writer, "FAILED")
	_, _ = fmt.Fprintf(writer, "error: %v\n", err)
}
