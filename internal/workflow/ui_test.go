package workflow

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncHooks provides channel-based synchronization for spinner tests
type syncHooks struct {
	started chan struct{}
	stopped chan struct{}
}

func newSyncHooks() *syncHooks {
	return &syncHooks{
		started: make(chan struct{}, 1),
		stopped: make(chan struct{}, 1),
	}
}

func (h *syncHooks) OnStart() {
	select {
	case h.started <- struct{}{}:
	default:
	}
}

func (h *syncHooks) OnStop() {
	select {
	case h.stopped <- struct{}{}:
	default:
	}
}

func (h *syncHooks) WaitForStart(t *testing.T) {
	t.Helper()
	select {
	case <-h.started:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for spinner to start")
	}
}

func (h *syncHooks) WaitForStop(t *testing.T) {
	t.Helper()
	select {
	case <-h.stopped:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for spinner to stop")
	}
}

func TestColorFunctions(t *testing.T) {
	tests := []struct {
		name      string
		colorFunc func(string) string
		input     string
		wantStart string
		wantEnd   string
	}{
		{
			name:      "Green wraps with green ANSI codes",
			colorFunc: Green,
			input:     "success",
			wantStart: "\033[32m",
			wantEnd:   "\033[0m",
		},
		{
			name:      "Red wraps with red ANSI codes",
			colorFunc: Red,
			input:     "error",
			wantStart: "\033[31m",
			wantEnd:   "\033[0m",
		},
		{
			name:      "Yellow wraps with yellow ANSI codes",
			colorFunc: Yellow,
			input:     "warning",
			wantStart: "\033[33m",
			wantEnd:   "\033[0m",
		},
		{
			name:      "Cyan wraps with cyan ANSI codes",
			colorFunc: Cyan,
			input:     "info",
			wantStart: "\033[36m",
			wantEnd:   "\033[0m",
		},
		{
			name:      "Bold wraps with bold ANSI codes",
			colorFunc: Bold,
			input:     "emphasis",
			wantStart: "\033[1m",
			wantEnd:   "\033[0m",
		},
		{
			name:      "Green handles empty string",
			colorFunc: Green,
			input:     "",
			wantStart: "\033[32m",
			wantEnd:   "\033[0m",
		},
		{
			name:      "Red handles special characters",
			colorFunc: Red,
			input:     "error: failed!\n",
			wantStart: "\033[31m",
			wantEnd:   "\033[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.colorFunc(tt.input)
			assert.True(t, strings.HasPrefix(got, tt.wantStart))
			assert.True(t, strings.HasSuffix(got, tt.wantEnd))
			assert.Contains(t, got, tt.input)
		})
	}
}

func TestNewSpinner(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "creates spinner with message",
			message: "Loading...",
		},
		{
			name:    "creates spinner with empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSpinner(tt.message)
			require.NotNil(t, got)
			assert.Equal(t, tt.message, got.message)
			assert.False(t, got.running)
			assert.NotNil(t, got.done)
		})
	}
}

func TestSpinner_Lifecycle(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Start and Stop cycle works correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewSpinnerWithDeps("Testing", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)
			assert.True(t, spinner.running)

			spinner.Stop()
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestSpinner_DoubleStart(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "double-start is idempotent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewSpinnerWithDeps("Testing", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)
			assert.True(t, spinner.running)

			spinner.Start()
			assert.True(t, spinner.running)

			spinner.Stop()
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestSpinner_DoubleStop(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "double-stop is idempotent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewSpinnerWithDeps("Testing", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)

			spinner.Stop()
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)

			spinner.Stop()
			assert.False(t, spinner.running)
		})
	}
}

func TestSpinner_Success(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "Success stops spinner and prints message",
			message: "Operation completed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewSpinnerWithDeps("Testing", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)

			spinner.Success(tt.message)
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestSpinner_Fail(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "Fail stops spinner and prints error message",
			message: "Operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewSpinnerWithDeps("Testing", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)

			spinner.Fail(tt.message)
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestNewStreamingSpinner(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "creates streaming spinner with message",
			message: "Processing...",
		},
		{
			name:    "creates streaming spinner with empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStreamingSpinner(tt.message)
			require.NotNil(t, got)
			assert.Equal(t, tt.message, got.message)
			assert.False(t, got.running)
			assert.NotNil(t, got.done)
		})
	}
}

