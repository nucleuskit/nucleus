package mcp

const (
	commandUseMCP      = "mcp"
	commandShortMCP    = "serve local Nucleus MCP tools over stdio"
	defaultDir         = "."
	nucleusMCPProtocol = "2024-11-05"
	serverName         = "nucleus"
	serverVersion      = "0.0.0"
	jsonRPCVersion     = "2.0"
)

const (
	flagStdio = "stdio"
)

const (
	flagHelpStdio = "serve MCP over stdio"
)

const (
	toolGetServiceDescription = "get_service_description"
	toolGetEditSurfaces       = "get_edit_surfaces"
	toolGetContracts          = "get_contracts"
	toolGetCapabilities       = "get_capabilities"
	toolTraceRoute            = "trace_route"
	toolTraceSymbol           = "trace_symbol"
	toolTraceCapability       = "trace_capability"
	toolImpactSymbol          = "impact_symbol"
	toolImpactFile            = "impact_file"
	toolImpactContract        = "impact_contract"
	toolFindSymbol            = "find_symbol"
	toolListCallers           = "list_callers"
	toolListCallees           = "list_callees"
	toolValidateDecision      = "validate_decision"
	toolListDecisions         = "list_decisions"
	toolGetReport             = "get_report"
	toolBuildPlan             = "build_plan"
	toolListRecipes           = "list_recipes"
	toolGetRecipe             = "get_recipe"
)

const (
	defaultRecipeDir = ".nucleus/recipes"
)

const (
	resultKindMCPEditSurfaces = "nucleus.mcp.edit_surfaces_result"
	resultKindMCPContracts    = "nucleus.mcp.contracts_result"
	resultKindMCPCapabilities = "nucleus.mcp.capabilities_result"
	resultKindMCPFindSymbol   = "nucleus.mcp.find_symbol_result"
	resultKindMCPCalls        = "nucleus.mcp.calls_result"
	schemaVersionMCPResult    = "mcp-result.v1"
	schemaRefMCPResult        = "contract/schema/mcp-result.v1.schema.json"
)
