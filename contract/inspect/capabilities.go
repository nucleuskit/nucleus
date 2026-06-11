package inspect

import (
	"sort"
	"strings"

	"github.com/nucleuskit/contract/manifest"
)

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

func manifestCapabilityProvider(m manifest.Manifest, capability string) string {
	if values, ok := m.Nucleus.Providers[capability]; ok {
		if text, _ := values["provider"].(string); strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	var value any
	switch capability {
	case "sql":
		value = m.Nucleus.SQL["provider"]
	case "mongo":
		value = m.Nucleus.Mongo["provider"]
	default:
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

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
		module := "anniext.cn/spelens-gud/nucleus/bridge/" + provider
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
		return "anniext.cn/spelens-gud/nucleus/runtime/http"
	case "grpc":
		return "anniext.cn/spelens-gud/nucleus/runtime/grpc"
	case "worker":
		return "anniext.cn/spelens-gud/nucleus/runtime/worker"
	case "log":
		return "anniext.cn/spelens-gud/nucleus/cap/log"
	case "trace":
		return "anniext.cn/spelens-gud/nucleus/cap/trace"
	case "config":
		return "anniext.cn/spelens-gud/nucleus/cap/config"
	case "httpclient":
		return "anniext.cn/spelens-gud/nucleus/cap/httpclient"
	case "transport":
		return "anniext.cn/spelens-gud/nucleus/cap/transport"
	case "discovery":
		return "anniext.cn/spelens-gud/nucleus/cap/discovery"
	case "metric":
		return "anniext.cn/spelens-gud/nucleus/cap/metric"
	case "auth":
		return "anniext.cn/spelens-gud/nucleus/cap/auth"
	case "health":
		return "anniext.cn/spelens-gud/nucleus/cap/health"
	case "sql":
		return "anniext.cn/spelens-gud/nucleus/cap/sql"
	case "redis":
		return "anniext.cn/spelens-gud/nucleus/cap/redis"
	case "mongo":
		return "anniext.cn/spelens-gud/nucleus/cap/mongo"
	case "kv":
		return "anniext.cn/spelens-gud/nucleus/cap/kv"
	case "mq":
		return "anniext.cn/spelens-gud/nucleus/cap/mq"
	case "store":
		return "anniext.cn/spelens-gud/nucleus/cap/store"
	case "lock":
		return "anniext.cn/spelens-gud/nucleus/cap/lock"
	case "sentinel":
		return "anniext.cn/spelens-gud/nucleus/cap/sentinel"
	case "errortracker":
		return "anniext.cn/spelens-gud/nucleus/cap/errortracker"
	case "profiler":
		return "anniext.cn/spelens-gud/nucleus/cap/profiler"
	default:
		return ""
	}
}

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
