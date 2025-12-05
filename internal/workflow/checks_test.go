package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePreCommitErrors(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{
			name:   "no errors",
			output: "",
			want:   nil,
		},
		{
			name: "single failed check",
			output: `check-yaml...........................................................Failed
- hook id: check-yaml
- exit code: 1`,
			want: []string{
				"check-yaml...........................................................Failed",
			},
		},
		{
			name: "multiple failed checks",
			output: `check-yaml...........................................................Failed
go fmt...............................................................Failed
✗ golangci-lint......................................................Failed`,
			want: []string{
				"check-yaml...........................................................Failed",
				"go fmt...............................................................Failed",
				"✗ golangci-lint......................................................Failed",
			},
		},
		{
			name: "mixed success and failure",
			output: `check-yaml...........................................................Passed
go fmt...............................................................Failed
check-merge-conflict.................................................Passed`,
			want: []string{
				"go fmt...............................................................Failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePreCommitErrors(tt.output)
			assert.Equal(t, tt.want, got)
		})
	}
}

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

func TestNewPreCommitChecker(t *testing.T) {
	tests := []struct {
		name       string
		workingDir string
	}{
		{
			name:       "with working directory",
			workingDir: "/tmp/test",
		},
		{
			name:       "without working directory",
			workingDir: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewPreCommitChecker(tt.workingDir)
			require.NotNil(t, checker)

			concreteChecker, ok := checker.(*preCommitChecker)
			require.True(t, ok)
			assert.Equal(t, tt.workingDir, concreteChecker.workingDir)
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

func TestPreCommitChecker_RunPreCommit_NotInstalled(t *testing.T) {
	checker := NewPreCommitChecker("/nonexistent/path/that/should/not/exist")
	ctx := context.Background()

	result, err := checker.RunPreCommit(ctx)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Passed)
}

func TestCIChecker_CheckCI_NotInstalled(t *testing.T) {
	checker := NewCIChecker("/nonexistent/path/that/should/not/exist", 1*time.Second)
	ctx := context.Background()

	result, err := checker.CheckCI(ctx, 123)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Passed)
}

func TestCIChecker_WaitForCI_Timeout(t *testing.T) {
	checker := NewCIChecker("/nonexistent/path/that/should/not/exist", 100*time.Millisecond)
	ctx := context.Background()

	result, err := checker.WaitForCI(ctx, 123, 200*time.Millisecond)
	require.Error(t, err)
	assert.Nil(t, result)
}
