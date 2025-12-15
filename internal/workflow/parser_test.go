package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutputParser(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates parser successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewOutputParser()
			assert.NotNil(t, got)
		})
	}
}

func TestOutputParser_ExtractJSON(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantErr     bool
		errContains string
		wantContain string
	}{
		{
			name:        "extracts structured_output from Claude CLI JSON envelope",
			output:      `{"type":"result","subtype":"success","is_error":false,"duration_ms":5328,"result":"","structured_output":{"summary":"test plan from envelope","contextType":"feature"}}`,
			wantErr:     false,
			wantContain: "test plan from envelope",
		},
		{
			name: "extracts direct JSON without markdown",
			output: `{
  "summary": "test plan",
  "contextType": "feature"
}`,
			wantErr:     false,
			wantContain: "test plan",
		},
		{
			name: "extracts JSON from markdown code block",
			output: `Here is the plan:

` + "```json" + `
{
  "summary": "test plan",
  "contextType": "feature"
}
` + "```" + `

That's the plan.`,
			wantErr:     false,
			wantContain: "test plan",
		},
		{
			name: "handles multiple JSON blocks and returns first valid",
			output: `First block is invalid:

` + "```json" + `
{
  "invalid": "missing brace"
` + "```" + `

Second block is valid:

` + "```json" + `
{
  "summary": "valid plan"
}
` + "```",
			wantErr:     false,
			wantContain: "valid plan",
		},
		{
			name:        "returns error when no JSON blocks found with output preview",
			output:      "No JSON here, just plain text",
			wantErr:     true,
			errContains: "Claude output preview",
		},
		{
			name: "returns error when JSON blocks are all invalid with output preview",
			output: `Invalid JSON:

` + "```json" + `
{invalid json}
` + "```",
			wantErr:     true,
			errContains: "Claude output preview",
		},
		{
			name: "skips empty JSON block and uses valid one",
			output: `Empty block:

` + "```json" + `

` + "```" + `

Valid block:

` + "```json" + `
{"summary": "test"}
` + "```",
			wantErr:     false,
			wantContain: "summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			got, err := parser.ExtractJSON(tt.output)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, got, tt.wantContain)
		})
	}
}

func TestOutputParser_ExtractJSON_FromFile(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantErr     bool
		errContains string
		wantContain string
	}{
		{
			name:        "extracts JSON from success output",
			filename:    "claude_output_success.txt",
			wantErr:     false,
			wantContain: "JWT authentication",
		},
		{
			name:        "returns error for output without JSON with preview",
			filename:    "claude_output_no_json.txt",
			wantErr:     true,
			errContains: "Claude output preview",
		},
		{
			name:        "extracts valid JSON from multiple blocks",
			filename:    "claude_output_multiple_blocks.txt",
			wantErr:     false,
			wantContain: "JWT authentication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("testdata", tt.filename))
			require.NoError(t, err)

			parser := NewOutputParser()
			got, err := parser.ExtractJSON(string(content))

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, got, tt.wantContain)
		})
	}
}

func TestOutputParser_ParsePlan(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantErr     bool
		errContains string
		wantSummary string
	}{
		{
			name:        "parses valid plan JSON",
			filename:    "valid_plan.json",
			wantErr:     false,
			wantSummary: "Add JWT authentication to the application",
		},
		{
			name:        "returns error for invalid plan JSON",
			filename:    "invalid_plan.json",
			wantErr:     true,
			errContains: "missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("testdata", tt.filename))
			require.NoError(t, err)

			parser := NewOutputParser()
			got, err := parser.ParsePlan(string(content))

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantSummary, got.Summary)
			assert.Equal(t, "feature", got.ContextType)
		})
	}
}

func TestOutputParser_ParsePlan_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		jsonStr     string
		wantErr     bool
		errContains string
	}{
		{
			name:        "returns error for malformed JSON",
			jsonStr:     "{invalid json}",
			wantErr:     true,
			errContains: "invalid JSON",
		},
		{
			name:        "returns error for empty summary",
			jsonStr:     `{"summary": "", "contextType": "feature"}`,
			wantErr:     true,
			errContains: "missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			got, err := parser.ParsePlan(tt.jsonStr)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestOutputParser_ParseImplementationSummary(t *testing.T) {
	tests := []struct {
		name        string
		jsonStr     string
		wantErr     bool
		errContains string
		wantSummary string
	}{
		{
			name: "parses valid implementation summary",
			jsonStr: `{
				"filesChanged": ["file1.go", "file2.go"],
				"linesAdded": 100,
				"linesRemoved": 50,
				"testsAdded": 5,
				"summary": "Implemented authentication feature"
			}`,
			wantErr:     false,
			wantSummary: "Implemented authentication feature",
		},
		{
			name: "returns error for missing summary",
			jsonStr: `{
				"filesChanged": ["file1.go"],
				"linesAdded": 10
			}`,
			wantErr:     true,
			errContains: "missing required field",
		},
		{
			name:        "returns error for malformed JSON",
			jsonStr:     "{invalid}",
			wantErr:     true,
			errContains: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			got, err := parser.ParseImplementationSummary(tt.jsonStr)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantSummary, got.Summary)
		})
	}
}

