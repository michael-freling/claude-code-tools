package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCIFailureClassifier_ClassifyResult(t *testing.T) {
	tests := []struct {
		name           string
		ciResult       *CIResult
		history        *CIFailureHistory
		wantCategory   CIFailureCategory
		wantNumReasons int
	}{
		{
			name: "failed jobs are code related",
			ciResult: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"build", "test"},
				Output:     `[]`,
			},
			history:        nil,
			wantCategory:   CategoryCodeRelated,
			wantNumReasons: 2,
		},
		{
			name: "cancelled job with short duration is infrastructure",
			ciResult: &CIResult{
				Passed:        false,
				Status:        "failure",
				CancelledJobs: []string{"lint"},
				Output:        `[{"name":"lint","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:00:10Z"}]`,
			},
			history:        nil,
			wantCategory:   CategoryInfrastructure,
			wantNumReasons: 1,
		},
		{
			name: "cancelled job with long duration is code related",
			ciResult: &CIResult{
				Passed:        false,
				Status:        "failure",
				CancelledJobs: []string{"test"},
				Output:        `[{"name":"test","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:10:00Z"}]`,
			},
			history:        nil,
			wantCategory:   CategoryCodeRelated,
			wantNumReasons: 1,
		},
		{
			name: "mixed failures with failed and cancelled jobs",
			ciResult: &CIResult{
				Passed:        false,
				Status:        "failure",
				FailedJobs:    []string{"build"},
				CancelledJobs: []string{"lint"},
				Output:        `[{"name":"lint","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:00:10Z"}]`,
			},
			history:        nil,
			wantCategory:   CategoryMixed,
			wantNumReasons: 2,
		},
		{
			name: "persistent failure detected",
			ciResult: &CIResult{
				Passed:     false,
				Status:     "failure",
				FailedJobs: []string{"test"},
				Output:     `[]`,
			},
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"test"}, Attempt: 1},
					{FailedJobs: []string{"test"}, Attempt: 2},
					{FailedJobs: []string{"test"}, Attempt: 3},
				},
			},
			wantCategory:   CategoryPersistent,
			wantNumReasons: 1,
		},
		{
			name: "cancelled job with no timing data defaults to infrastructure",
			ciResult: &CIResult{
				Passed:        false,
				Status:        "failure",
				CancelledJobs: []string{"deploy"},
				Output:        `[]`,
			},
			history:        nil,
			wantCategory:   CategoryInfrastructure,
			wantNumReasons: 1,
		},
		{
			name: "multiple cancelled jobs with different durations",
			ciResult: &CIResult{
				Passed:        false,
				Status:        "failure",
				CancelledJobs: []string{"lint", "test"},
				Output:        `[{"name":"lint","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:00:10Z"},{"name":"test","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:10:00Z"}]`,
			},
			history:        nil,
			wantCategory:   CategoryMixed,
			wantNumReasons: 2,
		},
		{
			name: "only infrastructure cancelled jobs",
			ciResult: &CIResult{
				Passed:        false,
				Status:        "failure",
				CancelledJobs: []string{"deploy", "notify"},
				Output:        `[{"name":"deploy","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:00:05Z"},{"name":"notify","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:00:08Z"}]`,
			},
			history:        nil,
			wantCategory:   CategoryInfrastructure,
			wantNumReasons: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCIFailureClassifier()
			result := c.ClassifyResult(tt.ciResult, tt.history)

			assert.Equal(t, tt.wantCategory, result.Category, "category mismatch")
			assert.Len(t, result.Reasons, tt.wantNumReasons, "number of reasons mismatch")
			assert.NotEmpty(t, result.RecommendedAction, "recommended action should not be empty")
		})
	}
}

