package trace

import (
	"errors"
	"os"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/graphquery"
)

func traceSymbol(config Config, value string) result {
	dir := stringValue(config.Dir, defaultDir)
	description, err := inspect.Describe(dir)
	if err != nil {
		return normalizeResult(result{
			Query:       query{Kind: "symbol", Value: value},
			Diagnostics: diagnostic.Diagnostics{errorDiagnostic("trace.describe_failed", err.Error())},
		})
	}
	resolved := graphquery.ResolveSymbol(description.SymbolGraph, value)
	if !resolved.OK {
		return normalizeResult(result{
			Query:       query{Kind: "symbol", Value: value},
			Candidates:  resolved.Candidates,
			Diagnostics: resolved.Diagnostic,
		})
	}
	incoming := graphquery.IncomingEdges(description.SymbolGraph, resolved.Node.ID, graphquery.EdgeKindCalls)
	outgoing := graphquery.OutgoingEdges(description.SymbolGraph, resolved.Node.ID, graphquery.EdgeKindCalls)
	edges := append([]inspect.SymbolEdge{}, incoming...)
	edges = append(edges, outgoing...)
	nodes := graphquery.NodesForEdges(description.SymbolGraph, edges, resolved.Node.ID)
	return normalizeResult(result{
		Query:   query{Kind: "symbol", Value: value, ResolvedID: resolved.Node.ID},
		Target:  &resolved.Node,
		Nodes:   nodes,
		Edges:   edges,
		Callers: hops(description.SymbolGraph, incoming, true),
		Callees: hops(description.SymbolGraph, outgoing, false),
	})
}

// SymbolForMCP returns the same structured trace result used by the CLI.
func SymbolForMCP(dir string, value string) any {
	return traceSymbol(Config{Dir: &dir}, value)
}

func traceRoute(config Config, value string) result {
	dir := stringValue(config.Dir, defaultDir)
	graph, err := inspect.BuildFlowGraphFromDir(dir)
	if err != nil {
		return normalizeResult(result{
			Query:       query{Kind: "route", Value: value},
			Diagnostics: diagnostic.Diagnostics{errorDiagnostic("trace.flow_graph_failed", err.Error())},
		})
	}
	route, ok := findRouteNode(graph, value)
	if !ok {
		return normalizeResult(result{
			Query:       query{Kind: "route", Value: value},
			Diagnostics: diagnostic.Diagnostics{errorDiagnostic("trace.route_not_found", "route was not found in flow graph")},
		})
	}
	edges := flowReachableEdges(graph, route.ID)
	nodes := flowNodesForEdges(graph, route.ID, edges)
	return normalizeResult(result{
		Query:     query{Kind: "route", Value: value, ResolvedID: route.ID},
		FlowNodes: nodes,
		FlowEdges: edges,
	})
}

// RouteForMCP returns the same structured route trace result used by the CLI.
func RouteForMCP(dir string, value string) any {
	return traceRoute(Config{Dir: &dir}, value)
}

func traceCapability(config Config, value string) result {
	dir := stringValue(config.Dir, defaultDir)
	m, err := manifest.Load(dir)
	if err != nil {
		code := "trace.manifest_load_failed"
		message := err.Error()
		if errors.Is(err, os.ErrNotExist) {
			code = "trace.manifest_missing"
			message = "nucleus.yaml is required to trace capability anchors"
		}
		return normalizeResult(result{
			Query:       query{Kind: "capability", Value: value},
			Diagnostics: diagnostic.Diagnostics{errorDiagnostic(code, message)},
		})
	}
	capability, ok := findCapability(m, value)
	if !ok {
		return normalizeResult(result{
			Query:       query{Kind: "capability", Value: value},
			Diagnostics: diagnostic.Diagnostics{errorDiagnostic("trace.capability_not_found", "capability was not found in nucleus.yaml")},
		})
	}
	if len(capability.Symbols) == 0 {
		return normalizeResult(result{
			Query:       query{Kind: "capability", Value: value, ResolvedID: capability.ID},
			Diagnostics: diagnostic.Diagnostics{warningDiagnostic("trace.capability_no_symbols", "capability has no symbol anchors")},
		})
	}
	description, err := inspect.Describe(dir)
	if err != nil {
		return normalizeResult(result{
			Query:       query{Kind: "capability", Value: value, ResolvedID: capability.ID},
			Diagnostics: diagnostic.Diagnostics{errorDiagnostic("trace.describe_failed", err.Error())},
		})
	}
	var diagnostics diagnostic.Diagnostics
	var candidates []inspect.SymbolNode
	var incoming []inspect.SymbolEdge
	var outgoing []inspect.SymbolEdge
	var anchorIDs []string
	for _, symbol := range capability.Symbols {
		symbolQuery := symbol.ID
		if symbolQuery == "" {
			symbolQuery = symbol.Name
		}
		resolved := graphquery.ResolveSymbol(description.SymbolGraph, symbolQuery)
		if resolved.OK {
			anchorIDs = append(anchorIDs, resolved.Node.ID)
			incoming = append(incoming, graphquery.IncomingEdges(description.SymbolGraph, resolved.Node.ID, graphquery.EdgeKindCalls)...)
			outgoing = append(outgoing, graphquery.OutgoingEdges(description.SymbolGraph, resolved.Node.ID, graphquery.EdgeKindCalls)...)
			continue
		}
		if len(resolved.Candidates) > 0 {
			candidates = append(candidates, resolved.Candidates...)
			diagnostics = append(diagnostics, errorDiagnostic("trace.symbol_ambiguous", "capability symbol matched multiple candidates; rerun mark with a stable symbol id"))
			continue
		}
		diagnostics = append(diagnostics, warningDiagnostic("trace.capability_symbol_unresolved", "capability symbol anchor is not present in the current symbol graph"))
	}
	edges := append([]inspect.SymbolEdge{}, incoming...)
	edges = append(edges, outgoing...)
	edges = uniqueEdges(edges)
	nodes := graphquery.NodesForEdges(description.SymbolGraph, edges, anchorIDs...)
	return normalizeResult(result{
		Query:       query{Kind: "capability", Value: value, ResolvedID: capability.ID},
		Nodes:       nodes,
		Edges:       edges,
		Callers:     hops(description.SymbolGraph, uniqueEdges(incoming), true),
		Callees:     hops(description.SymbolGraph, uniqueEdges(outgoing), false),
		Candidates:  candidates,
		Diagnostics: diagnostics,
	})
}

