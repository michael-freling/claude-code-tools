package workflow

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/michael-freling/claude-code-config/templates"
)

// PromptGenerator generates prompts for workflow phases
type PromptGenerator interface {
	GeneratePlanningPrompt(wfType WorkflowType, description string, feedback []string) (string, error)
	GenerateImplementationPrompt(plan *Plan) (string, error)
	GenerateRefactoringPrompt(plan *Plan) (string, error)
	GeneratePRSplitPrompt(metrics *PRMetrics) (string, error)
	GenerateFixCIPrompt(failures string) (string, error)
}

type promptGenerator struct {
	templates map[string]*template.Template
}

// NewPromptGenerator creates a new prompt generator using embedded templates
func NewPromptGenerator() (PromptGenerator, error) {
	pg := &promptGenerator{
		templates: make(map[string]*template.Template),
	}

	if err := pg.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return pg, nil
}

// loadTemplates loads all workflow templates from the embedded filesystem
func (p *promptGenerator) loadTemplates() error {
	templateNames := []string{
		"planning.tmpl",
		"implementation.tmpl",
		"refactoring.tmpl",
		"pr-split.tmpl",
		"fix-ci.tmpl",
	}

	for _, name := range templateNames {
		path := fmt.Sprintf("prompts/workflow/%s", name)
		content, err := templates.FS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", name, err)
		}

		tmpl, err := template.New(name).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		p.templates[name] = tmpl
	}

	return nil
}

// GeneratePlanningPrompt generates a prompt for the planning phase
func (p *promptGenerator) GeneratePlanningPrompt(wfType WorkflowType, description string, feedback []string) (string, error) {
	tmpl, ok := p.templates["planning.tmpl"]
	if !ok {
		return "", fmt.Errorf("planning template not loaded")
	}

	data := struct {
		Type        WorkflowType
		Description string
		Feedback    []string
	}{
		Type:        wfType,
		Description: description,
		Feedback:    feedback,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute planning template: %w", err)
	}

	return buf.String(), nil
}

// GenerateImplementationPrompt generates a prompt for the implementation phase
func (p *promptGenerator) GenerateImplementationPrompt(plan *Plan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("plan cannot be nil")
	}

	tmpl, ok := p.templates["implementation.tmpl"]
	if !ok {
		return "", fmt.Errorf("implementation template not loaded")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, plan); err != nil {
		return "", fmt.Errorf("failed to execute implementation template: %w", err)
	}

	return buf.String(), nil
}

// GenerateRefactoringPrompt generates a prompt for the refactoring phase
func (p *promptGenerator) GenerateRefactoringPrompt(plan *Plan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("plan cannot be nil")
	}

	tmpl, ok := p.templates["refactoring.tmpl"]
	if !ok {
		return "", fmt.Errorf("refactoring template not loaded")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, plan); err != nil {
		return "", fmt.Errorf("failed to execute refactoring template: %w", err)
	}

	return buf.String(), nil
}

// GeneratePRSplitPrompt generates a prompt for the PR split phase
func (p *promptGenerator) GeneratePRSplitPrompt(metrics *PRMetrics) (string, error) {
	if metrics == nil {
		return "", fmt.Errorf("metrics cannot be nil")
	}

	tmpl, ok := p.templates["pr-split.tmpl"]
	if !ok {
		return "", fmt.Errorf("pr-split template not loaded")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, metrics); err != nil {
		return "", fmt.Errorf("failed to execute pr-split template: %w", err)
	}

	return buf.String(), nil
}

// GenerateFixCIPrompt generates a prompt for fixing CI failures
func (p *promptGenerator) GenerateFixCIPrompt(failures string) (string, error) {
	if failures == "" {
		return "", fmt.Errorf("failures cannot be empty")
	}

	tmpl, ok := p.templates["fix-ci.tmpl"]
	if !ok {
		return "", fmt.Errorf("fix-ci template not loaded")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, failures); err != nil {
		return "", fmt.Errorf("failed to execute fix-ci template: %w", err)
	}

	return buf.String(), nil
}
