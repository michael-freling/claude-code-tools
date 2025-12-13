package workflow

import (
	"os"
	"strconv"
)

const (
	// DefaultMaxDescriptionLength sets the default maximum length for workflow descriptions.
	// 32KB (32768 bytes) represents approximately 4% of Claude's 200K token context window (~8,000 tokens).
	// This provides a reasonable limit for workflow descriptions while leaving ample room for:
	// - System prompts and instructions
	// - File contents and code context
	// - Response generation and output
	//
	// The limit can be overridden via the CLAUDE_WORKFLOW_MAX_DESCRIPTION_LENGTH environment variable.
	DefaultMaxDescriptionLength = 32768

	// MinDescriptionLength defines the minimum acceptable description length
	MinDescriptionLength = 1

	// EnvMaxDescriptionLength is the environment variable name for configuring the maximum description length
	EnvMaxDescriptionLength = "CLAUDE_WORKFLOW_MAX_DESCRIPTION_LENGTH"
)

// GetMaxDescriptionLength returns the configured maximum description length.
// It checks the CLAUDE_WORKFLOW_MAX_DESCRIPTION_LENGTH environment variable.
// If the environment variable is set and contains a valid positive integer, that value is returned.
// Otherwise, it returns DefaultMaxDescriptionLength.
// Invalid values (non-numeric, zero, or negative) silently fall back to the default.
func GetMaxDescriptionLength() int {
	envValue := os.Getenv(EnvMaxDescriptionLength)
	if envValue == "" {
		return DefaultMaxDescriptionLength
	}

	value, err := strconv.Atoi(envValue)
	if err != nil || value <= 0 {
		return DefaultMaxDescriptionLength
	}

	return value
}
