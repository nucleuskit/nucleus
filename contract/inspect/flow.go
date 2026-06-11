package inspect

import (
	"github.com/nucleuskit/contract/errors"
	"github.com/nucleuskit/contract/openapi"
)

// BuildFlowGraphFromDir builds a conservative flow graph from contracts and source hints.
func BuildFlowGraphFromDir(dir string) (FlowGraph, error) {
	routes, err := openapi.LoadRouteRegistry(dir)
	if err != nil {
		return FlowGraph{}, err
	}
	errorCodes, err := errors.Load(dir)
	if err != nil {
		return FlowGraph{}, err
	}
	graph := BuildFlowGraphFromContracts(routes, errorCodes)
	enrichFlowGraphFromSource(dir, routes, &graph)
	return graph, nil
}

// BuildFlowGraphFromContracts builds route, operation, parameter, and error facts from contracts.
func BuildFlowGraphFromContracts(routes []openapi.Route, errorCodes []errors.Code) FlowGraph {
	graph := FlowGraph{
		SchemaVersion: flowGraphSchemaVersion,
		Nodes:         make([]FlowNode, 0, len(routes)*3),
		Edges:         make([]FlowEdge, 0, len(routes)*2),
		Params:        []FlowFact{},
		ContextFields: []FlowFact{},
		ErrorPaths:    make([]FlowFact, 0, len(errorCodes)),
	}
	for _, route := range routes {
		endpoint := openapi.Endpoint{Method: route.Method, Path: route.Path, OperationID: route.OperationID}
		routeID := flowRouteID(endpoint)
		operationID := flowOperationID(endpoint)
		outboundID := routeID + flowUnknownOutboundID
		graph.Nodes = append(graph.Nodes,
			FlowNode{
				ID:          routeID,
				Kind:        flowNodeKindRoute,
				Name:        endpoint.Method + " " + endpoint.Path,
				Method:      endpoint.Method,
				Path:        endpoint.Path,
				OperationID: endpoint.OperationID,
				Source:      flowSourceOpenAPI,
				Inferred:    false,
			},
			FlowNode{
				ID:          operationID,
				Kind:        flowNodeKindHandler,
				Name:        chooseString(endpoint.OperationID, flowUnknownName),
				Method:      endpoint.Method,
				Path:        endpoint.Path,
				OperationID: endpoint.OperationID,
				Source:      flowSourceOpenAPI,
				Inferred:    false,
			},
			FlowNode{
				ID:       outboundID,
				Kind:     flowNodeKindOutbound,
				Name:     flowUnknownName,
				Source:   flowSourceStaticUnavailable,
				Inferred: false,
			},
		)
		graph.Edges = append(graph.Edges,
			FlowEdge{From: routeID, To: operationID, Kind: flowEdgeKindDispatch, Inferred: false},
			FlowEdge{From: operationID, To: outboundID, Kind: flowEdgeKindOutboundUnknown, Inferred: false},
		)
		for _, parameter := range route.Parameters {
			graph.Params = append(graph.Params, FlowFact{
				Name:        parameter.In + "." + parameter.Name,
				Kind:        flowFactKindOpenAPIParameter,
				Source:      flowSourceOpenAPI,
				Route:       route.Method + " " + route.Path,
				OperationID: route.OperationID,
				Required:    parameter.Required,
				SchemaType:  parameter.SchemaType,
				Inferred:    false,
			})
		}
		if route.RequestBodyRequired {
			graph.Params = append(graph.Params, FlowFact{
				Name:        flowRequestBodyName,
				Kind:        flowFactKindOpenAPIRequestBody,
				Source:      flowSourceOpenAPI,
				Route:       route.Method + " " + route.Path,
				OperationID: route.OperationID,
				Required:    true,
				Inferred:    false,
			})
		}
	}
	for _, code := range errorCodes {
		if code.Code == 0 {
			continue
		}
		graph.ErrorPaths = append(graph.ErrorPaths, FlowFact{
			Name:       code.Message,
			Kind:       flowFactKindErrorCode,
			Source:     flowSourceErrors,
			Code:       code.Code,
			HTTPStatus: code.HTTPStatus,
			Inferred:   false,
		})
	}
	return graph
}

func flowRouteID(endpoint openapi.Endpoint) string {
	return flowIDPrefixRoute + endpoint.Method + " " + endpoint.Path
}

func flowOperationID(endpoint openapi.Endpoint) string {
	if endpoint.OperationID != "" {
		return flowIDPrefixOperation + endpoint.OperationID
	}
	return flowIDPrefixOperation + endpoint.Method + " " + endpoint.Path
}

func updateFlowNode(graph *FlowGraph, id string, name string, source string, inferred bool) {
	for i := range graph.Nodes {
		if graph.Nodes[i].ID == id {
			graph.Nodes[i].Name = name
			graph.Nodes[i].Source = source
			graph.Nodes[i].Inferred = inferred
			return
		}
	}
}

func appendFlowNode(graph *FlowGraph, node FlowNode) {
	for _, existing := range graph.Nodes {
		if existing.ID == node.ID {
			return
		}
	}
	graph.Nodes = append(graph.Nodes, node)
}

func appendFlowEdge(graph *FlowGraph, edge FlowEdge) {
	for _, existing := range graph.Edges {
		if existing.From == edge.From && existing.To == edge.To && existing.Kind == edge.Kind {
			return
		}
	}
	graph.Edges = append(graph.Edges, edge)
}

func removeUnknownOutbound(graph *FlowGraph, routeID string) {
	outboundID := routeID + flowUnknownOutboundID
	nodes := graph.Nodes[:0]
	for _, node := range graph.Nodes {
		if node.ID != outboundID {
			nodes = append(nodes, node)
		}
	}
	graph.Nodes = nodes
	edges := graph.Edges[:0]
	for _, edge := range graph.Edges {
		if edge.From != outboundID && edge.To != outboundID {
			edges = append(edges, edge)
		}
	}
	graph.Edges = edges
}

func chooseString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
