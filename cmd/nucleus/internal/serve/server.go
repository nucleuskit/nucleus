package serve

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/nucleuskit/contract/inspect"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 30 * time.Second
	shutdownTimeout   = 5 * time.Second
)

type listener interface {
	net.Listener
}

func newHandler(description inspect.Description) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(pathHealthz, getOnly(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = writer.Write([]byte("OK\n"))
	}))
	mux.HandleFunc(pathReadyz, getOnly(func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, readyPayload(description))
	}))
	mux.HandleFunc(pathWellKnown, getOnly(func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, description)
	}))
	return mux
}

func listen(addr string) (listener, error) {
	return net.Listen("tcp", addr)
}

func serveListener(ctx context.Context, listener listener, handler http.Handler) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	errc := make(chan error, 1)
	go func() {
		errc <- server.Serve(listener)
	}()
	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errc
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func closeListener(listener listener) {
	if listener != nil {
		_ = listener.Close()
	}
}

func getOnly(handler http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.Header().Set("Allow", http.MethodGet)
			http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		handler(writer, request)
	}
}

func readyPayload(description inspect.Description) map[string]any {
	return map[string]any{
		"status":        "ready",
		"service":       description.Service.Name,
		"version":       description.Service.Version,
		"capabilities":  description.Capabilities,
		"endpoints":     len(description.Endpoints),
		"grpc_services": len(description.GRPCServices),
	}
}

func metadataEndpoints() []metadataEndpoint {
	return []metadataEndpoint{
		{Method: http.MethodGet, Path: pathHealthz, ContentType: "text/plain"},
		{Method: http.MethodGet, Path: pathReadyz, ContentType: "application/json"},
		{Method: http.MethodGet, Path: pathWellKnown, ContentType: "application/json"},
	}
}

func writeJSON(writer http.ResponseWriter, value any) {
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
