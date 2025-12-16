package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhasePrerequisitesMap(t *testing.T) {
	tests := []struct {
		name              string
		phase             Phase
		wantPrereqCount   int
		wantArtifactTypes []ArtifactType
	}{
		{
			name:              "planning has no prerequisites",
			phase:             PhasePlanning,
			wantPrereqCount:   0,
			wantArtifactTypes: []ArtifactType{},
		},
		{
			name:            "confirmation requires plan",
			phase:           PhaseConfirmation,
			wantPrereqCount: 1,
			wantArtifactTypes: []ArtifactType{
				ArtifactPlan,
			},
		},
		{
			name:            "implementation requires plan and approval",
			phase:           PhaseImplementation,
			wantPrereqCount: 2,
			wantArtifactTypes: []ArtifactType{
				ArtifactPlan,
				ArtifactApproval,
			},
		},
		{
			name:            "refactoring requires plan, approval, and implementation",
			phase:           PhaseRefactoring,
			wantPrereqCount: 3,
			wantArtifactTypes: []ArtifactType{
				ArtifactPlan,
				ArtifactApproval,
				ArtifactImplementation,
			},
		},
		{
			name:            "pr-split requires plan, approval, implementation, and pr",
			phase:           PhasePRSplit,
			wantPrereqCount: 4,
			wantArtifactTypes: []ArtifactType{
				ArtifactPlan,
				ArtifactApproval,
				ArtifactImplementation,
				ArtifactPR,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prereqs, exists := PhasePrerequisitesMap[tt.phase]
			assert.True(t, exists, "phase should exist in map")
			assert.Equal(t, tt.phase, prereqs.Phase)
			assert.Len(t, prereqs.Prerequisites, tt.wantPrereqCount)

			for i, wantType := range tt.wantArtifactTypes {
				assert.Equal(t, wantType, prereqs.Prerequisites[i].ArtifactType)
				assert.NotEmpty(t, prereqs.Prerequisites[i].Description)
			}
		})
	}
}

func TestPhasePrerequisitesMap_AllPhasesHaveDescriptions(t *testing.T) {
	for phase, prereqs := range PhasePrerequisitesMap {
		t.Run(string(phase), func(t *testing.T) {
			for i, prereq := range prereqs.Prerequisites {
				assert.NotEmpty(t, prereq.Description,
					"prerequisite %d for phase %s should have a description", i, phase)
			}
		})
	}
}

func TestPhasePrerequisitesMap_ContainsAllNonTerminalPhases(t *testing.T) {
	phases := []Phase{
		PhasePlanning,
		PhaseConfirmation,
		PhaseImplementation,
		PhaseRefactoring,
		PhasePRSplit,
	}

	for _, phase := range phases {
		t.Run(string(phase), func(t *testing.T) {
			_, exists := PhasePrerequisitesMap[phase]
			assert.True(t, exists, "phase %s should be in prerequisite map", phase)
		})
	}
}
