package workflow

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadExternalPlan loads and validates an external plan.json file
func LoadExternalPlan(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	if err := validatePlan(&plan); err != nil {
		return nil, fmt.Errorf("plan validation failed: %w", err)
	}

	return &plan, nil
}

func validatePlan(plan *Plan) error {
	if plan.Summary == "" {
		return fmt.Errorf("plan.Summary is required")
	}

	if len(plan.Phases) == 0 {
		return fmt.Errorf("plan.Phases is required and must not be empty")
	}

	if len(plan.WorkStreams) == 0 {
		return fmt.Errorf("plan.WorkStreams is required and must not be empty")
	}

	return nil
}
