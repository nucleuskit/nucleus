# AI-First Microservice Kernel

Nucleus treats an AI agent as a first-class service maintainer.

The project shape is designed so an agent can answer these questions before editing code:

- What public contracts define this service?
- Which generated files are stale?
- Which files are safe to edit?
- Which capabilities are declared and injected?
- Which commands must pass before the change is reviewable?

The framework is therefore centered on contracts, manifests, schemas, and evidence, rather than on hidden defaults or a large middleware bundle.

## Scenario Evidence

`nucleus scenario` derives reviewable scenario plans from OpenAPI routes, error catalogs, and inspection flow facts. The default command produces suggestions only; it does not start services, inject providers, or make network calls.

HTTP scenario execution is explicit evidence. A user or CI job must provide `--run-http --base-url ...` or `--cases ... --base-url ...`; the command then records request/response samples, assertion results, truncation metadata, and redacted headers. This keeps scenario checks aligned with the AI-safe loop without turning Nucleus into a hidden runtime harness or provider auto-wiring layer.
