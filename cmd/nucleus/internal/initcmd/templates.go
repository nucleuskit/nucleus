package initcmd

import (
	"fmt"
	"path/filepath"
	"strings"
)

func templateFiles(templateType string, name string, module string) (map[string]string, []string, error) {
	switch templateType {
	case templateService:
		return serviceTemplateFiles(name, module), []string{contractGenTarget, httpAdapterGenTarget}, nil
	case templateWorker:
		return workerTemplateFiles(name, module), nil, nil
	case templateLibrary:
		return libraryTemplateFiles(name, module), nil, nil
	default:
		return nil, nil, fmt.Errorf("unknown template %q", templateType)
	}
}

func serviceTemplateFiles(name string, module string) map[string]string {
	return map[string]string{
		"go.mod":                      goModTemplate(module, []moduleRequirement{{Path: nucleusHTTPModule, Version: nucleusHTTPVersion}}),
		"go.sum":                      goSumTemplate([]string{nucleusHTTPModule}),
		"nucleus.yaml":                serviceManifestTemplate(name),
		"api/openapi.yaml":            openAPITemplate(name),
		"api/errors.yaml":             serviceErrorsTemplate(),
		"configs/config.example.yaml": serviceConfigExampleTemplate(),
		"configs/README.md":           serviceConfigReadmeTemplate(),
		"docs/architecture.md":        serviceArchitectureDocTemplate(name),
		"docs/api.md":                 serviceAPIDocTemplate(),
		"deploy/Dockerfile":           serviceDockerfileTemplate(name),
		"test/integration/README.md":  serviceIntegrationReadmeTemplate(),
		"Makefile":                    serviceMakefileTemplate(name),
		filepath.ToSlash(filepath.Join("cmd", name, "main.go")): serviceMainTemplate(module),
		"internal/app/app.go":                         serviceAppTemplate(module),
		"internal/app/providers.go":                   serviceProvidersTemplate(module),
		"internal/app/routes.go":                      serviceRoutesTemplate(module),
		"internal/config/config.go":                   serviceConfigTemplate(),
		"internal/config/loader.go":                   serviceConfigLoaderTemplate(),
		"internal/config/validate.go":                 serviceConfigValidateTemplate(),
		"internal/domain/health/model.go":             serviceDomainHealthModelTemplate(),
		"internal/domain/health/service.go":           serviceDomainHealthServiceTemplate(),
		"internal/domain/health/repository.go":        serviceDomainHealthRepositoryTemplate(),
		"internal/domain/health/errors.go":            serviceDomainHealthErrorsTemplate(),
		"internal/domain/health/events.go":            serviceDomainHealthEventsTemplate(),
		"internal/usecase/health/usecase.go":          serviceUsecaseHealthTemplate(module),
		"internal/adapter/http/handler.go":            serviceHTTPHandlerTemplate(module),
		"internal/adapter/http/mapper.go":             serviceHTTPMapperTemplate(),
		"internal/adapter/http/validation.go":         serviceHTTPValidationTemplate(),
		"internal/adapter/http/errors.go":             serviceHTTPErrorsTemplate(),
		"internal/adapter/store/memory/repository.go": serviceMemoryRepositoryTemplate(module),
		"internal/component/clock/component.go":       serviceClockComponentTemplate(),
		"internal/server/http.go":                     serviceHTTPServerTemplate(),
	}
}

func workerTemplateFiles(name string, module string) map[string]string {
	return map[string]string{
		"go.mod":       goModTemplate(module, nil),
		"nucleus.yaml": workerManifestTemplate(name),
		filepath.ToSlash(filepath.Join("cmd", name, "main.go")): workerMainTemplate(module),
		"internal/worker/handler.go":                            workerHandlerTemplate(),
	}
}

func libraryTemplateFiles(name string, module string) map[string]string {
	packageName := packageNameFromServiceName(name)
	return map[string]string{
		"go.mod":            goModTemplate(module, nil),
		"nucleus.yaml":      libraryManifestTemplate(name),
		packageName + ".go": libraryTemplate(packageName, name),
	}
}

type moduleRequirement struct {
	Path    string
	Version string
}

