package gen

import (
	"net/http"

	runtimehttp "github.com/nucleuskit/http"
)

// RegisterRoutes binds generated OpenAPI routes to a handwritten handler.
func RegisterRoutes(server *runtimehttp.Server, handler Handler) {
	if server == nil || handler == nil {
		return
	}
	server.RegisterRoutes([]runtimehttp.Route{
		{Method: "GET", Path: "/hello/{name}", OperationID: "get_hello", Handler: func(request *http.Request) (any, error) {
			return handler.GetHello(request)
		}},
	})
}
