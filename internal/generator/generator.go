package generator

import (
	"fmt"
	"io/fs"
	"os"
)

type Generator struct {
	engine *Engine
}

func NewGenerator() (*Generator, error) {
	engine, err := NewEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	return &Generator{
		engine: engine,
	}, nil
}

func NewGeneratorWithFS(fsys fs.FS) (*Generator, error) {
	engine, err := NewEngineWithFS(fsys)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	return &Generator{
		engine: engine,
	}, nil
}

func (g *Generator) Generate(itemType ItemType, name string) error {
	content, err := g.engine.Generate(itemType, name)
	if err != nil {
		return fmt.Errorf("failed to generate %s %s: %w", itemType, name, err)
	}

	fmt.Fprintln(os.Stdout, content)
	return nil
}

func (g *Generator) List(itemType ItemType) []string {
	return g.engine.List(itemType)
}

func (g *Generator) GenerateAll(itemType ItemType) error {
	templates := g.engine.List(itemType)

	for _, name := range templates {
		content, err := g.engine.Generate(itemType, name)
		if err != nil {
			return fmt.Errorf("failed to generate %s %s: %w", itemType, name, err)
		}

		fmt.Fprintln(os.Stdout, content)
		fmt.Fprintln(os.Stdout, "---")
		fmt.Fprintln(os.Stdout)
	}

	return nil
}

func (g *Generator) GenerateRuleWithOptions(name string, opts GenerateOptions) (string, error) {
	return g.engine.GenerateRuleWithOptions(name, opts)
}

func (g *Generator) GetDefaultRules() []string {
	return g.engine.GetDefaultRules()
}

func (g *Generator) InitRulesDirectory(dir string, rules []string, force bool) error {
	return g.engine.InitRulesDirectory(dir, rules, force)
}
