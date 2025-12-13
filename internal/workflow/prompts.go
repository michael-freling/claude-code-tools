package workflow

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/michael-freling/claude-code-tools/internal/command"
	"github.com/michael-freling/claude-code-tools/internal/templates"
)

// PRCreationContext provides context for PR creation prompts.
type PRCreationContext struct {
	WorkflowType WorkflowType
	Branch       string
	BaseBranch   string
	Description  string
}

// PromptGenerator generates prompts for workflow phases
type PromptGenerator interface {
	GeneratePlanningPrompt(wfType WorkflowType, description string, feedback []string) (string, error)
	GenerateImplementationPrompt(plan *Plan) (string, error)
	GenerateRefactoringPrompt(plan *Plan) (string, error)
	GeneratePRSplitPrompt(metrics *PRMetrics, commits []command.Commit) (string, error)
	GenerateFixCIPrompt(failures string) (string, error)
	GenerateCreatePRPrompt(ctx *PRCreationContext) (string, error)
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

func (p *promptGenerator) loadTemplates() error {
	templateNames := []string{
		"planning.tmpl",
		"implementation.tmpl",
		"refactoring.tmpl",
		"pr-split.tmpl",
		"fix-ci.tmpl",
		"create-pr.tmpl",
	}

	for _, name := range templateNames {
		path := fmt.Sprintf("workflow/%s", name)
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

func (p *promptGenerator) GeneratePRSplitPrompt(metrics *PRMetrics, commits []command.Commit) (string, error) {
	if metrics == nil {
		return "", fmt.Errorf("metrics cannot be nil")
	}

	tmpl, ok := p.templates["pr-split.tmpl"]
	if !ok {
		return "", fmt.Errorf("pr-split template not loaded")
	}

	data := struct {
		Metrics *PRMetrics
		Commits []command.Commit
	}{
		Metrics: metrics,
		Commits: commits,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute pr-split template: %w", err)
	}

	return buf.String(), nil
}

func (p *promptGenerator) GenerateFixCIPrompt(failures string) (string, error) {
	if strings.TrimSpace(failures) == "" {
		return "", fmt.Errorf("failures cannot be empty")
	}

	tmpl, ok := p.templates["fix-ci.tmpl"]
	if !ok {
		return "", fmt.Errorf("fix-ci template not loaded")
	}

	data := struct {
		Failures string
	}{
		Failures: failures,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute fix-ci template: %w", err)
	}

	return buf.String(), nil
}

func (p *promptGenerator) GenerateCreatePRPrompt(ctx *PRCreationContext) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context cannot be nil")
	}

	if ctx.Branch == "" {
		return "", fmt.Errorf("branch cannot be empty")
	}
	if ctx.BaseBranch == "" {
		return "", fmt.Errorf("base branch cannot be empty")
	}

	tmpl, ok := p.templates["create-pr.tmpl"]
	if !ok {
		return "", fmt.Errorf("create-pr template not loaded")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("failed to execute create-pr template: %w", err)
	}

	return buf.String(), nil
}
