# Implementation Status

This document records what the current checkout actually implements. It is meant
to keep the public positioning honest: Nucleus is an AI-first microservice
kernel, not a production-complete replacement for go-zero, Kratos, go-kit, or a
full middleware platform.

## Status Key

| Status | Meaning |
| --- | --- |
| Implemented | Code exists, has tests, and is expected to work in the current checkout. |
| Published alpha | Code is available through published alpha module tags without local `replace` directives. |
| Scaffold | The CLI generates service-owned code or metadata, but does not wire a real provider automatically. |
| Fake/static | Useful for local tests, examples, or metadata checks; not a real external provider integration. |
| Metadata-only | Produces inspection or readiness evidence without starting business handlers or provider SDKs. |

## CLI Surface

| Command | Current status | Notes |
| --- | --- | --- |
| `nucleus init` | Implemented | Generates service, worker, and library templates. The service template is covered by tests that run validation, strict lint, and `go test ./...`. |
| `nucleus validate` | Implemented | Validates `nucleus.yaml`, `api/openapi.yaml`, `api/errors.yaml`, and lightweight proto facts. |
| `nucleus gen` | Implemented | Generates contract metadata, HTTP route binders, gRPC metadata, error metadata, docs, TypeScript schema exports, and minimal clients. |
| `nucleus lint` | Implemented | Checks architecture boundaries, route registration, capability graph, generated freshness, schema versions, and selected legacy imports. |
| `nucleus verify` | Implemented | Runs validate, strict lint, generated freshness, `go mod tidy`, import, build, and tests as evidence steps. |
| `nucleus describe` | Implemented | Emits manifest, contract, capability, module, edit-surface, freshness, and verification facts. |
| `nucleus plan` | Implemented | Produces task-oriented edit surfaces, commands, and risk notes. Natural-language classification is heuristic. |
| `nucleus apply` | Implemented | Applies plan file edits only after edit-surface, path traversal, and symlink checks. It deliberately does not execute shell commands. |
| `nucleus repair` | Implemented | Supports bounded repair for generated freshness and single explicit patch candidates with expected hashes. It is not a general code-repair engine. |
| `nucleus scenario` | Implemented | Builds OpenAPI/error-derived scenario suggestions and can run explicit HTTP scenario checks when a base URL or handler is provided. |
| `nucleus serve` | Metadata-only | Serves `/healthz`, `/readyz`, and `/.well-known/nucleus.json` metadata. It does not start generated business handlers. |
| `nucleus report` | Metadata-only | Summarizes local AI task quality and platform-readiness metadata without network calls. |
| `nucleus migrate` | Metadata-only | Produces migration checklists and readiness checks. It does not rewrite services. |
| `nucleus capability add` | Scaffold | Updates manifest metadata and generates service-owned component/app/docs scaffolds. Most providers still need explicit bridge wiring by the service. |

## Contract and Manifest Layer

| Area | Current status | Notes |
| --- | --- | --- |
| Manifest parsing | Implemented | Uses structured YAML parsing and validates required service metadata, capability names, dependencies, and AI edit surfaces. |
| OpenAPI inspection | Implemented | Loads route metadata, parameters, request-body facts, examples, and validation hints from `api/openapi.yaml`. It is a lightweight metadata parser, not a complete OpenAPI toolchain. |
| Proto inspection | Implemented | Extracts package, service, method, streaming, and `google.api.http` facts with a lightweight parser. It is not a replacement for `protoc`. |
| Error catalog | Implemented | Validates stable error codes, messages, and HTTP status mappings from `api/errors.yaml`. |
| Generated freshness | Implemented | Compares contract source hashes with generated target markers declared in `ai.generated`. |

## Runtime Modules

| Module | Current status | Notes |
| --- | --- | --- |
| `github.com/nucleuskit/http` | Published alpha | HTTP server, route registration, response envelopes, middleware, clients, CORS, SSE, static assets, and well-known metadata are implemented and tested. |
| `github.com/nucleuskit/grpc` | Published alpha | gRPC server/client wrappers, discovery resolver support, health/reflection options, and interceptor chains are implemented and tested. |
| `github.com/nucleuskit/worker` | Published alpha | Cron, interval, batch, consumer, stream, map-reduce, timing-wheel, hooks, and manager primitives are implemented and tested. |

The runtime modules use canonical imports such as `github.com/nucleuskit/cap`
and `github.com/nucleuskit/core`, and the current alpha tags are verified
without local replacements.

## Capabilities and Bridges

| Area | Current status | Notes |
| --- | --- | --- |
| `github.com/nucleuskit/cap` | Implemented | Small capability interfaces, options, no-op implementations, and tests exist for auth, config, discovery, health, HTTP client, KV, lock, log, metric, Mongo, MQ, profiler, Redis, Sentinel, SQL, store, trace, and transport. |
| In-memory/test bridges | Fake/static | `memory`, `memorylock`, `redis`, `kv`, and several health/config helpers are useful for tests and local examples. They should not be presented as external managed services. |
| Real provider bridges | Implemented | `goredis`, `sarama`, `sql`, `gorm`, `zap`, `otel`, `prometheus`, `redislock`, and `nacosofficial` contain real provider-facing adapter code and tests. |
| Static provider bridges | Fake/static | `nacos` provides local/static config and discovery behavior. It is not the official Nacos SDK integration. |
| Provider scaffolds | Scaffold | `capability add` records provider intent and generates service-owned compile-safe code. Except for the focused PostgreSQL scaffold, it does not automatically import or wire provider SDKs. |

## Examples

The checked-in `examples/service`, `examples/worker`, and `examples/library`
directories describe templates rather than hosting full generated services. To
inspect a runnable service today, generate one with:

```bash
nucleus init --template service --name demo-api --module example.com/demo-api --dir ./demo-api
cd ./demo-api
nucleus validate --dir .
nucleus lint --dir . --strict
nucleus verify --dir . --json
go test ./...
```

## Adoption Guidance

Use the current checkout as an AI-safe contract, manifest, generation, and
evidence loop. Treat it as a production framework replacement only after more
real services have validated the runtime modules, bridge modules, examples, and
upgrade path under production-like constraints.