func TestCIFailureClassifier_classifyCancelledJob(t *testing.T) {
	tests := []struct {
		name         string
		jobName      string
		detail       *CIJobDetail
		wantCategory CIFailureCategory
	}{
		{
			name:         "no timing data defaults to infrastructure",
			jobName:      "lint",
			detail:       nil,
			wantCategory: CategoryInfrastructure,
		},
		{
			name:    "short duration is infrastructure",
			jobName: "lint",
			detail: &CIJobDetail{
				Name:     "lint",
				Duration: 10 * time.Second,
			},
			wantCategory: CategoryInfrastructure,
		},
		{
			name:    "long duration is code related",
			jobName: "test",
			detail: &CIJobDetail{
				Name:     "test",
				Duration: 6 * time.Minute,
			},
			wantCategory: CategoryCodeRelated,
		},
		{
			name:    "medium duration test job is code related",
			jobName: "test-unit",
			detail: &CIJobDetail{
				Name:     "test-unit",
				Duration: 2 * time.Minute,
			},
			wantCategory: CategoryCodeRelated,
		},
		{
			name:    "medium duration non-test job is infrastructure",
			jobName: "deploy",
			detail: &CIJobDetail{
				Name:     "deploy",
				Duration: 2 * time.Minute,
			},
			wantCategory: CategoryInfrastructure,
		},
		{
			name:    "exactly at infrastructure threshold",
			jobName: "verify",
			detail: &CIJobDetail{
				Name:     "verify",
				Duration: 30 * time.Second,
			},
			wantCategory: CategoryInfrastructure,
		},
		{
			name:    "just over infrastructure threshold with build keyword",
			jobName: "build",
			detail: &CIJobDetail{
				Name:     "build",
				Duration: 31 * time.Second,
			},
			wantCategory: CategoryCodeRelated,
		},
		{
			name:    "medium duration e2e test is code related",
			jobName: "e2e-tests",
			detail: &CIJobDetail{
				Name:     "e2e-tests",
				Duration: 3 * time.Minute,
			},
			wantCategory: CategoryCodeRelated,
		},
		{
			name:    "medium duration integration test is code related",
			jobName: "integration-test",
			detail: &CIJobDetail{
				Name:     "integration-test",
				Duration: 90 * time.Second,
			},
			wantCategory: CategoryCodeRelated,
		},
		{
			name:    "medium duration check job is code related",
			jobName: "check-types",
			detail: &CIJobDetail{
				Name:     "check-types",
				Duration: 45 * time.Second,
			},
			wantCategory: CategoryCodeRelated,
		},
		{
			name:    "exactly at timeout threshold is code related",
			jobName: "test",
			detail: &CIJobDetail{
				Name:     "test",
				Duration: 5 * time.Minute,
			},
			wantCategory: CategoryCodeRelated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCIFailureClassifier()
			reason := c.classifyCancelledJob(tt.jobName, tt.detail)

			assert.Equal(t, tt.wantCategory, reason.Category, "category mismatch")
			assert.Equal(t, tt.jobName, reason.Job, "job name should match")
			assert.Equal(t, "CANCELLED", reason.Conclusion, "conclusion should be CANCELLED")
			assert.NotEmpty(t, reason.Explanation, "explanation should not be empty")
			if tt.detail != nil {
				assert.Equal(t, tt.detail.Duration, reason.Duration, "duration should match")
			}
		})
	}
}

func TestCIFailureHistory_IsPersistentFailure(t *testing.T) {
	tests := []struct {
		name      string
		history   *CIFailureHistory
		threshold int
		want      bool
	}{
		{
			name:      "empty history is not persistent",
			history:   &CIFailureHistory{},
			threshold: 3,
			want:      false,
		},
		{
			name: "less than threshold is not persistent",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"test"}},
					{FailedJobs: []string{"test"}},
				},
			},
			threshold: 3,
			want:      false,
		},
		{
			name: "threshold identical failures is persistent",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"test"}},
					{FailedJobs: []string{"test"}},
					{FailedJobs: []string{"test"}},
				},
			},
			threshold: 3,
			want:      true,
		},
		{
			name: "different jobs each time is not persistent",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"test"}},
					{FailedJobs: []string{"build"}},
					{FailedJobs: []string{"lint"}},
				},
			},
			threshold: 3,
			want:      false,
		},
		{
			name: "cancelled jobs also count for persistent",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{CancelledJobs: []string{"test"}},
					{CancelledJobs: []string{"test"}},
					{CancelledJobs: []string{"test"}},
				},
			},
			threshold: 3,
			want:      true,
		},
		{
			name: "mixed failed and cancelled is persistent if same",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"build"}, CancelledJobs: []string{"test"}},
					{FailedJobs: []string{"build"}, CancelledJobs: []string{"test"}},
					{FailedJobs: []string{"build"}, CancelledJobs: []string{"test"}},
				},
			},
			threshold: 3,
			want:      true,
		},
		{
			name: "multiple failed jobs in same order is persistent",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"build", "test"}},
					{FailedJobs: []string{"build", "test"}},
					{FailedJobs: []string{"build", "test"}},
				},
			},
			threshold: 3,
			want:      true,
		},
		{
			name: "multiple failed jobs in different order is still persistent",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"build", "test"}},
					{FailedJobs: []string{"test", "build"}},
					{FailedJobs: []string{"build", "test"}},
				},
			},
			threshold: 3,
			want:      true,
		},
		{
			name: "different number of jobs is not persistent",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"build", "test"}},
					{FailedJobs: []string{"build"}},
					{FailedJobs: []string{"build", "test"}},
				},
			},
			threshold: 3,
			want:      false,
		},
		{
			name: "more entries than threshold with persistent pattern",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"lint"}},
					{FailedJobs: []string{"test"}},
					{FailedJobs: []string{"test"}},
					{FailedJobs: []string{"test"}},
				},
			},
			threshold: 3,
			want:      true,
		},
		{
			name: "threshold 2 with two identical failures",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"build"}},
					{FailedJobs: []string{"build"}},
				},
			},
			threshold: 2,
			want:      true,
		},
		{
			name: "threshold 1 always returns true",
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"test"}},
				},
			},
			threshold: 1,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.history.IsPersistentFailure(tt.threshold)
			assert.Equal(t, tt.want, got, "IsPersistentFailure(%d) result mismatch", tt.threshold)
		})
	}
}

