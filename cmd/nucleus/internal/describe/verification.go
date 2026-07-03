package describe

func verificationCommands() []string {
	return []string{
		commandVerifyJSON,
	}
}

// verificationContract returns the verification contract.
func verificationContract() map[string]any {
	return map[string]any{
		verificationFieldCommands:       verificationCommands(),
		verificationFieldPipeline:       verificationPipeline(),
		verificationFieldProjectSource:  verificationProjectCommandSource,
		verificationFieldResultKind:     verificationResultKind,
		verificationFieldEvidenceSchema: verificationEvidenceSchema,
		verificationFieldOptional:       optionalEvidence(),
	}
}

// verificationPipeline returns the verification pipeline.
func verificationPipeline() []map[string]any {
	steps := []struct {
		id      string
		phase   string
		command string
	}{
		{phaseValidate, phaseValidate, commandValidate},
		{phaseLint, phaseLint, commandLintStrict},
		{phaseDecision, phaseDecision, commandDecisionValidate},
		{phaseGeneratedFreshness, phaseGeneratedFreshness, commandDescribeJSON},
	}
	pipeline := make([]map[string]any, 0, len(steps))
	for index, step := range steps {
		pipeline = append(pipeline, map[string]any{
			pipelineFieldID:        step.id,
			pipelineFieldSequence:  index + 1,
			pipelineFieldPhase:     step.phase,
			pipelineFieldCommand:   step.command,
			pipelineFieldSchemaRef: verificationEvidenceSchema,
			pipelineFieldProduces:  verificationResultKind,
		})
	}
	return pipeline
}

func optionalEvidence() []map[string]any {
	return []map[string]any{
		{
			pipelineFieldID:        "scenario_plan",
			pipelineFieldPhase:     phaseScenario,
			pipelineFieldCommand:   commandScenarioPlanJSON,
			pipelineFieldSchemaRef: "",
			pipelineFieldProduces:  "nucleus.scenario_plan_result",
			"required":             false,
		},
		{
			pipelineFieldID:        "http_scenario",
			pipelineFieldPhase:     phaseScenario,
			pipelineFieldCommand:   commandScenarioRunJSON,
			pipelineFieldSchemaRef: verificationEvidenceSchema,
			pipelineFieldProduces:  evidenceKindHTTPScenario,
			"required":             false,
		},
	}
}
