package mcp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/decision"
	describecmd "github.com/nucleuskit/nucleus/cmd/nucleus/internal/describe"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/graphquery"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/impact"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/plan"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/recipe"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/report"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/trace"
)

type toolDescriptor struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Tools returns MCP descriptors for all local Nucleus tools.
func (server *Server) Tools() []toolDescriptor {
	tools := []toolDescriptor{
		tool(toolGetServiceDescription, "Return the full local Nucleus service description.", objectSchema()),
		tool(toolGetEditSurfaces, "Return AI edit, readonly, and forbidden surfaces.", objectSchema()),
		tool(toolGetContracts, "Return declared contracts plus HTTP, gRPC, and error facts.", objectSchema()),
		tool(toolGetCapabilities, "Return manifest capabilities and capability graph facts.", objectSchema()),
		tool(toolTraceRoute, "Trace an HTTP route through the flow graph.", stringSchema("route", "Route query such as GET /orders.")),
		tool(toolTraceSymbol, "Trace callers and callees for a symbol.", stringSchema("symbol", "Stable symbol id or symbol name.")),
		tool(toolTraceCapability, "Trace callers and callees for a manifest capability anchor.", stringSchema("capability", "Capability id.")),
		tool(toolImpactSymbol, "Return symbol impact facts.", stringSchema("symbol", "Stable symbol id or symbol name.")),
		tool(toolImpactFile, "Return file impact facts.", stringSchema("path", "Relative file path.")),
		tool(toolImpactContract, "Return contract impact facts.", stringSchema("path", "Relative contract path.")),
		tool(toolFindSymbol, "Find symbols by stable id, name, package, or file.", findSymbolSchema()),
		tool(toolListCallers, "List direct callers for a symbol.", stringSchema("symbol", "Stable symbol id or symbol name.")),
		tool(toolListCallees, "List direct callees for a symbol.", stringSchema("symbol", "Stable symbol id or symbol name.")),
		tool(toolValidateDecision, "Validate structured decision evidence.", pathsSchema()),
		tool(toolListDecisions, "List and validate decision evidence summaries.", pathsSchema()),
		tool(toolGetReport, "Return local AI quality report facts.", aiTasksSchema()),
		tool(toolBuildPlan, "Build a Nucleus plan with impact summary.", buildPlanSchema()),
		tool(toolListRecipes, "List project and built-in read-only recipe knowledge.", recipeListSchema()),
		tool(toolGetRecipe, "Get one project or built-in read-only recipe by id.", stringSchema("id", "Recipe id.")),
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
	return tools
}

// CallTool executes one local MCP tool and returns structured JSON.
func (server *Server) CallTool(name string, args map[string]any) (any, error) {
	if args == nil {
		args = map[string]any{}
	}
	dir := server.dirArg(args)
	switch name {
	case toolGetServiceDescription:
		return describecmd.BuildOutput(describecmd.OutputOptions{Dir: dir})
	case toolGetEditSurfaces:
		description, err := inspect.Describe(dir)
		if err != nil {
			return nil, err
		}
		return mcpResult(resultKindMCPEditSurfaces, !description.Diagnostics.Failed(), description.Diagnostics, map[string]any{
			"edit_surfaces": description.EditSurfaces,
		}), nil
	case toolGetContracts:
		return server.getContracts(dir)
	case toolGetCapabilities:
		return server.getCapabilities(dir)
	case toolTraceRoute:
		return trace.RouteForMCP(dir, stringArg(args, "route")), nil
	case toolTraceSymbol:
		return trace.SymbolForMCP(dir, stringArg(args, "symbol")), nil
	case toolTraceCapability:
		return trace.CapabilityForMCP(dir, stringArg(args, "capability")), nil
	case toolImpactSymbol:
		return impact.SymbolForMCP(dir, stringArg(args, "symbol")), nil
	case toolImpactFile:
		return impact.FileForMCP(dir, stringArg(args, "path")), nil
	case toolImpactContract:
		return impact.ContractForMCP(dir, stringArg(args, "path")), nil
	case toolFindSymbol:
		return server.findSymbol(dir, stringArg(args, "query"), intArg(args, "limit", 50))
	case toolListCallers:
		return server.listCalls(dir, stringArg(args, "symbol"), true)
	case toolListCallees:
		return server.listCalls(dir, stringArg(args, "symbol"), false)
	case toolValidateDecision:
		return decision.ValidateForMCP(dir, stringSliceArg(args, "paths")), nil
	case toolListDecisions:
		return decision.ValidateForMCP(dir, stringSliceArg(args, "paths")), nil
	case toolGetReport:
		return report.BuildForMCP(dir, stringArg(args, "ai_tasks")), nil
	case toolBuildPlan:
		return plan.BuildOutput(plan.OutputOptions{Dir: dir, Task: stringArg(args, "task"), Executable: boolArg(args, "executable")})
	case toolListRecipes:
		return recipe.List(dir, recipe.Filter{Kind: stringArg(args, "kind"), Provider: stringArg(args, "provider")})
	case toolGetRecipe:
		return recipe.Get(dir, stringArg(args, "id"))
	default:
		return nil, fmt.Errorf("unknown tool %q", name)
	}
}

func (server *Server) getContracts(dir string) (any, error) {
	description, err := inspect.Describe(dir)
	if err != nil {
		return nil, err
	}
	var contracts []manifest.Contract
	if m, err := manifest.Load(dir); err == nil {
		contracts = m.Contracts
	}
	return mcpResult(resultKindMCPContracts, !description.Diagnostics.Failed(), description.Diagnostics, map[string]any{
		"contracts":     nonNil(contracts),
		"endpoints":     description.Endpoints,
		"grpc_services": description.GRPCServices,
		"error_codes":   description.ErrorCodes,
	}), nil
}

func (server *Server) getCapabilities(dir string) (any, error) {
	description, err := inspect.Describe(dir)
	if err != nil {
		return nil, err
	}
	var capabilities []manifest.Capability
	if m, err := manifest.Load(dir); err == nil {
		capabilities = m.Capabilities
	}
	return mcpResult(resultKindMCPCapabilities, !description.Diagnostics.Failed(), description.Diagnostics, map[string]any{
		"capabilities":       nonNil(capabilities),
		"capability_kinds":   description.Capabilities,
		"capability_graph":   description.CapabilityGraph,
		"decision_directory": defaultDecisionDir(),
	}), nil
}

func (server *Server) findSymbol(dir string, query string, limit int) (any, error) {
	description, err := inspect.Describe(dir)
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	var symbols []inspect.SymbolNode
	for _, node := range description.SymbolGraph.Nodes {
		if node.Kind == "package" || node.Kind == "file" {
			continue
		}
		if query == "" || symbolMatches(node, query) {
			symbols = append(symbols, node)
		}
	}
	sort.Slice(symbols, func(i, j int) bool { return symbols[i].ID < symbols[j].ID })
	if limit > 0 && len(symbols) > limit {
		symbols = symbols[:limit]
	}
	return mcpResult(resultKindMCPFindSymbol, !description.Diagnostics.Failed(), description.Diagnostics, map[string]any{
		"query":   query,
		"symbols": symbols,
	}), nil
}

func (server *Server) listCalls(dir string, symbol string, callers bool) (any, error) {
	description, err := inspect.Describe(dir)
	if err != nil {
		return nil, err
	}
	resolved := graphquery.ResolveSymbol(description.SymbolGraph, symbol)
	if !resolved.OK {
		diagnostics := mcpDiagnostics(description.Diagnostics, resolved.Diagnostic)
		return mcpResult(resultKindMCPCalls, false, diagnostics, map[string]any{
			"query":        symbol,
			"candidates":   resolved.Candidates,
			"relationship": chooseRelationship(callers),
		}), nil
	}
	edges := graphquery.OutgoingEdges(description.SymbolGraph, resolved.Node.ID, graphquery.EdgeKindCalls)
	if callers {
		edges = graphquery.IncomingEdges(description.SymbolGraph, resolved.Node.ID, graphquery.EdgeKindCalls)
	}
	diagnostics := mcpDiagnostics(description.Diagnostics)
	return mcpResult(resultKindMCPCalls, !diagnostics.Failed(), diagnostics, map[string]any{
		"query":        symbol,
		"relationship": chooseRelationship(callers),
		"target":       resolved.Node,
		"nodes":        graphquery.NodesForEdges(description.SymbolGraph, edges),
		"edges":        edges,
	}), nil
}

func mcpResult(resultKind string, ok bool, diagnostics diagnostic.Diagnostics, fields map[string]any) map[string]any {
	diagnostics = mcpDiagnostics(diagnostics)
	output := map[string]any{
		"result_kind":    resultKind,
		"schema_version": schemaVersionMCPResult,
		"schema_ref":     schemaRefMCPResult,
		"ok":             ok && !diagnostics.Failed(),
		"diagnostics":    diagnostics,
	}
	for key, value := range fields {
		output[key] = value
	}
	return output
}

func mcpDiagnostics(parts ...diagnostic.Diagnostics) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	for _, part := range parts {
		diagnostics = append(diagnostics, part...)
	}
	if diagnostics == nil {
		return diagnostic.Diagnostics{}
	}
	diagnostics.Sort()
	return diagnostics
}

