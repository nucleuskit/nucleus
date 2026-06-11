package app

import "net/http"

type Router struct{}

func (Router) Handle(method string, path string, handler func()) {}

type Logger struct{}

func (Logger) Info() {}

var log Logger

func RegisterRoutes(router Router) {
	router.Handle(http.MethodGet, "/hello/{name}", func() {
		log.Info()
	})
}