func TestStreamingSpinner_OnProgress(t *testing.T) {
	tests := []struct {
		name  string
		event ProgressEvent
	}{
		{
			name: "tool_use event with ToolInput",
			event: ProgressEvent{
				Type:      "tool_use",
				ToolName:  "Read",
				ToolInput: "/path/to/file.go",
			},
		},
		{
			name: "tool_use event without ToolInput",
			event: ProgressEvent{
				Type:     "tool_use",
				ToolName: "Bash",
			},
		},
		{
			name: "tool_result event with IsError true",
			event: ProgressEvent{
				Type:    "tool_result",
				Text:    "File not found",
				IsError: true,
			},
		},
		{
			name: "tool_result event with IsError false",
			event: ProgressEvent{
				Type:    "tool_result",
				Text:    "Success",
				IsError: false,
			},
		},
		{
			name: "text event",
			event: ProgressEvent{
				Type: "text",
				Text: "Some output",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewStreamingSpinnerWithDeps("Testing", nil, NewRealClock(), hooks)
			spinner.Start()
			hooks.WaitForStart(t)

			spinner.OnProgress(tt.event)

			if tt.event.Type == "tool_use" {
				if tt.event.ToolInput != "" {
					assert.Contains(t, spinner.lastTool, tt.event.ToolName)
				} else {
					assert.Equal(t, tt.event.ToolName, spinner.lastTool)
				}
			}

			spinner.Stop()
			hooks.WaitForStop(t)
		})
	}
}

