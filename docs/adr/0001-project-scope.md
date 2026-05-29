# ADR 0001: Project Scope

## Status

Accepted.

## Context

Nucleus needs a clear public boundary before the source is imported.

## Decision

Nucleus is an AI-first Go microservice kernel. It focuses on contract-driven service generation, manifest-driven metadata, capability protocols, thin runtime assembly, and verifiable AI-safe change loops.

Nucleus will not become a default middleware bundle, a business domain framework, a platform UI, or a private SDK compatibility layer.

## Consequences

New capabilities must fit one of the project areas: `core`, `contract`, `cap`, `bridge`, `runtime`, `cmd/nucleus`, `examples`, `template`, or `docs`.

Large scope changes require an ADR or design issue before implementation.
