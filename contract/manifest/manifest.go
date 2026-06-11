package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Manifest 清单
type Manifest struct {
	SchemaVersion string       `yaml:"schema_version" json:"schema_version"` // 架构版本
	Service       Service      `yaml:"service" json:"service"`               // 服务
	AI            AI           `yaml:"ai" json:"ai"`                         // AI 技能
	Nucleus       Nucleus      `yaml:"nucleus" json:"nucleus"`               // Nucleus 核心
	Capabilities  []string     `yaml:"capabilities" json:"capabilities"`     // 能力
	Dependencies  []Dependency `yaml:"dependencies" json:"dependencies"`     // 依赖
	Features      []string     `yaml:"features" json:"features"`             // 功能
}

// Service 服务
type Service struct {
	Name        string            `yaml:"name" json:"name"`                         // 名称
	Version     string            `yaml:"version" json:"version"`                   // 版本
	Env         string            `yaml:"env" json:"env,omitempty"`                 // 环境
	Owner       string            `yaml:"owner" json:"owner,omitempty"`             // 所有者
	Tier        string            `yaml:"tier" json:"tier,omitempty"`               // 层
	Namespace   string            `yaml:"namespace" json:"namespace,omitempty"`     // 命名空间
	Metadata    map[string]string `yaml:"metadata" json:"metadata,omitempty"`       // 元数据
	Description string            `yaml:"description" json:"description,omitempty"` // 描述
}

// Nucleus 核心
type Nucleus struct {
	PlatformURL string                    `yaml:"platform_url" json:"platform_url,omitempty"` // 平台 URL
	Registry    map[string]any            `yaml:"registry" json:"registry,omitempty"`         // 注册表
	Config      map[string]any            `yaml:"config" json:"config,omitempty"`             // 配置
	Providers   map[string]map[string]any `yaml:"providers" json:"providers,omitempty"`       // 提供者
	SQL         map[string]any            `yaml:"sql" json:"sql,omitempty"`                   // SQL
	Mongo       map[string]any            `yaml:"mongo" json:"mongo,omitempty"`               // MongoDB
	Gateway     map[string]any            `yaml:"gateway" json:"gateway,omitempty"`           // 网关
	Trace       map[string]any            `yaml:"trace" json:"trace,omitempty"`               // 追踪
	Log         map[string]any            `yaml:"log" json:"log,omitempty"`                   // 日志
	Metric      map[string]any            `yaml:"metric" json:"metric,omitempty"`             // 指标
}

// Dependency 依赖
type Dependency struct {
	Name     string `yaml:"name" json:"name"`         // 名称
	Contract string `yaml:"contract" json:"contract"` // 合约
	Required bool   `yaml:"required" json:"required"` // 是否必需
}

// AI 技能
type AI struct {
	Intent         string   `yaml:"intent" json:"intent,omitempty"`                   // 意图
	AllowedChanges []string `yaml:"allowed_changes" json:"allowed_changes,omitempty"` // 允许更改
	Readonly       []string `yaml:"readonly" json:"readonly,omitempty"`               // 只读
	Forbidden      []string `yaml:"forbidden" json:"forbidden,omitempty"`             // 禁止
	Generated      []string `yaml:"generated" json:"generated,omitempty"`             // 生成
}

// Load 加载清单
func Load(dir string) (Manifest, error) {
	path := filepath.Join(dir, "nucleus.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse nucleus.yaml: %w", err)
	}
	return manifest, nil
}
