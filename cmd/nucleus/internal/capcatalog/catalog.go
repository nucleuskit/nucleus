package capcatalog

import (
	"sort"
	"strings"
)

// Provider describes a provider name accepted for a capability scaffold.
type Provider struct {
	Name        string
	Description string
}

// Spec describes a supported Nucleus capability scaffold.
type Spec struct {
	Name            string
	DefaultProvider string
	Providers       []Provider
	Description     string
	DSNEnv          string
	Planning        bool
}

var specs = []Spec{
	{
		Name:            "http",
		DefaultProvider: "runtime/http",
		Description:     "inbound HTTP runtime",
		Providers: []Provider{
			{Name: "runtime/http", Description: "Nucleus HTTP runtime"},
		},
	},
	{
		Name:            "grpc",
		DefaultProvider: "runtime/grpc",
		Description:     "inbound gRPC runtime",
		Providers: []Provider{
			{Name: "runtime/grpc", Description: "Nucleus gRPC runtime"},
		},
	},
	{
		Name:            "worker",
		DefaultProvider: "runtime/worker",
		Description:     "worker runtime",
		Providers: []Provider{
			{Name: "runtime/worker", Description: "Nucleus worker runtime"},
		},
	},
	{
		Name:            "log",
		DefaultProvider: "zap",
		Description:     "structured logging",
		Planning:        true,
		Providers: []Provider{
			{Name: "zap", Description: "zap-compatible logging provider"},
		},
	},
	{
		Name:            "trace",
		DefaultProvider: "otel",
		Description:     "distributed tracing",
		Planning:        true,
		Providers: []Provider{
			{Name: "otel", Description: "OpenTelemetry tracing provider"},
		},
	},
	{
		Name:            "config",
		DefaultProvider: "file",
		Description:     "local or remote configuration",
		Planning:        true,
		Providers: []Provider{
			{Name: "file", Description: "local file configuration"},
			{Name: "nacos", Description: "Nacos-compatible configuration"},
			{Name: "nacosofficial", Description: "official Nacos client configuration"},
			{Name: "configkv", Description: "key-value configuration"},
			{Name: "acm", Description: "ACM-compatible configuration"},
		},
	},
	{
		Name:            "httpclient",
		DefaultProvider: "standard",
		Description:     "outbound HTTP client",
		Planning:        true,
		Providers: []Provider{
			{Name: "standard", Description: "standard-library HTTP client"},
		},
	},
	{
		Name:            "transport",
		DefaultProvider: "netdialer",
		Description:     "outbound transport policy",
		Planning:        true,
		Providers: []Provider{
			{Name: "netdialer", Description: "standard-library network dialer"},
		},
	},
	{
		Name:            "discovery",
		DefaultProvider: "nacos",
		Description:     "service discovery",
		Planning:        true,
		Providers: []Provider{
			{Name: "nacos", Description: "Nacos-compatible discovery"},
			{Name: "nacosofficial", Description: "official Nacos client discovery"},
		},
	},
	{
		Name:            "metric",
		DefaultProvider: "prometheus",
		Description:     "metrics",
		Planning:        true,
		Providers: []Provider{
			{Name: "prometheus", Description: "Prometheus metrics"},
			{Name: "otel", Description: "OpenTelemetry metrics"},
		},
	},
	{
		Name:            "auth",
		DefaultProvider: "security",
		Description:     "authentication and authorization",
		Planning:        true,
		Providers: []Provider{
			{Name: "security", Description: "service-owned security provider"},
		},
	},
	{
		Name:            "health",
		DefaultProvider: "noop",
		Description:     "health and readiness reporting",
		Planning:        true,
		Providers: []Provider{
			{Name: "noop", Description: "no-op health provider"},
		},
	},
	{
		Name:            "sql",
		DefaultProvider: "sql",
		Description:     "relational SQL persistence",
		DSNEnv:          "NUCLEUS_DATABASE_DSN",
		Planning:        true,
		Providers: []Provider{
			{Name: "sql", Description: "database/sql-compatible metadata provider"},
			{Name: "postgres", Description: "PostgreSQL database/sql provider"},
			{Name: "mysql", Description: "MySQL database/sql provider"},
			{Name: "gorm", Description: "GORM-compatible provider"},
		},
	},
	{
		Name:            "redis",
		DefaultProvider: "redis",
		Description:     "Redis cache or data structure client",
		DSNEnv:          "NUCLEUS_REDIS_DSN",
		Planning:        true,
		Providers: []Provider{
			{Name: "redis", Description: "Redis-compatible provider"},
			{Name: "goredis", Description: "go-redis-compatible provider"},
		},
	},
	{
		Name:            "mongo",
		DefaultProvider: "mongo",
		Description:     "MongoDB document persistence",
		DSNEnv:          "NUCLEUS_MONGO_DSN",
		Planning:        true,
		Providers: []Provider{
			{Name: "mongo", Description: "MongoDB provider"},
		},
	},
	{
		Name:            "kv",
		DefaultProvider: "kv",
		Description:     "key-value store",
		Planning:        true,
		Providers: []Provider{
			{Name: "kv", Description: "key-value provider"},
		},
	},
	{
		Name:            "mq",
		DefaultProvider: "kafka",
		Description:     "message producer and consumer",
		Planning:        true,
		Providers: []Provider{
			{Name: "kafka", Description: "Kafka-compatible provider"},
			{Name: "sarama", Description: "Sarama Kafka provider"},
			{Name: "nats", Description: "NATS provider"},
			{Name: "amqp", Description: "AMQP-compatible provider"},
		},
	},
	{
		Name:            "store",
		DefaultProvider: "memory",
		Description:     "generic store",
		Planning:        true,
		Providers: []Provider{
			{Name: "memory", Description: "in-memory store"},
			{Name: "cache", Description: "cache store"},
			{Name: "bloom", Description: "Bloom filter store"},
		},
	},
	{
		Name:            "lock",
		DefaultProvider: "memorylock",
		Description:     "distributed or local locking",
		Planning:        true,
		Providers: []Provider{
			{Name: "memorylock", Description: "local memory lock"},
			{Name: "redislock", Description: "Redis-backed lock"},
		},
	},
	{
		Name:            "sentinel",
		DefaultProvider: "sentinel",
		Description:     "rate limiting and circuit breaking",
		Planning:        true,
		Providers: []Provider{
			{Name: "sentinel", Description: "Sentinel-compatible provider"},
		},
	},
	{
		Name:            "errortracker",
		DefaultProvider: "sentry",
		Description:     "error reporting",
		Planning:        true,
		Providers: []Provider{
			{Name: "sentry", Description: "Sentry-compatible provider"},
		},
	},
	{
		Name:            "profiler",
		DefaultProvider: "pyroscope",
		Description:     "profiling",
		Planning:        true,
		Providers: []Provider{
			{Name: "pyroscope", Description: "Pyroscope-compatible provider"},
		},
	},
}

