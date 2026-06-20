package serve

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
)

type serveResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Mode          string                 `json:"mode"`
	Summary       serveSummary           `json:"summary"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
	Server        serveServer            `json:"server"`
	Description   inspect.Description    `json:"-"`
}

type serveSummary struct {
	Status           string   `json:"status"`
	Service          string   `json:"service,omitempty"`
	Version          string   `json:"version,omitempty"`
	Addr             string   `json:"addr"`
	ServedPaths      []string `json:"served_paths"`
	EndpointCount    int      `json:"endpoint_count"`
	GRPCServiceCount int      `json:"grpc_service_count"`
	CapabilityCount  int      `json:"capability_count"`
	GeneratedFresh   bool     `json:"generated_fresh"`
	Errors           int      `json:"errors"`
	Warnings         int      `json:"warnings"`
}

type serveServer struct {
	Listening         bool               `json:"listening"`
	NetworkScope      string             `json:"network_scope"`
	MetadataEndpoints []metadataEndpoint `json:"metadata_endpoints"`
}

type metadataEndpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
}

func renderHuman(stdout io.Writer, stderr io.Writer, result serveResult) {
	for _, item := range result.Diagnostics {
		_, _ = fmt.Fprintf(stderr, "%s %s %s: %s\n", item.Severity, item.Path, item.Code, item.Message)
	}
	if result.OK {
		_, _ = fmt.Fprintln(stdout, "OK")
	} else {
		_, _ = fmt.Fprintln(stderr, "FAILED")
	}
	_, _ = fmt.Fprintf(stdout, "mode: %s\n", result.Mode)
	if result.Summary.Service != "" {
		_, _ = fmt.Fprintf(stdout, "service: %s\n", result.Summary.Service)
	}
	if result.Summary.Version != "" {
		_, _ = fmt.Fprintf(stdout, "version: %s\n", result.Summary.Version)
	}
	_, _ = fmt.Fprintf(stdout, "addr: %s\n", result.Summary.Addr)
	if len(result.Summary.ServedPaths) > 0 {
		_, _ = fmt.Fprintf(stdout, "served: %s\n", strings.Join(result.Summary.ServedPaths, ", "))
	}
	_, _ = fmt.Fprintf(stdout, "metadata: endpoints=%d grpc_services=%d capabilities=%d generated_fresh=%t\n",
		result.Summary.EndpointCount,
		result.Summary.GRPCServiceCount,
		result.Summary.CapabilityCount,
		result.Summary.GeneratedFresh,
	)
	_, _ = fmt.Fprintf(stdout, "diagnostics: %d errors, %d warnings\n", result.Summary.Errors, result.Summary.Warnings)
}

func renderJSON(writer io.Writer, result serveResult, pretty bool) error {
	result = finalizeResult(result)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(result)
}

func finalizeResult(result serveResult) serveResult {
	result.ResultKind = resultKindServe
	result.SchemaVersion = schemaVersionServe
	result.SchemaRef = schemaRefServe
	if result.Mode == "" {
		result.Mode = modeServe
	}
	if result.Summary.Addr == "" {
		result.Summary.Addr = defaultAddr
	}
	if result.Summary.Status == "" {
		result.Summary.Status = "ready"
	}
	if result.Summary.ServedPaths == nil {
		result.Summary.ServedPaths = servedPaths()
	}
	if result.Server.NetworkScope == "" {
		result.Server.NetworkScope = networkScopeInvalid
	}
	if result.Server.MetadataEndpoints == nil {
		result.Server.MetadataEndpoints = metadataEndpoints()
	}
	if result.Diagnostics == nil {
		result.Diagnostics = diagnostic.Diagnostics{}
	}
	result.Diagnostics.Sort()
	result.OK = !result.Diagnostics.Failed()
	if !result.OK {
		result.Summary.Status = "failed"
	}
	result.Summary.Errors = result.Diagnostics.Count(diagnostic.SeverityError)
	result.Summary.Warnings = result.Diagnostics.Count(diagnostic.SeverityWarning)
	return result
}

func errorDiagnostic(path string, code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}
