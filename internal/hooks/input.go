package hooks

import (
	"encoding/json"
	"fmt"
	"io"
)

// ToolInput represents the input to a tool from Claude Code.
type ToolInput struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	parsed    map[string]interface{}
}

// ParseToolInput reads and parses tool input JSON from a reader.
func ParseToolInput(reader io.Reader) (*ToolInput, error) {
	var input ToolInput
	if err := json.NewDecoder(reader).Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	if input.ToolName == "" {
		return nil, fmt.Errorf("tool_name is required")
	}

	if len(input.ToolInput) > 0 {
		var parsed map[string]interface{}
		if err := json.Unmarshal(input.ToolInput, &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse tool_input: %w", err)
		}
		input.parsed = parsed
	}

	return &input, nil
}

// GetStringArg retrieves a string argument from the tool input.
// Returns the value and true if found, empty string and false if not found.
func (t *ToolInput) GetStringArg(name string) (string, bool) {
	if t.parsed == nil {
		return "", false
	}

	value, ok := t.parsed[name]
	if !ok {
		return "", false
	}

	strValue, ok := value.(string)
	if !ok {
		return "", false
	}

	return strValue, true
}

// GetBoolArg retrieves a boolean argument from the tool input.
// Returns the value and true if found, false and false if not found.
func (t *ToolInput) GetBoolArg(name string) (bool, bool) {
	if t.parsed == nil {
		return false, false
	}

	value, ok := t.parsed[name]
	if !ok {
		return false, false
	}

	boolValue, ok := value.(bool)
	if !ok {
		return false, false
	}

	return boolValue, true
}
