package generator

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successfully loads all templates",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEngine()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotNil(t, got.templates)
			assert.NotNil(t, got.templateNames)

			// Verify all item types have templates loaded
			itemTypes := []ItemType{ItemTypeSkill, ItemTypeAgent, ItemTypeCommand, ItemTypeRule}
			for _, itemType := range itemTypes {
				assert.Contains(t, got.templates, itemType)
				assert.Contains(t, got.templateNames, itemType)
			}

			// Verify rules config is loaded
			assert.NotNil(t, got.rulesConfig)
		})
	}
}

func TestNewEngineWithFS(t *testing.T) {
	tests := []struct {
		name        string
		fsys        fs.FS
		wantErr     bool
		errContains string
	}{
		{
			name:    "successfully loads embedded templates",
			fsys:    templatesFS,
			wantErr: false,
		},
		{
			name: "returns error when template has invalid syntax",
			fsys: fstest.MapFS{
				"prompts/skills/invalid.tmpl": &fstest.MapFile{
					Data: []byte("{{invalid template syntax"),
				},
			},
			wantErr:     true,
			errContains: "failed to load templates for skill",
		},
		{
			name: "returns error when partials file has invalid syntax",
			fsys: fstest.MapFS{
				"prompts/skills/_partials.tmpl": &fstest.MapFile{
					Data: []byte("{{define \"partial\"}}{{invalid"),
				},
			},
			wantErr:     true,
			errContains: "failed to load templates for skill",
		},
		{
			name: "successfully loads templates ignoring non-template files",
			fsys: fstest.MapFS{
				"prompts/skills/valid.tmpl": &fstest.MapFile{
					Data: []byte("Valid template content"),
				},
				"prompts/skills/README.md": &fstest.MapFile{
					Data: []byte("This is a readme"),
				},
				"prompts/skills/subdir/nested.tmpl": &fstest.MapFile{
					Data: []byte("Nested template"),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEngineWithFS(tt.fsys)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotNil(t, got.templates)
			assert.NotNil(t, got.templateNames)

			// Verify all item types have templates loaded
			itemTypes := []ItemType{ItemTypeSkill, ItemTypeAgent, ItemTypeCommand, ItemTypeRule}
			for _, itemType := range itemTypes {
				assert.Contains(t, got.templates, itemType)
				assert.Contains(t, got.templateNames, itemType)
			}

			// Verify rules config is loaded
			assert.NotNil(t, got.rulesConfig)
		})
	}
}

func TestEngine_List(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	tests := []struct {
		name     string
		itemType ItemType
		wantLen  int
	}{
		{
			name:     "list agents returns multiple templates",
			itemType: ItemTypeAgent,
			wantLen:  6,
		},
		{
			name:     "list commands returns multiple templates",
			itemType: ItemTypeCommand,
			wantLen:  6,
		},
		{
			name:     "list skills returns multiple templates",
			itemType: ItemTypeSkill,
			wantLen:  2,
		},
		{
			name:     "list rules returns multiple templates",
			itemType: ItemTypeRule,
			wantLen:  3,
		},
		{
			name:     "list unknown type returns empty slice",
			itemType: ItemType("unknown"),
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.List(tt.itemType)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestEngine_List_ValidateKnownTemplates(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	tests := []struct {
		name         string
		itemType     ItemType
		wantContains []string
	}{
		{
			name:     "agents includes known templates",
			itemType: ItemTypeAgent,
			wantContains: []string{
				"architecture-reviewer",
				"code-reviewer",
				"github-actions-workflow-engineer",
				"kubernetes-engineer",
				"software-architect",
				"software-engineer",
			},
		},
		{
			name:     "commands includes known templates",
			itemType: ItemTypeCommand,
			wantContains: []string{
				"feature",
				"fix",
				"document-guideline-monorepo",
				"refactor",
				"document-guideline",
				"split-pr",
			},
		},
		{
			name:     "skills includes known templates",
			itemType: ItemTypeSkill,
			wantContains: []string{
				"coding",
				"ci-error-fix",
			},
		},
		{
			name:     "rules includes known templates",
			itemType: ItemTypeRule,
			wantContains: []string{
				"coding-guidelines",
				"golang",
				"typescript",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.List(tt.itemType)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestEngine_Generate_Success(t *testing.T) {
	engine, err := NewEngine()
	require.NoError(t, err)

	tests := []struct {
		name         string
		itemType     ItemType
		templateName string
		wantContains string
	}{
		{
			name:         "generates coding skill template",
			itemType:     ItemTypeSkill,
			templateName: "coding",
			wantContains: "Coding skill:",
		},
		{
			name:         "generates ci-error-fix skill template",
			itemType:     ItemTypeSkill,
			templateName: "ci-error-fix",
			wantContains: "CI error",
		},
		{
			name:         "generates code-reviewer agent template",
			itemType:     ItemTypeAgent,
			templateName: "code-reviewer",
			wantContains: "code",
		},
		{
			name:         "generates feature command template",
			itemType:     ItemTypeCommand,
			templateName: "feature",
			wantContains: "feature",
		},
		{
			name:         "generates coding-guidelines rule template",
			itemType:     ItemTypeRule,
			templateName: "coding-guidelines",
			wantContains: "Coding Guidelines",
		},
		{
			name:         "generates golang rule template",
			itemType:     ItemTypeRule,
			templateName: "golang",
			wantContains: "Go Coding Guidelines",
		},
		{
			name:         "generates typescript rule template",
			itemType:     ItemTypeRule,
			templateName: "typescript",
			wantContains: "TypeScript Coding Guidelines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Generate(tt.itemType, tt.templateName)
			require.NoError(t, err)
			assert.NotEmpty(t, got)
			assert.Contains(t, got, tt.wantContains)
		})
	}
}

func TestEngine_Generate_Errors(t *testing.T) {
	tests := []struct {
		name         string
		fsys         fs.FS
		itemType     ItemType
		templateName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "returns error for non-existent template",
			fsys:         nil,
			itemType:     ItemTypeSkill,
			templateName: "non-existent-template",
			wantErr:      true,
			errContains:  "not found",
		},
		{
			name:         "returns error for non-existent agent template",
			fsys:         nil,
			itemType:     ItemTypeAgent,
			templateName: "invalid-agent",
			wantErr:      true,
			errContains:  "not found",
		},
		{
			name:         "returns error for non-existent command template",
			fsys:         nil,
			itemType:     ItemTypeCommand,
			templateName: "invalid-command",
			wantErr:      true,
			errContains:  "not found",
		},
		{
			name: "returns error when template execution fails",
			fsys: fstest.MapFS{
				"prompts/skills/broken.tmpl": &fstest.MapFile{
					Data: []byte("{{.NonExistentField}}"),
				},
			},
			itemType:     ItemTypeSkill,
			templateName: "broken",
			wantErr:      true,
			errContains:  "failed to execute template",
		},
		{
			name:         "returns error for unknown item type",
			fsys:         nil,
			itemType:     ItemType("unknown"),
			templateName: "test",
			wantErr:      true,
			errContains:  "no templates found for type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var engine *Engine
			var err error
			if tt.fsys != nil {
				engine, err = NewEngineWithFS(tt.fsys)
			} else {
				engine, err = NewEngine()
			}
			require.NoError(t, err)

			got, err := engine.Generate(tt.itemType, tt.templateName)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, got)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLoadRuleMetadata(t *testing.T) {
	tests := []struct {
		name        string
		fsys        fs.FS
		wantErr     bool
		errContains string
		validate    func(t *testing.T, config *RulesConfig)
	}{
		{
			name: "successfully loads metadata file",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`default_rules:
  - rule1
  - rule2
rules:
  rule1:
    name: "Rule 1"
    description: "First rule"
    filename: ".claude/rules/rule1.md"
    paths:
      - "**/*.go"
  rule2:
    name: "Rule 2"
    description: "Second rule"
    filename: ".claude/rules/rule2.md"
    paths:
      - "**/*.ts"
`),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, config *RulesConfig) {
				require.NotNil(t, config)
				assert.Len(t, config.DefaultRules, 2)
				assert.Contains(t, config.DefaultRules, "rule1")
				assert.Contains(t, config.DefaultRules, "rule2")
				assert.Len(t, config.Rules, 2)
				assert.Equal(t, "Rule 1", config.Rules["rule1"].Name)
				assert.Equal(t, "First rule", config.Rules["rule1"].Description)
				assert.Equal(t, ".claude/rules/rule1.md", config.Rules["rule1"].Filename)
				assert.Equal(t, []string{"**/*.go"}, config.Rules["rule1"].Paths)
			},
		},
		{
			name:    "returns empty config when metadata file does not exist",
			fsys:    fstest.MapFS{},
			wantErr: false,
			validate: func(t *testing.T, config *RulesConfig) {
				require.NotNil(t, config)
				assert.Empty(t, config.DefaultRules)
				assert.Empty(t, config.Rules)
			},
		},
		{
			name: "returns error when metadata has invalid YAML",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte("invalid: yaml: syntax: ["),
				},
			},
			wantErr:     true,
			errContains: "failed to parse metadata file",
		},
		{
			name: "handles metadata with only default_rules",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`default_rules:
  - rule1
`),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, config *RulesConfig) {
				require.NotNil(t, config)
				assert.Len(t, config.DefaultRules, 1)
				assert.Contains(t, config.DefaultRules, "rule1")
				assert.NotNil(t, config.Rules)
				assert.Empty(t, config.Rules)
			},
		},
		{
			name: "handles metadata with only rules",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`rules:
  rule1:
    name: "Rule 1"
    description: "First rule"
    filename: ".claude/rules/rule1.md"
    paths:
      - "**/*.go"
`),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, config *RulesConfig) {
				require.NotNil(t, config)
				assert.NotNil(t, config.DefaultRules)
				assert.Empty(t, config.DefaultRules)
				assert.Len(t, config.Rules, 1)
			},
		},
		{
			name: "handles empty metadata file",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(""),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, config *RulesConfig) {
				require.NotNil(t, config)
				assert.NotNil(t, config.DefaultRules)
				assert.NotNil(t, config.Rules)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadRuleMetadata(tt.fsys)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestEngine_GetRulesConfig(t *testing.T) {
	tests := []struct {
		name     string
		fsys     fs.FS
		validate func(t *testing.T, config *RulesConfig)
	}{
		{
			name: "returns loaded rules config from embedded FS",
			fsys: nil,
			validate: func(t *testing.T, config *RulesConfig) {
				require.NotNil(t, config)
				assert.NotNil(t, config.DefaultRules)
				assert.NotNil(t, config.Rules)
			},
		},
		{
			name: "returns rules config with metadata",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`default_rules:
  - test-rule
rules:
  test-rule:
    name: "Test Rule"
    description: "A test rule"
    filename: ".claude/rules/test.md"
    paths:
      - "**/*.go"
`),
				},
			},
			validate: func(t *testing.T, config *RulesConfig) {
				require.NotNil(t, config)
				assert.Len(t, config.DefaultRules, 1)
				assert.Contains(t, config.DefaultRules, "test-rule")
				assert.Len(t, config.Rules, 1)
				assert.Equal(t, "Test Rule", config.Rules["test-rule"].Name)
			},
		},
		{
			name: "returns empty config when no metadata exists",
			fsys: fstest.MapFS{},
			validate: func(t *testing.T, config *RulesConfig) {
				require.NotNil(t, config)
				assert.Empty(t, config.DefaultRules)
				assert.Empty(t, config.Rules)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var engine *Engine
			var err error
			if tt.fsys != nil {
				engine, err = NewEngineWithFS(tt.fsys)
			} else {
				engine, err = NewEngine()
			}
			require.NoError(t, err)

			got := engine.GetRulesConfig()
			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

func TestPathsToYAML(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{
			name:  "empty paths returns empty string",
			paths: []string{},
			want:  "",
		},
		{
			name:  "single path",
			paths: []string{"**/*.go"},
			want:  "**/*.go",
		},
		{
			name:  "multiple paths",
			paths: []string{"**/*.go", "**/*.ts"},
			want:  "**/*.go, **/*.ts",
		},
		{
			name:  "paths with special characters",
			paths: []string{"**/*.{ts,tsx}", "src/**/*.py"},
			want:  "**/*.{ts,tsx}, src/**/*.py",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathsToYAML(tt.paths)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEngine_Generate_RealRuleOutput(t *testing.T) {
	tests := []struct {
		name           string
		ruleName       string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:     "golang rule with paths",
			ruleName: "golang",
			wantContains: []string{
				"---",
				"paths: **/*.go",
				"# Go Coding Guidelines",
			},
		},
		{
			name:     "coding-guidelines rule without paths",
			ruleName: "coding-guidelines",
			wantContains: []string{
				"# Coding Guidelines",
			},
			wantNotContain: []string{
				"---",
				"paths:",
			},
		},
		{
			name:     "typescript rule with multiple paths",
			ruleName: "typescript",
			wantContains: []string{
				"---",
				"paths: **/*.ts, **/*.tsx",
				"# TypeScript Coding Guidelines",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine()
			require.NoError(t, err)

			output, err := engine.Generate(ItemTypeRule, tt.ruleName)
			require.NoError(t, err)

			t.Logf("Generated %s rule output:\n%s", tt.ruleName, output)

			for _, want := range tt.wantContains {
				assert.Contains(t, output, want)
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, output, notWant)
			}
		})
	}
}

func TestEngine_GetDefaultRules(t *testing.T) {
	tests := []struct {
		name string
		fsys fs.FS
		want []string
	}{
		{
			name: "returns default rules from metadata",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`default_rules:
  - rule1
  - rule2
rules:
  rule1:
    name: "Rule 1"
    description: "First rule"
    filename: ".claude/rules/rule1.md"
  rule2:
    name: "Rule 2"
    description: "Second rule"
    filename: ".claude/rules/rule2.md"
`),
				},
			},
			want: []string{"rule1", "rule2"},
		},
		{
			name: "returns empty slice when no default rules",
			fsys: fstest.MapFS{},
			want: []string{},
		},
		{
			name: "returns empty slice when metadata has no default_rules",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`rules:
  rule1:
    name: "Rule 1"
`),
				},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngineWithFS(tt.fsys)
			require.NoError(t, err)

			got := engine.GetDefaultRules()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEngine_GenerateRuleWithOptions(t *testing.T) {
	tests := []struct {
		name         string
		fsys         fs.FS
		ruleName     string
		opts         GenerateOptions
		wantContains []string
		wantErr      bool
		errContains  string
	}{
		{
			name: "generates rule with custom paths",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`rules:
  golang:
    name: "Go Guidelines"
    description: "Go coding standards"
    filename: ".claude/rules/golang.md"
    paths:
      - "**/*.go"
`),
				},
				"prompts/rules/_partials.tmpl": &fstest.MapFile{
					Data: []byte(`{{define "YAML_FRONTMATTER"}}---
{{- if .Paths}}
paths: {{pathsToYAML .Paths}}
{{- end}}
---
{{end}}`),
				},
				"prompts/rules/golang.tmpl": &fstest.MapFile{
					Data: []byte(`{{- template "YAML_FRONTMATTER" . -}}
# {{.Title}}
{{.Description}}`),
				},
			},
			ruleName: "golang",
			opts: GenerateOptions{
				Paths: []string{"src/**/*.go", "pkg/**/*.go"},
			},
			wantContains: []string{
				"---",
				"paths: src/**/*.go, pkg/**/*.go",
				"# Go Guidelines",
			},
			wantErr: false,
		},
		{
			name: "generates rule with default paths when no custom paths",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`rules:
  golang:
    name: "Go Guidelines"
    description: "Go coding standards"
    filename: ".claude/rules/golang.md"
    paths:
      - "**/*.go"
`),
				},
				"prompts/rules/_partials.tmpl": &fstest.MapFile{
					Data: []byte(`{{define "YAML_FRONTMATTER"}}---
{{- if .Paths}}
paths: {{pathsToYAML .Paths}}
{{- end}}
---
{{end}}`),
				},
				"prompts/rules/golang.tmpl": &fstest.MapFile{
					Data: []byte(`{{- template "YAML_FRONTMATTER" . -}}
# {{.Title}}
{{.Description}}`),
				},
			},
			ruleName: "golang",
			opts:     GenerateOptions{},
			wantContains: []string{
				"---",
				"paths: **/*.go",
				"# Go Guidelines",
			},
			wantErr: false,
		},
		{
			name:        "returns error for non-existent rule",
			fsys:        fstest.MapFS{},
			ruleName:    "non-existent",
			opts:        GenerateOptions{},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngineWithFS(tt.fsys)
			require.NoError(t, err)

			got, err := engine.GenerateRuleWithOptions(tt.ruleName, tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestEngine_InitRulesDirectory(t *testing.T) {
	tests := []struct {
		name        string
		fsys        fs.FS
		dir         string
		rules       []string
		force       bool
		setupFiles  map[string]string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, dir string)
	}{
		{
			name: "creates directory and writes rule files",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`rules:
  golang:
    name: "Go Guidelines"
    description: "Go coding standards"
    filename: "golang.md"
  common:
    name: "Common Guidelines"
    description: "General guidelines"
    filename: "common.md"
`),
				},
				"prompts/rules/_partials.tmpl": &fstest.MapFile{
					Data: []byte(`{{define "YAML_FRONTMATTER"}}---
{{- if .Paths}}
paths: {{pathsToYAML .Paths}}
{{- end}}
---
{{end}}`),
				},
				"prompts/rules/golang.tmpl": &fstest.MapFile{
					Data: []byte(`{{- template "YAML_FRONTMATTER" . -}}
# {{.Title}}`),
				},
				"prompts/rules/common.tmpl": &fstest.MapFile{
					Data: []byte(`{{- template "YAML_FRONTMATTER" . -}}
# {{.Title}}`),
				},
			},
			rules: []string{"golang", "common"},
			validate: func(t *testing.T, dir string) {
				golangPath := filepath.Join(dir, "golang.md")
				commonPath := filepath.Join(dir, "common.md")

				golangContent, err := os.ReadFile(golangPath)
				require.NoError(t, err)
				assert.Contains(t, string(golangContent), "# Go Guidelines")

				commonContent, err := os.ReadFile(commonPath)
				require.NoError(t, err)
				assert.Contains(t, string(commonContent), "# Common Guidelines")
			},
		},
		{
			name: "returns error when file exists and force is false",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`rules:
  golang:
    name: "Go Guidelines"
    filename: "golang.md"
`),
				},
				"prompts/rules/golang.tmpl": &fstest.MapFile{
					Data: []byte(`# {{.Title}}`),
				},
			},
			rules: []string{"golang"},
			setupFiles: map[string]string{
				"golang.md": "existing content",
			},
			force:       false,
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name: "overwrites file when force is true",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`rules:
  golang:
    name: "Go Guidelines"
    filename: "golang.md"
`),
				},
				"prompts/rules/_partials.tmpl": &fstest.MapFile{
					Data: []byte(`{{define "YAML_FRONTMATTER"}}---
---
{{end}}`),
				},
				"prompts/rules/golang.tmpl": &fstest.MapFile{
					Data: []byte(`{{- template "YAML_FRONTMATTER" . -}}
# {{.Title}}`),
				},
			},
			rules: []string{"golang"},
			setupFiles: map[string]string{
				"golang.md": "existing content",
			},
			force: true,
			validate: func(t *testing.T, dir string) {
				golangPath := filepath.Join(dir, "golang.md")
				content, err := os.ReadFile(golangPath)
				require.NoError(t, err)
				assert.Contains(t, string(content), "# Go Guidelines")
				assert.NotContains(t, string(content), "existing content")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dir := tmpDir
			if tt.dir != "" {
				dir = filepath.Join(tmpDir, tt.dir)
			}

			if tt.setupFiles != nil {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create test directory: %v", err)
				}
				for filename, content := range tt.setupFiles {
					path := filepath.Join(dir, filename)
					if err := os.WriteFile(path, []byte(content), 0644); err != nil {
						t.Fatalf("failed to create setup file %s: %v", filename, err)
					}
				}
			}

			engine, err := NewEngineWithFS(tt.fsys)
			require.NoError(t, err)

			err = engine.InitRulesDirectory(dir, tt.rules, tt.force)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, dir)
			}
		})
	}
}

