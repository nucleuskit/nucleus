package root

import (
	"bytes"
	"strings"
	"testing"
)

func TestMCPCommandIsWiredFromRoot(t *testing.T) {
	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mcp", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute mcp help: %v", err)
	}
	if !strings.Contains(stdout.String(), "--stdio") {
		t.Fatalf("mcp help missing --stdio:\n%s", stdout.String())
	}
}
