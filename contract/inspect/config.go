package inspect

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// collectConfigKeys 读取配置文件并返回配置键
func collectConfigKeys(dir string) []ConfigKey {
	configDir := filepath.Join(dir, configDirName)
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil
	}
	var keys []ConfigKey
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.Contains(name, configLocalNamePart) || strings.Contains(name, configSecretNamePart) {
			continue
		}
		if !strings.HasSuffix(name, configYAMLExtension) && !strings.HasSuffix(name, configYMLExtension) {
			continue
		}
		path := filepath.Join(configDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var root any
		if err := yaml.Unmarshal(data, &root); err != nil {
			continue
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			rel = path
		}
		keys = append(keys, configKeysFromValue(filepath.ToSlash(rel), "", root)...)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Key != keys[j].Key {
			return keys[i].Key < keys[j].Key
		}
		return keys[i].Source < keys[j].Source
	})
	return keys
}

func configKeysFromValue(source string, prefix string, value any) []ConfigKey {
	switch typed := value.(type) {
	case map[string]any:
		var keys []ConfigKey
		for key, item := range typed {
			child := key
			if prefix != "" {
				child = prefix + configKeySeparator + key
			}
			keys = append(keys, configKeysFromValue(source, child, item)...)
		}
		return keys
	case []any:
		if prefix == "" {
			return nil
		}
		env, fallback := configEnvPlaceholder(typed)
		return []ConfigKey{{Key: prefix, Source: source, Env: env, DefaultValue: fallback, Inferred: false}}
	default:
		if prefix == "" {
			return nil
		}
		env, fallback := configEnvPlaceholder(typed)
		return []ConfigKey{{Key: prefix, Source: source, Env: env, DefaultValue: fallback, Inferred: false}}
	}
}

func configEnvPlaceholder(value any) (string, string) {
	text, ok := value.(string)
	if !ok || !strings.HasPrefix(text, configEnvPrefix) || !strings.HasSuffix(text, configEnvSuffix) {
		if ok {
			return "", text
		}
		return "", ""
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(text, configEnvPrefix), configEnvSuffix)
	name, fallback, hasFallback := strings.Cut(inner, configEnvFallbackSep)
	if !hasFallback {
		return strings.TrimSpace(name), ""
	}
	return strings.TrimSpace(name), fallback
}
