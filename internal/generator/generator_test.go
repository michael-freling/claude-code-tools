package generator

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "creates generator successfully",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGenerator()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotNil(t, got.engine)
		})
	}
}

func TestNewGeneratorWithFS(t *testing.T) {
	tests := []struct {
		name        string
		fsys        fs.FS
		wantErr     bool
		errContains string
	}{
		{
			name:    "creates generator with embedded FS successfully",
			fsys:    templatesFS,
			wantErr: false,
		},
		{
			name: "returns error when template parsing fails",
			fsys: fstest.MapFS{
				"prompts/skills/invalid.tmpl": &fstest.MapFile{
					Data: []byte("{{invalid syntax"),
				},
			},
			wantErr:     true,
			errContains: "failed to create engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGeneratorWithFS(tt.fsys)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotNil(t, got.engine)
		})
	}
}

func TestGenerator_List(t *testing.T) {
	gen, err := NewGenerator()
	require.NoError(t, err)

	tests := []struct {
		name     string
		itemType ItemType
		wantLen  int
	}{
		{
			name:     "returns available agents",
			itemType: ItemTypeAgent,
			wantLen:  6,
		},
		{
			name:     "returns available commands",
			itemType: ItemTypeCommand,
			wantLen:  6,
		},
		{
			name:     "returns available skills",
			itemType: ItemTypeSkill,
			wantLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.List(tt.itemType)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestGenerator_Generate_Success(t *testing.T) {
	tests := []struct {
		name         string
		itemType     ItemType
		templateName string
		wantContains string
	}{
		{
			name:         "outputs coding skill content",
			itemType:     ItemTypeSkill,
			templateName: "coding",
			wantContains: "Coding skill:",
		},
		{
			name:         "outputs ci-error-fix skill content",
			itemType:     ItemTypeSkill,
			templateName: "ci-error-fix",
			wantContains: "CI error",
		},
		{
			name:         "outputs code-reviewer agent content",
			itemType:     ItemTypeAgent,
			templateName: "code-reviewer",
			wantContains: "code",
		},
		{
			name:         "outputs feature command content",
			itemType:     ItemTypeCommand,
			templateName: "feature",
			wantContains: "feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator()
			require.NoError(t, err)

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = gen.Generate(tt.itemType, tt.templateName)
			require.NoError(t, err)

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			got := buf.String()

			assert.NotEmpty(t, got)
			assert.Contains(t, got, tt.wantContains)
		})
	}
}

func TestGenerator_Generate_Errors(t *testing.T) {
	tests := []struct {
		name         string
		itemType     ItemType
		templateName string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "returns error for non-existent skill template",
			itemType:     ItemTypeSkill,
			templateName: "non-existent",
			wantErr:      true,
			errContains:  "not found",
		},
		{
			name:         "returns error for non-existent agent template",
			itemType:     ItemTypeAgent,
			templateName: "invalid-agent",
			wantErr:      true,
			errContains:  "not found",
		},
		{
			name:         "returns error for non-existent command template",
			itemType:     ItemTypeCommand,
			templateName: "invalid-command",
			wantErr:      true,
			errContains:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator()
			require.NoError(t, err)

			err = gen.Generate(tt.itemType, tt.templateName)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGenerator_GenerateAll_Success(t *testing.T) {
	tests := []struct {
		name         string
		itemType     ItemType
		wantContains []string
	}{
		{
			name:     "outputs all skill templates",
			itemType: ItemTypeSkill,
			wantContains: []string{
				"Coding skill:",
				"---",
			},
		},
		{
			name:     "outputs all agent templates",
			itemType: ItemTypeAgent,
			wantContains: []string{
				"---",
			},
		},
		{
			name:     "outputs all command templates",
			itemType: ItemTypeCommand,
			wantContains: []string{
				"---",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator()
			require.NoError(t, err)

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err = gen.GenerateAll(tt.itemType)
			require.NoError(t, err)

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			got := buf.String()

			assert.NotEmpty(t, got)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestGenerator_GenerateAll_Errors(t *testing.T) {
	tests := []struct {
		name        string
		fsys        fs.FS
		itemType    ItemType
		wantErr     bool
		errContains string
	}{
		{
			name: "returns error when template execution fails",
			fsys: fstest.MapFS{
				"prompts/skills/broken.tmpl": &fstest.MapFile{
					Data: []byte("{{.NonExistentField}}"),
				},
			},
			itemType:    ItemTypeSkill,
			wantErr:     true,
			errContains: "failed to generate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGeneratorWithFS(tt.fsys)
			require.NoError(t, err)

			err = gen.GenerateAll(tt.itemType)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGenerator_GenerateRuleWithOptions(t *testing.T) {
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
# {{.Title}}`),
				},
			},
			ruleName: "golang",
			opts: GenerateOptions{
				Paths: []string{"src/**/*.go"},
			},
			wantContains: []string{
				"---",
				"paths: src/**/*.go",
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
			gen, err := NewGeneratorWithFS(tt.fsys)
			require.NoError(t, err)

			got, err := gen.GenerateRuleWithOptions(tt.ruleName, tt.opts)
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

func TestGenerator_GetDefaultRules(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGeneratorWithFS(tt.fsys)
			require.NoError(t, err)

			got := gen.GetDefaultRules()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerator_InitRulesDirectory(t *testing.T) {
	tests := []struct {
		name        string
		fsys        fs.FS
		rules       []string
		force       bool
		setupFiles  map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "creates rules files",
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
			rules:   []string{"golang"},
			wantErr: false,
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
				"golang.md": "existing",
			},
			force:       false,
			wantErr:     true,
			errContains: "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.setupFiles != nil {
				for filename, content := range tt.setupFiles {
					err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
					require.NoError(t, err)
				}
			}

			gen, err := NewGeneratorWithFS(tt.fsys)
			require.NoError(t, err)

			err = gen.InitRulesDirectory(tmpDir, tt.rules, tt.force)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}
