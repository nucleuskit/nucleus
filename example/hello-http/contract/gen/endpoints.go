package gen

// Endpoint is generated from OpenAPI.
type Endpoint struct {
	Method      string
	Path        string
	OperationID string
}

// Endpoints lists generated HTTP endpoint metadata.
var Endpoints = []Endpoint{
	{Method: "GET", Path: "/hello/{name}", OperationID: "get_hello"},
}
