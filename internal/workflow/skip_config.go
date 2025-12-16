package workflow

// PhasePrerequisitesMap defines the prerequisites for each phase
var PhasePrerequisitesMap = map[Phase]PhasePrerequisites{
	PhasePlanning: {
		Phase:         PhasePlanning,
		Prerequisites: []PhasePrerequisite{},
	},
	PhaseConfirmation: {
		Phase: PhaseConfirmation,
		Prerequisites: []PhasePrerequisite{
			{
				ArtifactType: ArtifactPlan,
				Description:  "Plan must be created (plan.json must exist)",
			},
		},
	},
	PhaseImplementation: {
		Phase: PhaseImplementation,
		Prerequisites: []PhasePrerequisite{
			{
				ArtifactType: ArtifactPlan,
				Description:  "Plan must be created (plan.json must exist)",
			},
			{
				ArtifactType: ArtifactApproval,
				Description:  "Plan must be approved (confirmation phase completed)",
			},
		},
	},
	PhaseRefactoring: {
		Phase: PhaseRefactoring,
		Prerequisites: []PhasePrerequisite{
			{
				ArtifactType: ArtifactPlan,
				Description:  "Plan must be created (plan.json must exist)",
			},
			{
				ArtifactType: ArtifactApproval,
				Description:  "Plan must be approved (confirmation phase completed)",
			},
			{
				ArtifactType: ArtifactImplementation,
				Description:  "Implementation must be completed",
			},
		},
	},
	PhasePRSplit: {
		Phase: PhasePRSplit,
		Prerequisites: []PhasePrerequisite{
			{
				ArtifactType: ArtifactPlan,
				Description:  "Plan must be created (plan.json must exist)",
			},
			{
				ArtifactType: ArtifactApproval,
				Description:  "Plan must be approved (confirmation phase completed)",
			},
			{
				ArtifactType: ArtifactImplementation,
				Description:  "Implementation must be completed",
			},
			{
				ArtifactType: ArtifactPR,
				Description:  "PR must be created (refactoring phase completed)",
			},
		},
	},
}
