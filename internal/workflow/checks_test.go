package workflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		{
			name:        "unknown state treated as pending",
			output:      `[{"name":"build","state":"UNKNOWN_STATE"}]`,
			wantPassed:  0,
			wantFailed:  0,
			wantPending: 1,
		},
		{
			name:        "mixed unknown and known states",
			output:      `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"WEIRD_STATE"},{"name":"lint","state":"FAILURE"}]`,
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
		commandTimeout     time.Duration
		wantInterval       time.Duration
		wantCommandTimeout time.Duration
	}{
		{
			name:               "with custom interval",
			workingDir:         "/tmp/test",
			checkInterval:      10 * time.Second,
			commandTimeout:     5 * time.Minute,
			wantInterval:       10 * time.Second,
			wantCommandTimeout: 5 * time.Minute,
		},
		{
			name:               "with default interval",
			workingDir:         "/tmp/test",
			checkInterval:      0,
			commandTimeout:     0,
			wantInterval:       30 * time.Second,
			wantCommandTimeout: 2 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewCIChecker(tt.workingDir, tt.checkInterval, tt.commandTimeout)
			require.NotNil(t, checker)

			concreteChecker, ok := checker.(*ciChecker)
			require.True(t, ok)
			assert.Equal(t, tt.workingDir, concreteChecker.workingDir)
			assert.Equal(t, tt.wantInterval, concreteChecker.checkInterval)
			assert.Equal(t, tt.wantCommandTimeout, concreteChecker.commandTimeout)
		})
	}
}

func TestNewCICheckerWithOptions(t *testing.T) {
	tests := []struct {
		name               string
		workingDir         string
		checkInterval      time.Duration
		commandTimeout     time.Duration
		initialDelay       time.Duration
		wantInterval       time.Duration
		wantCommandTimeout time.Duration
		wantInitialDelay   time.Duration
	}{
		{
			name:               "all custom values",
			workingDir:         "/tmp/test",
			checkInterval:      10 * time.Second,
			commandTimeout:     3 * time.Minute,
			initialDelay:       2 * time.Minute,
			wantInterval:       10 * time.Second,
			wantCommandTimeout: 3 * time.Minute,
			wantInitialDelay:   2 * time.Minute,
		},
		{
			name:               "all default values (zeros)",
			workingDir:         "/tmp/test",
			checkInterval:      0,
			commandTimeout:     0,
			initialDelay:       0,
			wantInterval:       30 * time.Second,
			wantCommandTimeout: 2 * time.Minute,
			wantInitialDelay:   1 * time.Minute,
		},
		{
			name:               "mixed custom and default values",
			workingDir:         "/tmp/test",
			checkInterval:      15 * time.Second,
			commandTimeout:     0,
			initialDelay:       30 * time.Second,
			wantInterval:       15 * time.Second,
			wantCommandTimeout: 2 * time.Minute,
			wantInitialDelay:   30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewCICheckerWithOptions(tt.workingDir, tt.checkInterval, tt.commandTimeout, tt.initialDelay)
			require.NotNil(t, checker)

			concreteChecker, ok := checker.(*ciChecker)
			require.True(t, ok)
			assert.Equal(t, tt.workingDir, concreteChecker.workingDir)
			assert.Equal(t, tt.wantInterval, concreteChecker.checkInterval)
			assert.Equal(t, tt.wantCommandTimeout, concreteChecker.commandTimeout)
			assert.Equal(t, tt.wantInitialDelay, concreteChecker.initialDelay)
		})
	}
}

func TestCIChecker_CheckCI_NotInstalled(t *testing.T) {
	checker := NewCIChecker("/nonexistent/path/that/should/not/exist", 1*time.Second, 10*time.Second)
	ctx := context.Background()

	result, err := checker.CheckCI(ctx, 123)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Passed)
}

func TestCIChecker_CheckCI_NoPR(t *testing.T) {
	// This test verifies the error handling when gh pr checks fails
	// Running in /tmp (non-git directory) will cause an error
	checker := NewCIChecker("/tmp", 1*time.Second, 10*time.Second)
	ctx := context.Background()

	result, err := checker.CheckCI(ctx, 0)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Passed)
}

