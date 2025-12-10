package generator

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/michael-freling/claude-code-tools/internal/templates"
)

// ItemType represents the type of item to generate
type ItemType string

const (
	ItemTypeSkill   ItemType = "skill"
	ItemTypeAgent   ItemType = "agent"
	ItemTypeCommand ItemType = "command"
)

// TemplateData holds data to pass to templates
type TemplateData struct {
	Name string
	Type ItemType
}

var templatesFS = templates.FS

// Engine holds parsed templates and provides generation capabilities
type Engine struct {
	templates     map[ItemType]*template.Template
	templateNames map[ItemType][]string
}

// NewEngine creates a new template engine by loading and parsing all templates from embedded FS
func NewEngine() (*Engine, error) {
	return NewEngineWithFS(templatesFS)
}

// NewEngineWithFS creates a new template engine by loading and parsing all templates from the provided FS
func NewEngineWithFS(fsys fs.FS) (*Engine, error) {
	engine := &Engine{
		templates:     make(map[ItemType]*template.Template),
		templateNames: make(map[ItemType][]string),
	}

	itemTypes := []ItemType{ItemTypeSkill, ItemTypeAgent, ItemTypeCommand}

	for _, itemType := range itemTypes {
		tmpl, names, err := loadTemplatesForType(fsys, itemType)
		if err != nil {
			return nil, fmt.Errorf("failed to load templates for %s: %w", itemType, err)
		}
		engine.templates[itemType] = tmpl
		engine.templateNames[itemType] = names
	}

	return engine, nil
}

// loadTemplatesForType loads all templates for a specific item type from the provided FS
func loadTemplatesForType(fsys fs.FS, itemType ItemType) (*template.Template, []string, error) {
	dir := fmt.Sprintf("prompts/%ss", itemType)

	// Check if directory exists
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		// Directory doesn't exist, return empty template set
		return template.New(string(itemType)), []string{}, nil
	}

	var tmpl *template.Template

	// First pass: parse type-specific _partials.tmpl from the type's directory
	typePartialsPath := filepath.Join(dir, "_partials.tmpl")
	partialsContent, err := fs.ReadFile(fsys, typePartialsPath)
	if err == nil {
		tmpl = template.New(string(itemType))
		tmpl, err = tmpl.Parse(string(partialsContent))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse type-specific partials: %w", err)
		}
	} else {
		tmpl = template.New(string(itemType))
	}

	// Track template names from actual template files (not partials)
	var templateNames []string

	// Second pass: parse all other .tmpl files
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		// Skip _partials.tmpl as it's already parsed
		if entry.Name() == "_partials.tmpl" {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		content, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read template file %s: %w", filePath, err)
		}

		// Extract template name from filename (remove .tmpl extension)
		templateName := strings.TrimSuffix(entry.Name(), ".tmpl")
		templateNames = append(templateNames, templateName)

		// Parse template with the derived name
		_, err = tmpl.New(templateName).Parse(string(content))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse template %s: %w", filePath, err)
		}
	}

	return tmpl, templateNames, nil
}

// Generate executes a specific template and returns the result
func (e *Engine) Generate(itemType ItemType, name string) (string, error) {
	tmpl, ok := e.templates[itemType]
	if !ok {
		return "", fmt.Errorf("no templates found for type: %s", itemType)
	}

	// Check if the specific template exists
	templateToExecute := tmpl.Lookup(name)
	if templateToExecute == nil {
		return "", fmt.Errorf("template %s not found for type %s", name, itemType)
	}

	var result strings.Builder
	data := TemplateData{
		Name: name,
		Type: itemType,
	}

	err := templateToExecute.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return result.String(), nil
}

// List returns available template names for a given item type
func (e *Engine) List(itemType ItemType) []string {
	names, ok := e.templateNames[itemType]
	if !ok {
		return []string{}
	}

	return names
}