func goModTemplate(module string, requires []moduleRequirement) string {
	var builder strings.Builder
	builder.WriteString("module " + module + "\n\n")
	builder.WriteString("go " + defaultGoVersion + "\n")
	if len(requires) == 0 {
		return builder.String()
	}
	if len(requires) == 1 {
		item := requires[0]
		builder.WriteString("\nrequire " + item.Path + " " + item.Version + "\n")
		return builder.String()
	}
	builder.WriteString("\nrequire (\n")
	for _, item := range requires {
		builder.WriteString("\t" + item.Path + " " + item.Version + "\n")
	}
	builder.WriteString(")\n")
	return builder.String()
}

func goSumTemplate(modules []string) string {
	entries := map[string]string{
		nucleusHTTPModule: `github.com/nucleuskit/http v0.1.0-alpha.1.0.20260615170339-225ca98f40d7 h1:wWoSiKv5HihOSOstvkov883PfSKLveAuW3tTZb9dH/Q=
github.com/nucleuskit/http v0.1.0-alpha.1.0.20260615170339-225ca98f40d7/go.mod h1:M4WW38dQuFNIm2kf1O5JIrat+KKZ3ONarUyrtbMDmJo=
`,
	}
	var builder strings.Builder
	for _, module := range modules {
		builder.WriteString(entries[module])
	}
	return builder.String()
}

func serviceErrorsTemplate() string {
	return `errors:
  - code: 1000
    message: internal error
    http_status: 500
`
}

func openAPITemplate(name string) string {
	return fmt.Sprintf(`openapi: 3.0.3
info:
  title: %s
  version: %s
paths:
  /healthz:
    get:
      operationId: getHealthz
      responses:
        "200":
          description: ok
`, name, defaultServiceVersion)
}

func serviceManifestTemplate(name string) string {
	return fmt.Sprintf(`schema_version: "1.0"

service:
  name: %s
  version: "%s"
  tier: internal
  description: "Generated Nucleus service"

ai:
  intent: "Generated by nucleus init for a contract-first HTTP service"
  allowed_changes:
    - nucleus.yaml
    - go.mod
    - go.sum
    - api/**
    - cmd/**
    - configs/**
    - deploy/**
    - docs/**
    - internal/app/**
    - internal/config/**
    - internal/domain/**
    - internal/usecase/**
    - internal/component/**
    - internal/adapter/http/**
    - internal/adapter/store/**
    - internal/server/**
    - test/**
    - Makefile
  readonly:
    - contract/gen/**
    - internal/adapter/http/gen/**
  forbidden:
    - configs/*.local.yaml
    - configs/*secret*.yaml
    - bridge/legacy/**
  generated:
    - contract/gen
    - internal/adapter/http/gen

capabilities:
  - http
`, name, defaultServiceVersion)
}

func workerManifestTemplate(name string) string {
	return fmt.Sprintf(`schema_version: "1.0"

service:
  name: %s
  version: "%s"
  tier: internal
  description: "Generated Nucleus worker"

ai:
  intent: "Generated by nucleus init for an explicit worker entrypoint"
  allowed_changes:
    - nucleus.yaml
    - go.mod
    - cmd/**
    - internal/**
  readonly: []
  forbidden:
    - configs/*.local.yaml
    - configs/*secret*.yaml

nucleus:
  providers:
    worker:
      provider: local

capabilities:
  - worker
`, name, defaultServiceVersion)
}

func libraryManifestTemplate(name string) string {
	return fmt.Sprintf(`schema_version: "1.0"

service:
  name: %s
  version: "%s"
  tier: internal
  description: "Generated Nucleus library"

ai:
  intent: "Generated by nucleus init for a small reusable Go library"
  allowed_changes:
    - nucleus.yaml
    - go.mod
    - "*.go"
  readonly: []
  forbidden: []

capabilities: []
`, name, defaultServiceVersion)
}

func serviceMainTemplate(module string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"%s/internal/app"
	"%s/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	application := app.New(cfg)
	if err := application.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
`, module, module)
}

func serviceAppTemplate(module string) string {
	return fmt.Sprintf(`package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	runtimehttp "github.com/nucleuskit/http"

	"%s/internal/config"
	serverpkg "%s/internal/server"
)

type App struct {
	config config.Config
	router *runtimehttp.Server
	server *http.Server
}

