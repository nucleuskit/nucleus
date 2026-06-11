package openapi

// Endpoint is the minimal HTTP operation view exposed in service descriptions.
type Endpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	OperationID string `json:"operation_id,omitempty"`
}

// LoadEndpoints loads the minimal endpoint view from the OpenAPI route registry.
func LoadEndpoints(dir string) ([]Endpoint, error) {
	routes, err := LoadRouteRegistry(dir)
	if err != nil {
		return nil, err
	}
	endpoints := make([]Endpoint, 0, len(routes))
	for _, route := range routes {
		endpoints = append(endpoints, Endpoint{
			Method:      route.Method,
			Path:        route.Path,
			OperationID: route.OperationID,
		})
	}
	return endpoints, nil
}
