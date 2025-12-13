package workflow

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	InfrastructureDurationThreshold   = 30 * time.Second
	TimeoutDurationThreshold          = 5 * time.Minute
	DefaultPersistentFailureThreshold = 3
)

type CIFailureClassifier struct {
	PersistentThreshold int
}

func NewCIFailureClassifier() *CIFailureClassifier {
	return &CIFailureClassifier{
		PersistentThreshold: DefaultPersistentFailureThreshold,
	}
}

func (c *CIFailureClassifier) ClassifyResult(result *CIResult, history *CIFailureHistory) *ClassifiedCIResult {
	classified := &ClassifiedCIResult{
		CIResult: result,
		Reasons:  make([]CIFailureReason, 0),
	}

	jobDetails := c.parseJobDetails(result.Output)
	classified.JobDetails = jobDetails

	var infraCount, codeCount int

	for _, job := range result.FailedJobs {
		detail := c.findJobDetail(jobDetails, job)
		reason := CIFailureReason{
			Job:         job,
			Category:    CategoryCodeRelated,
			Conclusion:  "FAILURE",
			Explanation: "Job failed with errors - requires code fix",
		}
		if detail != nil {
			reason.Duration = detail.Duration
			reason.Conclusion = detail.Conclusion
		}
		classified.Reasons = append(classified.Reasons, reason)
		codeCount++
	}

	for _, job := range result.CancelledJobs {
		detail := c.findJobDetail(jobDetails, job)
		reason := c.classifyCancelledJob(job, detail)
		classified.Reasons = append(classified.Reasons, reason)

		if reason.Category == CategoryInfrastructure {
			infraCount++
		} else {
			codeCount++
		}
	}

	classified.Category = c.determineOverallCategory(infraCount, codeCount, history)
	classified.RecommendedAction = c.getRecommendedAction(classified.Category)

	return classified
}

func (c *CIFailureClassifier) classifyCancelledJob(jobName string, detail *CIJobDetail) CIFailureReason {
	reason := CIFailureReason{
		Job:        jobName,
		Conclusion: "CANCELLED",
	}

	if detail == nil {
		reason.Category = CategoryInfrastructure
		reason.Explanation = "Job was cancelled (no timing data available - assuming infrastructure issue)"
		return reason
	}

	reason.Duration = detail.Duration

	if detail.Duration > 0 && detail.Duration < InfrastructureDurationThreshold {
		reason.Category = CategoryInfrastructure
		reason.Explanation = "Job cancelled within 30 seconds of start - likely workflow superseded or runner issue"
		return reason
	}

	if detail.Duration >= TimeoutDurationThreshold {
		reason.Category = CategoryCodeRelated
		reason.Explanation = "Job ran for extended period before cancellation - likely timeout, infinite loop, or resource exhaustion"
		return reason
	}

	if c.jobNameSuggestsTimeout(jobName) {
		reason.Category = CategoryCodeRelated
		reason.Explanation = "Job name suggests test/build that may have timed out"
		return reason
	}

	reason.Category = CategoryInfrastructure
	reason.Explanation = "Job was cancelled - likely infrastructure issue (workflow concurrency, manual cancellation)"
	return reason
}

func (c *CIFailureClassifier) jobNameSuggestsTimeout(jobName string) bool {
	lower := strings.ToLower(jobName)
	timeoutKeywords := []string{"test", "build", "lint", "check", "e2e", "integration"}
	for _, keyword := range timeoutKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func (c *CIFailureClassifier) determineOverallCategory(infraCount, codeCount int, history *CIFailureHistory) CIFailureCategory {
	if history != nil && history.IsPersistentFailure(c.PersistentThreshold) {
		return CategoryPersistent
	}

	if infraCount > 0 && codeCount > 0 {
		return CategoryMixed
	}
	if codeCount > 0 {
		return CategoryCodeRelated
	}
	if infraCount > 0 {
		return CategoryInfrastructure
	}

	// Shouldn't happen, but default to code-related to be safe
	return CategoryCodeRelated
}

func (c *CIFailureClassifier) getRecommendedAction(category CIFailureCategory) string {
	switch category {
	case CategoryInfrastructure:
		return "Auto-retry CI - no code changes needed"
	case CategoryCodeRelated:
		return "Analyze failures and fix code issues"
	case CategoryMixed:
		return "Fix code issues first, then retry for infrastructure issues"
	case CategoryPersistent:
		return "Stop retrying - same failure pattern has occurred multiple times. Manual investigation required."
	default:
		return "Analyze failures and determine appropriate action"
	}
}

func (c *CIFailureClassifier) parseJobDetails(output string) []CIJobDetail {
	if output == "" {
		return nil
	}

	var checks []struct {
		Name        string `json:"name"`
		Conclusion  string `json:"conclusion"`
		Status      string `json:"status"`
		StartedAt   string `json:"startedAt"`
		CompletedAt string `json:"completedAt"`
	}

	if err := json.Unmarshal([]byte(output), &checks); err != nil {
		return nil
	}

	details := make([]CIJobDetail, 0, len(checks))
	for _, check := range checks {
		detail := CIJobDetail{
			Name:        check.Name,
			Conclusion:  check.Conclusion,
			Status:      check.Status,
			StartedAt:   parseTime(check.StartedAt),
			CompletedAt: parseTime(check.CompletedAt),
		}

		if detail.StartedAt != nil && detail.CompletedAt != nil {
			detail.Duration = detail.CompletedAt.Sub(*detail.StartedAt)
		}

		details = append(details, detail)
	}

	return details
}

func parseTime(timeStr string) *time.Time {
	if timeStr == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil
	}

	return &t
}

func (c *CIFailureClassifier) findJobDetail(details []CIJobDetail, jobName string) *CIJobDetail {
	for i := range details {
		if details[i].Name == jobName {
			return &details[i]
		}
	}
	return nil
}
