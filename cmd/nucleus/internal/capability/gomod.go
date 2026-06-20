package capability

import (
	"fmt"
	"os"
	"strings"
)

type moduleRequirement struct {
	Path    string
	Version string
}

func updateGoModRequires(path string, requires []moduleRequirement) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("go.mod is required and must be readable")
	}
	text := string(data)
	var missing []moduleRequirement
	for _, item := range requires {
		if strings.TrimSpace(item.Path) == "" || strings.TrimSpace(item.Version) == "" {
			return nil, fmt.Errorf("module requirement path and version are required")
		}
		if !strings.Contains(text, item.Path+" ") {
			missing = append(missing, item)
		}
	}
	if len(missing) == 0 {
		return data, nil
	}
	if strings.Contains(text, "\nrequire (\n") {
		blockStart := strings.Index(text, "\nrequire (\n")
		index := strings.LastIndex(text, "\n)")
		if index > blockStart {
			var builder strings.Builder
			for _, item := range missing {
				builder.WriteString("\t" + item.Path + " " + item.Version + "\n")
			}
			text = text[:index] + "\n" + builder.String() + text[index:]
			return []byte(text), nil
		}
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	var builder strings.Builder
	builder.WriteString(text)
	builder.WriteString("\nrequire (\n")
	for _, item := range missing {
		builder.WriteString("\t" + item.Path + " " + item.Version + "\n")
	}
	builder.WriteString(")\n")
	return []byte(builder.String()), nil
}

func updateGoSumEntries(path string, entries []string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("go.sum must be readable")
	}
	text := string(data)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	for _, entry := range entries {
		if strings.Contains(text, entry+"\n") {
			continue
		}
		text += entry + "\n"
	}
	return []byte(text), nil
}

func postgresGoSumEntries() []string {
	return []string{
		"github.com/lib/pq v1.10.9 h1:YXG7RB+JIjhP29X+OtkiDnYaXQwpS4JEWq7dtCCRUEw=",
		"github.com/lib/pq v1.10.9/go.mod h1:AlVN5x4E4T544tWzH6hKfbfQvm3HdbOxrmggDNAPY9o=",
	}
}
