package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// CIChecker checks CI status on GitHub
type CIChecker interface {
	// CheckCI checks CI status. If prNumber is 0, checks the current branch's PR.
	CheckCI(ctx context.Context, prNumber int) (*CIResult, error)
	// WaitForCI waits for CI to complete. If prNumber is 0, checks the current branch's PR.
	WaitForCI(ctx context.Context, prNumber int, timeout time.Duration) (*CIResult, error)
	// WaitForCIWithOptions waits for CI with options. If prNumber is 0, checks the current branch's PR.
	WaitForCIWithOptions(ctx context.Context, prNumber int, timeout time.Duration, opts CheckCIOptions) (*CIResult, error)
}

// CIResult represents the result of CI checks
type CIResult struct {
	Passed     bool
	Status     string
	FailedJobs []string
	Output     string
}

// ciChecker implements CIChecker interface
type ciChecker struct {
	workingDir    string
	checkInterval time.Duration
}

// NewCIChecker creates a new CI checker
func NewCIChecker(workingDir string, checkInterval time.Duration) CIChecker {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}
	return &ciChecker{
		workingDir:    workingDir,
		checkInterval: checkInterval,
	}
}

// CheckCI checks the current CI status. If prNumber is 0, checks the current branch's PR.
func (c *ciChecker) CheckCI(ctx context.Context, prNumber int) (*CIResult, error) {
	result := &CIResult{
		Passed:     false,
		FailedJobs: []string{},
	}

	var cmd *exec.Cmd
	if prNumber > 0 {
		cmd = exec.CommandContext(ctx, "gh", "pr", "checks", fmt.Sprintf("%d", prNumber))
	} else {
		cmd = exec.CommandContext(ctx, "gh", "pr", "checks")
	}
	if c.workingDir != "" {
		cmd.Dir = c.workingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	result.Output = output

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 127:
				return result, fmt.Errorf("gh CLI not found: is it installed?")
			case 8:
				// Exit code 8 means "checks pending" according to gh pr checks --help
				// Parse the output to get the current status
				if output != "" {
					result.Status, result.FailedJobs = parseCIOutput(output)
					result.Passed = result.Status == "success"
					return result, nil
				}
				// No output with exit code 8 likely means no PR found
				return result, fmt.Errorf("no PR found for the current branch: ensure a PR exists before checking CI status")
			case 1:
				// Exit code 1 can mean checks failed or other errors
				// If we got output, parse it as normal (this indicates failed checks)
				if output != "" {
					result.Status, result.FailedJobs = parseCIOutput(output)
					result.Passed = result.Status == "success"
					return result, nil
				}
				return result, fmt.Errorf("failed to check CI status: %w (stderr: %s)", err, stderr.String())
			}
		}
		return result, fmt.Errorf("failed to check CI status: %w (stderr: %s)", err, stderr.String())
	}

	result.Status, result.FailedJobs = parseCIOutput(output)
	result.Passed = result.Status == "success"

	return result, nil
}

// WaitForCI waits for CI to complete with polling. If prNumber is 0, checks the current branch's PR.
func (c *ciChecker) WaitForCI(ctx context.Context, prNumber int, timeout time.Duration) (*CIResult, error) {
	return c.WaitForCIWithOptions(ctx, prNumber, timeout, CheckCIOptions{})
}

// WaitForCIWithOptions waits for CI to complete with polling and optional e2e filtering. If prNumber is 0, checks the current branch's PR.
func (c *ciChecker) WaitForCIWithOptions(ctx context.Context, prNumber int, timeout time.Duration, opts CheckCIOptions) (*CIResult, error) {
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	if opts.E2ETestPattern == "" {
		opts.E2ETestPattern = "e2e|E2E|integration|Integration"
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	initialDelay := 1 * time.Minute
	time.Sleep(initialDelay)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("CI check timeout after %s", timeout)
		case <-ticker.C:
			result, err := c.CheckCI(ctx, prNumber)
			if err != nil {
				return nil, err
			}

			if result.Status == "success" || result.Status == "failure" {
				if opts.SkipE2E {
					result = filterE2EFailures(result, opts.E2ETestPattern)
				}
				return result, nil
			}
		}
	}
}

// parseCIOutput parses gh pr checks output to extract status and failed jobs
func parseCIOutput(output string) (string, []string) {
	lines := strings.Split(output, "\n")
	failedJobs := []string{}
	allPassed := true
	anyCompleted := false
	hasPending := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		status := fields[0]
		jobName := strings.Join(fields[1:], " ")

		switch status {
		case "✓", "pass", "success":
			anyCompleted = true
		case "✗", "fail", "failure":
			anyCompleted = true
			allPassed = false
			failedJobs = append(failedJobs, jobName)
		case "○", "*", "pending", "queued", "in_progress":
			hasPending = true
		}
	}

	if !anyCompleted || hasPending {
		return "pending", failedJobs
	}

	if allPassed {
		return "success", failedJobs
	}

	return "failure", failedJobs
}

// filterE2EFailures filters out e2e test failures from CI result
func filterE2EFailures(result *CIResult, e2ePattern string) *CIResult {
	if result.Passed {
		return result
	}

	e2eRegex, err := regexp.Compile(e2ePattern)
	if err != nil {
		return result
	}

	filteredJobs := []string{}
	for _, job := range result.FailedJobs {
		if !e2eRegex.MatchString(job) {
			filteredJobs = append(filteredJobs, job)
		}
	}

	filtered := &CIResult{
		Status:     result.Status,
		Output:     result.Output,
		FailedJobs: filteredJobs,
		Passed:     len(filteredJobs) == 0,
	}

	return filtered
}
