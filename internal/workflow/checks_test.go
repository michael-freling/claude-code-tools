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
			name: "all checks passed",
			output: `✓ build
✓ test
✓ lint`,
			wantStatus:     "success",
			wantFailedJobs: []string{},
		},
		{
			name: "some checks failed",
			output: `✓ build
✗ test
✓ lint`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"test"},
		},
		{
			name: "checks pending",
			output: `✓ build
○ test
○ lint`,
			wantStatus:     "pending",
			wantFailedJobs: []string{},
		},
		{
			name: "mixed status with pending",
			output: `✓ build
✗ test
○ lint`,
			wantStatus:     "pending",
			wantFailedJobs: []string{"test"},
		},
		{
			name: "multiple failed jobs",
			output: `✓ build
✗ test-unit
✗ test-integration
✓ lint`,
			wantStatus:     "failure",
			wantFailedJobs: []string{"test-unit", "test-integration"},
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

func TestNewCIChecker(t *testing.T) {
	tests := []struct {
		name          string
		workingDir    string
		checkInterval time.Duration
		wantInterval  time.Duration
	}{
		{
			name:          "with custom interval",
			workingDir:    "/tmp/test",
			checkInterval: 10 * time.Second,
			wantInterval:  10 * time.Second,
		},
		{
			name:          "with default interval",
			workingDir:    "/tmp/test",
			checkInterval: 0,
			wantInterval:  30 * time.Second,
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
	// Test that pending status from gh pr checks (exit code 8) is handled correctly
	// Exit code 8 means "checks pending" - when there's output, parse it
	output := `Vercel Preview Comments	pass	0	https://vercel.com/github
Test go/aninexus-gateway / Lint go/aninexus-gateway	pending	0	https://github.com/example/actions/runs/123
Test go/aninexus-gateway / Test go/aninexus-gateway	pending	0	https://github.com/example/actions/runs/123
Vercel – nooxac-gateway	pass	0	https://vercel.com/example	Deployment has completed`

	status, failedJobs := parseCIOutput(output)
	assert.Equal(t, "pending", status)
	assert.Empty(t, failedJobs)
}

func TestCIChecker_WaitForCI_Timeout(t *testing.T) {
	checker := NewCIChecker("/nonexistent/path/that/should/not/exist", 100*time.Millisecond)
	ctx := context.Background()

	result, err := checker.WaitForCI(ctx, 123, 200*time.Millisecond)
	require.Error(t, err)
	assert.Nil(t, result)
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
