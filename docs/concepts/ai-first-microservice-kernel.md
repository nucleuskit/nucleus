# AI-First Microservice Kernel

Nucleus treats an AI agent as a first-class service maintainer.

The project shape is designed so an agent can answer these questions before editing code:

- What public contracts define this service?
- Which generated files are stale?
- Which files are safe to edit?
- Which capabilities are declared and injected?
- Which commands must pass before the change is reviewable?

The framework is therefore centered on contracts, manifests, schemas, and evidence, rather than on hidden defaults or a large middleware bundle.
