package inspect

import (
	"sort"
	"strings"

	"github.com/nucleuskit/contract/manifest"
)

// capabilityGraph returns a list of capability nodes.
func capabilityGraph(m manifest.Manifest, imports []string) []CapabilityNode {
	nodes := make([]CapabilityNode, 0, len(m.Capabilities))
	for _, capability := range m.Capabilities {
		module := CapabilityModule(capability)
		matches := matchingImports(module, imports)
		provider := manifestCapabilityProvider(m, capability)
		if provider == "" {
			provider = capabilityProvider(capability, imports)
		}
		nodes = append(nodes, CapabilityNode{
			Capability: capability,
			Declared:   true,
			Imported:   len(matches) > 0 || module == "",
			Provider:   provider,
			Module:     module,
			Imports:    matches,
		})
	}
	return nodes
}

// manifestCapabilityProvider returns the capability provider for a declared capability.
func manifestCapabilityProvider(m manifest.Manifest, capability string) string {
	if values, ok := m.Nucleus.Providers[capability]; ok {
		if text, _ := values[providerConfigKey].(string); strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	var value any
	switch capability {
	case "sql":
		value = m.Nucleus.SQL[providerConfigKey]
	case "mongo":
		value = m.Nucleus.Mongo[providerConfigKey]
	default:
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

// capabilityProvider returns the capability provider for an undeclared capability.
func capabilityProvider(capability string, imports []string) string {
	providers := map[string][]string{
		"log":          {"zap"},
		"trace":        {"otel"},
		"config":       {"file", "nacos", "nacosofficial", "configkv", "acm"},
		"httpclient":   {},
		"transport":    {"netdialer"},
		"discovery":    {"nacos", "nacosofficial"},
		"metric":       {"prometheus", "otel"},
		"auth":         {"security"},
		"health":       {"noop"},
		"sql":          {"sql", "postgres", "gorm", "mysql"},
		"redis":        {"redis", "goredis"},
		"mongo":        {"mongo"},
		"kv":           {"kv"},
		"mq":           {"kafka", "sarama", "nats", "amqp"},
		"store":        {"memory", "cache", "bloom"},
		"lock":         {"memorylock", "redislock"},
		"sentinel":     {"sentinel"},
		"errortracker": {"sentry"},
		"profiler":     {"pyroscope"},
	}
	for _, provider := range providers[capability] {
		module := moduleBridgeRoot + "/" + provider
		if len(matchingImports(module, imports)) > 0 {
			return provider
		}
	}
	return ""
}

// CapabilityModule returns the canonical module path for a declared capability.
func CapabilityModule(capability string) string {
	switch capability {
	case "http":
		return moduleRuntime + "/http"
	case "grpc":
		return moduleRuntime + "/grpc"
	case "worker":
		return moduleRuntime + "/worker"
	case "log":
		return moduleCapRoot + "/log"
	case "trace":
		return moduleCapRoot + "/trace"
	case "config":
		return moduleCapRoot + "/config"
	case "httpclient":
		return moduleCapRoot + "/httpclient"
	case "transport":
		return moduleCapRoot + "/transport"
	case "discovery":
		return moduleCapRoot + "/discovery"
	case "metric":
		return moduleCapRoot + "/metric"
	case "auth":
		return moduleCapRoot + "/auth"
	case "health":
		return moduleCapRoot + "/health"
	case "sql":
		return moduleCapRoot + "/sql"
	case "redis":
		return moduleCapRoot + "/redis"
	case "mongo":
		return moduleCapRoot + "/mongo"
	case "kv":
		return moduleCapRoot + "/kv"
	case "mq":
		return moduleCapRoot + "/mq"
	case "store":
		return moduleCapRoot + "/store"
	case "lock":
		return moduleCapRoot + "/lock"
	case "sentinel":
		return moduleCapRoot + "/sentinel"
	case "errortracker":
		return moduleCapRoot + "/errortracker"
	case "profiler":
		return moduleCapRoot + "/profiler"
	default:
		return ""
	}
}

// matchingImports returns the matching imports for a module.
func matchingImports(module string, imports []string) []string {
	if module == "" {
		return nil
	}
	var matches []string
	for _, importPath := range imports {
		if importPath == module || strings.HasPrefix(importPath, module+"/") {
			matches = append(matches, importPath)
		}
	}
	sort.Strings(matches)
	return matches
}
