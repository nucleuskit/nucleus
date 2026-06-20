# Nucleus MCP Reference

Nucleus MCP is a tool layer over CLI/schema facts. Use it for quick access to service facts, not as a separate source of truth.

## Preferred Order

1. Use CLI/schema commands to understand the contract:

   ```bash
   nucleus describe --dir . --json --pretty
   nucleus plan --dir . --task "<task>" --json --executable
   ```

2. Use MCP tools for targeted facts:
   - describe service
   - endpoints
   - error catalog
   - capability recipes and scaffold commands
   - flow graph
   - executable plan
   - verification or repair wrappers

3. Return to CLI commands for generation, lint, verification, repair, and report evidence.

## Capability Recipes

Use `get_capability_recipe` before adding or changing a capability. Read these fields:

- `scaffold_supported`: whether `nucleus capability add` knows this capability.
- `scaffold_command`: default CLI command for the capability.
- `bridge_candidates[].scaffold_command`: provider-specific CLI commands.
- `wiring`: layer boundary guidance for app-level provider injection.

Run the scaffold command before manual provider wiring unless the command is unavailable or the manifest/edit surface blocks it.

## Consistency Rule

If MCP and CLI/schema disagree, trust CLI/schema and record the mismatch in the handoff.

## Safety Rule

MCP tools must not bypass:

- Contract-First edits
- Manifest-First edits
- allowed edit surfaces
- generated freshness
- verification commands
- evidence and rollback requirements
