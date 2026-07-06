package plan

import (
	"os"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
)

type impactSummary struct {
	Mode                  string               `json:"mode"`
	AffectedSymbols       []inspect.SymbolNode `json:"affected_symbols"`
	AffectedFiles         []string             `json:"affected_files"`
	AffectedRoutes        []routeImpact        `json:"affected_routes"`
	AffectedContracts     []string             `json:"affected_contracts"`
	AffectedTests         []inspect.SymbolNode `json:"affected_tests"`
	AffectedCapabilities  []string             `json:"affected_capabilities"`
	GraphEdges            []inspect.SymbolEdge `json:"graph_edges"`
	SuggestedVerification []string             `json:"suggested_verification"`
	Warnings              []string             `json:"warnings"`
}

type routeImpact struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	OperationID string `json:"operation_id,omitempty"`
}

func buildImpactSummary(dir string, task string, taskType string, description inspect.Description, requestedCapabilities []string, suggestedEdits []string, commands []string) impactSummary {
	summary := impactSummary{
		Mode:                  "best_effort",
		AffectedSymbols:       []inspect.SymbolNode{},
		AffectedFiles:         []string{},
		AffectedRoutes:        []routeImpact{},
		AffectedContracts:     affectedContracts(taskType, description, suggestedEdits),
		AffectedTests:         []inspect.SymbolNode{},
		AffectedCapabilities:  []string{},
		GraphEdges:            []inspect.SymbolEdge{},
		SuggestedVerification: append([]string{}, commands...),
		Warnings:              []string{},
	}

	manifestCapabilities := loadPlanCapabilities(dir)
	summary.AffectedCapabilities = affectedCapabilities(task, description, manifestCapabilities, requestedCapabilities)

	seedIDs := matchedSymbolIDs(task, description.SymbolGraph)
	seedIDs = append(seedIDs, capabilitySymbolIDs(description.SymbolGraph, manifestCapabilities, summary.AffectedCapabilities)...)
	edges := impactEdges(description.SymbolGraph, seedIDs)
	nodes := nodesForImpact(description.SymbolGraph, seedIDs, edges)
	summary.AffectedCapabilities = uniqueStrings(append(summary.AffectedCapabilities, capabilitiesForNodes(manifestCapabilities, nodes)...))
	summary.AffectedSymbols = nonTestSymbols(nodes)
	summary.AffectedTests = testSymbols(nodes)
	summary.AffectedFiles = filesForSymbols(nodes)
	summary.GraphEdges = uniquePlanEdges(edges)
	summary.AffectedRoutes = affectedRoutes(task, taskType, description)
	if len(summary.AffectedRoutes) > 0 {
		summary.AffectedContracts = uniqueSortedStrings(append(summary.AffectedContracts, "api/openapi.yaml"))
	}

	if len(summary.AffectedSymbols) == 0 && len(summary.AffectedRoutes) == 0 && len(summary.AffectedCapabilities) == 0 {
		summary.Warnings = append(summary.Warnings, "no direct graph match found for task text; inspect trace/impact before editing business logic")
	}
	return summary
}

func uniqueSortedStrings(values []string) []string {
	result := uniqueStrings(values)
	sort.Strings(result)
	return result
}

func affectedContracts(taskType string, description inspect.Description, suggestedEdits []string) []string {
	seen := map[string]bool{}
	var result []string
	add := func(value string) {
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		result = append(result, value)
	}
	for _, edit := range suggestedEdits {
		if strings.HasPrefix(edit, "api/") && !strings.ContainsAny(edit, "*?[]") {
			add(edit)
		}
	}
	switch taskType {
	case taskTypeHTTPEndpoint:
		add("api/openapi.yaml")
	case taskTypeErrorCatalog:
		add("api/errors.yaml")
	case taskTypeGRPCService:
		for _, service := range description.GRPCServices {
			add(service.Source)
		}
		if len(description.GRPCServices) == 0 {
			add("api/proto/*.proto")
		}
	}
	sort.Strings(result)
	return result
}

func loadPlanCapabilities(dir string) []manifest.Capability {
	m, err := manifest.Load(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return nil
	}
	return m.Capabilities
}

func affectedCapabilities(task string, description inspect.Description, capabilities []manifest.Capability, requested []string) []string {
	text := normalizeTaskText(task)
	seen := map[string]bool{}
	var result []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		result = append(result, value)
	}
	for _, capability := range requested {
		add(capability)
	}
	for _, capability := range description.Capabilities {
		if textContainsToken(text, capability) {
			add(capability)
		}
	}
	for _, capability := range capabilities {
		if textContainsToken(text, capability.ID) || textContainsToken(text, capability.Kind) {
			add(capability.ID)
		}
	}
	sort.Strings(result)
	return result
}

func matchedSymbolIDs(task string, graph inspect.SymbolGraph) []string {
	text := normalizeTaskText(task)
	var ids []string
	for _, node := range graph.Nodes {
		if node.Kind == "package" || node.Kind == "file" {
			continue
		}
		if textContainsToken(text, node.Name) || strings.Contains(strings.ToLower(task), strings.ToLower(node.ID)) {
			ids = append(ids, node.ID)
		}
	}
	sort.Strings(ids)
	return uniqueStrings(ids)
}