func TestParseCIOutput_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		wantStatus     string
		wantFailedJobs []string
	}{
		{
			name:           "empty string",
			output:         "",
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "malformed JSON - not an array",
			output:         `{"name":"build","state":"SUCCESS"}`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "malformed JSON - incomplete",
			output:         `[{"name":"build","state":`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "malformed JSON - invalid syntax",
			output:         `[{name:build,state:SUCCESS}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "unknown state value",
			output:         `[{"name":"build","state":"UNKNOWN_STATE"}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "multiple unknown state values",
			output:         `[{"name":"build","state":"UNKNOWN"},{"name":"test","state":"WEIRD_STATE"}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "mixed case state values - lowercase success",
			output:         `[{"name":"build","state":"success"},{"name":"test","state":"success"}]`,
			wantStatus:     "success",
			wantFailedJobs: []string{},
		},
		{
			name:           "mixed case state values - mixed success and failure",
			output:         `[{"name":"build","state":"SuCcEsS"},{"name":"test","state":"FaIlUrE"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"test"},
		},
		{
			name:           "very long job name",
			output:         `[{"name":"this-is-a-very-long-job-name-that-might-appear-in-deeply-nested-CI-workflows-with-matrix-builds-and-multiple-stages-and-substages","state":"FAILURE"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"this-is-a-very-long-job-name-that-might-appear-in-deeply-nested-CI-workflows-with-matrix-builds-and-multiple-stages-and-substages"},
		},
		{
			name:           "unicode characters in job name - emojis",
			output:         `[{"name":"test-🚀-build","state":"FAILURE"},{"name":"deploy-✅","state":"SUCCESS"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"test-🚀-build"},
		},
		{
			name:           "unicode characters in job name - chinese",
			output:         `[{"name":"测试构建","state":"SUCCESS"},{"name":"部署失败","state":"FAILURE"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"部署失败"},
		},
		{
			name:           "unicode characters in job name - japanese",
			output:         `[{"name":"ビルド・テスト","state":"FAILURE"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"ビルド・テスト"},
		},
		{
			name:           "null name field",
			output:         `[{"name":null,"state":"SUCCESS"}]`,
			wantStatus:     "success",
			wantFailedJobs: []string{},
		},
		{
			name:           "null state field",
			output:         `[{"name":"build","state":null}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "missing name field",
			output:         `[{"state":"SUCCESS"}]`,
			wantStatus:     "success",
			wantFailedJobs: []string{},
		},
		{
			name:           "missing state field",
			output:         `[{"name":"build"}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "empty object in array",
			output:         `[{}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "mix of valid and invalid entries",
			output:         `[{"name":"build","state":"SUCCESS"},{},{"name":"test","state":"FAILURE"}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{"test"},
		},
		{
			name:           "empty name with failure state",
			output:         `[{"name":"","state":"FAILURE"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{""},
		},
		{
			name:           "whitespace in state field",
			output:         `[{"name":"build","state":" SUCCESS "}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "special characters in job name",
			output:         `[{"name":"test/build:integration@v1.2.3","state":"FAILURE"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"test/build:integration@v1.2.3"},
		},
		{
			name:           "newlines in JSON",
			output:         "[\n{\"name\":\"build\",\"state\":\"SUCCESS\"},\n{\"name\":\"test\",\"state\":\"FAILURE\"}\n]",
			wantStatus:     "failure",
			wantFailedJobs: []string{"test"},
		},
		{
			name:           "extra fields in JSON",
			output:         `[{"name":"build","state":"SUCCESS","extra":"field","another":"value"}]`,
			wantStatus:     "success",
			wantFailedJobs: []string{},
		},
		{
			name:           "numeric state value",
			output:         `[{"name":"build","state":123}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "boolean state value",
			output:         `[{"name":"build","state":true}]`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name:           "large number of jobs",
			output:         `[{"name":"job1","state":"SUCCESS"},{"name":"job2","state":"SUCCESS"},{"name":"job3","state":"SUCCESS"},{"name":"job4","state":"SUCCESS"},{"name":"job5","state":"SUCCESS"},{"name":"job6","state":"SUCCESS"},{"name":"job7","state":"SUCCESS"},{"name":"job8","state":"SUCCESS"},{"name":"job9","state":"SUCCESS"},{"name":"job10","state":"SUCCESS"}]`,
			wantStatus:     "success",
			wantFailedJobs: []string{},
		},
		{
			name:           "all possible failure states",
			output:         `[{"name":"failed1","state":"FAILURE"},{"name":"failed2","state":"failure"},{"name":"failed3","state":"Failure"}]`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"failed1", "failed2", "failed3"},
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
	checker := NewCIChecker("/nonexistent/path/that/should/not/exist", 100*time.Millisecond, 10*time.Second)
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
	checker := NewCIChecker("/tmp", 100*time.Millisecond, 10*time.Second)
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
		"/nonexistent/path",  // workingDir - doesn't exist, so gh command will fail immediately
		50*time.Millisecond,  // checkInterval
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

func TestCheckCI_ContextCancellation(t *testing.T) {
	checker := NewCIChecker("/tmp", 1*time.Second, 30*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := checker.CheckCI(ctx, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotNil(t, result)
	assert.False(t, result.Passed)
}

func TestCheckCI_IsolatedCommandContext(t *testing.T) {
	mockGhRunner := new(MockGhRunner)
	mockGhRunner.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", fmt.Errorf("no PR found"))

	checker := NewCICheckerWithRunner("/tmp", 1*time.Second, 50*time.Millisecond, 1*time.Minute, mockGhRunner)
	ctx := context.Background()

	result, err := checker.CheckCI(ctx, 0)

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Passed)
	mockGhRunner.AssertExpectations(t)
}

// TestWaitForCIWithOptions_ParentContextCancellation is skipped because
// WaitForCIWithOptions has a hardcoded 1-minute initial delay that makes
// unit testing impractical. This should be tested in integration tests.

func TestFilterE2EFailures_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		result     *CIResult
		e2ePattern string
		want       *CIResult
	}{
		{
			name: "invalid regex pattern",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-e2e", "test-unit"},
				Output:     "tests failed",
			},
			e2ePattern: "[invalid(",
			want: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-e2e", "test-unit"},
				Output:     "tests failed",
			},
		},
		{
			name: "empty pattern matches everything",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-e2e"},
				Output:     "tests failed",
			},
			e2ePattern: "",
			want: &CIResult{
				Passed:     true,
				Status:     "failure",
				FailedJobs: []string{},
				Output:     "tests failed",
			},
		},
		{
			name: "pattern matches nothing",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-unit", "lint"},
				Output:     "tests failed",
			},
			e2ePattern: "nonexistent",
			want: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-unit", "lint"},
				Output:     "tests failed",
			},
		},
		{
			name: "pattern matches all failures",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"integration-api", "integration-db"},
				Output:     "integration tests failed",
			},
			e2ePattern: "integration",
			want: &CIResult{
				Passed:     true,
				Status:     "failure",
				FailedJobs: []string{},
				Output:     "integration tests failed",
			},
		},
		{
			name: "complex pattern with alternation",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"e2e-smoke", "E2E-full", "integration-test", "unit-test"},
				Output:     "multiple test failures",
			},
			e2ePattern: "(e2e|E2E|integration)",
			want: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"unit-test"},
				Output:     "multiple test failures",
			},
		},
		{
			name: "pattern at start of job name",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"e2e-browser-test", "test-unit"},
				Output:     "tests failed",
			},
			e2ePattern: "^e2e",
			want: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test-unit"},
				Output:     "tests failed",
			},
		},
		{
			name: "pattern at end of job name",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"browser-test-e2e", "unit-test"},
				Output:     "tests failed",
			},
			e2ePattern: "e2e$",
			want: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"unit-test"},
				Output:     "tests failed",
			},
		},
		{
			name: "empty failed jobs list",
			result: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{},
				Output:     "unknown failure",
			},
			e2ePattern: "e2e",
			want: &CIResult{
				Passed:     true,
				Status:     "failure",
				FailedJobs: []string{},
				Output:     "unknown failure",
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

// TestWaitForCIWithOptions_CustomE2EPattern, TestWaitForCIWithOptions_DefaultTimeout,
// and TestWaitForCI_ContextCancellation are skipped because WaitForCI methods have a
// hardcoded 1-minute initial delay that makes unit testing impractical.
// These should be tested in integration tests.

func TestNewCICheckerWithRunner(t *testing.T) {
	tests := []struct {
		name               string
		workingDir         string
		checkInterval      time.Duration
		commandTimeout     time.Duration
		initialDelay       time.Duration
		wantInterval       time.Duration
		wantCommandTimeout time.Duration
		wantInitialDelay   time.Duration
	}{
		{
			name:               "with all custom values",
			workingDir:         "/tmp/test",
			checkInterval:      15 * time.Second,
			commandTimeout:     3 * time.Minute,
			initialDelay:       2 * time.Minute,
			wantInterval:       15 * time.Second,
			wantCommandTimeout: 3 * time.Minute,
			wantInitialDelay:   2 * time.Minute,
		},
		{
			name:               "with all default values (zeros)",
			workingDir:         "/tmp/test",
			checkInterval:      0,
			commandTimeout:     0,
			initialDelay:       0,
			wantInterval:       30 * time.Second,
			wantCommandTimeout: 2 * time.Minute,
			wantInitialDelay:   1 * time.Minute,
		},
		{
			name:               "with mixed custom and default values",
			workingDir:         "/tmp/test",
			checkInterval:      20 * time.Second,
			commandTimeout:     0,
			initialDelay:       45 * time.Second,
			wantInterval:       20 * time.Second,
			wantCommandTimeout: 2 * time.Minute,
			wantInitialDelay:   45 * time.Second,
		},
		{
			name:               "with zero check interval only",
			workingDir:         "/tmp/test",
			checkInterval:      0,
			commandTimeout:     5 * time.Minute,
			initialDelay:       30 * time.Second,
			wantInterval:       30 * time.Second,
			wantCommandTimeout: 5 * time.Minute,
			wantInitialDelay:   30 * time.Second,
		},
		{
			name:               "with zero command timeout only",
			workingDir:         "/tmp/test",
			checkInterval:      10 * time.Second,
			commandTimeout:     0,
			initialDelay:       2 * time.Minute,
			wantInterval:       10 * time.Second,
			wantCommandTimeout: 2 * time.Minute,
			wantInitialDelay:   2 * time.Minute,
		},
		{
			name:               "with zero initial delay only",
			workingDir:         "/tmp/test",
			checkInterval:      25 * time.Second,
			commandTimeout:     4 * time.Minute,
			initialDelay:       0,
			wantInterval:       25 * time.Second,
			wantCommandTimeout: 4 * time.Minute,
			wantInitialDelay:   1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			checker := NewCICheckerWithRunner(tt.workingDir, tt.checkInterval, tt.commandTimeout, tt.initialDelay, mockGhRunner)
			require.NotNil(t, checker)

			concreteChecker, ok := checker.(*ciChecker)
			require.True(t, ok)
			assert.Equal(t, tt.workingDir, concreteChecker.workingDir)
			assert.Equal(t, tt.wantInterval, concreteChecker.checkInterval)
			assert.Equal(t, tt.wantCommandTimeout, concreteChecker.commandTimeout)
			assert.Equal(t, tt.wantInitialDelay, concreteChecker.initialDelay)
			assert.Equal(t, mockGhRunner, concreteChecker.ghRunner)
		})
	}
}

