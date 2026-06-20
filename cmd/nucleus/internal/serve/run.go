package serve

import (
	"fmt"
	"net"
	"strings"

	"github.com/nucleuskit/contract/inspect"
)

func buildResult(config Config, opts *options) serveResult {
	result := serveResult{
		Mode: modeFromOptions(opts),
		Summary: serveSummary{
			Addr:        addrFromOptions(opts),
			ServedPaths: servedPaths(),
		},
		Server: serveServer{
			MetadataEndpoints: metadataEndpoints(),
		},
	}
	scope, err := classifyNetworkScope(result.Summary.Addr)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, errorDiagnostic("", diagnosticAddrInvalid, fmt.Sprintf("invalid listen address %q: %v", result.Summary.Addr, err)))
	} else {
		result.Server.NetworkScope = scope
		if scope != networkScopeLoopback && (opts == nil || !opts.allowNonLocal) {
			result.Diagnostics = append(result.Diagnostics, errorDiagnostic("", diagnosticNonLocalAddr, fmt.Sprintf("refusing to expose metadata on non-loopback address %q without --%s", result.Summary.Addr, flagAllowNonLocal)))
		}
	}
	dir := stringValue(config.Dir, defaultDir)
	description, err := inspect.Describe(dir)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, errorDiagnostic("nucleus.yaml", diagnosticInspectFailed, fmt.Sprintf("inspect service metadata: %v", err)))
		return finalizeResult(result)
	}
	result.Description = description
	result.Summary.Service = description.Service.Name
	result.Summary.Version = description.Service.Version
	result.Summary.EndpointCount = len(description.Endpoints)
	result.Summary.GRPCServiceCount = len(description.GRPCServices)
	result.Summary.CapabilityCount = len(description.Capabilities)
	result.Summary.GeneratedFresh = generatedFresh(description.GeneratedFreshness)
	return finalizeResult(result)
}

func modeFromOptions(opts *options) string {
	if opts != nil && opts.mode != "" {
		return opts.mode
	}
	if opts != nil && opts.check {
		return modeCheck
	}
	return modeServe
}

func addrFromOptions(opts *options) string {
	if opts == nil || opts.addr == "" {
		return defaultAddr
	}
	addr := strings.TrimSpace(opts.addr)
	if addr == "" {
		return defaultAddr
	}
	return addr
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

func servedPaths() []string {
	return []string{pathHealthz, pathReadyz, pathWellKnown}
}

func generatedFresh(freshness []inspect.GeneratedFreshness) bool {
	if len(freshness) == 0 {
		return false
	}
	for _, item := range freshness {
		if !item.Fresh {
			return false
		}
	}
	return true
}

func classifyNetworkScope(addr string) (string, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return networkScopeInvalid, err
	}
	if strings.TrimSpace(port) == "" {
		return networkScopeInvalid, fmt.Errorf("missing port")
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return networkScopeNonLocal, nil
	}
	if strings.EqualFold(host, "localhost") {
		return networkScopeLoopback, nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return networkScopeNonLocal, nil
	}
	if ip.IsLoopback() {
		return networkScopeLoopback, nil
	}
	return networkScopeNonLocal, nil
}
