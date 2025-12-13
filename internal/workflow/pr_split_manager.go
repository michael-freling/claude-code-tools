package workflow

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

// PRSplitManager orchestrates the creation of split PRs with proper branch chains
type PRSplitManager interface {
	// ExecuteSplit executes the complete PR split workflow
	// It creates the parent branch, child branches, and all PRs
	ExecuteSplit(ctx context.Context, dir string, plan *PRSplitPlan, sourceBranch string, mainBranch string) (*PRSplitResult, error)

	// Rollback closes PRs and deletes branches created during a failed split
	Rollback(ctx context.Context, dir string, result *PRSplitResult) error
}

type prSplitManager struct {
	git command.GitRunner
	gh  command.GhRunner
}

// NewPRSplitManager creates a new PRSplitManager instance
func NewPRSplitManager(git command.GitRunner, gh command.GhRunner) PRSplitManager {
	return &prSplitManager{
		git: git,
		gh:  gh,
	}
}

// ExecuteSplit executes the complete PR split workflow
func (p *prSplitManager) ExecuteSplit(ctx context.Context, dir string, plan *PRSplitPlan, sourceBranch string, mainBranch string) (*PRSplitResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan cannot be nil")
	}
	if len(plan.ChildPRs) == 0 {
		return nil, fmt.Errorf("plan must have at least one child PR")
	}
	if sourceBranch == "" {
		return nil, fmt.Errorf("sourceBranch cannot be empty")
	}
	if mainBranch == "" {
		return nil, fmt.Errorf("mainBranch cannot be empty")
	}

	result := &PRSplitResult{
		BranchNames: make([]string, 0, 1+len(plan.ChildPRs)),
	}

	parentBranch := generateParentBranchName(sourceBranch)
	result.BranchNames = append(result.BranchNames, parentBranch)

	childBranches := make([]string, len(plan.ChildPRs))
	for i := range plan.ChildPRs {
		childBranches[i] = generateChildBranchName(sourceBranch, i)
		result.BranchNames = append(result.BranchNames, childBranches[i])
	}

	if err := p.git.CreateBranch(ctx, dir, parentBranch, mainBranch); err != nil {
		return result, fmt.Errorf("failed to create parent branch: %w", err)
	}

	if err := p.git.CheckoutBranch(ctx, dir, parentBranch); err != nil {
		return result, fmt.Errorf("failed to checkout parent branch: %w", err)
	}

	commitMsg := fmt.Sprintf("Parent PR for split: %s", plan.ParentTitle)
	if err := p.git.CommitEmpty(ctx, dir, commitMsg); err != nil {
		return result, fmt.Errorf("failed to create empty commit on parent branch: %w", err)
	}

	if err := p.git.Push(ctx, dir, parentBranch); err != nil {
		return result, fmt.Errorf("failed to push parent branch: %w", err)
	}

	baseBranch := parentBranch
	for i, childPlan := range plan.ChildPRs {
		childBranch := childBranches[i]

		if err := p.git.CreateBranch(ctx, dir, childBranch, baseBranch); err != nil {
			return result, fmt.Errorf("failed to create child branch %d: %w", i+1, err)
		}

		if err := p.git.CheckoutBranch(ctx, dir, childBranch); err != nil {
			return result, fmt.Errorf("failed to checkout child branch %d: %w", i+1, err)
		}

		if err := p.applyChildChanges(ctx, dir, plan.Strategy, childPlan, sourceBranch); err != nil {
			return result, fmt.Errorf("failed to apply changes to child branch %d: %w", i+1, err)
		}

		if err := p.git.Push(ctx, dir, childBranch); err != nil {
			return result, fmt.Errorf("failed to push child branch %d: %w", i+1, err)
		}

		baseBranch = childBranch
	}

	parentPRURL, err := p.gh.PRCreate(ctx, dir, plan.ParentTitle, plan.ParentDesc, parentBranch, mainBranch)
	if err != nil {
		return result, fmt.Errorf("failed to create parent PR: %w", err)
	}

	parentPRNumber, err := extractPRNumber(parentPRURL)
	if err != nil {
		return result, fmt.Errorf("failed to extract parent PR number: %w", err)
	}

	result.ParentPR = PRInfo{
		Number:      parentPRNumber,
		URL:         strings.TrimSpace(parentPRURL),
		Title:       plan.ParentTitle,
		Description: plan.ParentDesc,
	}

	baseBranch = parentBranch
	childPRLinks := make([]string, 0, len(plan.ChildPRs))
	for i, childPlan := range plan.ChildPRs {
		childBranch := childBranches[i]

		childPRURL, err := p.gh.PRCreate(ctx, dir, childPlan.Title, childPlan.Description, childBranch, baseBranch)
		if err != nil {
			return result, fmt.Errorf("failed to create child PR %d: %w", i+1, err)
		}

		childPRNumber, err := extractPRNumber(childPRURL)
		if err != nil {
			return result, fmt.Errorf("failed to extract child PR %d number: %w", i+1, err)
		}

		childPRInfo := PRInfo{
			Number:      childPRNumber,
			URL:         strings.TrimSpace(childPRURL),
			Title:       childPlan.Title,
			Description: childPlan.Description,
		}
		result.ChildPRs = append(result.ChildPRs, childPRInfo)

		childPRLinks = append(childPRLinks, fmt.Sprintf("- #%d - %s", childPRNumber, childPlan.Title))

		baseBranch = childBranch
	}

	updatedParentDesc := plan.ParentDesc + "\n\n## Child PRs\n\n" + strings.Join(childPRLinks, "\n")
	if err := p.gh.PREdit(ctx, dir, parentPRNumber, updatedParentDesc); err != nil {
		return result, fmt.Errorf("failed to update parent PR description: %w", err)
	}

	result.Summary = plan.Summary

	return result, nil
}

