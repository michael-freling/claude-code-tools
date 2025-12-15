package workflow

const PlanSchema = `{
  "type": "object",
  "properties": {
    "summary": {
      "type": "string",
      "description": "A one paragraph summary of what will be implemented"
    },
    "contextType": {
      "type": "string",
      "description": "The type of workflow (feature, fix, etc.)"
    },
    "architecture": {
      "type": "object",
      "properties": {
        "overview": {
          "type": "string",
          "description": "High-level architectural approach and design decisions"
        },
        "components": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "List of key components"
        }
      },
      "required": ["overview", "components"]
    },
    "phases": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "description": "Phase name"
          },
          "description": {
            "type": "string",
            "description": "What this phase accomplishes"
          },
          "estimatedFiles": {
            "type": "integer",
            "description": "Estimated number of files"
          },
          "estimatedLines": {
            "type": "integer",
            "description": "Estimated number of lines"
          }
        },
        "required": ["name", "description", "estimatedFiles", "estimatedLines"]
      }
    },
    "workStreams": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "description": "Work stream name"
          },
          "tasks": {
            "type": "array",
            "items": {
              "type": "string"
            },
            "description": "List of tasks"
          },
          "dependsOn": {
            "type": "array",
            "items": {
              "type": "string"
            },
            "description": "Optional dependencies"
          }
        },
        "required": ["name", "tasks"]
      }
    },
    "risks": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of risks"
    },
    "complexity": {
      "type": "string",
      "enum": ["small", "medium", "large"],
      "description": "Complexity level"
    },
    "estimatedTotalLines": {
      "type": "integer",
      "description": "Estimated total lines"
    },
    "estimatedTotalFiles": {
      "type": "integer",
      "description": "Estimated total files"
    }
  },
  "required": ["summary", "contextType", "phases", "complexity"]
}`

const ImplementationSummarySchema = `{
  "type": "object",
  "properties": {
    "filesChanged": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of files that were changed"
    },
    "linesAdded": {
      "type": "integer",
      "description": "Number of lines added"
    },
    "linesRemoved": {
      "type": "integer",
      "description": "Number of lines removed"
    },
    "testsAdded": {
      "type": "integer",
      "description": "Number of tests added"
    },
    "summary": {
      "type": "string",
      "description": "Brief summary of what was implemented"
    },
    "nextSteps": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Optional next steps"
    }
  },
  "required": ["filesChanged", "linesAdded", "linesRemoved", "testsAdded", "summary"]
}`

const RefactoringSummarySchema = `{
  "type": "object",
  "properties": {
    "filesChanged": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of files that were changed"
    },
    "improvementsMade": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of improvements made"
    },
    "summary": {
      "type": "string",
      "description": "Brief summary of refactoring changes"
    }
  },
  "required": ["filesChanged", "improvementsMade", "summary"]
}`

const PRSplitPlanSchema = `{
  "type": "object",
  "properties": {
    "strategy": {
      "type": "string",
      "enum": ["commits", "files"],
      "description": "Strategy for creating child branches"
    },
    "parentTitle": {
      "type": "string",
      "description": "Title for the parent PR"
    },
    "parentDescription": {
      "type": "string",
      "description": "Description for the parent PR"
    },
    "childPRs": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "properties": {
          "title": {
            "type": "string",
            "description": "Title for the child PR"
          },
          "description": {
            "type": "string",
            "description": "Description for the child PR"
          },
          "commits": {
            "type": "array",
            "items": {
              "type": "string"
            },
            "description": "Commit hashes to include in this child PR (for commits strategy)"
          },
          "files": {
            "type": "array",
            "items": {
              "type": "string"
            },
            "description": "File paths to include in this child PR (for files strategy)"
          }
        },
        "required": ["title", "description"]
      },
      "description": "List of child PRs to create"
    },
    "summary": {
      "type": "string",
      "description": "Summary of the split rationale"
    }
  },
  "required": ["strategy", "parentTitle", "parentDescription", "childPRs", "summary"]
}`

const PRSplitResultSchema = `{
  "type": "object",
  "properties": {
    "parentPR": {
      "type": "object",
      "properties": {
        "number": {
          "type": "integer",
          "description": "PR number"
        },
        "url": {
          "type": "string",
          "description": "PR URL"
        },
        "title": {
          "type": "string",
          "description": "PR title"
        },
        "description": {
          "type": "string",
          "description": "PR description"
        }
      },
      "required": ["number", "url", "title", "description"]
    },
    "childPRs": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "number": {
            "type": "integer",
            "description": "PR number"
          },
          "url": {
            "type": "string",
            "description": "PR URL"
          },
          "title": {
            "type": "string",
            "description": "PR title"
          },
          "description": {
            "type": "string",
            "description": "PR description"
          }
        },
        "required": ["number", "url", "title", "description"]
      }
    },
    "summary": {
      "type": "string",
      "description": "Brief summary of the split strategy"
    }
  },
  "required": ["parentPR", "childPRs", "summary"]
}`
