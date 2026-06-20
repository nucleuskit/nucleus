package plan

import (
	"strings"

	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/capcatalog"
)

func generatedOutputs(taskType string) []string {
	switch taskType {
	case taskTypeGRPCService:
		return []string{"internal/adapter/grpc/gen/**", "contract/gen/grpc.go"}
	case taskTypeHTTPEndpoint:
		return []string{"internal/adapter/http/gen/**", "contract/gen/endpoints.go"}
	case taskTypeErrorCatalog:
		return []string{"contract/gen/errors.go"}
	case taskTypeCapability:
		return []string{"nucleus.yaml", "internal/app/**", "internal/config/**", "internal/adapter/store/**", "configs/**", "deploy/**", "docs/**"}
	default:
		return []string{"contract/gen/**"}
	}
}

func commands(taskType string, task string, requestedCapabilities []string) []string {
	switch taskType {
	case taskTypeGRPCService:
		return []string{"nucleus gen --dir . --grpc", commandValidate, commandLintStrict, commandVerifyJSON}
	case taskTypeHTTPEndpoint:
		return []string{"nucleus gen --dir . --http --errors", commandValidate, commandLintStrict, commandVerifyJSON}
	case taskTypeErrorCatalog:
		return []string{"nucleus gen --dir . --errors", commandValidate, commandLintStrict, commandVerifyJSON}
	case taskTypeCapability:
		var items []string
		for _, capability := range requestedCapabilities {
			command := "nucleus capability add " + capability
			if provider := providerHint(task, capability); provider != "" {
				command += " --provider " + provider
			}
			items = append(items, command)
		}
		items = append(items, commandValidate)
		items = append(items, commandLintStrict, commandVerifyJSON)
		return items
	default:
		return []string{commandValidate, commandLintStrict, commandVerifyJSON}
	}
}

func risks(taskType string, description inspect.Description, requestedCapabilities []string) []string {
	risks := []string{}
	switch taskType {
	case taskTypeGRPCService:
		risks = append(risks, "gRPC 行为变更必须先更新 api/proto 并重新生成契约元数据")
	case taskTypeHTTPEndpoint:
		risks = append(risks, "HTTP 行为变更必须先更新 OpenAPI 与 errors.yaml")
	case taskTypeErrorCatalog:
		risks = append(risks, "错误码变更必须保持 api/errors.yaml 与响应映射稳定")
	case taskTypeCapability:
		risks = append(risks, "能力接入必须保持 manifest、cap 接口和 import graph 一致")
	}
	for _, capability := range requestedCapabilities {
		if !hasString(description.Capabilities, capability) {
			risks = append(risks, "manifest 未声明能力 "+capability)
		}
		if inspect.CapabilityModule(capability) == "" {
			risks = append(risks, "未知 capability "+capability+" 需要先确认是否属于 Nucleus 已支持能力；业务服务不得修改 Nucleus steering")
		}
	}
	if len(description.GeneratedFreshness) > 0 {
		for _, item := range description.GeneratedFreshness {
			if !item.Fresh {
				risks = append(risks, "生成物可能过期: "+item.Target)
			}
		}
	}
	return uniqueStrings(risks)
}

func capabilityContext(task string, description inspect.Description, capabilities []string) []map[string]any {
	items := make([]map[string]any, 0, len(capabilities))
	for _, capability := range capabilities {
		items = append(items, map[string]any{
			"name":          capability,
			"declared":      hasString(description.Capabilities, capability),
			"module":        strings.TrimPrefix(inspect.CapabilityModule(capability), modulePathPrefix),
			"provider_hint": providerHint(task, capability),
		})
	}
	return items
}

