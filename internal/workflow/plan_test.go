package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadExternalPlan(t *testing.T) {
	tests := []struct {
		name        string
		planContent string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *Plan)
	}{
		{
			name: "valid plan",
			planContent: `{
				"summary": "Test plan",
				"contextType": "feature",
				"architecture": {
					"overview": "Test overview",
					"components": ["component1"]
				},
				"phases": [
					{
						"name": "Phase 1",
						"description": "Description",
						"estimatedFiles": 5,
						"estimatedLines": 100
					}
				],
				"workStreams": [
					{
						"name": "Stream 1",
						"tasks": ["task1", "task2"]
					}
				],
				"risks": ["risk1"],
				"complexity": "medium",
				"estimatedTotalLines": 100,
				"estimatedTotalFiles": 5
			}`,
			wantErr: false,
			validate: func(t *testing.T, plan *Plan) {
				assert.Equal(t, "Test plan", plan.Summary)
				assert.Len(t, plan.Phases, 1)
				assert.Len(t, plan.WorkStreams, 1)
				assert.Equal(t, "Phase 1", plan.Phases[0].Name)
				assert.Equal(t, "Stream 1", plan.WorkStreams[0].Name)
			},
		},
		{
			name: "missing summary",
			planContent: `{
				"phases": [{"name": "Phase 1", "description": "Description", "estimatedFiles": 5, "estimatedLines": 100}],
				"workStreams": [{"name": "Stream 1", "tasks": ["task1"]}]
			}`,
			wantErr:     true,
			errContains: "plan.Summary is required",
		},
		{
			name: "missing phases",
			planContent: `{
				"summary": "Test plan",
				"workStreams": [{"name": "Stream 1", "tasks": ["task1"]}]
			}`,
			wantErr:     true,
			errContains: "plan.Phases is required",
		},
		{
			name: "empty phases",
			planContent: `{
				"summary": "Test plan",
				"phases": [],
				"workStreams": [{"name": "Stream 1", "tasks": ["task1"]}]
			}`,
			wantErr:     true,
			errContains: "plan.Phases is required",
		},
		{
			name: "missing workstreams",
			planContent: `{
				"summary": "Test plan",
				"phases": [{"name": "Phase 1", "description": "Description", "estimatedFiles": 5, "estimatedLines": 100}]
			}`,
			wantErr:     true,
			errContains: "plan.WorkStreams is required",
		},
		{
			name: "empty workstreams",
			planContent: `{
				"summary": "Test plan",
				"phases": [{"name": "Phase 1", "description": "Description", "estimatedFiles": 5, "estimatedLines": 100}],
				"workStreams": []
			}`,
			wantErr:     true,
			errContains: "plan.WorkStreams is required",
		},
		{
			name:        "invalid json",
			planContent: `{invalid json}`,
			wantErr:     true,
			errContains: "failed to parse plan JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			planPath := filepath.Join(tmpDir, "plan.json")

			err := os.WriteFile(planPath, []byte(tt.planContent), 0644)
			require.NoError(t, err)

			got, err := LoadExternalPlan(planPath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				if tt.validate != nil {
					tt.validate(t, got)
				}
			}
		})
	}
}

func TestLoadExternalPlan_FileNotFound(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "nonexistent file",
			path: "/nonexistent/plan.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadExternalPlan(tt.path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to read plan file")
			assert.Nil(t, got)
		})
	}
}

func TestValidatePlan(t *testing.T) {
	tests := []struct {
		name        string
		plan        *Plan
		wantErr     bool
		errContains string
	}{
		{
			name: "valid plan",
			plan: &Plan{
				Summary: "Test plan",
				Phases: []PlanPhase{
					{Name: "Phase 1"},
				},
				WorkStreams: []WorkStream{
					{Name: "Stream 1"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty summary",
			plan: &Plan{
				Summary: "",
				Phases: []PlanPhase{
					{Name: "Phase 1"},
				},
				WorkStreams: []WorkStream{
					{Name: "Stream 1"},
				},
			},
			wantErr:     true,
			errContains: "plan.Summary is required",
		},
		{
			name: "nil phases",
			plan: &Plan{
				Summary: "Test plan",
				Phases:  nil,
				WorkStreams: []WorkStream{
					{Name: "Stream 1"},
				},
			},
			wantErr:     true,
			errContains: "plan.Phases is required",
		},
		{
			name: "empty phases",
			plan: &Plan{
				Summary: "Test plan",
				Phases:  []PlanPhase{},
				WorkStreams: []WorkStream{
					{Name: "Stream 1"},
				},
			},
			wantErr:     true,
			errContains: "plan.Phases is required",
		},
		{
			name: "nil workstreams",
			plan: &Plan{
				Summary: "Test plan",
				Phases: []PlanPhase{
					{Name: "Phase 1"},
				},
				WorkStreams: nil,
			},
			wantErr:     true,
			errContains: "plan.WorkStreams is required",
		},
		{
			name: "empty workstreams",
			plan: &Plan{
				Summary: "Test plan",
				Phases: []PlanPhase{
					{Name: "Phase 1"},
				},
				WorkStreams: []WorkStream{},
			},
			wantErr:     true,
			errContains: "plan.WorkStreams is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlan(tt.plan)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadExternalPlan_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		plan Plan
	}{
		{
			name: "complete plan",
			plan: Plan{
				Summary:     "Feature implementation",
				ContextType: "feature",
				Architecture: Architecture{
					Overview:   "System overview",
					Components: []string{"component1", "component2"},
				},
				Phases: []PlanPhase{
					{
						Name:           "Foundation",
						Description:    "Base implementation",
						EstimatedFiles: 5,
						EstimatedLines: 100,
					},
					{
						Name:           "Integration",
						Description:    "Connect components",
						EstimatedFiles: 3,
						EstimatedLines: 50,
					},
				},
				WorkStreams: []WorkStream{
					{
						Name:      "Core",
						Tasks:     []string{"task1", "task2"},
						DependsOn: []string{},
					},
					{
						Name:      "UI",
						Tasks:     []string{"task3", "task4"},
						DependsOn: []string{"Core"},
					},
				},
				Risks:               []string{"risk1", "risk2"},
				Complexity:          "high",
				EstimatedTotalLines: 150,
				EstimatedTotalFiles: 8,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			planPath := filepath.Join(tmpDir, "plan.json")

			data, err := json.MarshalIndent(tt.plan, "", "  ")
			require.NoError(t, err)

			err = os.WriteFile(planPath, data, 0644)
			require.NoError(t, err)

			got, err := LoadExternalPlan(planPath)
			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, tt.plan.Summary, got.Summary)
			assert.Equal(t, tt.plan.ContextType, got.ContextType)
			assert.Equal(t, tt.plan.Architecture.Overview, got.Architecture.Overview)
			assert.Equal(t, tt.plan.Architecture.Components, got.Architecture.Components)
			assert.Equal(t, len(tt.plan.Phases), len(got.Phases))
			assert.Equal(t, len(tt.plan.WorkStreams), len(got.WorkStreams))
			assert.Equal(t, tt.plan.Risks, got.Risks)
			assert.Equal(t, tt.plan.Complexity, got.Complexity)
			assert.Equal(t, tt.plan.EstimatedTotalLines, got.EstimatedTotalLines)
			assert.Equal(t, tt.plan.EstimatedTotalFiles, got.EstimatedTotalFiles)
		})
	}
}