func TestCIFailureHistory_AddEntry(t *testing.T) {
	tests := []struct {
		name        string
		initial     *CIFailureHistory
		entry       CIFailureHistoryEntry
		wantLen     int
		wantLastIdx int
	}{
		{
			name:    "add to empty history",
			initial: &CIFailureHistory{},
			entry: CIFailureHistoryEntry{
				Timestamp:  time.Now(),
				Category:   CategoryCodeRelated,
				FailedJobs: []string{"test"},
				Attempt:    1,
			},
			wantLen:     1,
			wantLastIdx: 0,
		},
		{
			name: "add to existing history",
			initial: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"build"}},
				},
			},
			entry: CIFailureHistoryEntry{
				Timestamp:  time.Now(),
				Category:   CategoryInfrastructure,
				FailedJobs: []string{"test"},
				Attempt:    2,
			},
			wantLen:     2,
			wantLastIdx: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.AddEntry(tt.entry)

			assert.Len(t, tt.initial.Entries, tt.wantLen, "entry count mismatch")
			require.GreaterOrEqual(t, len(tt.initial.Entries), tt.wantLastIdx+1, "entry index out of range")

			lastEntry := tt.initial.Entries[tt.wantLastIdx]
			assert.Equal(t, tt.entry.Category, lastEntry.Category, "category mismatch")
			assert.Equal(t, tt.entry.FailedJobs, lastEntry.FailedJobs, "failed jobs mismatch")
			assert.Equal(t, tt.entry.CancelledJobs, lastEntry.CancelledJobs, "cancelled jobs mismatch")
			assert.Equal(t, tt.entry.Attempt, lastEntry.Attempt, "attempt number mismatch")
		})
	}
}

