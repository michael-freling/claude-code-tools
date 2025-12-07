package workflow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ErrCICheckTimeout is returned when CI check command times out
var ErrCICheckTimeout = errors.New("CI check command timed out")

// CIProgressEvent represents a CI check progress update
type CIProgressEvent struct {
	Type         string // "waiting", "checking", "retry", "status"
	Elapsed      time.Duration
	Message      string
	JobsPassed   int
	JobsFailed   int
	JobsPending  int
	RetryAttempt int
	NextCheckIn  time.Duration
}

// CIProgressCallback is called when CI check progress updates
type CIProgressCallback func(event CIProgressEvent)

// CIChecker checks CI status on GitHub
type CIChecker interface {
	// CheckCI checks CI status. If prNumber is 0, checks the current branch's PR.
	CheckCI(ctx context.Context, prNumber int) (*CIResult, error)
	// WaitForCI waits for CI to complete. If prNumber is 0, checks the current branch's PR.
	WaitForCI(ctx context.Context, prNumber int, timeout time.Duration) (*CIResult, error)
	// WaitForCIWithOptions waits for CI with options. If prNumber is 0, checks the current branch's PR.
	WaitForCIWithOptions(ctx context.Context, prNumber int, timeout time.Duration, opts CheckCIOptions) (*CIResult, error)
	// WaitForCIWithProgress waits for CI with progress reporting
	WaitForCIWithProgress(ctx context.Context, prNumber int, timeout time.Duration, opts CheckCIOptions, onProgress CIProgressCallback) (*CIResult, error)
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
	workingDir     string
	checkInterval  time.Duration
	commandTimeout time.Duration
	initialDelay   time.Duration
}

// NewCIChecker creates a new CI checker
func NewCIChecker(workingDir string, checkInterval time.Duration) CIChecker {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}
	return &ciChecker{
		workingDir:     workingDir,
		checkInterval:  checkInterval,
		commandTimeout: 2 * time.Minute,
		initialDelay:   1 * time.Minute,
	}
}

// NewCICheckerWithOptions creates a new CI checker with custom options (for testing)
func NewCICheckerWithOptions(workingDir string, checkInterval, commandTimeout, initialDelay time.Duration) CIChecker {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}
	if commandTimeout == 0 {
		commandTimeout = 2 * time.Minute
	}
	if initialDelay == 0 {
		initialDelay = 1 * time.Minute
	}
	return &ciChecker{
		workingDir:     workingDir,
		checkInterval:  checkInterval,
		commandTimeout: commandTimeout,
		initialDelay:   initialDelay,
	}
}