func TestOutputParser_ParseRefactoringSummary(t *testing.T) {
	tests := []struct {
		name        string
		jsonStr     string
		wantErr     bool
		errContains string
		wantSummary string
	}{
		{
			name: "parses valid refactoring summary",
			jsonStr: `{
				"filesChanged": ["file1.go"],
				"improvementsMade": ["Improved error handling", "Added tests"],
				"summary": "Refactored authentication code"
			}`,
			wantErr:     false,
			wantSummary: "Refactored authentication code",
		},
		{
			name: "returns error for missing summary",
			jsonStr: `{
				"filesChanged": ["file1.go"]
			}`,
			wantErr:     true,
			errContains: "missing required field",
		},
		{
			name:        "returns error for malformed JSON",
			jsonStr:     "not json",
			wantErr:     true,
			errContains: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			got, err := parser.ParseRefactoringSummary(tt.jsonStr)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantSummary, got.Summary)
		})
	}
}

func TestOutputParser_ParsePRSplitPlan(t *testing.T) {
	tests := []struct {
		name         string
		jsonStr      string
		wantErr      bool
		errContains  string
		wantStrategy SplitStrategy
		wantSummary  string
	}{
		{
			name: "parses valid PR split plan with commits strategy",
			jsonStr: `{
				"strategy": "commits",
				"parentTitle": "Parent PR Title",
				"parentDescription": "Parent PR Description",
				"childPRs": [
					{
						"title": "Child PR 1",
						"description": "First child PR",
						"commits": ["abc123", "def456"]
					},
					{
						"title": "Child PR 2",
						"description": "Second child PR",
						"commits": ["ghi789"]
					}
				],
				"summary": "Split into 2 PRs by commits"
			}`,
			wantErr:      false,
			wantStrategy: SplitByCommits,
			wantSummary:  "Split into 2 PRs by commits",
		},
		{
			name: "parses valid PR split plan with files strategy",
			jsonStr: `{
				"strategy": "files",
				"parentTitle": "Parent PR Title",
				"parentDescription": "Parent PR Description",
				"childPRs": [
					{
						"title": "Child PR 1",
						"description": "First child PR",
						"files": ["file1.go", "file2.go"]
					},
					{
						"title": "Child PR 2",
						"description": "Second child PR",
						"files": ["file3.go"]
					}
				],
				"summary": "Split into 2 PRs by files"
			}`,
			wantErr:      false,
			wantStrategy: SplitByFiles,
			wantSummary:  "Split into 2 PRs by files",
		},
		{
			name: "returns error for missing summary",
			jsonStr: `{
				"strategy": "commits",
				"parentTitle": "Parent",
				"parentDescription": "Description",
				"childPRs": []
			}`,
			wantErr:     true,
			errContains: "missing required field",
		},
		{
			name:        "returns error for empty object",
			jsonStr:     "{}",
			wantErr:     true,
			errContains: "missing required field",
		},
		{
			name:        "returns error for invalid JSON syntax",
			jsonStr:     "{invalid",
			wantErr:     true,
			errContains: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			got, err := parser.ParsePRSplitPlan(tt.jsonStr)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantStrategy, got.Strategy)
			assert.Equal(t, tt.wantSummary, got.Summary)
		})
	}
}