func TestCIFailureClassifier_parseJobDetails(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantLen   int
		wantFirst *CIJobDetail
	}{
		{
			name:    "valid JSON with all fields",
			output:  `[{"name":"test","conclusion":"FAILURE","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:05:00Z"}]`,
			wantLen: 1,
			wantFirst: &CIJobDetail{
				Name:       "test",
				Conclusion: "FAILURE",
				Status:     "completed",
				Duration:   5 * time.Minute,
			},
		},
		{
			name:    "valid JSON with missing optional fields",
			output:  `[{"name":"lint","conclusion":"SUCCESS","status":"completed"}]`,
			wantLen: 1,
			wantFirst: &CIJobDetail{
				Name:       "lint",
				Conclusion: "SUCCESS",
				Status:     "completed",
			},
		},
		{
			name:      "invalid JSON returns empty",
			output:    `not json`,
			wantLen:   0,
			wantFirst: nil,
		},
		{
			name:      "empty string returns empty",
			output:    "",
			wantLen:   0,
			wantFirst: nil,
		},
		{
			name:      "empty array returns empty",
			output:    `[]`,
			wantLen:   0,
			wantFirst: nil,
		},
		{
			name:    "multiple jobs",
			output:  `[{"name":"build","conclusion":"SUCCESS","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:02:00Z"},{"name":"test","conclusion":"FAILURE","status":"completed","startedAt":"2024-01-01T10:02:00Z","completedAt":"2024-01-01T10:07:00Z"}]`,
			wantLen: 2,
			wantFirst: &CIJobDetail{
				Name:       "build",
				Conclusion: "SUCCESS",
				Status:     "completed",
				Duration:   2 * time.Minute,
			},
		},
		{
			name:    "cancelled job",
			output:  `[{"name":"deploy","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:00:10Z"}]`,
			wantLen: 1,
			wantFirst: &CIJobDetail{
				Name:       "deploy",
				Conclusion: "CANCELLED",
				Status:     "completed",
				Duration:   10 * time.Second,
			},
		},
		{
			name:    "job with only startedAt",
			output:  `[{"name":"test","conclusion":"IN_PROGRESS","status":"in_progress","startedAt":"2024-01-01T10:00:00Z"}]`,
			wantLen: 1,
			wantFirst: &CIJobDetail{
				Name:       "test",
				Conclusion: "IN_PROGRESS",
				Status:     "in_progress",
				Duration:   0,
			},
		},
		{
			name:    "job with invalid timestamp format",
			output:  `[{"name":"test","conclusion":"SUCCESS","status":"completed","startedAt":"invalid","completedAt":"2024-01-01T10:00:00Z"}]`,
			wantLen: 1,
			wantFirst: &CIJobDetail{
				Name:       "test",
				Conclusion: "SUCCESS",
				Status:     "completed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCIFailureClassifier()
			details := c.parseJobDetails(tt.output)

			assert.Len(t, details, tt.wantLen, "number of details mismatch")

			if tt.wantFirst != nil && len(details) > 0 {
				assert.Equal(t, tt.wantFirst.Name, details[0].Name, "first job name mismatch")
				assert.Equal(t, tt.wantFirst.Conclusion, details[0].Conclusion, "first job conclusion mismatch")
				assert.Equal(t, tt.wantFirst.Status, details[0].Status, "first job status mismatch")
				assert.Equal(t, tt.wantFirst.Duration, details[0].Duration, "first job duration mismatch")
			}
		})
	}
}

func TestCIFailureClassifier_getRecommendedAction(t *testing.T) {
	tests := []struct {
		category CIFailureCategory
		contains string
	}{
		{CategoryInfrastructure, "Auto-retry"},
		{CategoryCodeRelated, "fix code"},
		{CategoryMixed, "Fix code"},
		{CategoryPersistent, "Stop retrying"},
	}

	c := NewCIFailureClassifier()
	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			action := c.getRecommendedAction(tt.category)
			assert.NotEmpty(t, action, "recommended action should not be empty")
			assert.Contains(t, action, tt.contains, "recommended action should contain expected text")
		})
	}
}

func TestCIFailureClassifier_findJobDetail(t *testing.T) {
	details := []CIJobDetail{
		{Name: "build", Conclusion: "SUCCESS"},
		{Name: "test", Conclusion: "FAILURE"},
		{Name: "lint", Conclusion: "SUCCESS"},
	}

	tests := []struct {
		name    string
		jobName string
		want    *CIJobDetail
	}{
		{
			name:    "find existing job",
			jobName: "test",
			want:    &CIJobDetail{Name: "test", Conclusion: "FAILURE"},
		},
		{
			name:    "job not found returns nil",
			jobName: "deploy",
			want:    nil,
		},
		{
			name:    "find first job",
			jobName: "build",
			want:    &CIJobDetail{Name: "build", Conclusion: "SUCCESS"},
		},
		{
			name:    "find last job",
			jobName: "lint",
			want:    &CIJobDetail{Name: "lint", Conclusion: "SUCCESS"},
		},
	}

	c := NewCIFailureClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.findJobDetail(details, tt.jobName)
			if tt.want == nil {
				assert.Nil(t, got, "should return nil for non-existent job")
			} else {
				require.NotNil(t, got, "should find existing job")
				assert.Equal(t, tt.want.Name, got.Name, "job name mismatch")
				assert.Equal(t, tt.want.Conclusion, got.Conclusion, "job conclusion mismatch")
			}
		})
	}
}

