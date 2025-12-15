package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// OutputParser interface for parsing Claude's output
type OutputParser interface {
	ExtractJSON(output string) (string, error)
	ParsePlan(jsonStr string) (*Plan, error)
	ParseImplementationSummary(jsonStr string) (*ImplementationSummary, error)
	ParseRefactoringSummary(jsonStr string) (*RefactoringSummary, error)
	ParsePRSplitPlan(jsonStr string) (*PRSplitPlan, error)
	ParsePRSplitResult(jsonStr string) (*PRSplitResult, error)
}

// outputParser implements OutputParser interface
type outputParser struct {
	jsonBlockRegex *regexp.Regexp
}

// NewOutputParser creates a new parser
func NewOutputParser() OutputParser {
	return &outputParser{
		jsonBlockRegex: regexp.MustCompile("(?s)```json\\s*\\n(.*?)```"),
	}
}

// claudeJSONResponse represents the JSON envelope from Claude CLI when using --output-format json
type claudeJSONResponse struct {
	Type             string          `json:"type"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
	IsError          bool            `json:"is_error"`
}

// ExtractJSON extracts JSON from output, handling Claude CLI JSON envelope format
func (p *outputParser) ExtractJSON(output string) (string, error) {
	trimmed := strings.TrimSpace(output)

	// Early detection: check for text-only before JSON parsing attempts
	if p.isTextOnlyResponse(trimmed) {
		preview := truncateOutput(output, 500)
		return "", fmt.Errorf("text-only response detected: Claude ignored schema constraint and returned text/markdown instead of required JSON format.\n\nClaude output preview:\n%s\n\n%w", preview, ErrParseJSON)
	}

	// First, try to parse as Claude CLI JSON envelope (from --output-format json)
	if json.Valid([]byte(trimmed)) {
		var envelope claudeJSONResponse
		if err := json.Unmarshal([]byte(trimmed), &envelope); err == nil {
			// Check if this is a Claude CLI JSON envelope with structured_output
			if envelope.Type == "result" && len(envelope.StructuredOutput) > 0 {
				return string(envelope.StructuredOutput), nil
			}
		}
		// If not a Claude envelope, return as-is (direct JSON)
		return trimmed, nil
	}

	// Fall back to looking for markdown code blocks
	blocks := p.findJSONBlocks(output)
	if len(blocks) == 0 {
		preview := truncateOutput(output, 500)
		return "", fmt.Errorf("no JSON blocks found in output.\n\nClaude output preview:\n%s\n\n%w", preview, ErrParseJSON)
	}

	for _, block := range blocks {
		trimmed := strings.TrimSpace(block)
		if trimmed == "" {
			continue
		}

		if json.Valid([]byte(trimmed)) {
			return trimmed, nil
		}
	}

	preview := truncateOutput(output, 500)
	return "", fmt.Errorf("no valid JSON found in output.\n\nClaude output preview:\n%s\n\n%w", preview, ErrParseJSON)
}

// truncateOutput truncates output to maxLen characters with ellipsis
func truncateOutput(output string, maxLen int) string {
	if len(output) == 0 {
		return "(empty output)"
	}
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + fmt.Sprintf("...\n(truncated, showing first %d chars)", maxLen)
}

// ParsePlan parses a Plan from JSON string
func (p *outputParser) ParsePlan(jsonStr string) (*Plan, error) {
	var plan Plan
	if err := p.unmarshalJSON(jsonStr, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	if plan.Summary == "" {
		return nil, fmt.Errorf("plan missing required field 'summary': %w", ErrParseJSON)
	}

	return &plan, nil
}

// ParseImplementationSummary parses an ImplementationSummary from JSON string
func (p *outputParser) ParseImplementationSummary(jsonStr string) (*ImplementationSummary, error) {
	var summary ImplementationSummary
	if err := p.unmarshalJSON(jsonStr, &summary); err != nil {
		return nil, fmt.Errorf("failed to parse implementation summary: %w", err)
	}

	if summary.Summary == "" {
		return nil, fmt.Errorf("implementation summary missing required field 'summary': %w", ErrParseJSON)
	}

	return &summary, nil
}

// ParseRefactoringSummary parses a RefactoringSummary from JSON string
func (p *outputParser) ParseRefactoringSummary(jsonStr string) (*RefactoringSummary, error) {
	var summary RefactoringSummary
	if err := p.unmarshalJSON(jsonStr, &summary); err != nil {
		return nil, fmt.Errorf("failed to parse refactoring summary: %w", err)
	}

	if summary.Summary == "" {
		return nil, fmt.Errorf("refactoring summary missing required field 'summary': %w", ErrParseJSON)
	}

	return &summary, nil
}

// ParsePRSplitPlan parses a PRSplitPlan from JSON string
func (p *outputParser) ParsePRSplitPlan(jsonStr string) (*PRSplitPlan, error) {
	var plan PRSplitPlan
	if err := p.unmarshalJSON(jsonStr, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse PR split plan: %w", err)
	}

	if plan.Summary == "" {
		return nil, fmt.Errorf("PR split plan missing required field 'summary': %w", ErrParseJSON)
	}

	return &plan, nil
}

// ParsePRSplitResult parses a PRSplitResult from JSON string
func (p *outputParser) ParsePRSplitResult(jsonStr string) (*PRSplitResult, error) {
	var result PRSplitResult
	if err := p.unmarshalJSON(jsonStr, &result); err != nil {
		return nil, fmt.Errorf("failed to parse PR split result: %w", err)
	}

	if result.Summary == "" {
		return nil, fmt.Errorf("PR split result missing required field 'summary': %w", ErrParseJSON)
	}

	return &result, nil
}

// findJSONBlocks finds all JSON code blocks in the output
func (p *outputParser) findJSONBlocks(output string) []string {
	matches := p.jsonBlockRegex.FindAllStringSubmatch(output, -1)
	blocks := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 {
			blocks = append(blocks, match[1])
		}
	}

	return blocks
}

// unmarshalJSON unmarshals JSON string into target
func (p *outputParser) unmarshalJSON(jsonStr string, target interface{}) error {
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// isTextOnlyResponse detects when Claude ignored the JSON schema constraint
// and returned natural language instead. Uses heuristics including markdown
// indicators, sentence patterns, and common refusal phrases.
// Note: May have false negatives if text contains { or [ characters.
func (p *outputParser) isTextOnlyResponse(output string) bool {
	// Check for common Claude refusal/explanation patterns
	lowerOutput := strings.ToLower(output)
	refusalPatterns := []string{
		"i cannot",
		"i apologize",
		"i'm unable",
		"unfortunately,",
	}
	for _, pattern := range refusalPatterns {
		if strings.Contains(lowerOutput, pattern) {
			return true
		}
	}

	// Check if the output contains common indicators of text-only responses
	// Look for markdown headers, paragraphs, or sentences without any JSON-like structures

	// If output contains { or [, it might contain JSON-like structures
	if strings.Contains(output, "{") || strings.Contains(output, "[") {
		return false
	}

	// Check for common text-only indicators
	hasMarkdownHeaders := strings.Contains(output, "#")
	hasSentences := strings.Contains(output, ".") && len(output) > 50
	hasMultipleLines := strings.Count(output, "\n") > 2

	// If it has text-like characteristics and no JSON structures, it's likely text-only
	return (hasMarkdownHeaders || hasSentences) && hasMultipleLines
}