func requestedCapabilities(task string) []string {
	candidates := capcatalog.PlanningNames()
	var capabilities []string
	lowerTask := strings.ToLower(task)
	mongoRequested := containsAny(lowerTask, "mongodb", "mongo", "文档库")
	for _, candidate := range candidates {
		if strings.Contains(lowerTask, candidate) || strings.Contains(task, candidate+"能力") {
			capabilities = append(capabilities, candidate)
		}
	}
	if !mongoRequested && containsAny(lowerTask, "postgres", "postgre", "pg", "mysql", "入库", "持久化", "数据库", "database") {
		capabilities = append(capabilities, "sql")
	}
	if mongoRequested {
		capabilities = append(capabilities, "mongo")
	}
	if containsAny(lowerTask, "kafka", "sarama", "nats", "amqp", "rabbit", "消息", "队列") {
		capabilities = append(capabilities, "mq")
	}
	if containsAny(lowerTask, "prometheus", "metrics", "metric", "指标") {
		capabilities = append(capabilities, "metric")
	}
	if containsAny(lowerTask, "config", "配置", "acm") {
		capabilities = append(capabilities, "config")
	}
	if containsAny(lowerTask, "nacos", "service discovery", "注册", "发现") {
		capabilities = append(capabilities, "discovery")
	}
	if containsAny(lowerTask, "auth", "认证", "鉴权", "权限") {
		capabilities = append(capabilities, "auth")
	}
	if containsAny(lowerTask, "redis") {
		capabilities = append(capabilities, "redis")
	}
	if containsAny(lowerTask, "lock", "锁") {
		capabilities = append(capabilities, "lock")
	}
	if containsAny(lowerTask, "sentinel", "限流", "熔断") {
		capabilities = append(capabilities, "sentinel")
	}
	if containsAny(lowerTask, "sentry", "errortracker", "错误追踪") {
		capabilities = append(capabilities, "errortracker")
	}
	if containsAny(lowerTask, "pyroscope", "profiler", "profile", "性能剖析") {
		capabilities = append(capabilities, "profiler")
	}
	return uniqueStrings(capabilities)
}

func providerHint(task string, capability string) string {
	lowerTask := strings.ToLower(task)
	switch capability {
	case "sql":
		switch {
		case containsAny(lowerTask, "postgres", "postgre", "pg"):
			return "postgres"
		case containsAny(lowerTask, "mysql"):
			return "mysql"
		case containsAny(lowerTask, "gorm"):
			return "gorm"
		default:
			return capcatalog.DefaultProvider(capability)
		}
	case "mongo":
		return "mongo"
	case "redis":
		if containsAny(lowerTask, "goredis", "go-redis") {
			return "goredis"
		}
		return "redis"
	case "mq":
		switch {
		case containsAny(lowerTask, "sarama"):
			return "sarama"
		case containsAny(lowerTask, "nats"):
			return "nats"
		case containsAny(lowerTask, "amqp", "rabbit"):
			return "amqp"
		default:
			return "kafka"
		}
	case "config":
		switch {
		case containsAny(lowerTask, "nacosofficial"):
			return "nacosofficial"
		case containsAny(lowerTask, "nacos"):
			return "nacos"
		case containsAny(lowerTask, "acm"):
			return "acm"
		case containsAny(lowerTask, "kv"):
			return "configkv"
		default:
			return "file"
		}
	case "discovery":
		if containsAny(lowerTask, "nacosofficial") {
			return "nacosofficial"
		}
		return "nacos"
	case "metric":
		if containsAny(lowerTask, "otel") {
			return "otel"
		}
		return "prometheus"
	case "log":
		return "zap"
	case "trace":
		return "otel"
	case "httpclient":
		return "standard"
	case "transport":
		return "netdialer"
	case "auth":
		return "security"
	case "health":
		return "noop"
	case "kv":
		return "kv"
	case "store":
		switch {
		case containsAny(lowerTask, "cache"):
			return "cache"
		case containsAny(lowerTask, "bloom"):
			return "bloom"
		default:
			return "memory"
		}
	case "lock":
		if containsAny(lowerTask, "redis") {
			return "redislock"
		}
		return "memorylock"
	case "sentinel":
		return "sentinel"
	case "errortracker":
		return "sentry"
	case "profiler":
		return "pyroscope"
	default:
		return capcatalog.DefaultProvider(capability)
	}
}

func taskType(task string) string {
	switch {
	case containsAny(task, "grpc", "proto", "rpc"):
		return taskTypeGRPCService
	case containsAny(task, "cap", "能力", "bridge"):
		return taskTypeCapability
	case containsAny(task, "错误码", "error"):
		return taskTypeErrorCatalog
	case containsAny(task, "接口", "endpoint", "http", "api"):
		return taskTypeHTTPEndpoint
	default:
		return taskTypeGeneral
	}
}

func contractFirst(taskType string) bool {
	return taskType == taskTypeHTTPEndpoint || taskType == taskTypeGRPCService || taskType == taskTypeErrorCatalog
}