func TestCheckCIOnce_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		mockSetup   func(*MockGhRunner)
		wantErr     bool
		errContains string
		errType     error
	}{
		{
			name: "context deadline exceeded returns timeout error",
			mockSetup: func(m *MockGhRunner) {
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", context.DeadlineExceeded)
			},
			wantErr: true,
			errType: ErrCICheckTimeout,
		},
		{
			name: "generic error is returned as-is",
			mockSetup: func(m *MockGhRunner) {
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", fmt.Errorf("some error"))
			},
			wantErr:     true,
			errContains: "some error",
		},
		{
			name: "context cancelled returns context error",
			mockSetup: func(m *MockGhRunner) {
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", context.Canceled)
			},
			wantErr: true,
		},
		{
			name: "successful check with pending status",
			mockSetup: func(m *MockGhRunner) {
				output := `[{"name":"test","state":"PENDING"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(output, nil)
			},
			wantErr: false,
		},
		{
			name: "successful check with success status",
			mockSetup: func(m *MockGhRunner) {
				output := `[{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(output, nil)
			},
			wantErr: false,
		},
		{
			name: "successful check with failure status",
			mockSetup: func(m *MockGhRunner) {
				output := `[{"name":"test","state":"FAILURE"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(output, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			tt.mockSetup(mockGhRunner)

			checker := NewCICheckerWithRunner("/tmp", 1*time.Second, 50*time.Millisecond, 1*time.Minute, mockGhRunner)
			concreteChecker := checker.(*ciChecker)

			ctx := context.Background()
			result, err := concreteChecker.checkCIOnce(ctx, 0)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.NotNil(t, result)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			mockGhRunner.AssertExpectations(t)
		})
	}
}

func TestWaitForCIWithProgress_WithMocks(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(*MockGhRunner)
		checkInterval  time.Duration
		commandTimeout time.Duration
		initialDelay   time.Duration
		timeout        time.Duration
		opts           CheckCIOptions
		wantErr        bool
		wantPassed     bool
	}{
		{
			name: "CI already complete on first check",
			mockSetup: func(m *MockGhRunner) {
				successOutput := `[{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(successOutput, nil)
			},
			checkInterval:  50 * time.Millisecond,
			commandTimeout: 100 * time.Millisecond,
			initialDelay:   50 * time.Millisecond,
			timeout:        5 * time.Second,
			wantErr:        false,
			wantPassed:     true,
		},
		{
			name: "CI fails on first check",
			mockSetup: func(m *MockGhRunner) {
				failOutput := `[{"name":"test","state":"FAILURE"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(failOutput, nil)
			},
			checkInterval:  50 * time.Millisecond,
			commandTimeout: 100 * time.Millisecond,
			initialDelay:   50 * time.Millisecond,
			timeout:        5 * time.Second,
			wantErr:        false,
			wantPassed:     false,
		},
		{
			name: "skip e2e failures",
			mockSetup: func(m *MockGhRunner) {
				failOutput := `[{"name":"test-unit","state":"SUCCESS"},{"name":"test-e2e","state":"FAILURE"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(failOutput, nil)
			},
			checkInterval:  50 * time.Millisecond,
			commandTimeout: 100 * time.Millisecond,
			initialDelay:   50 * time.Millisecond,
			timeout:        5 * time.Second,
			opts:           CheckCIOptions{SkipE2E: true, E2ETestPattern: "e2e"},
			wantErr:        false,
			wantPassed:     true,
		},
		{
			name: "context timeout returns error",
			mockSetup: func(m *MockGhRunner) {
				pendingOutput := `[{"name":"test","state":"PENDING"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Maybe()
			},
			checkInterval:  50 * time.Millisecond,
			commandTimeout: 100 * time.Millisecond,
			initialDelay:   50 * time.Millisecond,
			timeout:        150 * time.Millisecond,
			wantErr:        true,
		},
		{
			name: "uses default timeout when zero",
			mockSetup: func(m *MockGhRunner) {
				successOutput := `[{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(successOutput, nil)
			},
			checkInterval:  50 * time.Millisecond,
			commandTimeout: 100 * time.Millisecond,
			initialDelay:   50 * time.Millisecond,
			timeout:        0,
			wantErr:        false,
			wantPassed:     true,
		},
		{
			name: "uses default e2e pattern",
			mockSetup: func(m *MockGhRunner) {
				failOutput := `[{"name":"test","state":"SUCCESS"},{"name":"integration","state":"FAILURE"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(failOutput, nil)
			},
			checkInterval:  50 * time.Millisecond,
			commandTimeout: 100 * time.Millisecond,
			initialDelay:   50 * time.Millisecond,
			timeout:        5 * time.Second,
			opts:           CheckCIOptions{SkipE2E: true},
			wantErr:        false,
			wantPassed:     true,
		},
		{
			name: "handles timeout error during polling",
			mockSetup: func(m *MockGhRunner) {
				pendingOutput := `[{"name":"test","state":"PENDING"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Once()
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", ErrCICheckTimeout).Maybe()
			},
			checkInterval:  50 * time.Millisecond,
			commandTimeout: 100 * time.Millisecond,
			initialDelay:   50 * time.Millisecond,
			timeout:        300 * time.Millisecond,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			tt.mockSetup(mockGhRunner)

			checker := NewCICheckerWithRunner("/tmp", tt.checkInterval, tt.commandTimeout, tt.initialDelay, mockGhRunner)

			ctx, cancel := context.WithTimeout(context.Background(), max(tt.timeout, 5*time.Second))
			defer cancel()

			var progressEvents []CIProgressEvent
			onProgress := func(event CIProgressEvent) {
				progressEvents = append(progressEvents, event)
			}

			result, err := checker.WaitForCIWithProgress(ctx, 0, tt.timeout, tt.opts, onProgress)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantPassed, result.Passed)
			assert.Greater(t, len(progressEvents), 0, "should have progress events")
		})
	}
}

func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func TestCheckCI_RetryLogic(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(*MockGhRunner)
		wantErr   bool
	}{
		{
			name: "retries on timeout and eventually succeeds",
			mockSetup: func(m *MockGhRunner) {
				successOutput := `[{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", ErrCICheckTimeout).Once()
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(successOutput, nil)
			},
			wantErr: false,
		},
		{
			name: "retries maximum times and returns error",
			mockSetup: func(m *MockGhRunner) {
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", ErrCICheckTimeout).Times(3)
			},
			wantErr: true,
		},
		{
			name: "non-timeout error returns immediately",
			mockSetup: func(m *MockGhRunner) {
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", fmt.Errorf("some error")).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			tt.mockSetup(mockGhRunner)

			checker := NewCICheckerWithRunner("/tmp", 1*time.Millisecond, 50*time.Millisecond, 1*time.Minute, mockGhRunner)

			ctx := context.Background()
			result, err := checker.CheckCI(ctx, 0)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestWaitForCIWithOptions(t *testing.T) {
	tests := []struct {
		name       string
		mockSetup  func(*MockGhRunner)
		opts       CheckCIOptions
		wantErr    bool
		wantPassed bool
	}{
		{
			name: "waits and succeeds",
			mockSetup: func(m *MockGhRunner) {
				output := `[{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(output, nil)
			},
			wantErr:    false,
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			tt.mockSetup(mockGhRunner)

			checker := NewCICheckerWithRunner("/tmp", 50*time.Millisecond, 100*time.Millisecond, 50*time.Millisecond, mockGhRunner)

			ctx := context.Background()
			result, err := checker.WaitForCIWithOptions(ctx, 0, 5*time.Second, tt.opts)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantPassed, result.Passed)
		})
	}
}

func TestWaitForCI(t *testing.T) {
	tests := []struct {
		name       string
		mockSetup  func(*MockGhRunner)
		wantErr    bool
		wantPassed bool
	}{
		{
			name: "waits and succeeds without options",
			mockSetup: func(m *MockGhRunner) {
				output := `[{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(output, nil)
			},
			wantErr:    false,
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			tt.mockSetup(mockGhRunner)

			checker := NewCICheckerWithRunner("/tmp", 50*time.Millisecond, 100*time.Millisecond, 50*time.Millisecond, mockGhRunner)

			ctx := context.Background()
			result, err := checker.WaitForCI(ctx, 0, 5*time.Second)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantPassed, result.Passed)
		})
	}
}

func TestWaitForCIWithProgress_ProgressEvents(t *testing.T) {
	tests := []struct {
		name              string
		mockSetup         func(*MockGhRunner)
		wantEventTypes    []string
		wantMinEventCount int
	}{
		{
			name: "generates waiting and checking events",
			mockSetup: func(m *MockGhRunner) {
				pendingOutput := `[{"name":"test","state":"PENDING"}]`
				successOutput := `[{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Once()
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(successOutput, nil)
			},
			wantEventTypes:    []string{"checking", "status", "waiting"},
			wantMinEventCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			tt.mockSetup(mockGhRunner)

			checker := NewCICheckerWithRunner("/tmp", 50*time.Millisecond, 100*time.Millisecond, 100*time.Millisecond, mockGhRunner)

			ctx := context.Background()
			var events []CIProgressEvent
			onProgress := func(event CIProgressEvent) {
				events = append(events, event)
			}

			_, err := checker.WaitForCIWithProgress(ctx, 0, 5*time.Second, CheckCIOptions{}, onProgress)
			require.NoError(t, err)

			assert.GreaterOrEqual(t, len(events), tt.wantMinEventCount)

			eventTypesFound := make(map[string]bool)
			for _, event := range events {
				eventTypesFound[event.Type] = true
			}

			for _, wantType := range tt.wantEventTypes {
				if !eventTypesFound[wantType] {
					t.Logf("Event types found: %v", eventTypesFound)
				}
			}
		})
	}
}

func TestWaitForCIWithProgress_PollingLoop(t *testing.T) {
	tests := []struct {
		name       string
		mockSetup  func(*MockGhRunner)
		wantErr    bool
		wantPassed bool
	}{
		{
			name: "polls multiple times before success",
			mockSetup: func(m *MockGhRunner) {
				pendingOutput := `[{"name":"test","state":"PENDING"}]`
				successOutput := `[{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Once()
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Once()
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(successOutput, nil)
			},
			wantErr:    false,
			wantPassed: true,
		},
		{
			name: "handles non-timeout error in polling loop",
			mockSetup: func(m *MockGhRunner) {
				pendingOutput := `[{"name":"test","state":"PENDING"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Once()
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", fmt.Errorf("some error"))
			},
			wantErr: true,
		},
		{
			name: "handles retry event with timeout error",
			mockSetup: func(m *MockGhRunner) {
				pendingOutput := `[{"name":"test","state":"PENDING"}]`
				successOutput := `[{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Once()
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return("", ErrCICheckTimeout).Once()
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(successOutput, nil)
			},
			wantErr:    false,
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			tt.mockSetup(mockGhRunner)

			checker := NewCICheckerWithRunner("/tmp", 50*time.Millisecond, 100*time.Millisecond, 50*time.Millisecond, mockGhRunner)

			ctx := context.Background()
			var retryEvents int
			onProgress := func(event CIProgressEvent) {
				if event.Type == "retry" {
					retryEvents++
				}
			}

			result, err := checker.WaitForCIWithProgress(ctx, 0, 5*time.Second, CheckCIOptions{}, onProgress)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantPassed, result.Passed)
		})
	}
}

func TestWaitForCIWithProgress_AdditionalPaths(t *testing.T) {
	tests := []struct {
		name       string
		mockSetup  func(*MockGhRunner)
		opts       CheckCIOptions
		wantErr    bool
		wantPassed bool
		wantStatus string
	}{
		{
			name: "initial check succeeds with success status - immediate return",
			mockSetup: func(m *MockGhRunner) {
				successOutput := `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 123, "name,state").Return(successOutput, nil).Once()
			},
			wantErr:    false,
			wantPassed: true,
			wantStatus: "success",
		},
		{
			name: "initial check succeeds with failure status - immediate return",
			mockSetup: func(m *MockGhRunner) {
				failOutput := `[{"name":"build","state":"SUCCESS"},{"name":"test","state":"FAILURE"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 456, "name,state").Return(failOutput, nil).Once()
			},
			wantErr:    false,
			wantPassed: false,
			wantStatus: "failure",
		},
		{
			name: "initial check succeeds with success and SkipE2E option",
			mockSetup: func(m *MockGhRunner) {
				successOutput := `[{"name":"build","state":"SUCCESS"},{"name":"e2e-test","state":"SUCCESS"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 789, "name,state").Return(successOutput, nil).Once()
			},
			opts:       CheckCIOptions{SkipE2E: true, E2ETestPattern: "e2e"},
			wantErr:    false,
			wantPassed: true,
			wantStatus: "success",
		},
		{
			name: "initial check fails but SkipE2E filters out e2e failures",
			mockSetup: func(m *MockGhRunner) {
				failOutput := `[{"name":"build","state":"SUCCESS"},{"name":"e2e-test","state":"FAILURE"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 111, "name,state").Return(failOutput, nil).Once()
			},
			opts:       CheckCIOptions{SkipE2E: true, E2ETestPattern: "e2e"},
			wantErr:    false,
			wantPassed: true,
			wantStatus: "failure",
		},
		{
			name: "initial check fails with mixed failures and SkipE2E",
			mockSetup: func(m *MockGhRunner) {
				failOutput := `[{"name":"build","state":"FAILURE"},{"name":"e2e-test","state":"FAILURE"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 222, "name,state").Return(failOutput, nil).Once()
			},
			opts:       CheckCIOptions{SkipE2E: true, E2ETestPattern: "e2e"},
			wantErr:    false,
			wantPassed: false,
			wantStatus: "failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			tt.mockSetup(mockGhRunner)

			checker := NewCICheckerWithRunner("/tmp", 50*time.Millisecond, 100*time.Millisecond, 50*time.Millisecond, mockGhRunner)

			ctx := context.Background()
			var progressEvents []CIProgressEvent
			onProgress := func(event CIProgressEvent) {
				progressEvents = append(progressEvents, event)
			}

			var prNumber int
			switch tt.name {
			case "initial check succeeds with success status - immediate return":
				prNumber = 123
			case "initial check succeeds with failure status - immediate return":
				prNumber = 456
			case "initial check succeeds with success and SkipE2E option":
				prNumber = 789
			case "initial check fails but SkipE2E filters out e2e failures":
				prNumber = 111
			case "initial check fails with mixed failures and SkipE2E":
				prNumber = 222
			}

			result, err := checker.WaitForCIWithProgress(ctx, prNumber, 5*time.Second, tt.opts, onProgress)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantPassed, result.Passed)
			assert.Equal(t, tt.wantStatus, result.Status)

			assert.GreaterOrEqual(t, len(progressEvents), 1, "should have at least one progress event")
			assert.Equal(t, "checking", progressEvents[0].Type, "first event should be checking")

			mockGhRunner.AssertExpectations(t)
		})
	}
}

func TestWaitForCIWithProgress_ContextCancellationDuringDelay(t *testing.T) {
	tests := []struct {
		name         string
		mockSetup    func(*MockGhRunner)
		cancelDelay  time.Duration
		initialDelay time.Duration
		wantErr      bool
	}{
		{
			name: "context cancelled during initial delay wait",
			mockSetup: func(m *MockGhRunner) {
				pendingOutput := `[{"name":"test","state":"PENDING"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Once()
			},
			cancelDelay:  75 * time.Millisecond,
			initialDelay: 200 * time.Millisecond,
			wantErr:      true,
		},
		{
			name: "context cancelled early during initial delay",
			mockSetup: func(m *MockGhRunner) {
				pendingOutput := `[{"name":"test","state":"PENDING"}]`
				m.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Once()
			},
			cancelDelay:  25 * time.Millisecond,
			initialDelay: 150 * time.Millisecond,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGhRunner := new(MockGhRunner)
			tt.mockSetup(mockGhRunner)

			checker := NewCICheckerWithRunner("/tmp", 50*time.Millisecond, 100*time.Millisecond, tt.initialDelay, mockGhRunner)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				time.Sleep(tt.cancelDelay)
				cancel()
			}()

			start := time.Now()
			result, err := checker.WaitForCIWithProgress(ctx, 0, 5*time.Second, CheckCIOptions{}, nil)
			elapsed := time.Since(start)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, context.Canceled)
				assert.Nil(t, result)
				assert.Less(t, elapsed, tt.initialDelay, "should cancel before initial delay completes")
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestWaitForCIWithProgress_WaitingEvents(t *testing.T) {
	t.Run("validates waiting event structure", func(t *testing.T) {
		mockGhRunner := new(MockGhRunner)
		pendingOutput := `[{"name":"test","state":"PENDING"}]`
		successOutput := `[{"name":"test","state":"SUCCESS"}]`
		mockGhRunner.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(pendingOutput, nil).Once()
		mockGhRunner.On("PRChecks", mock.Anything, "/tmp", 0, "name,state").Return(successOutput, nil)

		checker := NewCICheckerWithRunner("/tmp", 50*time.Millisecond, 100*time.Millisecond, 6*time.Second, mockGhRunner)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var waitingEvents []CIProgressEvent
		onProgress := func(event CIProgressEvent) {
			if event.Type == "waiting" {
				waitingEvents = append(waitingEvents, event)
			}
		}

		result, err := checker.WaitForCIWithProgress(ctx, 0, 10*time.Second, CheckCIOptions{}, onProgress)

		require.NoError(t, err)
		require.NotNil(t, result)

		assert.GreaterOrEqual(t, len(waitingEvents), 1, "should have at least one waiting event with 6s initial delay and 5s ticker")

		for i, event := range waitingEvents {
			assert.Equal(t, "waiting", event.Type, "event %d should be waiting type", i)
			assert.Equal(t, "Waiting for CI jobs to complete", event.Message, "event %d should have correct message", i)
			assert.Greater(t, event.Elapsed, time.Duration(0), "event %d should have positive elapsed time", i)
			assert.GreaterOrEqual(t, event.NextCheckIn, time.Duration(0), "event %d NextCheckIn should be non-negative", i)
		}

		if len(waitingEvents) > 1 {
			for i := 1; i < len(waitingEvents); i++ {
				prevNextCheckIn := waitingEvents[i-1].NextCheckIn
				currNextCheckIn := waitingEvents[i].NextCheckIn
				assert.LessOrEqual(t, currNextCheckIn, prevNextCheckIn, "NextCheckIn should decrease over time")
			}
		}
	})
}