func TestEngine_Generate_RuleTemplateData(t *testing.T) {
	tests := []struct {
		name         string
		fsys         fs.FS
		templateName string
		wantContains []string
		wantErr      bool
	}{
		{
			name: "generates rule with paths in frontmatter",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`rules:
  test-rule:
    name: "Test Rule"
    description: "A test rule"
    filename: ".claude/rules/test.md"
    paths:
      - "**/*.go"
      - "**/*.ts"
`),
				},
				"prompts/rules/_partials.tmpl": &fstest.MapFile{
					Data: []byte(`{{define "YAML_FRONTMATTER"}}---
{{- if .Paths}}
paths: {{pathsToYAML .Paths}}
{{- end}}
---
{{end}}`),
				},
				"prompts/rules/test-rule.tmpl": &fstest.MapFile{
					Data: []byte(`{{- template "YAML_FRONTMATTER" . -}}
# {{.Title}}

{{.Description}}`),
				},
			},
			templateName: "test-rule",
			wantContains: []string{
				"---",
				"paths: **/*.go, **/*.ts",
				"---",
				"# Test Rule",
				"A test rule",
			},
			wantErr: false,
		},
		{
			name: "generates rule without paths when empty",
			fsys: fstest.MapFS{
				"prompts/rules/_metadata.yaml": &fstest.MapFile{
					Data: []byte(`rules:
  common:
    name: "Common Guidelines"
    description: "General guidelines"
    filename: ".claude/rules/common.md"
`),
				},
				"prompts/rules/_partials.tmpl": &fstest.MapFile{
					Data: []byte(`{{define "YAML_FRONTMATTER"}}---
{{- if .Paths}}
paths: {{pathsToYAML .Paths}}
{{- end}}
---
{{end}}`),
				},
				"prompts/rules/common.tmpl": &fstest.MapFile{
					Data: []byte(`{{- template "YAML_FRONTMATTER" . -}}
# {{.Title}}

{{.Description}}`),
				},
			},
			templateName: "common",
			wantContains: []string{
				"---",
				"---",
				"# Common Guidelines",
				"General guidelines",
			},
			wantErr: false,
		},
		{
			name: "generates rule with default metadata when not found",
			fsys: fstest.MapFS{
				"prompts/rules/_partials.tmpl": &fstest.MapFile{
					Data: []byte(`{{define "YAML_FRONTMATTER"}}---
{{- if .Paths}}
paths: {{pathsToYAML .Paths}}
{{- end}}
---
{{end}}`),
				},
				"prompts/rules/unknown.tmpl": &fstest.MapFile{
					Data: []byte(`{{- template "YAML_FRONTMATTER" . -}}
# {{.Name}}

{{if .Description}}{{.Description}}{{else}}No description{{end}}`),
				},
			},
			templateName: "unknown",
			wantContains: []string{
				"---",
				"---",
				"# unknown",
				"No description",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngineWithFS(tt.fsys)
			require.NoError(t, err)

			got, err := engine.Generate(ItemTypeRule, tt.templateName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want, "Expected output to contain: %s", want)
			}
		})
	}
}
