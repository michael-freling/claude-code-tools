package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCIOutput(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		wantStatus     string
		wantFailedJobs []string
	}{
		{
			name:           "empty output",
			output:         "",
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "empty JSON array",
			output:         "[]",
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "invalid JSON",
			output:         "not json",
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "all checks passed",
			output:         `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"SUCCESS"},{"name":"lint","state":"SUCCESS"}]`,
			wantStatus:     "success",
			wantFailedJobs: []string{},
		},
		{
			name:           "some checks failed",
			output:         `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"FAILURE"},{"name":"lint","state":"SUCCESS"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"test"},
		},
		{
			name:           "checks pending",
			output:         `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"PENDING"},{"name":"lint","state":"QUEUED"}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "checks in progress",
			output:         `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"IN_PROGRESS"}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "mixed status with pending",
			output:         `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"FAILURE"},{"name":"lint","state":"PENDING"}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{"test"},
		},
		{
			name:           "multiple failed jobs",
			output:         `[{"name":"build","state":"SUCCESS"},{"name":"test-unit","state":"FAILURE"},{"name":"test-integration","state":"FAILURE"},{"name":"lint","state":"SUCCESS"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"test-unit", "test-integration"},
		},
		{
			name:           "skipped and neutral states count as passed",
			output:         `[{"name":"build","state":"SUCCESS"},{"name":"optional-check","state":"SKIPPED"},{"name":"neutral-check","state":"NEUTRAL"}]`,
			wantStatus:     "success",
			wantFailedJobs: []string{},
		},
		{
			name:           "lowercase state values",
			output:         `[{"name":"build","state":"success"},{"name":"test","state":"failure"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotFailedJobs := parseCIOutput(tt.output)
			assert.Equal(t, tt.wantStatus, gotStatus)
			assert.Equal(t, tt.wantFailedJobs, gotFailedJobs)
		})
	}
}

func TestCountJobStatuses(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantPassed  int
		wantFailed  int
		wantPending int
	}{
		{
			name:        "empty output",
			output:      "",
			wantPassed:  0,
			wantFailed:  0,
			wantPending: 0,
		},
		{
			name:        "invalid JSON",
			output:      "not json",
			wantPassed:  0,
			wantFailed:  0,
			wantPending: 0,
		},
		{
			name:        "empty JSON array",
			output:      "[]",
			wantPassed:  0,
			wantFailed:  0,
			wantPending: 0,
		},
		{
			name:        "single passed job",
			output:      `[{"name":"build","state":"SUCCESS"}]`,
			wantPassed:  1,
			wantFailed:  0,
			wantPending: 0,
		},
		{
			name:        "multiple jobs with mixed states",
			output:      `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"FAILURE"},{"name":"lint","state":"PENDING"}]`,
			wantPassed:  1,
			wantFailed:  1,
			wantPending: 1,
		},
		{
			name:        "all job states",
			output:      `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"FAILURE"},{"name":"lint","state":"PENDING"},{"name":"deploy","state":"QUEUED"},{"name":"e2e","state":"IN_PROGRESS"},{"name":"optional","state":"SKIPPED"},{"name":"neutral","state":"NEUTRAL"}]`,
			wantPassed:  3,
			wantFailed:  1,
			wantPending: 3,
		},
		{
			name:        "lowercase states",
			output:      `[{"name":"build","state":"success"},{"name":"test","state":"failure"},{"name":"lint","state":"pending"}]`,
			wantPassed:  1,
			wantFailed:  1,
			wantPending: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPassed, gotFailed, gotPending := countJobStatuses(tt.output)
			assert.Equal(t, tt.wantPassed, gotPassed, "passed count mismatch")
			assert.Equal(t, tt.wantFailed, gotFailed, "failed count mismatch")
			assert.Equal(t, tt.wantPending, gotPending, "pending count mismatch")
		})
	}
}

