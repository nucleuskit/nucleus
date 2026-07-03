# ADR 0002: Agent-Native Protocol Layer

## Status

Accepted.

## Context

The project is pre-release and has not been adopted as a production framework. Early implementation and documentation still contained scaffold, provider, bridge, and platform-readiness concepts that conflict with the desired direction.

AI agents need structured facts, graph context, edit boundaries, decisions, and evidence. They do not need Nucleus to choose an ORM, database driver, service registry, directory layout, or provider SDK.

## Decision

Nucleus will be implemented as an agent-native protocol layer that can be added to any Go project.

The primary workflow is:

```text
adopt -> describe graph -> trace -> impact -> plan -> decision -> verify
```

Nucleus owns:

- protocol schemas
- local manifest indexes
- source graph inspection
- edit-surface policy
- generated freshness metadata
- technical decision evidence
- local verification evidence
- local quality reports

Nucleus does not own:

- project scaffolding
- provider SDK adapters
- bridge modules
- ORM or driver choices
- application wiring
- repository implementations
- migrations
- platform upload or control-plane workflows

## Consequences

The pre-release codebase may remove incompatible commands and modules instead of preserving compatibility.

`nucleus.yaml` v2 becomes a thin protocol index. Provider, library, and driver choices live in `.nucleus/decisions/*.yaml`, not in the manifest.

Capability kinds are advisory vocabulary data, not Go-coded enums. Unknown kinds and unknown provider decisions are valid when their schema and evidence are complete.

`bridge` is removed from the core module boundary. Provider knowledge may exist as recipes or external examples, but recipes cannot execute commands or write files.
