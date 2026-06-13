package openapi

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Route describes an HTTP operation with request metadata used by inspection.
type Route struct {
	Method              string      `json:"method"`
	Path                string      `json:"path"`
	OperationID         string      `json:"operation_id,omitempty"`
	Parameters          []Parameter `json:"parameters,omitempty"`
	RequestBodyRequired bool        `json:"request_body_required,omitempty"`
	Priority            int         `json:"priority,omitempty"`
}

// Parameter describes an OpenAPI operation parameter.
type Parameter struct {
	Name       string `json:"name"`
	In         string `json:"in"`
	Required   bool   `json:"required,omitempty"`
	SchemaType string `json:"schema_type,omitempty"`
}

type routeDocument struct {
	Paths map[string]routePathItem `yaml:"paths"`
}

type routePathItem struct {
	Parameters []routeParameter `yaml:"parameters"`
	Get        *routeOperation  `yaml:"get"`
	Post       *routeOperation  `yaml:"post"`
	Put        *routeOperation  `yaml:"put"`
	Patch      *routeOperation  `yaml:"patch"`
	Delete     *routeOperation  `yaml:"delete"`
	Head       *routeOperation  `yaml:"head"`
	Options    *routeOperation  `yaml:"options"`
}

type routeOperation struct {
	OperationID string           `yaml:"operationId"`
	Parameters  []routeParameter `yaml:"parameters"`
	RequestBody routeRequestBody `yaml:"requestBody"`
	Responses   map[string]any   `yaml:"responses"`
	Priority    int              `yaml:"x-nucleus-priority"`
}

type routeParameter struct {
	Name     string      `yaml:"name"`
	In       string      `yaml:"in"`
	Required bool        `yaml:"required"`
	Schema   routeSchema `yaml:"schema"`
}

type routeSchema struct {
	Type string `yaml:"type"`
}

type routeRequestBody struct {
	Required bool `yaml:"required"`
}

// LoadRouteRegistry loads HTTP routes from api/openapi.yaml.
func LoadRouteRegistry(dir string) ([]Route, error) {
	path := filepath.Join(dir, apiDirName, openAPIFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var doc routeDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	var routes []Route
	for path, item := range doc.Paths {
		pathParameters := convertParameters(item.Parameters)
		for method, operation := range item.operations() {
			parameters := append([]Parameter{}, pathParameters...)
			parameters = append(parameters, convertParameters(operation.Parameters)...)
			routes = append(routes, Route{
				Method:              strings.ToUpper(method),
				Path:                path,
				OperationID:         operation.OperationID,
				Parameters:          parameters,
				RequestBodyRequired: operation.RequestBody.Required,
				Priority:            operation.Priority,
			})
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	return routes, nil
}

func (item routePathItem) operations() map[string]routeOperation {
	candidates := map[string]*routeOperation{
		methodGet:     item.Get,
		methodPost:    item.Post,
		methodPut:     item.Put,
		methodPatch:   item.Patch,
		methodDelete:  item.Delete,
		methodHead:    item.Head,
		methodOptions: item.Options,
	}
	operations := make(map[string]routeOperation, len(candidates))
	for method, operation := range candidates {
		if operation != nil {
			operations[method] = *operation
		}
	}
	return operations
}

func convertParameters(parameters []routeParameter) []Parameter {
	converted := make([]Parameter, 0, len(parameters))
	for _, parameter := range parameters {
		converted = append(converted, Parameter{
			Name:       parameter.Name,
			In:         parameter.In,
			Required:   parameter.Required,
			SchemaType: parameter.Schema.Type,
		})
	}
	return converted
}
