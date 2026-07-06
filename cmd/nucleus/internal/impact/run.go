package impact

import (
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/graphquery"
)

func impactSymbol(config Config, value string) result {
	dir := stringValue(config.Dir, defaultDir)
	description, err := inspect.Describe(dir)
	if err != nil {
		return normalizeResult(result{Query: query{Kind: "symbol", Value: value}, Diagnostics: diagnostic.Diagnostics{errorDiagnostic("impact.describe_failed", err.Error())}})
	}
	resolved := graphquery.ResolveSymbol(description.SymbolGraph, value)
	if !resolved.OK {
		return normalizeResult(result{Query: query{Kind: "symbol", Value: value}, Candidates: resolved.Candidates, Diagnostics: resolved.Diagnostic})
	}
	return impactResolvedSymbol(description.SymbolGraph, resolved.Node, query{Kind: "symbol", Value: value, ResolvedID: resolved.Node.ID})
}

// SymbolForMCP returns the same structured impact result used by the CLI.
func SymbolForMCP(dir string, value string) any {
	return impactSymbol(Config{Dir: &dir}, value)
}

func impactFile(config Config, value string) result {
	dir := stringValue(config.Dir, defaultDir)
	description, err := inspect.Describe(dir)
	if err != nil {
		return normalizeResult(result{Query: query{Kind: "file", Value: value}, Diagnostics: diagnostic.Diagnostics{errorDiagnostic("impact.describe_failed", err.Error())}})
	}
	fileNodes := graphquery.NodeByFile(description.SymbolGraph, value)
	if len(fileNodes) == 0 {
		return normalizeResult(result{Query: query{Kind: "file", Value: value}, Diagnostics: diagnostic.Diagnostics{errorDiagnostic("impact.file_not_found", "file was not found in symbol graph")}})
	}
	edges := append([]inspect.SymbolEdge{}, graphquery.OutgoingEdges(description.SymbolGraph, fileNodes[0].ID, "declares")...)
	var symbols []inspect.SymbolNode
	for _, edge := range edges {
		symbols = append(symbols, graphquery.NodesForEdges(description.SymbolGraph, []inspect.SymbolEdge{edge})...)
	}
	expandedEdges := append([]inspect.SymbolEdge{}, edges...)
	for _, node := range symbols {
		if node.ID == fileNodes[0].ID {
			continue
		}
		expandedEdges = append(expandedEdges, symbolImpactEdges(description.SymbolGraph, node.ID)...)
	}
	affected := graphquery.NodesForEdges(description.SymbolGraph, expandedEdges, fileNodes[0].ID)
	tests := testNodes(affected)
	return normalizeResult(result{
		Query:           query{Kind: "file", Value: value, ResolvedID: fileNodes[0].ID},
		AffectedSymbols: affected,
		AffectedFiles:   graphquery.FilesForNodes(affected),
		AffectedTests:   tests,
		Edges:           uniqueSymbolEdges(expandedEdges),
	})
}

// FileForMCP returns the same structured file impact result used by the CLI.
func FileForMCP(dir string, value string) any {
	return impactFile(Config{Dir: &dir}, value)
}

func impactContract(config Config, value string) result {
	dir := stringValue(config.Dir, defaultDir)
	graph, err := inspect.BuildFlowGraphFromDir(dir)
	if err != nil {
		return normalizeResult(result{Query: query{Kind: "contract", Value: value}, Diagnostics: diagnostic.Diagnostics{errorDiagnostic("impact.flow_graph_failed", err.Error())}})
	}
	routes := routeNodes(graph)
	edges := graph.Edges
	if len(routes) == 0 {
		return normalizeResult(result{Query: query{Kind: "contract", Value: value}, AffectedRoutes: []inspect.FlowNode{}, FlowEdges: []inspect.FlowEdge{}, Diagnostics: diagnostic.Diagnostics{warningDiagnostic("impact.contract_no_routes", "contract produced no route facts")}})
	}
	return normalizeResult(result{
		Query:          query{Kind: "contract", Value: value},
		AffectedFiles:  []string{normalizePath(value)},
		AffectedRoutes: routes,
		FlowEdges:      edges,
	})
}

// ContractForMCP returns the same structured contract impact result used by the CLI.
func ContractForMCP(dir string, value string) any {
	return impactContract(Config{Dir: &dir}, value)
}

func impactResolvedSymbol(graph inspect.SymbolGraph, target inspect.SymbolNode, q query) result {
	edges := symbolImpactEdges(graph, target.ID)
	nodes := graphquery.NodesForEdges(graph, edges, target.ID)
	tests := testNodes(nodes)
	return normalizeResult(result{
		Query:           q,
		Target:          &target,
		AffectedSymbols: nodes,
		AffectedFiles:   graphquery.FilesForNodes(nodes),
		AffectedTests:   tests,
		Edges:           uniqueSymbolEdges(edges),
	})
}

func symbolImpactEdges(graph inspect.SymbolGraph, id string) []inspect.SymbolEdge {
	var edges []inspect.SymbolEdge
	edges = append(edges, graphquery.IncomingEdges(graph, id, graphquery.EdgeKindCalls, graphquery.EdgeKindTests)...)
	edges = append(edges, graphquery.OutgoingEdges(graph, id, graphquery.EdgeKindCalls)...)
	edges = append(edges, graphquery.IncomingEdges(graph, id, "implements")...)
	edges = append(edges, graphquery.OutgoingEdges(graph, id, "implements")...)
	return uniqueSymbolEdges(edges)
}

func uniqueSymbolEdges(edges []inspect.SymbolEdge) []inspect.SymbolEdge {
	seen := map[string]bool{}
	var result []inspect.SymbolEdge
	for _, edge := range edges {
		key := edge.From + "\x00" + edge.To + "\x00" + edge.Kind
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, edge)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].From == result[j].From {
			if result[i].Kind == result[j].Kind {
				return result[i].To < result[j].To
			}
			return result[i].Kind < result[j].Kind
		}
		return result[i].From < result[j].From
	})
	return result
}

func testNodes(nodes []inspect.SymbolNode) []inspect.SymbolNode {
	var result []inspect.SymbolNode
	for _, node := range nodes {
		if node.Kind == "function" && strings.HasPrefix(node.Name, "Test") {
			result = append(result, node)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func routeNodes(graph inspect.FlowGraph) []inspect.FlowNode {
	var result []inspect.FlowNode
	for _, node := range graph.Nodes {
		if node.Kind == routeNodeKind {
			result = append(result, node)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func normalizePath(value string) string {
	return strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"), "./")
}

func errorDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: code, Message: message}
}

func warningDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: diagnostic.SeverityWarning, Code: code, Message: message}
}
