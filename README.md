# Claude Code Configuration Generator

A CLI tool to generate prompts for creating Claude Code skills, agents, and commands.

## Overview

This generator outputs PROMPTS to stdout that you can give to Claude to create skills, agents, and commands. It does NOT create the files directly - instead, it generates instructions that Claude can use.

## Installation

```bash
go build -o generator cmd/generator/main.go
```

## Usage

### Agents

```bash
# List all available agents
./generator agents list

# Generate prompt for a specific agent
./generator agents golang-engineer
./generator agents software-architect
```

Available agents:
- `golang-engineer` - Go development with full verification and testing
- `golang-code-reviewer` - Review Go code for best practices
- `typescript-engineer` - TypeScript/Next.js development with testing
- `typescript-code-reviewer` - Review TypeScript code
- `software-architect` - Design software architecture and API specifications
- `architecture-reviewer` - Review architectural designs
- `github-actions-workflow-engineer` - Create and test GitHub Actions workflows
- `kubernetes-engineer` - Kubernetes deployment and configuration

### Commands

```bash
# List all available commands
./generator commands list

# Generate prompt for a specific command
./generator commands feature
./generator commands fix
```

Available commands:
- `feature` - Add or update a feature with architecture design and review
- `fix` - Fix a bug by reproducing, understanding root cause, and planning fixes
- `refactor` - Refactor code with structured workflow
- `document-guideline` - Create comprehensive project guidelines
- `document-guideline-monorepo` - Create guidelines for monorepo subprojects
- `split-pr` - Split large PRs into smaller, reviewable child PRs

### Skills

```bash
# List all available skills
./generator skills list

# Generate prompt for a specific skill
./generator skills coding
./generator skills ci-error-fix
```

Available skills:
- `coding` - Iterative development with Test-Driven Development (TDD)
- `ci-error-fix` - Fix CI errors systematically

### Custom Templates

You can use your own templates by specifying a custom template directory:

```bash
./generator agents --template-dir /path/to/templates golang-engineer
./generator commands -t /path/to/templates feature
```

## How It Works

1. The generator uses Go templates with embedded filesystem
2. Shared rules (COMMON_RULES, CODING_RULES, GOLANG_RULES, TYPESCRIPT_RULES, etc.) are defined in `_partials.tmpl`
3. Each template includes relevant shared rules using `{{template "RULE_NAME"}}`
4. Output is always to stdout - no files are written

## Project Structure

```
.
├── cmd/generator/
│   └── main.go              # CLI entry point
├── internal/generator/
│   ├── generator.go         # High-level Generator wrapper
│   ├── template.go          # Template Engine
│   └── *_test.go            # Tests
├── templates/
│   ├── embed.go             # Embedded templates filesystem
│   └── prompts/
│       ├── _partials.tmpl   # Shared rule definitions
│       ├── agents/          # Agent templates
│       ├── commands/        # Command templates
│       └── skills/          # Skill templates
└── outputs/                 # Generated markdown outputs
    ├── agents/
    ├── commands/
    └── skills/
```

## Development

### Build

```bash
go build ./...
```

### Test

```bash
go test ./...
```

### Verify

```bash
go vet ./...
go fmt ./...
```
