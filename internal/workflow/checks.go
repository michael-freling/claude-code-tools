package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/michael-freling/claude-code-tools/internal/command"
)

// ciCheck represents a single CI check from gh pr checks --json output
type ciCheck struct {
	Name        string `json:"name"`
	State       string `json:"state"`
	Conclusion  string `json:"conclusion"`
	StartedAt   string `json:"startedAt"`
	CompletedAt string `json:"completedAt"`
}

// CIProgressEvent represents a CI check progress update
type CIProgressEvent struct {
	Type          string // "waiting", "checking", "retry", "status"
	Elapsed       time.Duration
	Message       string
	JobsPassed    int
	JobsFailed    int
	JobsPending   int
	JobsCancelled int
	RetryAttempt  int
	NextCheckIn   time.Duration
}

// CIProgressCallback is called when CI check progress updates
type CIProgressCallback func(event CIProgressEvent)

// NoPRError represents the error when no PR is found for the current branch
type NoPRError struct {
	Branch string
	Msg    string
}

func (e *NoPRError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	if e.Branch == "" {
		return "no PR found for current branch"
	}
	return fmt.Sprintf("no PR found for branch %s", e.Branch)
}

// IsNoPRError checks if an error is a NoPRError or wraps one
func IsNoPRError(err error) bool {
	var noPRErr *NoPRError
	return errors.As(err, &noPRErr)
}

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
	Passed        bool
	Status        string
	FailedJobs    []string
	CancelledJobs []string
	Output        string
}

// ciChecker implements CIChecker interface
type ciChecker struct {
	workingDir     string
	checkInterval  time.Duration
	commandTimeout time.Duration
	initialDelay   time.Duration
	ghRunner       command.GhRunner
	clock          Clock
}

// NewCIChecker creates a new CI checker
func NewCIChecker(workingDir string, checkInterval time.Duration, commandTimeout time.Duration) CIChecker {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}
	if commandTimeout == 0 {
		commandTimeout = 2 * time.Minute
	}
	cmdRunner := command.NewRunner()
	return &ciChecker{
		workingDir:     workingDir,
		checkInterval:  checkInterval,
		commandTimeout: commandTimeout,
		initialDelay:   1 * time.Minute,
		ghRunner:       command.NewGhRunner(cmdRunner),
		clock:          NewRealClock(),
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
	cmdRunner := command.NewRunner()
	return &ciChecker{
		workingDir:     workingDir,
		checkInterval:  checkInterval,
		commandTimeout: commandTimeout,
		initialDelay:   initialDelay,
		ghRunner:       command.NewGhRunner(cmdRunner),
		clock:          NewRealClock(),
	}
}

// NewCICheckerWithRunner creates a new CI checker with injected GhRunner (for testing)
func NewCICheckerWithRunner(workingDir string, checkInterval, commandTimeout, initialDelay time.Duration, ghRunner command.GhRunner) CIChecker {
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
		ghRunner:       ghRunner,
		clock:          NewRealClock(),
	}
}

