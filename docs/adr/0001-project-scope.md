# ADR 0001: Project Scope

## Status

Accepted.

## Context

Nucleus needs a clear public boundary before the source is imported.

## Decision

Nucleus is an agent-native Go microservice protocol layer. It focuses on contract-driven behavior facts, manifest-driven protocol indexes, source graph inspection, decision evidence, edit-surface policy, and verifiable AI-safe change loops.

Nucleus will not become a default middleware bundle, a business domain framework, a platform UI, a provider SDK collection, a project scaffold, or a private SDK compatibility layer.

## Consequences

New capabilities must fit one of the project areas: `contract`, `cmd/nucleus`, `docs`, local protocol schemas, graph inspection, decision evidence, or narrowly scoped runtime adapters explicitly chosen by a user project.

Provider choices, library choices, drivers, directory layouts, repository implementations, migrations, Dockerfiles, and application wiring are not Nucleus-owned concerns.

Large scope changes require an ADR or design issue before implementation.