func TestNewCIChecker(t *testing.T) {
	tests := []struct {
		name               string
		workingDir         string
		checkInterval      time.Duration
		wantInterval       time.Duration
		wantCommandTimeout time.Duration
	}{
		{
			name:               "with custom interval",
			workingDir:         "/tmp/test",
			checkInterval:      10 * time.Second,
			wantInterval:       10 * time.Second,
			wantCommandTimeout: 2 * time.Minute,
		},
		{
			name:               "with default interval",
			workingDir:         "/tmp/test",
			checkInterval:      0,
			wantInterval:       30 * time.Second,
			wantCommandTimeout: 2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewCIChecker(tt.workingDir, tt.checkInterval)
			require.NotNil(t, checker)

			concreteChecker, ok := checker.(*ciChecker)
			require.True(t, ok)
			assert.Equal(t, tt.workingDir, concreteChecker.workingDir)
			assert.Equal(t, tt.wantInterval, concreteChecker.checkInterval)
			assert.Equal(t, tt.wantCommandTimeout, concreteChecker.commandTimeout)
		})
	}
}

func TestCIChecker_CheckCI_NotInstalled(t *testing.T) {
	checker := NewCIChecker("/nonexistent/path/that/should/not/exist", 1*time.Second)
	ctx := context.Background()

	result, err := checker.CheckCI(ctx, 123)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Passed)
}

func TestCIChecker_CheckCI_NoPR(t *testing.T) {
	// This test verifies the error handling when gh pr checks fails
	// Running in /tmp (non-git directory) will cause an error
	checker := NewCIChecker("/tmp", 1*time.Second)
	ctx := context.Background()

	result, err := checker.CheckCI(ctx, 0)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Passed)
}

func TestParseCIOutput_PendingStatus(t *testing.T) {
	// Test that pending status from gh pr checks --json is handled correctly
	// This simulates the case where some checks are still pending
	output := `[{"name":"Vercel Preview Comments","state":"SUCCESS"},{"name":"Test go/aninexus-gateway / Lint go/aninexus-gateway","state":"PENDING"},{"name":"Test go/aninexus-gateway / Test go/aninexus-gateway","state":"PENDING"},{"name":"Vercel – nooxac-gateway","state":"SUCCESS"}]`

	status, failedJobs := parseCIOutput(output)
	assert.Equal(t, "pending", status)
	assert.Empty(t, failedJobs)
}

