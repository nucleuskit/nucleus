package inspect

// Description 服务描述
type Description struct {
	FlowGraph *FlowGraph `json:"flow_graph,omitempty"` // 服务流图
}

// FlowGraph 服务流图
type FlowGraph struct {
	SchemaVersion string `json:"schema_version"` // 架构版本
}

// BuildFlowGraphFromDir 创建服务流图
func BuildFlowGraphFromDir(dir string) (FlowGraph, error) {
	return FlowGraph{}, nil
}

// Describe 描述服务
func Describe(dir string) (Description, error) {
	return Description{}, nil
}
