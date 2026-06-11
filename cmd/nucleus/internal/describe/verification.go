package describe

func verificationCommands() []string {
	return []string{
		"nucleus validate --dir .",
		"nucleus lint --dir . --strict",
		"nucleus verify --dir . --json",
	}
}

// verificationContract returns the verification contract.
func verificationContract() map[string]any {
	return map[string]any{
		"commands":        verificationCommands(),
		"pipeline":        verificationPipeline(),
		"result_kind":     "nucleus.verify_result",
		"evidence_schema": "contract/schema/evidence.schema.json",
	}
}

// verificationPipeline returns the verification pipeline.
func verificationPipeline() []map[string]any {
	steps := []struct {
		id      string
		phase   string
		command string
	}{
		{"validate", "validate", "nucleus validate --dir ."},
		{"lint", "lint", "nucleus lint --dir . --strict"},
		{"generated_freshness", "generated_freshness", "nucleus describe --dir . --json"},
		{"tidy", "tidy", "go mod tidy"},
		{"import", "import", "go list ./..."},
		{"build", "build", "go test ./... -run ^$"},
		{"test", "test", "go test ./..."},
	}
	pipeline := make([]map[string]any, 0, len(steps))
	for index, step := range steps {
		pipeline = append(pipeline, map[string]any{
			"id":         step.id,
			"sequence":   index + 1,
			"phase":      step.phase,
			"command":    step.command,
			"schema_ref": "contract/schema/evidence.schema.json",
			"produces":   "nucleus.verify_result",
		})
	}
	return pipeline
}