func TestCIFailureClassifier_jobNameSuggestsTimeout(t *testing.T) {
	tests := []struct {
		jobName string
		want    bool
	}{
		{"test", true},
		{"test-unit", true},
		{"integration-test", true},
		{"build", true},
		{"build-linux", true},
		{"lint", true},
		{"lint-go", true},
		{"check", true},
		{"check-types", true},
		{"e2e", true},
		{"e2e-tests", true},
		{"integration", true},
		{"deploy", false},
		{"notify", false},
		{"release", false},
		{"publish", false},
		{"TEST", true},
		{"Build", true},
		{"LINT", true},
	}

	c := NewCIFailureClassifier()
	for _, tt := range tests {
		t.Run(tt.jobName, func(t *testing.T) {
			got := c.jobNameSuggestsTimeout(tt.jobName)
			assert.Equal(t, tt.want, got, "jobNameSuggestsTimeout(%q) result mismatch", tt.jobName)
		})
	}
}

func TestCIFailureClassifier_determineOverallCategory(t *testing.T) {
	tests := []struct {
		name       string
		infraCount int
		codeCount  int
		history    *CIFailureHistory
		want       CIFailureCategory
	}{
		{
			name:       "no failures defaults to code related",
			infraCount: 0,
			codeCount:  0,
			history:    nil,
			want:       CategoryCodeRelated,
		},
		{
			name:       "only infrastructure failures",
			infraCount: 2,
			codeCount:  0,
			history:    nil,
			want:       CategoryInfrastructure,
		},
		{
			name:       "only code failures",
			infraCount: 0,
			codeCount:  3,
			history:    nil,
			want:       CategoryCodeRelated,
		},
		{
			name:       "mixed failures",
			infraCount: 1,
			codeCount:  1,
			history:    nil,
			want:       CategoryMixed,
		},
		{
			name:       "persistent failure overrides everything",
			infraCount: 1,
			codeCount:  1,
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"test"}},
					{FailedJobs: []string{"test"}},
					{FailedJobs: []string{"test"}},
				},
			},
			want: CategoryPersistent,
		},
		{
			name:       "non-persistent history does not override",
			infraCount: 0,
			codeCount:  1,
			history: &CIFailureHistory{
				Entries: []CIFailureHistoryEntry{
					{FailedJobs: []string{"test"}},
					{FailedJobs: []string{"build"}},
				},
			},
			want: CategoryCodeRelated,
		},
	}

	c := NewCIFailureClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.determineOverallCategory(tt.infraCount, tt.codeCount, tt.history)
			assert.Equal(t, tt.want, got, "determineOverallCategory result mismatch")
		})
	}
}

func TestNewCIFailureClassifier(t *testing.T) {
	c := NewCIFailureClassifier()

	require.NotNil(t, c, "classifier should not be nil")
	assert.Equal(t, DefaultPersistentFailureThreshold, c.PersistentThreshold, "should use default threshold")
}

func TestClassifiedCIResult_Integration(t *testing.T) {
	c := NewCIFailureClassifier()

	ciResult := &CIResult{
		Passed:        false,
		Status:        "failure",
		FailedJobs:    []string{"build"},
		CancelledJobs: []string{"test", "lint"},
		Output:        `[{"name":"build","conclusion":"FAILURE","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:05:00Z"},{"name":"test","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:10:00Z"},{"name":"lint","conclusion":"CANCELLED","status":"completed","startedAt":"2024-01-01T10:00:00Z","completedAt":"2024-01-01T10:00:10Z"}]`,
	}

	result := c.ClassifyResult(ciResult, nil)

	require.NotNil(t, result, "result should not be nil")
	assert.Equal(t, CategoryMixed, result.Category, "should be mixed category")
	assert.Len(t, result.Reasons, 3, "should have 3 reasons")
	assert.NotEmpty(t, result.RecommendedAction, "should have recommended action")
	assert.Len(t, result.JobDetails, 3, "should have 3 job details")

	buildReason := result.Reasons[0]
	assert.Equal(t, "build", buildReason.Job)
	assert.Equal(t, CategoryCodeRelated, buildReason.Category)
	assert.Equal(t, 5*time.Minute, buildReason.Duration)

	testReason := result.Reasons[1]
	assert.Equal(t, "test", testReason.Job)
	assert.Equal(t, CategoryCodeRelated, testReason.Category)
	assert.Equal(t, 10*time.Minute, testReason.Duration)

	lintReason := result.Reasons[2]
	assert.Equal(t, "lint", lintReason.Job)
	assert.Equal(t, CategoryInfrastructure, lintReason.Category)
	assert.Equal(t, 10*time.Second, lintReason.Duration)
}
