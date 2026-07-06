package graphquery

import (
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
)

const (
	EdgeKindCalls = "calls"
	EdgeKindTests = "tests"
)

// SymbolResolution is the result of resolving a user-supplied symbol query.
type SymbolResolution struct {
	Node       inspect.SymbolNode
	Candidates []inspect.SymbolNode
	Diagnostic diagnostic.Diagnostics
	OK         bool
}

// ResolveSymbol resolves stable symbol IDs and short names without guessing on ambiguity.
func ResolveSymbol(graph inspect.SymbolGraph, query string) SymbolResolution {
	query = strings.TrimSpace(query)
	if query == "" {
		return SymbolResolution{Diagnostic: diagnostic.Diagnostics{errorDiagnostic("graph.symbol_query_required", "symbol query is required")}}
	}
	if strings.HasPrefix(query, "go://") {
		for _, node := range graph.Nodes {
			if node.ID == query {
				return SymbolResolution{Node: node, OK: true}
			}
		}
		return SymbolResolution{Diagnostic: diagnostic.Diagnostics{errorDiagnostic("graph.symbol_not_found", "symbol id was not found")}}
	}
	var candidates []inspect.SymbolNode
	for _, node := range graph.Nodes {
		if node.Name == query || strings.HasSuffix(node.ID, "#"+query) {
			candidates = append(candidates, node)
		}
	}
	sortNodes(candidates)
	switch len(candidates) {
	case 0:
		return SymbolResolution{Diagnostic: diagnostic.Diagnostics{errorDiagnostic("graph.symbol_not_found", "symbol name was not found")}}
	case 1:
		return SymbolResolution{Node: candidates[0], OK: true}
	default:
		return SymbolResolution{
			Candidates: candidates,
			Diagnostic: diagnostic.Diagnostics{errorDiagnostic("graph.symbol_ambiguous", "symbol name matched multiple candidates; rerun with a stable symbol id")},
		}
	}
}

// IncomingEdges returns sorted graph edges pointing to id. When kinds are provided, only those kinds are returned.
func IncomingEdges(graph inspect.SymbolGraph, id string, kinds ...string) []inspect.SymbolEdge {
	allowed := edgeKindSet(kinds)
	var result []inspect.SymbolEdge
	for _, edge := range graph.Edges {
		if edge.To == id && (len(allowed) == 0 || allowed[edge.Kind]) {
			result = append(result, edge)
		}
	}
	sortEdges(result)
	return result
}

// OutgoingEdges returns sorted graph edges leaving id. When kinds are provided, only those kinds are returned.
func OutgoingEdges(graph inspect.SymbolGraph, id string, kinds ...string) []inspect.SymbolEdge {
	allowed := edgeKindSet(kinds)
	var result []inspect.SymbolEdge
	for _, edge := range graph.Edges {
		if edge.From == id && (len(allowed) == 0 || allowed[edge.Kind]) {
			result = append(result, edge)
		}
	}
	sortEdges(result)
	return result
}

// NodesForEdges returns stable unique nodes referenced by edges plus optional explicit ids.
func NodesForEdges(graph inspect.SymbolGraph, edges []inspect.SymbolEdge, ids ...string) []inspect.SymbolNode {
	nodeByID := map[string]inspect.SymbolNode{}
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = node
	}
	seen := map[string]bool{}
	var result []inspect.SymbolNode
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
	for _, id := range ids {
		add(id)
	}
	for _, edge := range edges {
		add(edge.From)
		add(edge.To)
	}
	sortNodes(result)
	return result
}

// FilesForNodes returns sorted files attached to symbol nodes.
func FilesForNodes(nodes []inspect.SymbolNode) []string {
	seen := map[string]bool{}
	var result []string
	for _, node := range nodes {
		if node.File == "" || seen[node.File] {
			continue
		}
		seen[node.File] = true
		result = append(result, node.File)
	}
	sort.Strings(result)
	return result
}

// NodeByFile returns file nodes that match the relative file path.
func NodeByFile(graph inspect.SymbolGraph, path string) []inspect.SymbolNode {
	path = strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"), "./")
	var result []inspect.SymbolNode
	for _, node := range graph.Nodes {
		if node.File == path && node.Kind == "file" {
			result = append(result, node)
		}
	}
	sortNodes(result)
	return result
}

func edgeKindSet(kinds []string) map[string]bool {
	set := map[string]bool{}
	for _, kind := range kinds {
		set[kind] = true
	}
	return set
}

func sortNodes(nodes []inspect.SymbolNode) {
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
}

func sortEdges(edges []inspect.SymbolEdge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			if edges[i].Kind == edges[j].Kind {
				return edges[i].To < edges[j].To
			}
			return edges[i].Kind < edges[j].Kind
		}
		return edges[i].From < edges[j].From
	})
}

func errorDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: code, Message: message}
}
