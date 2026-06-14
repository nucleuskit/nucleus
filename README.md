# Nucleus

[![CI](https://github.com/nucleuskit/nucleus/actions/workflows/ci.yml/badge.svg)](https://github.com/nucleuskit/nucleus/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Nucleus is an AI-first Go microservice kernel for contract-driven service generation, evolution, and verification.

It is designed for AI agents, CI systems, and human reviewers to work from the same source of truth: service contracts, manifests, generated freshness, safe edit surfaces, and verification evidence.

> Status: pre-alpha. The public repository is being prepared for the first source import under `github.com/nucleuskit/nucleus`.

## Why Nucleus

Traditional Go microservice frameworks are optimized for humans writing code inside an IDE. Nucleus is optimized for AI agents changing services safely and repeatably.

Nucleus focuses on four rules:

- **Contract-first**: OpenAPI, protobuf, and error definitions are the source of truth for external behavior.
- **Manifest-first**: service identity, capabilities, dependencies, and AI edit surfaces are declared in `nucleus.yaml`.
- **Thin kernel**: core lifecycle, identity, errors, context, and response types stay small and dependency-aware.
- **AI-safe loop**: `describe -> plan -> lint -> verify` makes service changes reviewable, reproducible, and bounded.

Nucleus is not a full-stack middleware bundle. Redis, Kafka, SQL, Nacos, tracing exporters, and similar infrastructure are connected through explicit capability interfaces and optional bridges.

## Core Concepts

### Contract

A Nucleus service treats these files as public behavior contracts:

- `api/openapi.yaml` for HTTP APIs
- `api/proto/*.proto` for gRPC APIs
- `api/errors.yaml` for stable error codes and HTTP mappings

Generated handlers, types, clients, and freshness metadata are derived from these contracts.

### Manifest

`nucleus.yaml` describes the service identity and machine-readable operating surface:

- service name, version, tier, and ownership
- declared capabilities and providers
- dependencies and contract snapshots
- edit surfaces for AI agents
- verification commands for CI and local review

### Capability Protocol

Nucleus separates capability declaration from infrastructure implementation:

- `github.com/nucleuskit/cap/*` defines small interfaces, options, and no-op implementations.
- `github.com/nucleuskit/bridge/*` provides optional adapters.
- Application wiring injects bridges into runtime code.
- Domain code stays independent from transport and infrastructure SDKs.

### AI-Safe Change Loop

The intended workflow inside a service directory is:

```text
nucleus describe --json
nucleus validate
nucleus plan --task "change request" --json
nucleus gen
nucleus lint
nucleus verify
```

The goal is not just to generate code. The goal is to produce evidence that the change respected contracts, manifests, generated freshness, dependency boundaries, and verification commands.

`describe` emits the service facts that agents and reviewers consume. `validate`
checks manifest and contract source legality before later workflow steps. `lint`
checks project conventions and risk rules, while `verify` executes the validation,
lint, build, and test evidence pipeline.

For the bundled example, run `nucleus validate --dir example/hello-http`.
Successful human output includes a short validation summary; `--json` emits the
same result with stable `ok`, `summary`, and `diagnostics` fields.

## Project Shape

The main repository hosts the CLI, examples, and public documentation:

```text
api/                 Contract files
cmd/nucleus/         CLI implementation
example/             Runnable examples
docs/                Concepts, ADRs, plans, and platform mapping
```

The kernel, contract, capability, bridge, and runtime packages are published as
separate Go modules:

- `github.com/nucleuskit/core`
- `github.com/nucleuskit/contract`
- `github.com/nucleuskit/cap`
- `github.com/nucleuskit/bridge`
- `github.com/nucleuskit/http`
- `github.com/nucleuskit/grpc`
- `github.com/nucleuskit/worker`

## Roadmap

The early public roadmap is intentionally narrow:

- Publish the source under `github.com/nucleuskit/nucleus`.
- Stabilize the first `nucleus validate`, `gen`, `describe`, `plan`, `lint`, and `verify` loop.
- Keep `core` standard-library-only.
- Make `examples/hello-http` the first runnable contract-first service.
- Add focused capability protocols before adding optional bridges.
- Publish `v0.1.0-alpha.1` once the public module path, CI, examples, and docs are aligned.

## Contributing

Nucleus is early, but contributions should already follow the project boundaries:

- Start with contracts and manifests when changing external behavior.
- Keep kernel code small and dependency-aware.
- Prefer explicit capabilities over hidden framework defaults.
- Add tests or verification evidence with behavior changes.
- Do not add compatibility shims for private or legacy internal SDKs.

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## License

Nucleus is licensed under the [Apache License 2.0](LICENSE).

## Star History

<a href="https://www.star-history.com/?repos=nucleuskit%2Fnucleus&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=nucleuskit/nucleus&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=nucleuskit/nucleus&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=nucleuskit/nucleus&type=date&legend=top-left" />
 </picture>
</a>