func TestOutputParser_ParsePRSplitResult(t *testing.T) {
	tests := []struct {
		name        string
		jsonStr     string
		wantErr     bool
		errContains string
		wantSummary string
	}{
		{
			name: "parses valid PR split result",
			jsonStr: `{
				"parentPR": {
					"number": 123,
					"url": "https://github.com/repo/pull/123",
					"title": "Parent PR",
					"description": "Parent description"
				},
				"childPRs": [
					{
						"number": 124,
						"url": "https://github.com/repo/pull/124",
						"title": "Child PR 1",
						"description": "Child description"
					}
				],
				"summary": "Split PR into 2 parts"
			}`,
			wantErr:     false,
			wantSummary: "Split PR into 2 parts",
		},
		{
			name: "returns error for missing summary",
			jsonStr: `{
				"parentPR": {
					"number": 123
				}
			}`,
			wantErr:     true,
			errContains: "missing required field",
		},
		{
			name:        "returns error for empty object missing summary",
			jsonStr:     "{}",
			wantErr:     true,
			errContains: "missing required field",
		},
		{
			name:        "returns error for invalid JSON syntax",
			jsonStr:     "{invalid json",
			wantErr:     true,
			errContains: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			got, err := parser.ParsePRSplitResult(tt.jsonStr)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantSummary, got.Summary)
		})
	}
}

func TestOutputParser_findJSONBlocks(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantCount int
	}{
		{
			name: "finds single JSON block",
			output: `Text before
` + "```json" + `
{"key": "value"}
` + "```" + `
Text after`,
			wantCount: 1,
		},
		{
			name: "finds multiple JSON blocks",
			output: `First block:
` + "```json" + `
{"key1": "value1"}
` + "```" + `

Second block:
` + "```json" + `
{"key2": "value2"}
` + "```",
			wantCount: 2,
		},
		{
			name:      "finds no JSON blocks",
			output:    "No JSON blocks here",
			wantCount: 0,
		},
		{
			name: "handles JSON block with newlines",
			output: `Complex JSON:
` + "```json" + `
{
  "key": "value",
  "nested": {
    "field": "data"
  }
}
` + "```",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			p, ok := parser.(*outputParser)
			require.True(t, ok)

			got := p.findJSONBlocks(tt.output)
			assert.Len(t, got, tt.wantCount)
		})
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		maxLen   int
		expected string
	}{
		{
			name:     "returns empty output message for empty string",
			output:   "",
			maxLen:   100,
			expected: "(empty output)",
		},
		{
			name:     "returns full output when under max length",
			output:   "short output",
			maxLen:   100,
			expected: "short output",
		},
		{
			name:     "truncates output when over max length",
			output:   "this is a very long output that should be truncated",
			maxLen:   20,
			expected: "this is a very long ...\n(truncated, showing first 20 chars)",
		},
		{
			name:     "returns exact length output unchanged",
			output:   "exact",
			maxLen:   5,
			expected: "exact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateOutput(tt.output, tt.maxLen)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestOutputParser_unmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonStr     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "unmarshals valid JSON",
			jsonStr: `{"summary": "test"}`,
			wantErr: false,
		},
		{
			name:        "returns error for invalid JSON",
			jsonStr:     "not json",
			wantErr:     true,
			errContains: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			p, ok := parser.(*outputParser)
			require.True(t, ok)

			var target Plan
			err := p.unmarshalJSON(tt.jsonStr, &target)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestOutputParser_isTextOnlyResponse(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name: "detects text with markdown headers and multiple lines",
			output: `# Summary

This is a text-only response that explains the task.
It has multiple lines and paragraphs.

## Details

More information here.`,
			want: true,
		},
		{
			name: "detects pure sentences without JSON",
			output: `This is a comprehensive analysis of the feature request.
The implementation would require several components to work together.
Each component has its own set of dependencies and requirements.
Additional details are provided here.`,
			want: true,
		},
		{
			name:   "returns false for text containing opening brace",
			output: "This text has a { character in it.\nIt should not be detected as text-only.",
			want:   false,
		},
		{
			name:   "returns false for text containing opening bracket",
			output: "This text has a [ character in it.\nIt should not be detected as text-only.",
			want:   false,
		},
		{
			name:   "returns false for short text",
			output: "Short text.",
			want:   false,
		},
		{
			name:   "returns false for empty text",
			output: "",
			want:   false,
		},
		{
			name:   "returns false for minimal text with few lines",
			output: "Line 1\nLine 2",
			want:   false,
		},
		{
			name: "returns false for text without sentences or headers",
			output: `word1
word2
word3
word4`,
			want: false,
		},
		{
			name: "detects text with periods and multiple lines but no JSON",
			output: `First sentence here.
Second sentence continues.
Third line with more text.
Fourth line completes the response.`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewOutputParser()
			p, ok := parser.(*outputParser)
			require.True(t, ok)

			got := p.isTextOnlyResponse(tt.output)
			assert.Equal(t, tt.want, got)
		})
	}
}
