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

// ExtractJSON extracts JSON from markdown code blocks
func (p *outputParser) ExtractJSON(output string) (string, error) {
	blocks := p.findJSONBlocks(output)
	if len(blocks) == 0 {
		return "", fmt.Errorf("no JSON blocks found in output: %w", ErrParseJSON)
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

	return "", fmt.Errorf("no valid JSON found in output: %w", ErrParseJSON)
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
		return fmt.Errorf("invalid JSON: %w: %w", err, ErrParseJSON)
	}
	return nil
}
