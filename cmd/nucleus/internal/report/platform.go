package report

import (
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/contract/openapi"
	"github.com/nucleuskit/contract/proto"
)

type platformReadinessReport struct {
	Service               string                       `json:"service"`
	Version               string                       `json:"version"`
	EndpointCount         int                          `json:"endpoint_count"`
	GRPCServiceCount      int                          `json:"grpc_service_count"`
	CapabilityCount       int                          `json:"capability_count"`
	Capabilities          []string                     `json:"capabilities"`
	GeneratedFresh        bool                         `json:"generated_fresh"`
	GeneratedFreshness    []inspect.GeneratedFreshness `json:"generated_freshness"`
	VerificationCommands  []string                     `json:"verification_commands"`
	PlatformMapping       string                       `json:"platform_mapping"`
	ReleaseMatrix         []string                     `json:"release_matrix"`
	PlatformUploadPayload platformUploadPayload        `json:"platform_upload_payload"`
	ReleaseDryRun         releaseDryRunPayload         `json:"release_dry_run"`
	ReadinessGates        []readinessGate              `json:"readiness_gates"`
	RiskGates             []riskGate                   `json:"risk_gates"`
	ProviderStrategy      []providerStrategy           `json:"provider_strategy"`
	ControlPlaneIncluded  bool                         `json:"control_plane_included"`
	ExternalSDKRequired   bool                         `json:"external_sdk_required"`
	ProductionBridgeScope string                       `json:"production_bridge_scope"`
}

