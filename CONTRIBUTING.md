# Contributing to Nucleus

Thanks for helping improve Nucleus.

Nucleus is an AI-first Go microservice kernel. Contributions should preserve the project boundary: contract-driven service generation, manifest-driven metadata, thin runtime assembly, explicit capability protocols, and verifiable AI-safe changes.

## Project Status

Nucleus is pre-alpha. Public APIs, CLI output schemas, examples, and module layout may change before `v1.0.0`.

Before starting a large change, open an issue or discussion so maintainers can align on the intended contract, compatibility impact, and verification plan.

## Development Principles

- Change public HTTP or gRPC behavior contract-first.
- Keep `core` standard-library-only.
- Do not introduce default middleware bundles.
- Add new infrastructure through `cap/*` interfaces and optional `bridge/*` implementations.
- Keep runtime packages independent from bridge packages.
- Keep generated code fresh and reproducible.
- Include tests, command output, or verification evidence for behavior changes.

## Contribution Workflow

1. Fork the repository.
2. Create a topic branch from `main`.
3. Make focused changes.
4. Run the relevant verification commands.
5. Open a pull request using the PR template.

Recommended commands once the source tree is available:

```bash
go test ./...
go test ./... -race -count=1
go run ./cmd/nucleus validate --dir .
go run ./cmd/nucleus lint --dir .
go run ./cmd/nucleus verify --dir . --json
```

If you change contracts, generated code, schema files, CLI flags, or directory layout, update the related docs in the same pull request.

## Pull Request Expectations

Every pull request should explain:

- what changed
- why it is needed
- what contracts or manifests were affected
- what verification was run
- what risks remain

Small pull requests are easier to review and merge.

## Commit Style

Use short, descriptive commit messages:

```text
feat: add manifest capability lint
fix: preserve generated freshness hash
docs: clarify contract-first workflow
```

## Security

Do not report vulnerabilities in public issues. Follow [SECURITY.md](SECURITY.md).