func capabilitySymbolIDs(graph inspect.SymbolGraph, capabilities []manifest.Capability, affected []string) []string {
	if len(affected) == 0 {
		return nil
	}
	affectedSet := map[string]bool{}
	for _, value := range affected {
		affectedSet[value] = true
	}
	var queries []string
	for _, capability := range capabilities {
		if !affectedSet[capability.ID] && !affectedSet[capability.Kind] {
			continue
		}
		for _, symbol := range capability.Symbols {
			if symbol.ID != "" {
				queries = append(queries, symbol.ID)
			} else if symbol.Name != "" {
				queries = append(queries, symbol.Name)
			}
		}
	}
	var ids []string
	for _, query := range queries {
		ids = append(ids, resolvePlanSymbolID(graph, query)...)
	}
	sort.Strings(ids)
	return uniqueStrings(ids)
}

func resolvePlanSymbolID(graph inspect.SymbolGraph, query string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if strings.HasPrefix(query, "go://") {
		for _, node := range graph.Nodes {
			if node.ID == query {
				return []string{node.ID}
			}
		}
		return nil
	}
	var ids []string
	for _, node := range graph.Nodes {
		if node.Name == query || strings.HasSuffix(node.ID, "#"+query) {
			ids = append(ids, node.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func impactEdges(graph inspect.SymbolGraph, seedIDs []string) []inspect.SymbolEdge {
	if len(seedIDs) == 0 {
		return nil
	}
	seed := map[string]bool{}
	for _, id := range seedIDs {
		seed[id] = true
	}
	var edges []inspect.SymbolEdge
	for _, edge := range graph.Edges {
		if edge.Kind != "calls" && edge.Kind != "tests" && edge.Kind != "implements" && edge.Kind != "accepts" && edge.Kind != "returns" {
			continue
		}
		if seed[edge.From] || seed[edge.To] {
			edges = append(edges, edge)
		}
	}
	return uniquePlanEdges(edges)
}

func capabilitiesForNodes(capabilities []manifest.Capability, nodes []inspect.SymbolNode) []string {
	nodeIDs := map[string]bool{}
	nodeNames := map[string]bool{}
	for _, node := range nodes {
		nodeIDs[node.ID] = true
		nodeNames[node.Name] = true
	}
	var result []string
	for _, capability := range capabilities {
		for _, symbol := range capability.Symbols {
			if symbol.ID != "" && nodeIDs[symbol.ID] || symbol.Name != "" && nodeNames[symbol.Name] {
				result = append(result, capability.ID)
				break
			}
		}
	}
	sort.Strings(result)
	return uniqueStrings(result)
}

func nodesForImpact(graph inspect.SymbolGraph, seedIDs []string, edges []inspect.SymbolEdge) []inspect.SymbolNode {
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
	for _, id := range seedIDs {
		add(id)
	}
	for _, edge := range edges {
		add(edge.From)
		add(edge.To)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func nonTestSymbols(nodes []inspect.SymbolNode) []inspect.SymbolNode {
	var result []inspect.SymbolNode
	for _, node := range nodes {
		if isTestSymbol(node) {
			continue
		}
		result = append(result, node)
	}
	return result
}

func testSymbols(nodes []inspect.SymbolNode) []inspect.SymbolNode {
	var result []inspect.SymbolNode
	for _, node := range nodes {
		if isTestSymbol(node) {
			result = append(result, node)
		}
	}
	return result
}

func isTestSymbol(node inspect.SymbolNode) bool {
	return node.Kind == "function" && (strings.HasPrefix(node.Name, "Test") || strings.HasSuffix(node.File, "_test.go"))
}

func filesForSymbols(nodes []inspect.SymbolNode) []string {
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

func affectedRoutes(task string, taskType string, description inspect.Description) []routeImpact {
	text := normalizeTaskText(task)
	includeAll := taskType == taskTypeHTTPEndpoint && len(description.Endpoints) == 1
	var routes []routeImpact
	for _, endpoint := range description.Endpoints {
		routeText := endpoint.Method + " " + endpoint.Path + " " + endpoint.OperationID
		if includeAll || routeMatchesTask(text, routeText, endpoint.Path, endpoint.OperationID) {
			routes = append(routes, routeImpact{Method: endpoint.Method, Path: endpoint.Path, OperationID: endpoint.OperationID})
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Method == routes[j].Method {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})
	return routes
}

func routeMatchesTask(text string, routeText string, path string, operationID string) bool {
	if textContainsToken(text, operationID) || strings.Contains(text, strings.ToLower(routeText)) {
		return true
	}
	for _, token := range splitPlanTokens(strings.Trim(path, "/")) {
		if token == "" || len(token) < 3 {
			continue
		}
		if textContainsToken(text, token) || textContainsToken(text, strings.TrimSuffix(token, "s")) {
			return true
		}
	}
	return false
}

func uniquePlanEdges(edges []inspect.SymbolEdge) []inspect.SymbolEdge {
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

func normalizeTaskText(task string) string {
	return " " + strings.Join(splitPlanTokens(task), " ") + " "
}

func textContainsToken(text string, token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	tokens := splitPlanTokens(token)
	if len(tokens) == 0 {
		return false
	}
	return strings.Contains(text, " "+strings.Join(tokens, " ")+" ") || strings.Contains(text, " "+tokens[len(tokens)-1]+" ")
}

func splitPlanTokens(value string) []string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer("/", " ", "_", " ", "-", " ", ".", " ", "#", " ", ":", " ", "{", " ", "}", " ", "(", " ", ")", " ", "\"", " ", "'", " ")
	value = replacer.Replace(value)
	return strings.Fields(value)
}
