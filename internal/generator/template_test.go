package generator

import (
	"io/fs"
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
			itemTypes := []ItemType{ItemTypeSkill, ItemTypeAgent, ItemTypeCommand}
			for _, itemType := range itemTypes {
				assert.Contains(t, got.templates, itemType)
				assert.Contains(t, got.templateNames, itemType)
			}
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
			itemTypes := []ItemType{ItemTypeSkill, ItemTypeAgent, ItemTypeCommand}
			for _, itemType := range itemTypes {
				assert.Contains(t, got.templates, itemType)
				assert.Contains(t, got.templateNames, itemType)
			}
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
			wantLen:  8,
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
				"golang-code-reviewer",
				"kubernetes-engineer",
				"architecture-reviewer",
				"typescript-code-reviewer",
				"software-architect",
				"github-actions-workflow-engineer",
				"typescript-engineer",
				"golang-engineer",
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
			name:         "generates golang-code-reviewer agent template",
			itemType:     ItemTypeAgent,
			templateName: "golang-code-reviewer",
			wantContains: "Go",
		},
		{
			name:         "generates feature command template",
			itemType:     ItemTypeCommand,
			templateName: "feature",
			wantContains: "feature",
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
