package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// validWorkflowNameRegex ensures alphanumeric characters and hyphens only
	validWorkflowNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
)

// ValidateWorkflowName validates a workflow name
// Rules:
// - 1-64 characters
// - Alphanumeric and hyphens only
// - Cannot start or end with hyphen
// - No path traversal sequences
func ValidateWorkflowName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidWorkflowName)
	}

	if len(name) > 64 {
		return fmt.Errorf("%w: name too long (max 64 characters)", ErrInvalidWorkflowName)
	}

	// Check for path traversal first (more specific error message)
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("%w: name cannot contain path traversal sequences", ErrInvalidWorkflowName)
	}

	if !validWorkflowNameRegex.MatchString(name) {
		return fmt.Errorf("%w: must contain only alphanumeric characters and hyphens, and cannot start or end with hyphen", ErrInvalidWorkflowName)
	}

	return nil
}

// ValidateWorkflowType validates a workflow type
func ValidateWorkflowType(wfType WorkflowType) error {
	if wfType != WorkflowTypeFeature && wfType != WorkflowTypeFix {
		return fmt.Errorf("%w: must be 'feature' or 'fix'", ErrInvalidWorkflowType)
	}
	return nil
}

// ValidateDescription validates a workflow description
// Rules:
// - Minimum MinDescriptionLength characters
// - Maximum configurable length (default DefaultMaxDescriptionLength, override via CLAUDE_WORKFLOW_MAX_DESCRIPTION_LENGTH)
// - Cannot be empty
func ValidateDescription(desc string) error {
	if desc == "" {
		return fmt.Errorf("description cannot be empty")
	}

	maxLength := GetMaxDescriptionLength()
	if len(desc) > maxLength {
		overLimit := len(desc) - maxLength
		return fmt.Errorf("description too long: %d characters (max %d characters, %d over limit)", len(desc), maxLength, overLimit)
	}

	return nil
}