func New(cfg config.Config) *App {
	router := runtimehttp.NewServer()
	handler := NewHTTPHandler(NewHealthUsecase())
	RegisterHTTPRoutes(router, handler)
	return &App{
		config: cfg,
		router: router,
		server: serverpkg.NewHTTPServer(cfg.Server.Addr, router),
	}
}

func (a *App) Run(ctx context.Context) error {
	errc := make(chan error, 1)
	go func() {
		errc <- a.server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return a.Shutdown(shutdownCtx)
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
`, module, module)
}

func serviceProvidersTemplate(module string) string {
	return fmt.Sprintf(`package app

import (
	httpadapter "%s/internal/adapter/http"
	domainhealth "%s/internal/domain/health"
	healthusecase "%s/internal/usecase/health"
)

func NewHealthUsecase() *healthusecase.Usecase {
	return healthusecase.NewUsecase(domainhealth.NewService())
}

func NewHTTPHandler(health *healthusecase.Usecase) *httpadapter.Handler {
	return httpadapter.NewHandler(health)
}
`, module, module, module)
}

func serviceRoutesTemplate(module string) string {
	return fmt.Sprintf(`package app

import (
	runtimehttp "github.com/nucleuskit/http"

	httpadapter "%s/internal/adapter/http"
	httpgen "%s/internal/adapter/http/gen"
)

func RegisterHTTPRoutes(server *runtimehttp.Server, handler *httpadapter.Handler) {
	httpgen.RegisterRoutes(server, handler)
}
`, module, module)
}

func serviceConfigTemplate() string {
	return `package config

type Config struct {
	Server ServerConfig ` + "`json:\"server\" yaml:\"server\"`" + `
}

type ServerConfig struct {
	Addr string ` + "`json:\"addr\" yaml:\"addr\"`" + `
}

func Default() Config {
	return Config{Server: ServerConfig{Addr: "` + defaultHTTPAddress + `"}}
}
`
}

func serviceConfigLoaderTemplate() string {
	return `package config

import "os"

func Load() (Config, error) {
	cfg := Default()
	if value := os.Getenv("NUCLEUS_HTTP_ADDR"); value != "" {
		cfg.Server.Addr = value
	}
	return cfg, cfg.Validate()
}
`
}

func serviceConfigValidateTemplate() string {
	return `package config

import "fmt"

func (c Config) Validate() error {
	if c.Server.Addr == "" {
		return fmt.Errorf("server.addr is required")
	}
	return nil
}
`
}

func serviceDomainHealthModelTemplate() string {
	return `package health

type Status struct {
	Status string ` + "`json:\"status\"`" + `
}
`
}

func serviceDomainHealthServiceTemplate() string {
	return `package health

import "context"

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) Check(context.Context) (Status, error) {
	return Status{Status: "ok"}, nil
}
`
}

func serviceDomainHealthRepositoryTemplate() string {
	return `package health

import "context"

type Repository interface {
	Status(context.Context) (Status, error)
}
`
}

func serviceDomainHealthErrorsTemplate() string {
	return `package health

const ErrUnavailable = "health unavailable"
`
}

func serviceDomainHealthEventsTemplate() string {
	return `package health

type CheckedEvent struct {
	Status Status
}
`
}

func serviceUsecaseHealthTemplate(module string) string {
	return fmt.Sprintf(`package health

import (
	"context"

	domainhealth "%s/internal/domain/health"
)

type Usecase struct {
	service domainhealth.Service
}

func NewUsecase(service domainhealth.Service) *Usecase {
	return &Usecase{service: service}
}

func (u *Usecase) Get(ctx context.Context) (domainhealth.Status, error) {
	return u.service.Check(ctx)
}
`, module)
}

func serviceHTTPHandlerTemplate(module string) string {
	return fmt.Sprintf(`package http

import (
	"net/http"

	healthusecase "%s/internal/usecase/health"
)

type Handler struct {
	health *healthusecase.Usecase
}

func NewHandler(health *healthusecase.Usecase) *Handler {
	return &Handler{health: health}
}

func (h *Handler) GetHealthz(request *http.Request) (any, error) {
	return h.health.Get(request.Context())
}
`, module)
}

func serviceHTTPMapperTemplate() string {
	return `package http

type HealthResponse struct {
	Status string ` + "`json:\"status\"`" + `
}
`
}

func serviceHTTPValidationTemplate() string {
	return `package http

func validateHealthRequest() error {
	return nil
}
`
}

func serviceHTTPErrorsTemplate() string {
	return `package http

func mapError(err error) error {
	return err
}
`
}

func serviceMemoryRepositoryTemplate(module string) string {
	return fmt.Sprintf(`package memory

import (
	"context"

	domainhealth "%s/internal/domain/health"
)

type HealthRepository struct{}

func NewHealthRepository() *HealthRepository {
	return &HealthRepository{}
}

func (HealthRepository) Status(context.Context) (domainhealth.Status, error) {
	return domainhealth.Status{Status: "ok"}, nil
}
`, module)
}

func serviceClockComponentTemplate() string {
	return `package clock

import "time"

type Clock struct{}

func (Clock) Now() time.Time {
	return time.Now()
}
`
}

func serviceHTTPServerTemplate() string {
	return `package server

import "net/http"

func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,
	}
}
`
}

func serviceConfigExampleTemplate() string {
	return `server:
  addr: ${NUCLEUS_HTTP_ADDR:-` + defaultHTTPAddress + `}
`
}

func serviceConfigReadmeTemplate() string {
	return `# Config

` + "`configs/config.example.yaml`" + ` is safe to commit and must only contain placeholders.
Local files such as ` + "`configs/config.local.yaml`" + ` are forbidden by ` + "`nucleus.yaml`" + ` edit surfaces.
`
}

func serviceArchitectureDocTemplate(name string) string {
	return fmt.Sprintf(`# %s Architecture

This service is generated by Nucleus.

- Contracts live in `+"`api/`"+`.
- Handwritten business code lives in `+"`internal/domain`"+` and `+"`internal/usecase`"+`.
- Protocol adapters live in `+"`internal/adapter`"+`.
- Generated HTTP glue lives in `+"`internal/adapter/http/gen`"+`.
`, name)
}

func serviceAPIDocTemplate() string {
	return `# API

The HTTP API contract is ` + "`api/openapi.yaml`" + `. Run ` + "`nucleus gen --dir .`" + ` after changing contracts.
`
}

func serviceDockerfileTemplate(name string) string {
	return fmt.Sprintf(`FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN go build -o /out/%s ./cmd/%s

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/%s /%s
ENTRYPOINT ["/%s"]
`, name, name, name, name, name)
}

func serviceIntegrationReadmeTemplate() string {
	return `# Integration Tests

Put external-dependency tests here and guard them with explicit build tags.
`
}

func serviceMakefileTemplate(name string) string {
	return fmt.Sprintf(`SERVICE := %s

.PHONY: run gen lint verify test

run:
	go run ./cmd/$(SERVICE)

gen:
	nucleus gen --dir .

lint:
	nucleus lint --dir . --strict

verify:
	nucleus verify --dir . --json

test:
	go test ./...
`, name)
}

func workerMainTemplate(module string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"log"

	localworker "%s/internal/worker"
)

func main() {
	handler := localworker.HandlerFunc(func(context.Context, localworker.Message) error {
		return nil
	})
	if err := handler.Handle(context.Background(), localworker.Message{ID: "startup", Name: "startup"}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("nucleus worker initialized")
}
`, module)
}

func workerHandlerTemplate() string {
	return `package worker

import (
	"context"
	"errors"
)

// ErrNilHandler is returned when a nil HandlerFunc is invoked.
var ErrNilHandler = errors.New("worker handler is nil")

// Message is the transport-neutral input delivered to a worker handler.
type Message struct {
	ID         string
	Name       string
	Payload    []byte
	Attributes map[string]string
}

// Handler processes one worker message.
type Handler interface {
	Handle(context.Context, Message) error
}

// HandlerFunc adapts a function to the Handler interface.
type HandlerFunc func(context.Context, Message) error

// Handle processes one message or returns ErrNilHandler for a nil function.
func (fn HandlerFunc) Handle(ctx context.Context, message Message) error {
	if fn == nil {
		return ErrNilHandler
	}
	return fn(ctx, message)
}
`
}

func libraryTemplate(packageName string, name string) string {
	return fmt.Sprintf(`package %s

func Name() string {
	return %q
}
`, packageName, name)
}

func packageNameFromServiceName(name string) string {
	return strings.NewReplacer("-", "_", ".", "_").Replace(name)
}