// CheckCI checks the current CI status. If prNumber is 0, checks the current branch's PR.
func (c *ciChecker) CheckCI(ctx context.Context, prNumber int) (*CIResult, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * 5 * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		result, err := c.checkCIOnce(ctx, prNumber)
		if err == nil {
			return result, nil
		}

		if errors.Is(err, ErrCICheckTimeout) {
			lastErr = err
			continue
		}

		return result, err
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (c *ciChecker) checkCIOnce(ctx context.Context, prNumber int) (*CIResult, error) {
	result := &CIResult{
		Passed:     false,
		FailedJobs: []string{},
	}

	cmdCtx, cancel := context.WithTimeout(ctx, c.commandTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if prNumber > 0 {
		cmd = exec.CommandContext(cmdCtx, "gh", "pr", "checks", fmt.Sprintf("%d", prNumber))
	} else {
		cmd = exec.CommandContext(cmdCtx, "gh", "pr", "checks")
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
		if errors.Is(err, context.DeadlineExceeded) || (cmdCtx.Err() == context.DeadlineExceeded) {
			return result, ErrCICheckTimeout
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.String() == "signal: killed" {
				return result, ErrCICheckTimeout
			}

			switch exitErr.ExitCode() {
			case 127:
				return result, fmt.Errorf("gh CLI not found: is it installed?")
			case 8:
				if output != "" {
					result.Status, result.FailedJobs = parseCIOutput(output)
					result.Passed = result.Status == "success"
					return result, nil
				}
				return result, fmt.Errorf("no PR found for the current branch: ensure a PR exists before checking CI status")
			case 1:
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
	return c.WaitForCIWithProgress(ctx, prNumber, timeout, opts, nil)
}

// WaitForCIWithProgress waits for CI with progress reporting
func (c *ciChecker) WaitForCIWithProgress(ctx context.Context, prNumber int, timeout time.Duration, opts CheckCIOptions, onProgress CIProgressCallback) (*CIResult, error) {
	if timeout == 0 {
		timeout = 30 * time.Minute
	}

	if opts.E2ETestPattern == "" {
		opts.E2ETestPattern = "e2e|E2E|integration|Integration"
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()

	// Check CI status immediately first - if already complete, return right away
	if onProgress != nil {
		onProgress(CIProgressEvent{
			Type:    "checking",
			Elapsed: time.Since(startTime),
			Message: "Checking CI status",
		})
	}

	result, err := c.CheckCI(ctx, prNumber)
	if err != nil && !errors.Is(err, ErrCICheckTimeout) {
		return nil, err
	}

	if err == nil {
		passed, failed, pending := countJobStatuses(result.Output)
		if onProgress != nil {
			onProgress(CIProgressEvent{
				Type:        "status",
				Elapsed:     time.Since(startTime),
				Message:     fmt.Sprintf("CI status: %s", result.Status),
				JobsPassed:  passed,
				JobsFailed:  failed,
				JobsPending: pending,
				NextCheckIn: c.checkInterval,
			})
		}

		// If CI already completed, return immediately
		if result.Status == "success" || result.Status == "failure" {
			if opts.SkipE2E {
				result = filterE2EFailures(result, opts.E2ETestPattern)
			}
			return result, nil
		}
	}

	// CI is pending or had a timeout error - wait before polling
	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	initialDelayTimer := time.NewTimer(c.initialDelay)
	defer initialDelayTimer.Stop()

	progressTicker := time.NewTicker(5 * time.Second)
	defer progressTicker.Stop()

	waitStartTime := time.Now()

	// Wait for initial delay before starting polling loop
	for {
		select {
		case <-initialDelayTimer.C:
			goto checkLoop
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-progressTicker.C:
			if onProgress != nil {
				elapsed := time.Since(startTime)
				waitElapsed := time.Since(waitStartTime)
				remaining := c.initialDelay - waitElapsed
				if remaining < 0 {
					remaining = 0
				}
				onProgress(CIProgressEvent{
					Type:        "waiting",
					Elapsed:     elapsed,
					Message:     "Waiting for CI jobs to complete",
					NextCheckIn: remaining,
				})
			}
		}
	}

checkLoop:
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("CI check timeout after %s", timeout)
		case <-ticker.C:
			if onProgress != nil {
				onProgress(CIProgressEvent{
					Type:    "checking",
					Elapsed: time.Since(startTime),
					Message: "Checking CI status",
				})
			}

			result, err := c.CheckCI(ctx, prNumber)
			if err != nil {
				if errors.Is(err, ErrCICheckTimeout) {
					if onProgress != nil {
						onProgress(CIProgressEvent{
							Type:    "retry",
							Elapsed: time.Since(startTime),
							Message: "Command timeout, retrying",
						})
					}
					continue
				}
				return nil, err
			}

			passed, failed, pending := countJobStatuses(result.Output)
			if onProgress != nil {
				onProgress(CIProgressEvent{
					Type:        "status",
					Elapsed:     time.Since(startTime),
					Message:     fmt.Sprintf("CI status: %s", result.Status),
					JobsPassed:  passed,
					JobsFailed:  failed,
					JobsPending: pending,
					NextCheckIn: c.checkInterval,
				})
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

// countJobStatuses counts passed, failed, and pending jobs from CI output
func countJobStatuses(output string) (passed, failed, pending int) {
	lines := strings.Split(output, "\n")

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

		switch status {
		case "✓", "pass", "success":
			passed++
		case "✗", "fail", "failure":
			failed++
		case "○", "*", "pending", "queued", "in_progress":
			pending++
		}
	}

	return passed, failed, pending
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
