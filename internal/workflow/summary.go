package workflow

import (
	"context"
	"fmt"
	"os"
)

// gatherSummaryData collects and aggregates workflow execution data into a comprehensive summary
func gatherSummaryData(ctx context.Context, o *Orchestrator, workflowName string) (*WorkflowSummary, error) {
	summary := &WorkflowSummary{
		WorkflowName: workflowName,
		PRType:       PRSummaryTypeNone,
		FilesChanged: []string{},
		ChildPRs:     []PRInfo{},
		Phases:       []PhaseStats{},
	}

	var implSummary ImplementationSummary
	implErr := o.stateManager.LoadPhaseOutput(workflowName, PhaseImplementation, &implSummary)
	if implErr == nil {
		summary.FilesChanged = implSummary.FilesChanged
		summary.LinesAdded = implSummary.LinesAdded
		summary.LinesRemoved = implSummary.LinesRemoved
		summary.TestsAdded = implSummary.TestsAdded
	} else {
		o.logger.Verbose("Warning: Could not load implementation data: %v", implErr)
	}

	var splitResult PRSplitResult
	splitErr := o.stateManager.LoadPhaseOutput(workflowName, PhasePRSplit, &splitResult)
	if splitErr == nil {
		summary.PRType = PRSummaryTypeSplit
		summary.MainPR = &splitResult.ParentPR
		summary.ChildPRs = splitResult.ChildPRs
		return summary, nil
	}

	singlePR, err := getSinglePRInfo(ctx, o)
	if err != nil {
		o.logger.Verbose("Warning: Could not get single PR info: %v", err)
		return summary, nil
	}

	if singlePR != nil {
		summary.PRType = PRSummaryTypeSingle
		summary.MainPR = singlePR
	}

	return summary, nil
}

// getSinglePRInfo attempts to find a PR for the current branch
func getSinglePRInfo(ctx context.Context, o *Orchestrator) (*PRInfo, error) {
	workingDir := o.config.BaseDir
	if _, err := os.Stat(workingDir); err != nil {
		return nil, fmt.Errorf("working directory does not exist: %w", err)
	}

	branch, err := o.gitRunner.GetCurrentBranch(ctx, workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	cmdPRs, err := o.ghRunner.ListPRs(ctx, workingDir, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	if len(cmdPRs) == 0 {
		return nil, nil
	}

	cmdPR := cmdPRs[0]
	return &PRInfo{
		Number: cmdPR.Number,
		URL:    cmdPR.URL,
		Title:  cmdPR.Title,
		Branch: cmdPR.HeadRefName,
	}, nil
}

// formatWorkflowSummary formats the workflow summary for display
func formatWorkflowSummary(summary *WorkflowSummary) string {
	if summary == nil {
		return ""
	}

	var result string

	result += "═══════════════════════════════════════════════════\n"
	result += Bold("Workflow Summary: ") + summary.WorkflowName + "\n"
	result += "═══════════════════════════════════════════════════\n"
	result += "\n"

	prSection := formatPRSection(summary)
	if prSection != "" {
		result += prSection
		result += "\n"
	}

	statsSection := formatStatsSection(summary)
	if statsSection != "" {
		result += statsSection
		result += "\n"
	}

	phaseTimings := formatPhaseTimings(summary)
	if phaseTimings != "" {
		result += phaseTimings
		result += "\n"
	}

	result += Bold("Total Duration: ") + Yellow(FormatDuration(summary.TotalDuration)) + "\n"

	return result
}

// formatPRSection formats the PR information section
func formatPRSection(summary *WorkflowSummary) string {
	if summary == nil || summary.PRType == PRSummaryTypeNone {
		return ""
	}

	var result string
	result += Bold("Pull Requests:") + "\n"

	switch summary.PRType {
	case PRSummaryTypeSingle:
		if summary.MainPR != nil {
			result += fmt.Sprintf("  Main PR: %s - %s\n",
				Cyan(fmt.Sprintf("#%d", summary.MainPR.Number)),
				summary.MainPR.Title)
			result += fmt.Sprintf("          %s\n", Cyan(summary.MainPR.URL))
		}

	case PRSummaryTypeSplit:
		if summary.MainPR != nil {
			result += fmt.Sprintf("  Main PR: %s - %s\n",
				Cyan(fmt.Sprintf("#%d", summary.MainPR.Number)),
				summary.MainPR.Title)
			result += fmt.Sprintf("          %s\n\n", Cyan(summary.MainPR.URL))
		}

		if len(summary.ChildPRs) > 0 {
			result += "  Child PRs:\n"
			for _, pr := range summary.ChildPRs {
				result += fmt.Sprintf("    • %s - %s\n",
					Cyan(fmt.Sprintf("#%d", pr.Number)),
					pr.Title)
				result += fmt.Sprintf("      %s\n", Cyan(pr.URL))
			}
		}
	}

	return result
}

// formatStatsSection formats the implementation statistics section
func formatStatsSection(summary *WorkflowSummary) string {
	if summary == nil {
		return ""
	}

	if len(summary.FilesChanged) == 0 && summary.LinesAdded == 0 &&
		summary.LinesRemoved == 0 && summary.TestsAdded == 0 {
		return ""
	}

	var result string
	result += Bold("Implementation Stats:") + "\n"
	result += fmt.Sprintf("  Files Changed: %s\n", Green(fmt.Sprintf("%d", len(summary.FilesChanged))))
	result += fmt.Sprintf("  Lines Added:   %s\n", Green(fmt.Sprintf("+%d", summary.LinesAdded)))
	result += fmt.Sprintf("  Lines Removed: %s\n", fmt.Sprintf("-%d", summary.LinesRemoved))
	result += fmt.Sprintf("  Tests Added:   %s\n", Green(fmt.Sprintf("%d", summary.TestsAdded)))

	return result
}

// formatPhaseTimings formats the phase execution details section
func formatPhaseTimings(summary *WorkflowSummary) string {
	if summary == nil || len(summary.Phases) == 0 {
		return ""
	}

	var result string
	result += Bold("Phase Execution:") + "\n"

	for _, phase := range summary.Phases {
		statusIcon := Green("✓")
		if !phase.Success {
			statusIcon = Red("✗")
		}

		attemptStr := "1 attempt"
		if phase.Attempts > 1 {
			attemptStr = fmt.Sprintf("%d attempts", phase.Attempts)
		}

		result += fmt.Sprintf("  %s %s    %s (%s)\n",
			statusIcon,
			phase.Name,
			Yellow(FormatDuration(phase.Duration)),
			attemptStr)
	}

	return result
}

// displayWorkflowSummary gathers and displays the workflow execution summary
func (o *Orchestrator) displayWorkflowSummary(ctx context.Context, workflowName string) {
	summary, err := gatherSummaryData(ctx, o, workflowName)
	if err != nil {
		o.logger.Verbose("Warning: Could not gather summary data: %v", err)
		return
	}

	formatted := formatWorkflowSummary(summary)
	if formatted != "" {
		fmt.Printf("\n%s\n", formatted)
	}
}
