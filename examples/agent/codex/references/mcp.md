# Nucleus MCP Reference

Nucleus MCP is a local stdio tool layer over CLI/schema facts. Use it for quick structured access to service facts, not as a separate source of truth.

Start it from a service root:

```bash
nucleus --dir . mcp --stdio
```

The server uses Content-Length framed JSON-RPC and currently supports `initialize`, `ping`, `tools/list`, and `tools/call`.

## Preferred Order

1. Use CLI/schema commands for durable evidence:

   ```bash
   nucleus describe --dir . --json --pretty
   nucleus plan --dir . --task "<task>" --json --executable
   nucleus verify --dir . --json
   ```

2. Use MCP tools for targeted local facts:
   - `get_service_description`
   - `get_edit_surfaces`
   - `get_contracts`
   - `get_capabilities`
   - `trace_route`
   - `trace_symbol`
   - `trace_capability`
   - `impact_symbol`
   - `impact_file`
   - `impact_contract`
   - `find_symbol`
   - `list_callers`
   - `list_callees`
   - `validate_decision`
   - `list_decisions`
   - `get_report`
   - `build_plan`
   - `list_recipes`
   - `get_recipe`

3. Return to CLI commands for generation, lint, verification, repair, execution, and report artifacts.

## Result Envelope

Every MCP `structuredContent` payload must be self-describing:

- `result_kind`
- `schema_version`
- `schema_ref`
- `ok`
- `diagnostics`

Tools that mirror a CLI command use that CLI result schema. MCP-only aggregate tools use `contract/schema/mcp-result.v1.schema.json`; recipe views use `contract/schema/recipe-result.v1.schema.json`.

## Capability Decisions

Use MCP capability or decision tools only as structured views over local facts. Capability tools can expose semantic kinds and existing symbols, but must not choose providers or scaffold implementation code.

Read these fields when available:

- declared capability `id`, `kind`, `intent`, and symbols
- decision evidence path and validation status
- affected graph nodes and edges
- allowed edit surfaces and blocked edits
- recipe candidates as reference knowledge only

## Recipe Boundary

Recipes are read-only knowledge from project `.nucleus/recipes/*.yaml`, `.yml`, `.json`, or built-in data-only recipes. Project recipes override built-ins with the same id.

Recipe tools must not:

- execute commands
- write files
- modify `nucleus.yaml`
- modify `.nucleus/decisions`
- modify `go.mod` or `go.sum`
- produce accepted or locked decisions

Unknown recipe fields such as executable `commands` should fail validation. Suggested verification belongs under `suggest.verification`.

## Consistency Rule

If MCP and CLI/schema disagree, trust CLI/schema and record the mismatch in the handoff.

## Safety Rule

MCP tools must not bypass:

- contract-first edits
- manifest-first edits
- allowed edit surfaces
- generated freshness
- verification commands
- evidence and rollback requirements
