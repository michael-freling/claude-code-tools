package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *WorkflowError
		want string
	}{
		{
			name: "returns message",
			err: &WorkflowError{
				Message: "test error message",
			},
			want: "test error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestArtifactType(t *testing.T) {
	tests := []struct {
		name         string
		artifactType ArtifactType
		want         string
	}{
		{
			name:         "plan artifact",
			artifactType: ArtifactPlan,
			want:         "plan.json",
		},
		{
			name:         "approval artifact",
			artifactType: ArtifactApproval,
			want:         "approval",
		},
		{
			name:         "implementation artifact",
			artifactType: ArtifactImplementation,
			want:         "implementation",
		},
		{
			name:         "pr artifact",
			artifactType: ArtifactPR,
			want:         "pr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.artifactType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPhasePrerequisites(t *testing.T) {
	tests := []struct {
		name          string
		prerequisites PhasePrerequisites
		wantPhase     Phase
		wantPrereqLen int
		wantFirstType ArtifactType
		wantFirstDesc string
	}{
		{
			name: "confirmation prerequisites",
			prerequisites: PhasePrerequisites{
				Phase: PhaseConfirmation,
				Prerequisites: []PhasePrerequisite{
					{
						ArtifactType: ArtifactPlan,
						Description:  "Plan must be created",
					},
				},
			},
			wantPhase:     PhaseConfirmation,
			wantPrereqLen: 1,
			wantFirstType: ArtifactPlan,
			wantFirstDesc: "Plan must be created",
		},
		{
			name: "implementation prerequisites",
			prerequisites: PhasePrerequisites{
				Phase: PhaseImplementation,
				Prerequisites: []PhasePrerequisite{
					{
						ArtifactType: ArtifactPlan,
						Description:  "Plan must be created",
					},
					{
						ArtifactType: ArtifactApproval,
						Description:  "Plan must be approved",
					},
				},
			},
			wantPhase:     PhaseImplementation,
			wantPrereqLen: 2,
			wantFirstType: ArtifactPlan,
			wantFirstDesc: "Plan must be created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantPhase, tt.prerequisites.Phase)
			assert.Len(t, tt.prerequisites.Prerequisites, tt.wantPrereqLen)
			if tt.wantPrereqLen > 0 {
				assert.Equal(t, tt.wantFirstType, tt.prerequisites.Prerequisites[0].ArtifactType)
				assert.Equal(t, tt.wantFirstDesc, tt.prerequisites.Prerequisites[0].Description)
			}
		})
	}
}