func TestStreamingSpinner_Lifecycle(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Start and Stop cycle works correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewStreamingSpinnerWithDeps("Testing", nil, NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)
			assert.True(t, spinner.running)

			spinner.Stop()
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestStreamingSpinner_Success(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		toolCallCount int
		wantToolCount int
	}{
		{
			name:          "Success stops spinner and prints message with stats",
			message:       "Operation completed successfully",
			toolCallCount: 1,
			wantToolCount: 1,
		},
		{
			name:          "Success with no tool calls",
			message:       "Operation completed",
			toolCallCount: 0,
			wantToolCount: 0,
		},
		{
			name:          "Success with multiple tool calls",
			message:       "Operation completed",
			toolCallCount: 5,
			wantToolCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewStreamingSpinnerWithDeps("Testing", nil, NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)

			for i := 0; i < tt.toolCallCount; i++ {
				spinner.OnProgress(ProgressEvent{
					Type:     "tool_use",
					ToolName: "Read",
				})
			}

			assert.Equal(t, tt.wantToolCount, spinner.toolCount)

			spinner.Success(tt.message)
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestStreamingSpinner_Fail(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "Fail stops spinner and prints error message",
			message: "Operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewStreamingSpinnerWithDeps("Testing", nil, NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)

			spinner.Fail(tt.message)
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "0 seconds",
			duration: 0 * time.Second,
			want:     "0s",
		},
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			want:     "30s",
		},
		{
			name:     "90 seconds",
			duration: 90 * time.Second,
			want:     "1m 30s",
		},
		{
			name:     "3600 seconds",
			duration: 3600 * time.Second,
			want:     "1h 0m 0s",
		},
		{
			name:     "3661 seconds",
			duration: 3661 * time.Second,
			want:     "1h 1m 1s",
		},
		{
			name:     "1 hour exactly",
			duration: 1 * time.Hour,
			want:     "1h 0m 0s",
		},
		{
			name:     "2 hours 30 minutes 45 seconds",
			duration: 2*time.Hour + 30*time.Minute + 45*time.Second,
			want:     "2h 30m 45s",
		},
		{
			name:     "1 minute exactly",
			duration: 1 * time.Minute,
			want:     "1m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatPhase(t *testing.T) {
	tests := []struct {
		name  string
		phase Phase
		total int
		want  string
	}{
		{
			name:  "PhasePlanning",
			phase: PhasePlanning,
			total: 5,
			want:  "Phase 1/5: Planning",
		},
		{
			name:  "PhaseConfirmation",
			phase: PhaseConfirmation,
			total: 5,
			want:  "Phase 2/5: Confirmation",
		},
		{
			name:  "PhaseImplementation",
			phase: PhaseImplementation,
			total: 5,
			want:  "Phase 3/5: Implementation",
		},
		{
			name:  "PhaseRefactoring",
			phase: PhaseRefactoring,
			total: 5,
			want:  "Phase 4/5: Refactoring",
		},
		{
			name:  "PhasePRSplit",
			phase: PhasePRSplit,
			total: 5,
			want:  "Phase 5/5: PR Split",
		},
		{
			name:  "PhaseCompleted",
			phase: PhaseCompleted,
			total: 5,
			want:  "Phase 0/5: Completed",
		},
		{
			name:  "PhaseFailed",
			phase: PhaseFailed,
			total: 5,
			want:  "Phase 0/5: Failed",
		},
		{
			name:  "Unknown phase",
			phase: Phase("UNKNOWN"),
			total: 5,
			want:  "Phase 0/5: UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPhase(tt.phase, tt.total)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateForDisplay(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "long string truncated with ellipsis",
			input:  "this is a very long string that needs truncation",
			maxLen: 20,
			want:   "this is a very lo...",
		},
		{
			name:   "string with newlines converted to spaces",
			input:  "line1\nline2\nline3",
			maxLen: 50,
			want:   "line1 line2 line3",
		},
		{
			name:   "string with multiple spaces collapsed",
			input:  "hello    world   test",
			maxLen: 50,
			want:   "hello world test",
		},
		{
			name:   "string with tabs converted to spaces",
			input:  "hello\tworld",
			maxLen: 50,
			want:   "hello world",
		},
		{
			name:   "exact length string unchanged",
			input:  "12345678901234567890",
			maxLen: 20,
			want:   "12345678901234567890",
		},
		{
			name:   "string longer by one character truncated",
			input:  "123456789012345678901",
			maxLen: 20,
			want:   "12345678901234567...",
		},
		{
			name:   "very small maxLen",
			input:  "hello world",
			maxLen: 5,
			want:   "he...",
		},
		{
			name:   "maxLen of 4 truncates to single char plus ellipsis",
			input:  "hello",
			maxLen: 4,
			want:   "h...",
		},
		{
			name:   "maxLen of 3 produces only ellipsis",
			input:  "hello",
			maxLen: 3,
			want:   "...",
		},
		{
			name:   "newlines and spaces with truncation",
			input:  "line1\nline2\nline3 with very long content that needs truncation",
			maxLen: 30,
			want:   "line1 line2 line3 with very...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateForDisplay(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
			if len(tt.input) > tt.maxLen {
				assert.LessOrEqual(t, len(got), tt.maxLen)
			}
		})
	}
}

func TestFormatPlanSummary(t *testing.T) {
	tests := []struct {
		name         string
		plan         *Plan
		wantContains []string
	}{
		{
			name: "complete plan with all fields",
			plan: &Plan{
				Summary:             "Add authentication feature",
				Complexity:          "Medium",
				EstimatedTotalLines: 500,
				EstimatedTotalFiles: 10,
				Architecture: Architecture{
					Overview:   "JWT-based authentication",
					Components: []string{"Auth Handler", "Token Service"},
				},
				Phases: []PlanPhase{
					{
						Name:           "Setup",
						Description:    "Initial setup",
						EstimatedFiles: 3,
						EstimatedLines: 150,
					},
				},
				WorkStreams: []WorkStream{
					{
						Name:      "Backend",
						Tasks:     []string{"Create API", "Add middleware"},
						DependsOn: []string{"Setup"},
					},
				},
				Risks: []string{"Security vulnerability", "Performance impact"},
			},
			wantContains: []string{
				"Plan Summary",
				"Add authentication feature",
				"Complexity: ",
				"Medium",
				"~500 lines across 10 files",
				"Architecture",
				"JWT-based authentication",
				"Auth Handler",
				"Token Service",
				"Phases (1 total)",
				"Setup",
				"3 files, ~150 lines",
				"Work Streams",
				"Backend",
				"Create API",
				"Dependencies: Setup",
				"Risks",
				"Security vulnerability",
				"Performance impact",
			},
		},
		{
			name: "minimal plan without optional fields",
			plan: &Plan{
				Summary:             "Simple feature",
				Complexity:          "Low",
				EstimatedTotalLines: 50,
				EstimatedTotalFiles: 2,
				Phases: []PlanPhase{
					{
						Name:           "Implementation",
						EstimatedFiles: 2,
						EstimatedLines: 50,
					},
				},
			},
			wantContains: []string{
				"Plan Summary",
				"Simple feature",
				"Complexity: ",
				"Low",
				"~50 lines across 2 files",
				"Phases (1 total)",
				"Implementation",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPlanSummary(tt.plan)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestFormatWorkflowStatus(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-30 * time.Second)

	tests := []struct {
		name         string
		state        *WorkflowState
		wantContains []string
	}{
		{
			name: "completed workflow with green status",
			state: &WorkflowState{
				Name:         "feature/auth",
				Type:         WorkflowTypeFeature,
				Description:  "Add authentication",
				CurrentPhase: PhaseCompleted,
				CreatedAt:    earlier,
				Phases: map[Phase]*PhaseState{
					PhasePlanning: {
						Status:   StatusCompleted,
						Attempts: 1,
					},
				},
			},
			wantContains: []string{
				"Workflow: ",
				"feature/auth",
				"Type: feature",
				"Description: Add authentication",
				"Status: ",
				"Completed",
				"Current Phase: Completed",
				"Elapsed: ",
				"Phase History:",
			},
		},
		{
			name: "failed workflow with red status",
			state: &WorkflowState{
				Name:         "fix/bug",
				Type:         WorkflowTypeFix,
				Description:  "Fix critical bug",
				CurrentPhase: PhaseFailed,
				CreatedAt:    earlier,
				Error: &WorkflowError{
					Message:     "Build failed",
					Phase:       PhaseImplementation,
					Recoverable: false,
				},
				Phases: map[Phase]*PhaseState{},
			},
			wantContains: []string{
				"Workflow: ",
				"fix/bug",
				"Type: fix",
				"Status: ",
				"Failed",
				"Error: ",
				"Build failed",
			},
		},
		{
			name: "in progress workflow with yellow status",
			state: &WorkflowState{
				Name:         "feature/new",
				Type:         WorkflowTypeFeature,
				Description:  "New feature",
				CurrentPhase: PhaseImplementation,
				CreatedAt:    earlier,
				Phases: map[Phase]*PhaseState{
					PhaseImplementation: {
						Status:   StatusInProgress,
						Attempts: 1,
					},
				},
			},
			wantContains: []string{
				"Workflow: ",
				"feature/new",
				"Status: ",
				"In Progress",
				"Current Phase: Implementation",
			},
		},
		{
			name: "workflow with recoverable error shows recovery hint",
			state: &WorkflowState{
				Name:         "feature/retry",
				Type:         WorkflowTypeFeature,
				Description:  "Retry test",
				CurrentPhase: PhaseImplementation,
				CreatedAt:    earlier,
				Error: &WorkflowError{
					Message:     "Temporary failure",
					Phase:       PhaseImplementation,
					Recoverable: true,
				},
				Phases: map[Phase]*PhaseState{},
			},
			wantContains: []string{
				"Error: ",
				"Temporary failure",
				"This error is recoverable. Use 'resume' to retry.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatWorkflowStatus(tt.state)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestGetPhaseNumber(t *testing.T) {
	tests := []struct {
		name  string
		phase Phase
		want  int
	}{
		{
			name:  "PhasePlanning returns 1",
			phase: PhasePlanning,
			want:  1,
		},
		{
			name:  "PhaseConfirmation returns 2",
			phase: PhaseConfirmation,
			want:  2,
		},
		{
			name:  "PhaseImplementation returns 3",
			phase: PhaseImplementation,
			want:  3,
		},
		{
			name:  "PhaseRefactoring returns 4",
			phase: PhaseRefactoring,
			want:  4,
		},
		{
			name:  "PhasePRSplit returns 5",
			phase: PhasePRSplit,
			want:  5,
		},
		{
			name:  "PhaseCompleted returns 0",
			phase: PhaseCompleted,
			want:  0,
		},
		{
			name:  "PhaseFailed returns 0",
			phase: PhaseFailed,
			want:  0,
		},
		{
			name:  "Unknown phase returns 0",
			phase: Phase("UNKNOWN"),
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPhaseNumber(tt.phase)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPhaseName(t *testing.T) {
	tests := []struct {
		name  string
		phase Phase
		want  string
	}{
		{
			name:  "PhasePlanning returns Planning",
			phase: PhasePlanning,
			want:  "Planning",
		},
		{
			name:  "PhaseConfirmation returns Confirmation",
			phase: PhaseConfirmation,
			want:  "Confirmation",
		},
		{
			name:  "PhaseImplementation returns Implementation",
			phase: PhaseImplementation,
			want:  "Implementation",
		},
		{
			name:  "PhaseRefactoring returns Refactoring",
			phase: PhaseRefactoring,
			want:  "Refactoring",
		},
		{
			name:  "PhasePRSplit returns PR Split",
			phase: PhasePRSplit,
			want:  "PR Split",
		},
		{
			name:  "PhaseCompleted returns Completed",
			phase: PhaseCompleted,
			want:  "Completed",
		},
		{
			name:  "PhaseFailed returns Failed",
			phase: PhaseFailed,
			want:  "Failed",
		},
		{
			name:  "Unknown phase returns the phase string",
			phase: Phase("CUSTOM"),
			want:  "CUSTOM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPhaseName(tt.phase)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatSectionHeader(t *testing.T) {
	tests := []struct {
		name         string
		title        string
		wantContains []string
		wantPrefix   bool
	}{
		{
			name:  "creates header with underline",
			title: "Summary",
			wantContains: []string{
				"Summary",
				"───",
			},
			wantPrefix: true,
		},
		{
			name:  "longer title has longer underline",
			title: "Architecture Overview",
			wantContains: []string{
				"Architecture Overview",
				"─────────────────────",
			},
			wantPrefix: true,
		},
		{
			name:  "empty title",
			title: "",
			wantContains: []string{
				"────",
			},
			wantPrefix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSectionHeader(tt.title)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
			assert.True(t, strings.Contains(got, "\n"))
		})
	}
}

func TestIndentText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		spaces int
		want   string
	}{
		{
			name:   "single line indented by 2 spaces",
			text:   "hello",
			spaces: 2,
			want:   "  hello",
		},
		{
			name:   "multiple lines indented by 2 spaces",
			text:   "line1\nline2\nline3",
			spaces: 2,
			want:   "  line1\n  line2\n  line3",
		},
		{
			name:   "indented by 4 spaces",
			text:   "hello\nworld",
			spaces: 4,
			want:   "    hello\n    world",
		},
		{
			name:   "empty lines not indented",
			text:   "line1\n\nline3",
			spaces: 2,
			want:   "  line1\n\n  line3",
		},
		{
			name:   "zero spaces no indentation",
			text:   "hello",
			spaces: 0,
			want:   "hello",
		},
		{
			name:   "text with trailing newline",
			text:   "hello\n",
			spaces: 2,
			want:   "  hello\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentText(tt.text, tt.spaces)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStreamingSpinner_OnProgress_ToolCount(t *testing.T) {
	tests := []struct {
		name          string
		events        []ProgressEvent
		wantToolCount int
	}{
		{
			name: "single tool_use event increments count",
			events: []ProgressEvent{
				{Type: "tool_use", ToolName: "Read"},
			},
			wantToolCount: 1,
		},
		{
			name: "multiple tool_use events increment count",
			events: []ProgressEvent{
				{Type: "tool_use", ToolName: "Read"},
				{Type: "tool_use", ToolName: "Bash"},
				{Type: "tool_use", ToolName: "Edit"},
			},
			wantToolCount: 3,
		},
		{
			name: "tool_result events do not increment count",
			events: []ProgressEvent{
				{Type: "tool_result", Text: "Success"},
				{Type: "tool_result", Text: "Done"},
			},
			wantToolCount: 0,
		},
		{
			name: "mixed events only count tool_use",
			events: []ProgressEvent{
				{Type: "tool_use", ToolName: "Read"},
				{Type: "tool_result", Text: "Success"},
				{Type: "text", Text: "Processing"},
				{Type: "tool_use", ToolName: "Bash"},
			},
			wantToolCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewStreamingSpinnerWithDeps("Testing", nil, NewRealClock(), hooks)
			spinner.Start()
			hooks.WaitForStart(t)

			for _, event := range tt.events {
				spinner.OnProgress(event)
			}

			assert.Equal(t, tt.wantToolCount, spinner.toolCount)

			spinner.Stop()
			hooks.WaitForStop(t)
		})
	}
}

func TestStreamingSpinner_OnProgress_LongToolInput(t *testing.T) {
	tests := []struct {
		name          string
		event         ProgressEvent
		wantLastTool  string
		containsEllip bool
	}{
		{
			name: "long tool input is truncated for display",
			event: ProgressEvent{
				Type:      "tool_use",
				ToolName:  "Read",
				ToolInput: "/this/is/a/very/long/path/that/should/be/truncated/for/display/purposes/file.go",
			},
			wantLastTool:  "Read:",
			containsEllip: true,
		},
		{
			name: "short tool input is not truncated",
			event: ProgressEvent{
				Type:      "tool_use",
				ToolName:  "Bash",
				ToolInput: "ls -la",
			},
			wantLastTool:  "Bash: ls -la",
			containsEllip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewStreamingSpinnerWithDeps("Testing", nil, NewRealClock(), hooks)
			spinner.Start()
			hooks.WaitForStart(t)

			spinner.OnProgress(tt.event)

			assert.Contains(t, spinner.lastTool, tt.wantLastTool)
			if tt.containsEllip {
				assert.Contains(t, spinner.lastTool, "...")
			}

			spinner.Stop()
			hooks.WaitForStop(t)
		})
	}
}

func TestStreamingSpinner_DoubleStart(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "double-start is idempotent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewStreamingSpinnerWithDeps("Testing", nil, NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)
			assert.True(t, spinner.running)

			spinner.Start()
			assert.True(t, spinner.running)

			spinner.Stop()
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestStreamingSpinner_DoubleStop(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "double-stop is idempotent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewStreamingSpinnerWithDeps("Testing", nil, NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)

			spinner.Stop()
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)

			spinner.Stop()
			assert.False(t, spinner.running)
		})
	}
}

func TestFormatDuration_Rounding(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "rounds down from 499ms",
			duration: 30*time.Second + 499*time.Millisecond,
			want:     "30s",
		},
		{
			name:     "rounds up from 500ms",
			duration: 30*time.Second + 500*time.Millisecond,
			want:     "31s",
		},
		{
			name:     "rounds up from 999ms",
			duration: 30*time.Second + 999*time.Millisecond,
			want:     "31s",
		},
		{
			name:     "complex duration with milliseconds",
			duration: 1*time.Hour + 23*time.Minute + 45*time.Second + 678*time.Millisecond,
			want:     "1h 23m 46s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateForDisplay_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "empty string with maxLen 10",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "whitespace only string",
			input:  "    ",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "string with only newlines",
			input:  "\n\n\n",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "mixed whitespace characters",
			input:  " \t\n \t\n ",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "very short maxLen still truncates",
			input:  "hello world",
			maxLen: 5,
			want:   "he...",
		},
		{
			name:   "maxLen exactly 3 produces ellipsis",
			input:  "hello world",
			maxLen: 3,
			want:   "...",
		},
		{
			name:   "string with newlines and truncation",
			input:  "hello\nworld\ntest",
			maxLen: 10,
			want:   "hello w...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateForDisplay(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatPlanSummary_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		plan         *Plan
		wantContains []string
	}{
		{
			name: "plan with empty architecture overview but with components",
			plan: &Plan{
				Summary:             "Feature",
				Complexity:          "Low",
				EstimatedTotalLines: 100,
				EstimatedTotalFiles: 5,
				Architecture: Architecture{
					Overview:   "",
					Components: []string{"Component A", "Component B"},
				},
				Phases: []PlanPhase{
					{
						Name:           "Phase 1",
						EstimatedFiles: 5,
						EstimatedLines: 100,
					},
				},
			},
			wantContains: []string{
				"Architecture",
				"Components:",
				"Component A",
				"Component B",
			},
		},
		{
			name: "plan with empty work stream dependencies",
			plan: &Plan{
				Summary:             "Feature",
				Complexity:          "Low",
				EstimatedTotalLines: 100,
				EstimatedTotalFiles: 5,
				Phases: []PlanPhase{
					{
						Name:           "Phase 1",
						EstimatedFiles: 5,
						EstimatedLines: 100,
					},
				},
				WorkStreams: []WorkStream{
					{
						Name:      "Backend",
						Tasks:     []string{"Task 1"},
						DependsOn: []string{},
					},
				},
			},
			wantContains: []string{
				"Work Streams",
				"Backend",
				"Task 1",
			},
		},
		{
			name: "plan with multiple phases and descriptions",
			plan: &Plan{
				Summary:             "Complex feature",
				Complexity:          "High",
				EstimatedTotalLines: 1000,
				EstimatedTotalFiles: 50,
				Phases: []PlanPhase{
					{
						Name:           "Phase 1",
						Description:    "First phase description",
						EstimatedFiles: 10,
						EstimatedLines: 200,
					},
					{
						Name:           "Phase 2",
						Description:    "Second phase description",
						EstimatedFiles: 20,
						EstimatedLines: 400,
					},
					{
						Name:           "Phase 3",
						Description:    "",
						EstimatedFiles: 20,
						EstimatedLines: 400,
					},
				},
			},
			wantContains: []string{
				"Phases (3 total)",
				"Phase 1",
				"First phase description",
				"Phase 2",
				"Second phase description",
				"Phase 3",
				"10 files, ~200 lines",
				"20 files, ~400 lines",
			},
		},
		{
			name: "plan with empty risks array",
			plan: &Plan{
				Summary:             "Feature",
				Complexity:          "Low",
				EstimatedTotalLines: 100,
				EstimatedTotalFiles: 5,
				Phases: []PlanPhase{
					{
						Name:           "Phase 1",
						EstimatedFiles: 5,
						EstimatedLines: 100,
					},
				},
				Risks: []string{},
			},
			wantContains: []string{
				"Feature",
				"Phase 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPlanSummary(tt.plan)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestFormatWorkflowStatus_PhaseHistory(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-30 * time.Second)

	tests := []struct {
		name         string
		state        *WorkflowState
		wantContains []string
	}{
		{
			name: "workflow with multiple phase states",
			state: &WorkflowState{
				Name:         "feature/test",
				Type:         WorkflowTypeFeature,
				Description:  "Test workflow",
				CurrentPhase: PhaseImplementation,
				CreatedAt:    earlier,
				Phases: map[Phase]*PhaseState{
					PhasePlanning: {
						Status:   StatusCompleted,
						Attempts: 1,
					},
					PhaseConfirmation: {
						Status:   StatusCompleted,
						Attempts: 1,
					},
					PhaseImplementation: {
						Status:   StatusInProgress,
						Attempts: 2,
					},
				},
			},
			wantContains: []string{
				"Phase History:",
				"Planning",
				"Confirmation",
				"Implementation",
				"(attempts: 2)",
			},
		},
		{
			name: "workflow with skipped phase",
			state: &WorkflowState{
				Name:         "feature/test",
				Type:         WorkflowTypeFeature,
				Description:  "Test workflow",
				CurrentPhase: PhaseImplementation,
				CreatedAt:    earlier,
				Phases: map[Phase]*PhaseState{
					PhasePlanning: {
						Status:   StatusCompleted,
						Attempts: 1,
					},
					PhaseConfirmation: {
						Status:   StatusSkipped,
						Attempts: 0,
					},
					PhaseImplementation: {
						Status:   StatusInProgress,
						Attempts: 1,
					},
				},
			},
			wantContains: []string{
				"Phase History:",
				"Planning",
				"Confirmation",
				"Implementation",
			},
		},
		{
			name: "workflow with failed phase",
			state: &WorkflowState{
				Name:         "feature/test",
				Type:         WorkflowTypeFeature,
				Description:  "Test workflow",
				CurrentPhase: PhaseFailed,
				CreatedAt:    earlier,
				Phases: map[Phase]*PhaseState{
					PhasePlanning: {
						Status:   StatusCompleted,
						Attempts: 1,
					},
					PhaseConfirmation: {
						Status:   StatusFailed,
						Attempts: 3,
					},
				},
			},
			wantContains: []string{
				"Phase History:",
				"Planning",
				"Confirmation",
				"(attempts: 3)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatWorkflowStatus(tt.state)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestColorFunctions_MultipleWrapping(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Green and Bold combined",
			input: "test",
		},
		{
			name:  "Red and Bold combined",
			input: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			greenBold := Green(Bold(tt.input))
			assert.Contains(t, greenBold, tt.input)
			assert.Contains(t, greenBold, ansiGreen)
			assert.Contains(t, greenBold, ansiBold)

			boldRed := Bold(Red(tt.input))
			assert.Contains(t, boldRed, tt.input)
			assert.Contains(t, boldRed, ansiRed)
			assert.Contains(t, boldRed, ansiBold)
		})
	}
}

func TestFormatPlanSummary_LargeScale(t *testing.T) {
	tests := []struct {
		name         string
		plan         *Plan
		wantContains []string
	}{
		{
			name: "plan with many phases",
			plan: &Plan{
				Summary:             "Large refactoring",
				Complexity:          "High",
				EstimatedTotalLines: 5000,
				EstimatedTotalFiles: 100,
				Phases: []PlanPhase{
					{Name: "Phase 1", EstimatedFiles: 20, EstimatedLines: 1000},
					{Name: "Phase 2", EstimatedFiles: 20, EstimatedLines: 1000},
					{Name: "Phase 3", EstimatedFiles: 20, EstimatedLines: 1000},
					{Name: "Phase 4", EstimatedFiles: 20, EstimatedLines: 1000},
					{Name: "Phase 5", EstimatedFiles: 20, EstimatedLines: 1000},
				},
			},
			wantContains: []string{
				"Phases (5 total)",
				"Phase 1",
				"Phase 2",
				"Phase 3",
				"Phase 4",
				"Phase 5",
				"~5000 lines across 100 files",
			},
		},
		{
			name: "plan with many work streams",
			plan: &Plan{
				Summary:             "Multi-stream project",
				Complexity:          "High",
				EstimatedTotalLines: 3000,
				EstimatedTotalFiles: 60,
				Phases: []PlanPhase{
					{Name: "Phase 1", EstimatedFiles: 60, EstimatedLines: 3000},
				},
				WorkStreams: []WorkStream{
					{Name: "Frontend", Tasks: []string{"Task 1", "Task 2"}},
					{Name: "Backend", Tasks: []string{"Task 3", "Task 4"}},
					{Name: "Database", Tasks: []string{"Task 5", "Task 6"}},
					{Name: "Testing", Tasks: []string{"Task 7", "Task 8"}},
				},
			},
			wantContains: []string{
				"Work Streams",
				"Frontend",
				"Backend",
				"Database",
				"Testing",
			},
		},
		{
			name: "plan with many risks",
			plan: &Plan{
				Summary:             "Risky feature",
				Complexity:          "High",
				EstimatedTotalLines: 2000,
				EstimatedTotalFiles: 40,
				Phases: []PlanPhase{
					{Name: "Phase 1", EstimatedFiles: 40, EstimatedLines: 2000},
				},
				Risks: []string{
					"Risk 1: Data migration complexity",
					"Risk 2: Breaking API changes",
					"Risk 3: Performance degradation",
					"Risk 4: Security vulnerabilities",
					"Risk 5: Backward compatibility",
				},
			},
			wantContains: []string{
				"Risks",
				"Risk 1",
				"Risk 2",
				"Risk 3",
				"Risk 4",
				"Risk 5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPlanSummary(tt.plan)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestIndentText_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		spaces int
		want   string
	}{
		{
			name:   "text with consecutive newlines",
			text:   "line1\n\n\nline2",
			spaces: 2,
			want:   "  line1\n\n\n  line2",
		},
		{
			name:   "text with only newlines",
			text:   "\n\n\n",
			spaces: 2,
			want:   "\n\n\n",
		},
		{
			name:   "single newline",
			text:   "\n",
			spaces: 2,
			want:   "\n",
		},
		{
			name:   "text with leading newline",
			text:   "\nhello",
			spaces: 2,
			want:   "\n  hello",
		},
		{
			name:   "large indentation",
			text:   "test",
			spaces: 10,
			want:   "          test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentText(tt.text, tt.spaces)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSpinner_StopWithoutStart(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Stop without Start is safe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spinner := NewSpinner("Testing")
			assert.False(t, spinner.running)

			spinner.Stop()
			assert.False(t, spinner.running)
		})
	}
}

func TestStreamingSpinner_StopWithoutStart(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Stop without Start is safe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spinner := NewStreamingSpinner("Testing")
			assert.False(t, spinner.running)

			spinner.Stop()
			assert.False(t, spinner.running)
		})
	}
}

func TestCISpinner_formatMessage(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		event        CIProgressEvent
		wantContains []string
	}{
		{
			name:    "waiting event shows initial delay and next check-in",
			message: "Waiting for CI",
			event: CIProgressEvent{
				Type:        "waiting",
				Elapsed:     30 * time.Second,
				NextCheckIn: 15 * time.Second,
			},
			wantContains: []string{
				"Waiting for CI",
				"30s",
				"initial delay",
				"checking in 15s",
			},
		},
		{
			name:    "checking event shows checking status",
			message: "Checking CI",
			event: CIProgressEvent{
				Type:    "checking",
				Elapsed: 45 * time.Second,
			},
			wantContains: []string{
				"Checking CI",
				"45s",
				"checking...",
			},
		},
		{
			name:    "retry event shows retry after timeout",
			message: "Retrying CI",
			event: CIProgressEvent{
				Type:    "retry",
				Elapsed: 60 * time.Second,
			},
			wantContains: []string{
				"Retrying CI",
				"1m 0s",
				"retrying after timeout",
			},
		},
		{
			name:    "status event shows job counts",
			message: "CI Status",
			event: CIProgressEvent{
				Type:        "status",
				Elapsed:     90 * time.Second,
				JobsPassed:  5,
				JobsFailed:  1,
				JobsPending: 2,
			},
			wantContains: []string{
				"CI Status",
				"1m 30s",
				"5 passed",
				"1 failed",
				"2 pending",
			},
		},
		{
			name:    "unknown event type shows basic message",
			message: "Unknown",
			event: CIProgressEvent{
				Type:    "unknown",
				Elapsed: 10 * time.Second,
			},
			wantContains: []string{
				"Unknown",
				"10s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spinner := &CISpinner{
				message:   tt.message,
				lastEvent: tt.event,
			}

			got := spinner.formatMessage()
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestCISpinner_OnProgress(t *testing.T) {
	tests := []struct {
		name  string
		event CIProgressEvent
	}{
		{
			name: "updates lastEvent on progress",
			event: CIProgressEvent{
				Type:        "status",
				Elapsed:     30 * time.Second,
				JobsPassed:  3,
				JobsFailed:  0,
				JobsPending: 1,
			},
		},
		{
			name: "handles waiting event",
			event: CIProgressEvent{
				Type:        "waiting",
				Elapsed:     15 * time.Second,
				NextCheckIn: 10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spinner := NewCISpinner("Testing")

			spinner.OnProgress(tt.event)

			assert.Equal(t, tt.event, spinner.lastEvent)
		})
	}
}

func TestCISpinner_Lifecycle(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Start and Stop cycle works correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewCISpinnerWithDeps("Testing CI", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)
			assert.True(t, spinner.running)

			spinner.Stop()
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestCISpinner_DoubleStart(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "double-start is idempotent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewCISpinnerWithDeps("Testing", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)
			assert.True(t, spinner.running)

			spinner.Start()
			assert.True(t, spinner.running)

			spinner.Stop()
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestNewCISpinner(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "creates CI spinner with message",
			message: "Waiting for CI",
		},
		{
			name:    "creates CI spinner with empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewCISpinner(tt.message)
			require.NotNil(t, got)
			assert.Equal(t, tt.message, got.message)
			assert.False(t, got.running)
			assert.NotNil(t, got.done)
		})
	}
}

func TestCISpinner_Success(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "Success stops spinner and prints message",
			message: "CI checks passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewCISpinnerWithDeps("Testing", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)

			spinner.Success(tt.message)
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestCISpinner_Fail(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "Fail stops spinner and prints error message",
			message: "CI checks failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewCISpinnerWithDeps("Testing", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)

			spinner.Fail(tt.message)
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)
		})
	}
}

func TestCISpinner_DoubleStop(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "double-stop is idempotent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := newSyncHooks()
			spinner := NewCISpinnerWithDeps("Testing", NewRealClock(), hooks)

			spinner.Start()
			hooks.WaitForStart(t)

			spinner.Stop()
			hooks.WaitForStop(t)
			assert.False(t, spinner.running)

			spinner.Stop()
			assert.False(t, spinner.running)
		})
	}
}

func TestCISpinner_StopWithoutStart(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Stop without Start is safe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spinner := NewCISpinner("Testing")
			assert.False(t, spinner.running)

			spinner.Stop()
			assert.False(t, spinner.running)
		})
	}
}
