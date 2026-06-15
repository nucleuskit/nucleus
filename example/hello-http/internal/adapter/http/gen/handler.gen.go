package gen

import "net/http"

// Handler is implemented by the handwritten HTTP adapter.
type Handler interface {
	// GetHello handles the get_hello operation.
	GetHello(request *http.Request) (any, error)
}
