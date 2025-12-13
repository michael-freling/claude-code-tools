package workflow

import (
	"encoding/json"
	"strings"
	"time"
)

// Classification thresholds
const (
	// InfrastructureDurationThreshold - jobs cancelled within this duration are likely infrastructure issues
	InfrastructureDurationThreshold = 30 * time.Second

	// TimeoutDurationThreshold - jobs running longer than this before cancellation likely hit timeouts
	TimeoutDurationThreshold = 5 * time.Minute

	// DefaultPersistentFailureThreshold - number of identical failures before marking as persistent
	DefaultPersistentFailureThreshold = 3
)

// CIFailureClassifier classifies CI failures
type CIFailureClassifier struct {
	PersistentThreshold int
}

// NewCIFailureClassifier creates a new classifier with default settings
func NewCIFailureClassifier() *CIFailureClassifier {
	return &CIFailureClassifier{
		PersistentThreshold: DefaultPersistentFailureThreshold,
	}
}

// ClassifyResult analyzes a CIResult and returns a ClassifiedCIResult
func (c *CIFailureClassifier) ClassifyResult(result *CIResult, history *CIFailureHistory) *ClassifiedCIResult {
	classified := &ClassifiedCIResult{
		CIResult: result,
		Reasons:  make([]CIFailureReason, 0),
	}

	// Parse job details from output
	jobDetails := c.parseJobDetails(result.Output)
	classified.JobDetails = jobDetails

	// Classify each job
	var infraCount, codeCount int

	// Classify failed jobs (always code-related)
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

	// Classify cancelled jobs (need analysis)
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

	// Determine overall category
	classified.Category = c.determineOverallCategory(infraCount, codeCount, history)
	classified.RecommendedAction = c.getRecommendedAction(classified.Category)

	return classified
}

// classifyCancelledJob determines why a job was cancelled
func (c *CIFailureClassifier) classifyCancelledJob(jobName string, detail *CIJobDetail) CIFailureReason {
	reason := CIFailureReason{
		Job:        jobName,
		Conclusion: "CANCELLED",
	}

	if detail == nil {
		// No timing info available - assume infrastructure (conservative)
		reason.Category = CategoryInfrastructure
		reason.Explanation = "Job was cancelled (no timing data available - assuming infrastructure issue)"
		return reason
	}

	reason.Duration = detail.Duration

	// Heuristic 1: Very short duration = likely infrastructure
	if detail.Duration > 0 && detail.Duration < InfrastructureDurationThreshold {
		reason.Category = CategoryInfrastructure
		reason.Explanation = "Job cancelled within 30 seconds of start - likely workflow superseded or runner issue"
		return reason
	}

	// Heuristic 2: Long duration = likely timeout or code issue
	if detail.Duration >= TimeoutDurationThreshold {
		reason.Category = CategoryCodeRelated
		reason.Explanation = "Job ran for extended period before cancellation - likely timeout, infinite loop, or resource exhaustion"
		return reason
	}

	// Heuristic 3: Medium duration - check job name for hints
	if c.jobNameSuggestsTimeout(jobName) {
		reason.Category = CategoryCodeRelated
		reason.Explanation = "Job name suggests test/build that may have timed out"
		return reason
	}

	// Default: assume infrastructure for medium-duration cancellations
	reason.Category = CategoryInfrastructure
	reason.Explanation = "Job was cancelled - likely infrastructure issue (workflow concurrency, manual cancellation)"
	return reason
}

// jobNameSuggestsTimeout checks if job name suggests it might timeout
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

// determineOverallCategory determines the overall failure category
func (c *CIFailureClassifier) determineOverallCategory(infraCount, codeCount int, history *CIFailureHistory) CIFailureCategory {
	// Check for persistent failure first
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

// getRecommendedAction returns the recommended action based on category
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

// parseJobDetails parses job timing information from CI output
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
			Name:       check.Name,
			Conclusion: check.Conclusion,
			Status:     check.Status,
		}

		if check.StartedAt != "" {
			if t, err := time.Parse(time.RFC3339, check.StartedAt); err == nil {
				detail.StartedAt = &t
			}
		}
		if check.CompletedAt != "" {
			if t, err := time.Parse(time.RFC3339, check.CompletedAt); err == nil {
				detail.CompletedAt = &t
			}
		}

		// Calculate duration if both times are available
		if detail.StartedAt != nil && detail.CompletedAt != nil {
			detail.Duration = detail.CompletedAt.Sub(*detail.StartedAt)
		}

		details = append(details, detail)
	}

	return details
}

// findJobDetail finds job detail by name
func (c *CIFailureClassifier) findJobDetail(details []CIJobDetail, jobName string) *CIJobDetail {
	for i := range details {
		if details[i].Name == jobName {
			return &details[i]
		}
	}
	return nil
}