func symbolMatches(node inspect.SymbolNode, query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(strings.ToLower(node.ID), query) ||
		strings.Contains(strings.ToLower(node.Name), query) ||
		strings.Contains(strings.ToLower(node.Package), query) ||
		strings.Contains(strings.ToLower(node.File), query)
}

func chooseRelationship(callers bool) string {
	if callers {
		return "callers"
	}
	return "callees"
}

func (server *Server) dirArg(args map[string]any) string {
	if value := stringArg(args, "dir"); value != "" {
		return value
	}
	return server.dir
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return strings.TrimSpace(value)
}

func boolArg(args map[string]any, key string) bool {
	value, _ := args[key].(bool)
	return value
}

func intArg(args map[string]any, key string, fallback int) int {
	switch value := args[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return fallback
	}
}

func stringSliceArg(args map[string]any, key string) []string {
	raw, ok := args[key].([]any)
	if !ok {
		if single := stringArg(args, key); single != "" {
			return []string{single}
		}
		return nil
	}
	var result []string
	for _, item := range raw {
		value, ok := item.(string)
		if ok && strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func defaultDecisionDir() string {
	return ".nucleus/decisions"
}

func tool(name string, description string, inputSchema map[string]any) toolDescriptor {
	return toolDescriptor{Name: name, Description: description, InputSchema: inputSchema}
}

func objectSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{"dir": map[string]any{"type": "string"}},
	}
}

func stringSchema(name string, description string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			name:  map[string]any{"type": "string", "description": description},
			"dir": map[string]any{"type": "string"},
		},
		"required": []string{name},
	}
}

func findSymbolSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
			"limit": map[string]any{"type": "integer", "minimum": 1},
			"dir":   map[string]any{"type": "string"},
		},
	}
}

func pathsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"paths": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"dir":   map[string]any{"type": "string"},
		},
	}
}

func aiTasksSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ai_tasks": map[string]any{"type": "string"},
			"dir":      map[string]any{"type": "string"},
		},
	}
}

func buildPlanSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task":       map[string]any{"type": "string"},
			"executable": map[string]any{"type": "boolean"},
			"dir":        map[string]any{"type": "string"},
		},
		"required": []string{"task"},
	}
}

func recipeListSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kind":     map[string]any{"type": "string"},
			"provider": map[string]any{"type": "string"},
			"dir":      map[string]any{"type": "string"},
		},
	}
}
