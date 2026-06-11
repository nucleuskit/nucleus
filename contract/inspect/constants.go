package inspect

const (
	descriptionSchemaVersion = "1.0"
	flowGraphSchemaVersion   = "1.0"
)

const (
	goModFileName = "go.mod"
)

const (
	contractSourceAPI       = "api"
	contractPathOpenAPI     = "api/openapi.yaml"
	contractPathErrors      = "api/errors.yaml"
	contractProtoDir        = "api/proto"
	contractProtoFileSuffix = ".proto"
)

const (
	defaultPolicyOutboundKey = "outbound"
	providerConfigKey        = "provider"
)

const (
	commandValidate = "nucleus validate --dir ."
	commandLint     = "nucleus lint --dir ."
	commandGoTest   = "go test ./..."
)
const (
	capabilitySQL   = "sql"
	capabilityMongo = "mongo"
)

const (
	moduleRoot       = "anniext.cn/spelens-gud/nucleus"
	moduleBridgeRoot = moduleRoot + "/bridge"
	moduleCapRoot    = moduleRoot + "/cap"
	moduleRuntime    = moduleRoot + "/runtime"
)
const (
	configDirName        = "configs"
	configLocalNamePart  = "local"
	configSecretNamePart = "secret"
	configYAMLExtension  = ".yaml"
	configYMLExtension   = ".yml"
	configEnvPrefix      = "${"
	configEnvSuffix      = "}"
	configEnvFallbackSep = ":-"
	configKeySeparator   = "."
)
const (
	generatedHTTPAdapterTarget = "internal/adapter/http/gen"
	generatedGRPCAdapterTarget = "internal/adapter/grpc/gen"
	contractGeneratedTarget    = "contract/gen"
	runtimeHTTPServerSource    = "runtime/http/server.go"
	responseEnvelopeName       = "runtime/http.ResponseEnvelope"
)

const (
	flowNodeKindRoute      = "route"
	flowNodeKindHandler    = "handler"
	flowNodeKindOutbound   = "outbound"
	flowNodeKindDomain     = "domain"
	flowNodeKindResponse   = "response"
	flowNodeKindCapability = "capability"
)

const (
	flowEdgeKindDispatch        = "dispatch"
	flowEdgeKindOutboundUnknown = "outbound_unknown"
	flowEdgeKindCall            = "call"
	flowEdgeKindOutboundCall    = "outbound_call"
	flowEdgeKindEncodesResponse = "encodes_response"
	flowEdgeKindUsesCapability  = "uses_capability"
)

const (
	flowFactKindOpenAPIParameter   = "openapi_parameter"
	flowFactKindOpenAPIRequestBody = "openapi_request_body"
	flowFactKindErrorCode          = "error_code"
)

const (
	flowIDPrefixRoute      = "route:"
	flowIDPrefixOperation  = "operation:"
	flowIDPrefixDomain     = "domain:"
	flowIDPartResponse     = ":response:"
	flowIDPartCapability   = ":capability:"
	flowUnknownOutboundID  = ":outbound:unknown"
	flowHTTPClientOutbound = ":outbound:httpclient.Do"
)

const (
	flowParamPrefixPath   = "path."
	flowParamPrefixQuery  = "query."
	flowParamPrefixHeader = "header."
	flowRequestBodyName   = "body.required"
)

const (
	flowSourceOpenAPI           = contractPathOpenAPI
	flowSourceErrors            = contractPathErrors
	flowSourceStaticUnavailable = "static analysis unavailable"
	flowSourceConfidence        = "source"
	flowUnknownName             = "unknown"
	flowHTTPClientDoName        = "httpclient.Do"
	capabilityLog               = "log"
	generatedRouteBinderName    = "generated route binder dispatch"
	wellKnownNucleusPath        = "/.well-known/nucleus.json"
)

const (
	generatedFreshnessReasonMissingMarker = "missing freshness marker"
	generatedFreshnessReasonHashDiffers   = "source hash differs from generated marker"
)
const (
	goSourceExtension     = ".go"
	goTestSourceExtension = "_test.go"
	sourceLineSeparator   = ":"
	routeKeySeparator     = " "
	handlerNameSuffix     = " handler"
	domainSourcePathPart  = "internal/domain/"
)

const (
	selectorRegisterRoutes    = "RegisterRoutes"
	selectorHandle            = "Handle"
	selectorRegisterWellKnown = "RegisterWellKnown"
	selectorHTTPClientDo      = "Do"
	selectorLogDebug          = "Debug"
	selectorLogInfo           = "Info"
	selectorLogWarn           = "Warn"
	selectorLogError          = "Error"
	identifierHTTP            = "http"
	identifierBlankImport     = "_"
	identifierDotImport       = "."
)

const (
	selectorHTTPMethodGet    = "MethodGet"
	selectorHTTPMethodPost   = "MethodPost"
	selectorHTTPMethodPut    = "MethodPut"
	selectorHTTPMethodPatch  = "MethodPatch"
	selectorHTTPMethodDelete = "MethodDelete"
)

const (
	httpMethodGet    = "GET"
	httpMethodPost   = "POST"
	httpMethodPut    = "PUT"
	httpMethodPatch  = "PATCH"
	httpMethodDelete = "DELETE"
)

const (
	skipDirGit         = ".git"
	skipDirGitNexus    = ".gitnexus"
	skipDirVendor      = "vendor"
	skipDirNodeModules = "node_modules"
	skipPathGit        = "/.git/"
	skipPathIdea       = "/.idea/"
	skipPathCursor     = "/.cursor/"
)
