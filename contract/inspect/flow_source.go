package inspect

import (
	"strings"

	"github.com/nucleuskit/contract/openapi"
)

type sourceFunction struct {
	Name           string
	Source         string
	Params         []string
	Calls          []string
	Domain         bool
	UsesHTTPClient bool
}

type sourceRouteHandler struct {
	Method      string
	Path        string
	OperationID string
	Name        string
	Source      string
	UsesLog     bool
}

func enrichFlowGraphFromSource(dir string, routes []openapi.Route, graph *FlowGraph) {
	functions := collectSourceFunctions(dir)
	handlers := collectRuntimeHTTPHandlers(dir, routes)
	if len(functions) == 0 {
		enrichFlowGraphFromRuntimeHandlers(routes, handlers, graph)
		return
	}
	byName := map[string]sourceFunction{}
	for _, function := range functions {
		if _, exists := byName[function.Name]; !exists {
			byName[function.Name] = function
		}
	}
	for _, route := range routes {
		endpoint := openapi.Endpoint{Method: route.Method, Path: route.Path, OperationID: route.OperationID}
		handlerName := exportedOperationName(route.OperationID)
		if handlerName == "" {
			continue
		}
		handler, ok := byName[handlerName]
		if !ok {
			continue
		}
		handlerID := flowOperationID(endpoint)
		updateFlowNode(graph, handlerID, handler.Name, handler.Source, true)
		for i := range graph.Params {
			if graph.Params[i].OperationID == route.OperationID {
				target := paramTarget(handler, strings.TrimPrefix(graph.Params[i].Name, flowParamPrefixPath))
				if target == "" {
					target = paramTarget(handler, strings.TrimPrefix(graph.Params[i].Name, flowParamPrefixQuery))
				}
				if target == "" {
					target = paramTarget(handler, strings.TrimPrefix(graph.Params[i].Name, flowParamPrefixHeader))
				}
				if target != "" {
					graph.Params[i].Target = target
					graph.Params[i].Confidence = flowSourceConfidence
					graph.Params[i].Inferred = true
				}
			}
		}
		for _, call := range handler.Calls {
			callee, ok := byName[call]
			if !ok || !callee.Domain {
				continue
			}
			domainID := flowIDPrefixDomain + callee.Name
			appendFlowNode(graph, FlowNode{
				ID:          domainID,
				Kind:        flowNodeKindDomain,
				Name:        callee.Name,
				Method:      route.Method,
				Path:        route.Path,
				OperationID: route.OperationID,
				Source:      callee.Source,
				Inferred:    true,
			})
			appendFlowEdge(graph, FlowEdge{From: handlerID, To: domainID, Kind: flowEdgeKindCall, Inferred: true})
			if callee.UsesHTTPClient {
				outboundID := domainID + flowHTTPClientOutbound
				appendFlowNode(graph, FlowNode{
					ID:          outboundID,
					Kind:        flowNodeKindOutbound,
					Name:        flowHTTPClientDoName,
					Method:      route.Method,
					Path:        route.Path,
					OperationID: route.OperationID,
					Source:      callee.Source,
					Inferred:    true,
				})
				appendFlowEdge(graph, FlowEdge{From: domainID, To: outboundID, Kind: flowEdgeKindOutboundCall, Inferred: true})
			}
		}
	}
	enrichFlowGraphFromRuntimeHandlers(routes, handlers, graph)
}

func enrichFlowGraphFromRuntimeHandlers(routes []openapi.Route, handlers []sourceRouteHandler, graph *FlowGraph) {
	if len(handlers) == 0 {
		return
	}
	byRoute := map[string]sourceRouteHandler{}
	for _, handler := range handlers {
		byRoute[handler.Method+routeKeySeparator+handler.Path] = handler
	}
	for _, route := range routes {
		handler, ok := byRoute[route.Method+routeKeySeparator+route.Path]
		if !ok {
			continue
		}
		endpoint := openapi.Endpoint{Method: route.Method, Path: route.Path, OperationID: route.OperationID}
		handlerID := flowOperationID(endpoint)
		updateFlowNode(graph, handlerID, handler.Name, handler.Source, true)
		removeUnknownOutbound(graph, flowRouteID(endpoint))
		responseID := handlerID + flowIDPartResponse + "envelope"
		appendFlowNode(graph, FlowNode{
			ID:          responseID,
			Kind:        flowNodeKindResponse,
			Name:        responseEnvelopeName,
			Method:      route.Method,
			Path:        route.Path,
			OperationID: route.OperationID,
			Source:      runtimeHTTPServerSource,
			Inferred:    false,
		})
		appendFlowEdge(graph, FlowEdge{From: handlerID, To: responseID, Kind: flowEdgeKindEncodesResponse, Inferred: false})
		if handler.UsesLog {
			capID := handlerID + flowIDPartCapability + capabilityLog
			appendFlowNode(graph, FlowNode{
				ID:          capID,
				Kind:        flowNodeKindCapability,
				Name:        capabilityLog,
				Method:      route.Method,
				Path:        route.Path,
				OperationID: route.OperationID,
				Source:      handler.Source,
				Inferred:    true,
			})
			appendFlowEdge(graph, FlowEdge{From: handlerID, To: capID, Kind: flowEdgeKindUsesCapability, Inferred: true})
		}
	}
}