type readinessGate struct {
	ID       string `json:"id"`
	Pass     bool   `json:"pass"`
	Artifact string `json:"artifact,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type riskGate struct {
	ID       string `json:"id"`
	Pass     bool   `json:"pass"`
	Severity string `json:"severity"`
	Reason   string `json:"reason"`
}

type providerStrategy struct {
	Capability string   `json:"capability"`
	Provider   string   `json:"provider,omitempty"`
	SDKStatus  string   `json:"sdk_status"`
	Gaps       []string `json:"gaps"`
}

type platformUploadPayload struct {
	Artifact             string             `json:"artifact"`
	NetworkRequired      bool               `json:"network_required"`
	Service              manifest.Service   `json:"service"`
	Capabilities         []string           `json:"capabilities"`
	Endpoints            []openapi.Endpoint `json:"endpoints"`
	GRPCServices         []proto.Service    `json:"grpc_services"`
	GeneratedFresh       bool               `json:"generated_fresh"`
	VerificationCommands []string           `json:"verification_commands"`
}

type releaseDryRunPayload struct {
	Artifact             string   `json:"artifact"`
	NetworkRequired      bool     `json:"network_required"`
	ReleaseMatrix        []string `json:"release_matrix"`
	VerificationCommands []string `json:"verification_commands"`
}

func buildPlatformReadinessResult(dir string) reportResult {
	description, err := inspect.Describe(dir)
	if err != nil {
		return finalizeReportResult(reportResult{
			Mode: reportModePlatformReadiness,
			Diagnostics: diagnostic.Diagnostics{
				reportErrorDiagnostic(safeReportPath(dir), reportDiagnosticInspectFailed, safeErrorMessage(err)),
			},
		})
	}
	report := platformReadinessFromDescription(description)
	summary := reportSummaryFromPlatformReadiness(report)
	return finalizeReportResult(reportResult{
		Mode:              reportModePlatformReadiness,
		Summary:           summary,
		PlatformReadiness: &report,
	})
}

func platformReadinessFromDescription(description inspect.Description) platformReadinessReport {
	generatedFresh := generatedFreshnessPass(description.GeneratedFreshness)
	releaseMatrix := []string{"linux/amd64", "linux/arm64"}
	platformPayload := platformUploadPayload{
		Artifact:             "local:artifacts/nucleus/platform-upload-payload.json",
		NetworkRequired:      false,
		Service:              description.Service,
		Capabilities:         copyStringSlice(description.Capabilities),
		Endpoints:            copyEndpointSlice(description.Endpoints),
		GRPCServices:         copyGRPCServiceSlice(description.GRPCServices),
		GeneratedFresh:       generatedFresh,
		VerificationCommands: copyStringSlice(description.Verification.Commands),
	}
	releaseDryRun := releaseDryRunPayload{
		Artifact:             "local:artifacts/nucleus/release-dry-run.json",
		NetworkRequired:      false,
		ReleaseMatrix:        releaseMatrix,
		VerificationCommands: copyStringSlice(description.Verification.Commands),
	}
	return platformReadinessReport{
		Service:               description.Service.Name,
		Version:               description.Service.Version,
		EndpointCount:         len(description.Endpoints),
		GRPCServiceCount:      len(description.GRPCServices),
		CapabilityCount:       len(description.Capabilities),
		Capabilities:          copyStringSlice(description.Capabilities),
		GeneratedFresh:        generatedFresh,
		GeneratedFreshness:    description.GeneratedFreshness,
		VerificationCommands:  copyStringSlice(description.Verification.Commands),
		PlatformMapping:       "docs/platform-mapping.md",
		ReleaseMatrix:         releaseMatrix,
		PlatformUploadPayload: platformPayload,
		ReleaseDryRun:         releaseDryRun,
		ReadinessGates:        buildReadinessGates(generatedFresh, description.Verification.Commands),
		RiskGates:             buildRiskGates(generatedFresh),
		ProviderStrategy:      buildProviderStrategy(description.CapabilityGraph),
		ControlPlaneIncluded:  false,
		ExternalSDKRequired:   false,
		ProductionBridgeScope: "capability protocol metadata only; provider SDKs stay optional bridge/user code",
	}
}

func generatedFreshnessPass(items []inspect.GeneratedFreshness) bool {
	for _, item := range items {
		if !item.Fresh {
			return false
		}
	}
	return true
}

func reportSummaryFromPlatformReadiness(report platformReadinessReport) reportSummary {
	readinessPassed, readinessFailed := gateCounts(report.ReadinessGates)
	riskPassed, riskFailed := riskGateCounts(report.RiskGates)
	return reportSummary{
		EndpointCount:       report.EndpointCount,
		GRPCServiceCount:    report.GRPCServiceCount,
		CapabilityCount:     report.CapabilityCount,
		GeneratedFresh:      report.GeneratedFresh,
		ReadinessGateCount:  len(report.ReadinessGates),
		ReadinessGatePassed: readinessPassed,
		ReadinessGateFailed: readinessFailed,
		RiskGateCount:       len(report.RiskGates),
		RiskGatePassed:      riskPassed,
		RiskGateFailed:      riskFailed,
	}
}

func buildReadinessGates(generatedFresh bool, verificationCommands []string) []readinessGate {
	return []readinessGate{
		{
			ID:       "platform_upload_payload",
			Pass:     true,
			Artifact: "local:artifacts/nucleus/platform-upload-payload.json",
			Reason:   "local artifact metadata generated; no platform network call required",
		},
		{
			ID:       "release_dry_run",
			Pass:     true,
			Artifact: "local:artifacts/nucleus/release-dry-run.json",
			Reason:   "release dry-run metadata is local-only",
		},
		{
			ID:     "generated_freshness",
			Pass:   generatedFresh,
			Reason: chooseBoolReason(generatedFresh, "generated artifacts are fresh", "generated artifacts are stale or missing"),
		},
		{
			ID:     "verification_commands",
			Pass:   len(verificationCommands) > 0,
			Reason: chooseBoolReason(len(verificationCommands) > 0, "verification commands are declared", "verification commands are missing"),
		},
	}
}

func buildRiskGates(generatedFresh bool) []riskGate {
	return []riskGate{
		{
			ID:       "external_provider_sdk",
			Pass:     true,
			Severity: "info",
			Reason:   "provider SDKs remain optional bridge or user code; report does not import SDKs",
		},
		{
			ID:       "control_plane_network",
			Pass:     true,
			Severity: "info",
			Reason:   "platform upload and release dry-run are local artifacts only",
		},
		{
			ID:       "generated_freshness",
			Pass:     generatedFresh,
			Severity: chooseBoolReason(generatedFresh, "info", "warning"),
			Reason:   chooseBoolReason(generatedFresh, "generated artifacts are fresh", "stale generated artifacts block release readiness"),
		},
	}
}

func buildProviderStrategy(nodes []inspect.CapabilityNode) []providerStrategy {
	strategies := make([]providerStrategy, 0, len(nodes))
	for _, node := range nodes {
		strategy := providerStrategy{
			Capability: node.Capability,
			Provider:   node.Provider,
			SDKStatus:  providerSDKStatus(node.Provider),
			Gaps: []string{
				"health_check: define provider-specific readiness probe before production rollout",
				"fallback: define noop/local fallback behavior for provider outage",
				"observability: define provider metrics, traces, and error labels",
			},
		}
		strategies = append(strategies, strategy)
	}
	return strategies
}

func providerSDKStatus(provider string) string {
	switch strings.TrimSpace(provider) {
	case "":
		return "optional_no_provider_detected"
	case "noop", "memory", "file":
		return "optional_local_provider_detected"
	default:
		return "optional_external_provider_detected"
	}
}

func gateCounts(gates []readinessGate) (int, int) {
	passed := 0
	failed := 0
	for _, gate := range gates {
		if gate.Pass {
			passed++
		} else {
			failed++
		}
	}
	return passed, failed
}

func riskGateCounts(gates []riskGate) (int, int) {
	passed := 0
	failed := 0
	for _, gate := range gates {
		if gate.Pass {
			passed++
		} else {
			failed++
		}
	}
	return passed, failed
}

func chooseBoolReason(ok bool, pass string, fail string) string {
	if ok {
		return pass
	}
	return fail
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func copyEndpointSlice(values []openapi.Endpoint) []openapi.Endpoint {
	if len(values) == 0 {
		return []openapi.Endpoint{}
	}
	return append([]openapi.Endpoint(nil), values...)
}

func copyGRPCServiceSlice(values []proto.Service) []proto.Service {
	if len(values) == 0 {
		return []proto.Service{}
	}
	return append([]proto.Service(nil), values...)
}
