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
    "prNumber": {
      "type": "integer",
      "description": "PR number created for the implementation"
    },
    "prUrl": {
      "type": "string",
      "description": "URL of the PR created for the implementation"
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
  "required": ["filesChanged", "linesAdded", "linesRemoved", "testsAdded", "prNumber", "prUrl", "summary"]
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
