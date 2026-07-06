package plan

import (
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/capvocab"
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
		return []string{"nucleus.yaml", ".nucleus/decisions/*.json", "docs/**"}
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
		return []string{commandDescribeFlow, commandValidate, commandLintStrict, commandVerifyJSON}
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
		risks = append(risks, "能力接入必须声明抽象接口、decision evidence、影响面和验证命令")
		risks = append(risks, "provider/library/driver 只能写入 decision evidence，不能写入 manifest")
	}
	for _, capability := range requestedCapabilities {
		if !hasString(description.Capabilities, capability) {
			risks = append(risks, "manifest 未声明能力 "+capability)
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
			"name":                capability,
			"declared":            hasString(description.Capabilities, capability),
			"decision_required":   true,
			"decision_schema_ref": decisionSchemaRef,
			"provider_selection":  "decision_only",
			"implementation":      "user_or_ai_defined_interface",
		})
	}
	return items
}

func requestedCapabilities(task string) []string {
	return capvocab.MatchTask(task)
}

func taskType(task string) string {
	switch {
	case containsAny(task, "grpc", "proto", "rpc"):
		return taskTypeGRPCService
	case containsAny(task, "cap", "能力"):
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