// applyChildChanges applies changes to a child branch based on the strategy
func (p *prSplitManager) applyChildChanges(ctx context.Context, dir string, strategy SplitStrategy, childPlan ChildPRPlan, sourceBranch string) error {
	switch strategy {
	case SplitByCommits:
		for _, commitHash := range childPlan.Commits {
			if err := p.git.CherryPick(ctx, dir, commitHash); err != nil {
				return fmt.Errorf("failed to cherry-pick commit %s: %w", commitHash, err)
			}
		}
	case SplitByFiles:
		if err := p.git.CheckoutFiles(ctx, dir, sourceBranch, childPlan.Files); err != nil {
			return fmt.Errorf("failed to checkout files: %w", err)
		}

		if err := p.git.CommitAll(ctx, dir, childPlan.Title); err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}
	default:
		return fmt.Errorf("unknown split strategy: %s", strategy)
	}

	return nil
}

// Rollback closes PRs and deletes branches created during a failed split
func (p *prSplitManager) Rollback(ctx context.Context, dir string, result *PRSplitResult) error {
	var errs []error

	for i := len(result.ChildPRs) - 1; i >= 0; i-- {
		if err := p.gh.PRClose(ctx, dir, result.ChildPRs[i].Number); err != nil {
			errs = append(errs, fmt.Errorf("failed to close child PR %d: %w", result.ChildPRs[i].Number, err))
		}
	}

	if result.ParentPR.Number > 0 {
		if err := p.gh.PRClose(ctx, dir, result.ParentPR.Number); err != nil {
			errs = append(errs, fmt.Errorf("failed to close parent PR %d: %w", result.ParentPR.Number, err))
		}
	}

	for i := len(result.BranchNames) - 1; i >= 0; i-- {
		branch := result.BranchNames[i]
		if err := p.git.DeleteRemoteBranch(ctx, dir, branch); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete remote branch %s: %w", branch, err))
		}
	}

	for i := len(result.BranchNames) - 1; i >= 0; i-- {
		branch := result.BranchNames[i]
		if err := p.git.DeleteBranch(ctx, dir, branch, true); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete local branch %s: %w", branch, err))
		}
	}

	if len(errs) > 0 {
		errMsgs := make([]string, len(errs))
		for i, err := range errs {
			errMsgs[i] = err.Error()
		}
		return fmt.Errorf("rollback encountered errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// extractPRNumber extracts PR number from GitHub PR URL
func extractPRNumber(prURL string) (int, error) {
	prURL = strings.TrimSpace(prURL)
	if prURL == "" {
		return 0, fmt.Errorf("PR URL is empty")
	}

	matches := prNumberRegex.FindStringSubmatch(prURL)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid PR URL format: %s", prURL)
	}

	prNumber, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse PR number: %w", err)
	}

	return prNumber, nil
}

// generateParentBranchName generates the parent branch name
func generateParentBranchName(sourceBranch string) string {
	return fmt.Sprintf("split/%s/parent", sourceBranch)
}

// generateChildBranchName generates a child branch name
func generateChildBranchName(sourceBranch string, index int) string {
	return fmt.Sprintf("split/%s/child-%d", sourceBranch, index+1)
}
