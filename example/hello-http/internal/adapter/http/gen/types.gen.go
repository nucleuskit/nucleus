package gen

// RouteParameter is generated OpenAPI route parameter metadata.
type RouteParameter struct {
	Name       string
	In         string
	Required   bool
	SchemaType string
}

// ServerRoute is generated HTTP route metadata for adapter registration.
type ServerRoute struct {
	Method              string
	Path                string
	OperationID         string
	HandlerName         string
	Parameters          []RouteParameter
	RequestBodyRequired bool
	Priority            int
}

// ServerRoutes lists generated HTTP server route metadata.
var ServerRoutes = []ServerRoute{
	{Method: "GET", Path: "/hello/{name}", OperationID: "get_hello", HandlerName: "GetHello", Parameters: []RouteParameter{{Name: "name", In: "path", Required: true, SchemaType: "string"}, {Name: "trace_id", In: "header", Required: false, SchemaType: "string"}}, RequestBodyRequired: false, Priority: 10},
}