// All returns every known capability scaffold spec sorted by capability name.
func All() []Spec {
	values := make([]Spec, len(specs))
	copy(values, specs)
	sort.Slice(values, func(i, j int) bool {
		return values[i].Name < values[j].Name
	})
	return values
}

// Names returns every known capability name sorted lexicographically.
func Names() []string {
	return namesMatching(func(Spec) bool { return true })
}

// PlanningNames returns capability names considered by natural-language plans.
func PlanningNames() []string {
	return namesMatching(func(spec Spec) bool { return spec.Planning })
}

// Lookup returns the spec for name after normalizing whitespace and case.
func Lookup(name string) (Spec, bool) {
	normalized := Normalize(name)
	for _, spec := range specs {
		if spec.Name == normalized {
			return spec, true
		}
	}
	return Spec{}, false
}

// Normalize returns the canonical spelling for capability or provider values.
func Normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// DefaultProvider returns the default provider for a capability.
func DefaultProvider(capability string) string {
	spec, ok := Lookup(capability)
	if !ok {
		return ""
	}
	return spec.DefaultProvider
}

// ProviderSupported reports whether provider is accepted for spec.
func ProviderSupported(spec Spec, provider string) bool {
	_, ok := spec.Provider(provider)
	return ok
}

// Provider returns the provider metadata for name.
func (s Spec) Provider(name string) (Provider, bool) {
	normalized := Normalize(name)
	for _, provider := range s.Providers {
		if provider.Name == normalized {
			return provider, true
		}
	}
	return Provider{}, false
}

// ProviderNames returns every provider name for the spec.
func (s Spec) ProviderNames() []string {
	names := make([]string, 0, len(s.Providers))
	for _, provider := range s.Providers {
		names = append(names, provider.Name)
	}
	sort.Strings(names)
	return names
}

func namesMatching(include func(Spec) bool) []string {
	var names []string
	for _, spec := range specs {
		if include(spec) {
			names = append(names, spec.Name)
		}
	}
	sort.Strings(names)
	return names
}