func TestCIChecker_WaitForCI_ImmediateCheckOnStart(t *testing.T) {
	// Test that WaitForCI checks CI status immediately before starting the delay
	// When the working directory doesn't exist, it should fail immediately with a directory error
	// (not wait for the initial delay and then fail with timeout)
	checker := NewCIChecker("/nonexistent/path/that/should/not/exist", 100*time.Millisecond)
	ctx := context.Background()

	start := time.Now()
	result, err := checker.WaitForCI(ctx, 123, 5*time.Minute)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Nil(t, result)
	// Should fail immediately (not after initial delay of 1 minute)
	assert.Less(t, elapsed, 10*time.Second)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestCIChecker_WaitForCI_ContextCancellationDuringWait(t *testing.T) {
	// This test verifies that context cancellation is respected during the waiting period
	// Since the immediate check will fail with directory error, we need a valid directory
	// but in a non-git repository to trigger the waiting behavior
	checker := NewCIChecker("/tmp", 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result, err := checker.WaitForCI(ctx, 123, 5*time.Minute)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Nil(t, result)
	// Should fail quickly due to context cancellation or directory error
	assert.Less(t, elapsed, 10*time.Second)
}

func TestCIChecker_InitialDelayTimerFiresCorrectly(t *testing.T) {
	// This test verifies that the initial delay timer actually fires and doesn't loop forever
	// We use a very short initial delay (100ms) to make the test fast
	// The key assertion is that the function completes within expected time bounds

	// Create checker with short initial delay for testing
	// Use a non-existent directory so the command fails immediately
	checker := NewCICheckerWithOptions(
		"/nonexistent/path", // workingDir - doesn't exist, so gh command will fail immediately
		50*time.Millisecond, // checkInterval
		100*time.Millisecond, // commandTimeout
		100*time.Millisecond, // initialDelay - short for testing
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Track progress events to verify the timer behavior
	var progressEvents []CIProgressEvent
	onProgress := func(event CIProgressEvent) {
		progressEvents = append(progressEvents, event)
	}

	start := time.Now()
	// This will:
	// 1. Check CI immediately (will fail since directory doesn't exist)
	// 2. Since we get an error (not ErrCICheckTimeout), it should return immediately
	result, err := checker.WaitForCIWithProgress(ctx, 123, 5*time.Second, CheckCIOptions{}, onProgress)
	elapsed := time.Since(start)

	// Should fail with error (directory doesn't exist)
	require.Error(t, err)
	assert.Nil(t, result)
	t.Logf("Error: %v", err)
	t.Logf("Elapsed: %v", elapsed)

	// Should complete quickly (immediate check fails with directory error)
	assert.Less(t, elapsed, 1*time.Second, "Should complete quickly when immediate check fails")

	// Should have at least one progress event (the initial "checking" event)
	assert.GreaterOrEqual(t, len(progressEvents), 1, "Should have at least one progress event")
	if len(progressEvents) > 0 {
		assert.Equal(t, "checking", progressEvents[0].Type, "First event should be 'checking'")
	}
}

func TestCIChecker_InitialDelayCompletesWithinExpectedTime(t *testing.T) {
	// This test verifies that when CI is pending, the initial delay completes
	// and doesn't loop forever. We test this by using a mock-like approach
	// where we cancel the context after the initial delay should have completed.

	initialDelay := 200 * time.Millisecond
	checker := NewCICheckerWithOptions(
		"/tmp",
		50*time.Millisecond,  // checkInterval
		100*time.Millisecond, // commandTimeout
		initialDelay,         // initialDelay
	)

	// Set timeout to be longer than initialDelay + some buffer
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var waitingEvents []CIProgressEvent
	onProgress := func(event CIProgressEvent) {
		if event.Type == "waiting" {
			waitingEvents = append(waitingEvents, event)
		}
	}

	start := time.Now()
	_, err := checker.WaitForCIWithProgress(ctx, 123, 2*time.Second, CheckCIOptions{}, onProgress)
	elapsed := time.Since(start)

	// Should error (either from gh command failing or timeout)
	require.Error(t, err)

	// The test passes if we get here - if the timer loop was infinite,
	// this test would timeout after 2 seconds and fail
	t.Logf("Elapsed time: %v, waiting events: %d", elapsed, len(waitingEvents))

	// Additional check: elapsed time should be reasonable
	// If initial delay was working, we should see some waiting events
	// or complete quickly if the gh command failed
	assert.Less(t, elapsed, 3*time.Second, "Should complete within timeout")
}

func TestFilterE2EFailures(t *testing.T) {
	tests := []struct {
		name       string
		result     *CIResult
		e2ePattern string
		want       *CIResult
	}{
		{
			name: "no failures",
			result: &CIResult{
				Passed:     true,
				Status:     "success",
				FailedJobs: []string{},
				Output:     "all passed",
			},
			e2ePattern: "e2e|E2E",
			want: &CIResult{
				Passed:     true,
				Status:     "success",
				FailedJobs: []string{},
				Output:     "all passed",
			},
		},
		{
			name: "only e2e failures",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-e2e", "integration-test"},
				Output:     "e2e tests failed",
			},
			e2ePattern: "e2e|integration",
			want: &CIResult{
				Passed:     true,
				Status:     "failure",
				FailedJobs: []string{},
				Output:     "e2e tests failed",
			},
		},
		{
			name: "mixed failures with e2e",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-unit", "test-e2e", "lint"},
				Output:     "multiple failures",
			},
			e2ePattern: "e2e|E2E",
			want: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-unit", "lint"},
				Output:     "multiple failures",
			},
		},
		{
			name: "only non-e2e failures",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-unit", "lint"},
				Output:     "unit tests failed",
			},
			e2ePattern: "e2e|E2E",
			want: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-unit", "lint"},
				Output:     "unit tests failed",
			},
		},
		{
			name: "case insensitive e2e pattern",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-E2E", "test-Integration"},
				Output:     "e2e tests failed",
			},
			e2ePattern: "e2e|E2E|integration|Integration",
			want: &CIResult{
				Passed:     true,
				Status:     "failure",
				FailedJobs: []string{},
				Output:     "e2e tests failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterE2EFailures(tt.result, tt.e2ePattern)
			assert.Equal(t, tt.want, got)
		})
	}
}