// CapabilityForMCP returns the same structured capability trace result used by the CLI.
func CapabilityForMCP(dir string, value string) any {
	return traceCapability(Config{Dir: &dir}, value)
}

func hops(graph inspect.SymbolGraph, edges []inspect.SymbolEdge, incoming bool) []traceHop {
	nodeByID := map[string]inspect.SymbolNode{}
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = node
	}
	result := make([]traceHop, 0, len(edges))
	for _, edge := range edges {
		id := edge.To
		if incoming {
			id = edge.From
		}
		node, ok := nodeByID[id]
		if !ok {
			continue
		}
		result = append(result, traceHop{Node: node, Edge: edge})
	}
	return result
}

func findRouteNode(graph inspect.FlowGraph, value string) (inspect.FlowNode, bool) {
	value = strings.TrimSpace(value)
	for _, node := range graph.Nodes {
		if node.Kind != routeNodeKind {
			continue
		}
		if node.Name == value || strings.EqualFold(node.Method+" "+node.Path, value) || node.ID == value {
			return node, true
		}
	}
	return inspect.FlowNode{}, false
}

func flowReachableEdges(graph inspect.FlowGraph, routeID string) []inspect.FlowEdge {
	seenNodes := map[string]bool{routeID: true}
	var result []inspect.FlowEdge
	changed := true
	for changed {
		changed = false
		for _, edge := range graph.Edges {
			if !seenNodes[edge.From] {
				continue
			}
			if containsFlowEdge(result, edge) {
				continue
			}
			result = append(result, edge)
			if !seenNodes[edge.To] {
				seenNodes[edge.To] = true
				changed = true
			}
		}
	}
	return result
}

func flowNodesForEdges(graph inspect.FlowGraph, routeID string, edges []inspect.FlowEdge) []inspect.FlowNode {
	nodeByID := map[string]inspect.FlowNode{}
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = node
	}
	seen := map[string]bool{}
	var result []inspect.FlowNode
	add := func(id string) {
		if seen[id] {
			return
		}
		node, ok := nodeByID[id]
		if !ok {
			return
		}
		seen[id] = true
		result = append(result, node)
	}
	add(routeID)
	for _, edge := range edges {
		add(edge.From)
		add(edge.To)
	}
	return result
}

func containsFlowEdge(edges []inspect.FlowEdge, want inspect.FlowEdge) bool {
	for _, edge := range edges {
		if edge.From == want.From && edge.To == want.To && edge.Kind == want.Kind {
			return true
		}
	}
	return false
}

func findCapability(m manifest.Manifest, id string) (manifest.Capability, bool) {
	id = strings.TrimSpace(id)
	for _, capability := range m.Capabilities {
		if capability.ID == id {
			return capability, true
		}
	}
	return manifest.Capability{}, false
}

func uniqueEdges(edges []inspect.SymbolEdge) []inspect.SymbolEdge {
	seen := map[string]bool{}
	var result []inspect.SymbolEdge
	for _, edge := range edges {
		key := edge.From + "\x00" + edge.Kind + "\x00" + edge.To
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, edge)
	}
	return result
}

func errorDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: code, Message: message}
}

func warningDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: diagnostic.SeverityWarning, Code: code, Message: message}
}