// NewCICheckerWithClock creates a new CI checker with injected GhRunner and Clock (for testing)
func NewCICheckerWithClock(workingDir string, checkInterval, commandTimeout, initialDelay time.Duration, ghRunner command.GhRunner, clock Clock) CIChecker {
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
		ghRunner:       ghRunner,
		clock:          clock,
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
			case <-c.clock.After(backoff):
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

	output, err := c.ghRunner.PRChecks(cmdCtx, c.workingDir, prNumber, "name,state,conclusion,startedAt,completedAt")
	result.Output = output

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || (cmdCtx.Err() == context.DeadlineExceeded) {
			return result, ErrCICheckTimeout
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.String() == "signal: killed" {
				return result, ErrCICheckTimeout
			}

			switch exitErr.ExitCode() {
			case 127:
				return result, fmt.Errorf("gh CLI not found: is it installed?")
			case 8:
				return result, &NoPRError{
					Branch: "",
					Msg:    "no PR found for the current branch: ensure a PR exists before checking CI status",
				}
			case 1:
				return result, err
			}
		}

		return result, err
	}

	result.Status, result.FailedJobs, result.CancelledJobs = parseCIOutput(output)
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

	startTime := c.clock.Now()

	// Check CI status immediately first - if already complete, return right away
	if onProgress != nil {
		onProgress(CIProgressEvent{
			Type:    "checking",
			Elapsed: c.clock.Since(startTime),
			Message: "Checking CI status",
		})
	}

	result, err := c.CheckCI(ctx, prNumber)
	if err != nil && !errors.Is(err, ErrCICheckTimeout) {
		return nil, err
	}

	if err == nil {
		passed, failed, pending, cancelled := countJobStatuses(result.Output)
		if onProgress != nil {
			onProgress(CIProgressEvent{
				Type:          "status",
				Elapsed:       c.clock.Since(startTime),
				Message:       fmt.Sprintf("CI status: %s", result.Status),
				JobsPassed:    passed,
				JobsFailed:    failed,
				JobsPending:   pending,
				JobsCancelled: cancelled,
				NextCheckIn:   c.checkInterval,
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
	ticker := c.clock.NewTicker(c.checkInterval)
	defer ticker.Stop()

	initialDelayTimer := c.clock.NewTimer(c.initialDelay)
	defer initialDelayTimer.Stop()

	progressTicker := c.clock.NewTicker(5 * time.Second)
	defer progressTicker.Stop()

	waitStartTime := c.clock.Now()

	// Wait for initial delay before starting polling loop
	for {
		select {
		case <-initialDelayTimer.C():
			goto checkLoop
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-progressTicker.C():
			if onProgress != nil {
				elapsed := c.clock.Since(startTime)
				waitElapsed := c.clock.Since(waitStartTime)
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
		case <-ticker.C():
			if onProgress != nil {
				onProgress(CIProgressEvent{
					Type:    "checking",
					Elapsed: c.clock.Since(startTime),
					Message: "Checking CI status",
				})
			}

			result, err := c.CheckCI(ctx, prNumber)
			if err != nil {
				if errors.Is(err, ErrCICheckTimeout) {
					if onProgress != nil {
						onProgress(CIProgressEvent{
							Type:    "retry",
							Elapsed: c.clock.Since(startTime),
							Message: "Command timeout, retrying",
						})
					}
					continue
				}
				return nil, err
			}

			passed, failed, pending, cancelled := countJobStatuses(result.Output)
			if onProgress != nil {
				onProgress(CIProgressEvent{
					Type:          "status",
					Elapsed:       c.clock.Since(startTime),
					Message:       fmt.Sprintf("CI status: %s", result.Status),
					JobsPassed:    passed,
					JobsFailed:    failed,
					JobsPending:   pending,
					JobsCancelled: cancelled,
					NextCheckIn:   c.checkInterval,
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

// parseCIOutput parses gh pr checks --json output to extract status, failed jobs, and cancelled jobs
// The output is expected to be JSON array: [{"name":"build","state":"SUCCESS"},...]
// State values: SUCCESS, FAILURE, PENDING, QUEUED, IN_PROGRESS, SKIPPED, NEUTRAL, CANCELLED
func parseCIOutput(output string) (string, []string, []string) {
	var checks []ciCheck
	if err := json.Unmarshal([]byte(output), &checks); err != nil {
		// If JSON parsing fails, return pending (safest default)
		return "pending", []string{}, []string{}
	}

	if len(checks) == 0 {
		return "pending", []string{}, []string{}
	}

	failedJobs := []string{}
	cancelledJobs := []string{}
	allPassed := true
	hasPending := false

	for _, check := range checks {
		state := strings.ToUpper(check.State)
		switch state {
		case "SUCCESS", "SKIPPED", "NEUTRAL":
			// These are considered passing states
		case "FAILURE":
			allPassed = false
			failedJobs = append(failedJobs, check.Name)
		case "CANCELLED":
			allPassed = false
			cancelledJobs = append(cancelledJobs, check.Name)
		case "PENDING", "QUEUED", "IN_PROGRESS", "":
			hasPending = true
		default:
			// Unknown state, treat as pending to be safe
			hasPending = true
		}
	}

	if hasPending {
		return "pending", failedJobs, cancelledJobs
	}

	if allPassed {
		return "success", failedJobs, cancelledJobs
	}

	return "failure", failedJobs, cancelledJobs
}

// countJobStatuses counts passed, failed, pending, and cancelled jobs from CI JSON output
func countJobStatuses(output string) (passed, failed, pending, cancelled int) {
	var checks []ciCheck
	if err := json.Unmarshal([]byte(output), &checks); err != nil {
		return 0, 0, 0, 0
	}

	for _, check := range checks {
		state := strings.ToUpper(check.State)
		switch state {
		case "SUCCESS", "SKIPPED", "NEUTRAL":
			passed++
		case "FAILURE":
			failed++
		case "CANCELLED":
			cancelled++
		case "PENDING", "QUEUED", "IN_PROGRESS", "":
			pending++
		default:
			// Unknown state, count as pending
			pending++
		}
	}

	return passed, failed, pending, cancelled
}

// filterE2EFailures filters out e2e test failures and cancellations from CI result
func filterE2EFailures(result *CIResult, e2ePattern string) *CIResult {
	if result.Passed {
		return result
	}

	e2eRegex, err := regexp.Compile(e2ePattern)
	if err != nil {
		return result
	}

	filteredFailedJobs := []string{}
	for _, job := range result.FailedJobs {
		if !e2eRegex.MatchString(job) {
			filteredFailedJobs = append(filteredFailedJobs, job)
		}
	}

	filteredCancelledJobs := []string{}
	for _, job := range result.CancelledJobs {
		if !e2eRegex.MatchString(job) {
			filteredCancelledJobs = append(filteredCancelledJobs, job)
		}
	}

	filtered := &CIResult{
		Status:        result.Status,
		Output:        result.Output,
		FailedJobs:    filteredFailedJobs,
		CancelledJobs: filteredCancelledJobs,
		Passed:        len(filteredFailedJobs) == 0 && len(filteredCancelledJobs) == 0,
	}

	return filtered
}
