package workflow

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ANSI color codes
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
)

// Green returns a green colored string
func Green(s string) string {
	return ansiGreen + s + ansiReset
}

// Red returns a red colored string
func Red(s string) string {
	return ansiRed + s + ansiReset
}

// Yellow returns a yellow colored string
func Yellow(s string) string {
	return ansiYellow + s + ansiReset
}

// Cyan returns a cyan colored string
func Cyan(s string) string {
	return ansiCyan + s + ansiReset
}

// Bold returns a bold string
func Bold(s string) string {
	return ansiBold + s + ansiReset
}

// SpinnerHooks provides callbacks for spinner lifecycle events (for testing)
type SpinnerHooks interface {
	OnStart()
	OnStop()
}

// noopHooks is the default implementation that does nothing
type noopHooks struct{}

func (noopHooks) OnStart() {}
func (noopHooks) OnStop()  {}

// Spinner provides a simple spinner for long-running operations
type Spinner struct {
	message string
	done    chan bool
	running bool
	mu      sync.Mutex
	clock   Clock
	hooks   SpinnerHooks
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return NewSpinnerWithDeps(message, NewRealClock(), noopHooks{})
}

// NewSpinnerWithDeps creates a new spinner with explicit dependencies
func NewSpinnerWithDeps(message string, clock Clock, hooks SpinnerHooks) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
		running: false,
		clock:   clock,
		hooks:   hooks,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.done = make(chan bool)
	s.mu.Unlock()

	go func() {
		s.hooks.OnStart()
		defer s.hooks.OnStop()

		ticker := s.clock.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C():
				fmt.Printf("\r%s %s", frames[i%len(frames)], s.message)
				i++
			}
		}
	}()
}

// Stop stops the spinner and clears the line
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.done)
	fmt.Print("\r\033[K")
}

// Success stops the spinner and shows a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Printf("%s %s\n", Green("✓"), message)
}

// Fail stops the spinner and shows a failure message
func (s *Spinner) Fail(message string) {
	s.Stop()
	fmt.Printf("%s %s\n", Red("✗"), message)
}

// FormatPhase formats the current phase indicator
func FormatPhase(current Phase, total int) string {
	phaseNum := getPhaseNumber(current)
	return fmt.Sprintf("Phase %d/%d: %s", phaseNum, total, getPhaseName(current))
}

// getPhaseNumber returns the phase number for display
func getPhaseNumber(phase Phase) int {
	switch phase {
	case PhasePlanning:
		return 1
	case PhaseConfirmation:
		return 2
	case PhaseImplementation:
		return 3
	case PhaseRefactoring:
		return 4
	case PhasePRSplit:
		return 5
	default:
		return 0
	}
}

// getPhaseName returns a human-readable phase name
func getPhaseName(phase Phase) string {
	switch phase {
	case PhasePlanning:
		return "Planning"
	case PhaseConfirmation:
		return "Confirmation"
	case PhaseImplementation:
		return "Implementation"
	case PhaseRefactoring:
		return "Refactoring"
	case PhasePRSplit:
		return "PR Split"
	case PhaseCompleted:
		return "Completed"
	case PhaseFailed:
		return "Failed"
	default:
		return string(phase)
	}
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// FormatPlanSummary formats a plan with full details
func FormatPlanSummary(plan *Plan) string {
	var b strings.Builder

	// Summary section
	b.WriteString(formatSectionHeader("Plan Summary"))
	b.WriteString("\n")
	b.WriteString(plan.Summary)
	b.WriteString("\n\n")

	// Complexity and totals
	b.WriteString(fmt.Sprintf("Complexity: %s\n", Bold(plan.Complexity)))
	b.WriteString(fmt.Sprintf("Total: ~%d lines across %d files\n", plan.EstimatedTotalLines, plan.EstimatedTotalFiles))

	// Architecture section (if available)
	if plan.Architecture.Overview != "" || len(plan.Architecture.Components) > 0 {
		b.WriteString("\n")
		b.WriteString(formatSectionHeader("Architecture"))
		b.WriteString("\n")

		if plan.Architecture.Overview != "" {
			b.WriteString("Overview:\n")
			b.WriteString(indentText(plan.Architecture.Overview, 2))
			b.WriteString("\n")
		}

		if len(plan.Architecture.Components) > 0 {
			b.WriteString("\nComponents:\n")
			for _, component := range plan.Architecture.Components {
				b.WriteString(fmt.Sprintf("  • %s\n", component))
			}
		}
	}

	// Phases section
	b.WriteString("\n")
	b.WriteString(formatSectionHeader(fmt.Sprintf("Phases (%d total)", len(plan.Phases))))
	b.WriteString("\n")

	for i, phase := range plan.Phases {
		b.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, Bold(phase.Name)))
		b.WriteString(fmt.Sprintf("   %d files, ~%d lines\n", phase.EstimatedFiles, phase.EstimatedLines))
		if phase.Description != "" {
			b.WriteString(indentText(phase.Description, 3))
			b.WriteString("\n")
		}
	}

	// Work Streams section (if available)
	if len(plan.WorkStreams) > 0 {
		b.WriteString("\n")
		b.WriteString(formatSectionHeader("Work Streams"))
		b.WriteString("\n")

		for _, ws := range plan.WorkStreams {
			b.WriteString(fmt.Sprintf("\n%s:\n", Bold(ws.Name)))
			for _, task := range ws.Tasks {
				b.WriteString(fmt.Sprintf("  • %s\n", task))
			}
			if len(ws.DependsOn) > 0 {
				b.WriteString(fmt.Sprintf("  Dependencies: %s\n", strings.Join(ws.DependsOn, ", ")))
			}
		}
	}

	// Risks section (if available)
	if len(plan.Risks) > 0 {
		b.WriteString("\n")
		b.WriteString(formatSectionHeader("Risks"))
		b.WriteString("\n")

		for _, risk := range plan.Risks {
			b.WriteString(fmt.Sprintf("  • %s\n", risk))
		}
	}

	return b.String()
}

// formatSectionHeader creates a section header with a line underneath
func formatSectionHeader(title string) string {
	return fmt.Sprintf("%s\n%s", Bold(title), strings.Repeat("─", len(title)+4))
}

// indentText adds indentation to each line of text
func indentText(text string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// StreamingSpinner provides a spinner that also displays streaming progress events
type StreamingSpinner struct {
	message   string
	done      chan bool
	running   bool
	mu        sync.Mutex
	lastTool  string
	toolCount int
	startTime time.Time
	logger    Logger
	clock     Clock
	hooks     SpinnerHooks
}

// NewStreamingSpinner creates a new streaming spinner with the given message
func NewStreamingSpinner(message string) *StreamingSpinner {
	return NewStreamingSpinnerWithDeps(message, nil, NewRealClock(), noopHooks{})
}

// NewStreamingSpinnerWithLogger creates a new streaming spinner with a logger for verbose output
func NewStreamingSpinnerWithLogger(message string, logger Logger) *StreamingSpinner {
	return NewStreamingSpinnerWithDeps(message, logger, NewRealClock(), noopHooks{})
}

// NewStreamingSpinnerWithDeps creates a new streaming spinner with explicit dependencies
func NewStreamingSpinnerWithDeps(message string, logger Logger, clock Clock, hooks SpinnerHooks) *StreamingSpinner {
	return &StreamingSpinner{
		message:   message,
		done:      make(chan bool),
		running:   false,
		startTime: clock.Now(),
		logger:    logger,
		clock:     clock,
		hooks:     hooks,
	}
}

// Start begins the streaming spinner animation
func (s *StreamingSpinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.done = make(chan bool)
	s.startTime = s.clock.Now()
	s.mu.Unlock()

	go func() {
		s.hooks.OnStart()
		defer s.hooks.OnStop()

		ticker := s.clock.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C():
				s.mu.Lock()
				elapsed := s.clock.Since(s.startTime).Round(time.Second)
				displayMsg := s.message
				if s.lastTool != "" {
					displayMsg = fmt.Sprintf("%s [%s]", s.message, s.lastTool)
				}
				fmt.Printf("\r%s %s (%s)", frames[i%len(frames)], displayMsg, elapsed)
				s.mu.Unlock()
				i++
			}
		}
	}()
}

// OnProgress handles a progress event and updates the display
func (s *StreamingSpinner) OnProgress(event ProgressEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch event.Type {
	case "tool_use":
		s.toolCount++
		// Truncate only for spinner status line (needs to fit on one line)
		if event.ToolInput != "" {
			s.lastTool = fmt.Sprintf("%s: %s", event.ToolName, truncateForDisplay(event.ToolInput, 40))
		} else {
			s.lastTool = event.ToolName
		}
		// Print full tool call on a new line - no truncation
		if event.ToolInput != "" {
			fmt.Printf("\r\033[K  %s %s %s\n", Cyan("→"), event.ToolName, event.ToolInput)
		} else {
			fmt.Printf("\r\033[K  %s %s\n", Cyan("→"), event.ToolName)
		}
	case "tool_result":
		if event.IsError {
			// Print full error message - no truncation
			fmt.Printf("\r\033[K  %s %s\n", Red("✗"), event.Text)
		}
		// Don't print successful tool results to avoid clutter
	case "text":
		// Show raw Claude text output in verbose mode
		if s.logger != nil && s.logger.IsVerbose() && event.Text != "" {
			// Clear the current line and print the text output
			fmt.Printf("\r\033[K")
			// Print each line with indentation for better readability
			lines := strings.Split(strings.TrimSpace(event.Text), "\n")
			for _, line := range lines {
				if line != "" {
					fmt.Printf("  %s %s\n", Yellow("│"), line)
				}
			}
		}
	}
}

// Stop stops the spinner and clears the line
func (s *StreamingSpinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.done)
	fmt.Print("\r\033[K")
}

// Success stops the spinner and shows a success message with stats
func (s *StreamingSpinner) Success(message string) {
	s.mu.Lock()
	toolCount := s.toolCount
	elapsed := s.clock.Since(s.startTime)
	s.mu.Unlock()

	s.Stop()
	if toolCount > 0 {
		fmt.Printf("%s %s (%d tool calls, %s)\n", Green("✓"), message, toolCount, FormatDuration(elapsed))
	} else {
		fmt.Printf("%s %s (%s)\n", Green("✓"), message, FormatDuration(elapsed))
	}
}

// Fail stops the spinner and shows a failure message
func (s *StreamingSpinner) Fail(message string) {
	s.Stop()
	fmt.Printf("%s %s\n", Red("✗"), message)
}

// truncateForDisplay truncates a string for display purposes
func truncateForDisplay(s string, maxLen int) string {
	// Remove newlines and extra whitespace
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// CISpinner displays CI waiting progress
type CISpinner struct {
	message   string
	startTime time.Time
	done      chan bool
	running   bool
	mu        sync.Mutex
	lastEvent CIProgressEvent
	clock     Clock
	hooks     SpinnerHooks
}

// NewCISpinner creates a new CI spinner with the given message
func NewCISpinner(message string) *CISpinner {
	return NewCISpinnerWithDeps(message, NewRealClock(), noopHooks{})
}

// NewCISpinnerWithDeps creates a new CI spinner with explicit dependencies
func NewCISpinnerWithDeps(message string, clock Clock, hooks SpinnerHooks) *CISpinner {
	return &CISpinner{
		message:   message,
		done:      make(chan bool),
		running:   false,
		startTime: clock.Now(),
		clock:     clock,
		hooks:     hooks,
	}
}

// Start begins the CI spinner animation
func (s *CISpinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.done = make(chan bool)
	s.startTime = s.clock.Now()
	s.mu.Unlock()

	go func() {
		s.hooks.OnStart()
		defer s.hooks.OnStop()

		ticker := s.clock.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C():
				s.mu.Lock()
				displayMsg := s.formatMessage()
				s.mu.Unlock()
				fmt.Printf("\r%s %s", frames[i%len(frames)], displayMsg)
				i++
			}
		}
	}()
}

// formatMessage formats the current spinner message based on the last event
func (s *CISpinner) formatMessage() string {
	event := s.lastEvent
	elapsed := FormatDuration(event.Elapsed)

	switch event.Type {
	case "waiting":
		nextCheckIn := FormatDuration(event.NextCheckIn)
		return fmt.Sprintf("%s (%s) [initial delay, checking in %s]", s.message, elapsed, nextCheckIn)
	case "checking":
		return fmt.Sprintf("%s (%s) [checking...]", s.message, elapsed)
	case "retry":
		return fmt.Sprintf("%s (%s) [retrying after timeout]", s.message, elapsed)
	case "status":
		return fmt.Sprintf("%s (%s) [%d passed, %d failed, %d pending]",
			s.message, elapsed, event.JobsPassed, event.JobsFailed, event.JobsPending)
	default:
		return fmt.Sprintf("%s (%s)", s.message, elapsed)
	}
}

// OnProgress handles a progress event and updates the display
func (s *CISpinner) OnProgress(event CIProgressEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastEvent = event
}

// Stop stops the spinner and clears the line
func (s *CISpinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.done)
	fmt.Print("\r\033[K")
}

// Success stops the spinner and shows a success message
func (s *CISpinner) Success(message string) {
	s.Stop()
	fmt.Printf("%s %s\n", Green("✓"), message)
}

// Fail stops the spinner and shows a failure message
func (s *CISpinner) Fail(message string) {
	s.Stop()
	fmt.Printf("%s %s\n", Red("✗"), message)
}

// FormatWorkflowStatus formats a workflow state with colors
func FormatWorkflowStatus(state *WorkflowState) string {
	var b strings.Builder

	b.WriteString(Bold("Workflow: ") + state.Name + "\n")
	b.WriteString(fmt.Sprintf("Type: %s\n", state.Type))
	b.WriteString(fmt.Sprintf("Description: %s\n", state.Description))

	statusStr := ""
	switch state.CurrentPhase {
	case PhaseCompleted:
		statusStr = Green("Completed")
	case PhaseFailed:
		statusStr = Red("Failed")
	default:
		statusStr = Yellow("In Progress")
	}
	b.WriteString(fmt.Sprintf("Status: %s\n", statusStr))

	b.WriteString(fmt.Sprintf("Current Phase: %s\n", getPhaseName(state.CurrentPhase)))

	elapsed := time.Since(state.CreatedAt)
	b.WriteString(fmt.Sprintf("Elapsed: %s\n", FormatDuration(elapsed)))

	if state.Error != nil {
		b.WriteString("\n" + Red("Error: ") + state.Error.Message + "\n")
		if state.Error.Recoverable {
			b.WriteString(Yellow("This error is recoverable. Use 'resume' to retry.\n"))
		}
	}

	b.WriteString("\nPhase History:\n")
	phases := []Phase{PhasePlanning, PhaseConfirmation, PhaseImplementation, PhaseRefactoring, PhasePRSplit}
	for _, phase := range phases {
		phaseState, ok := state.Phases[phase]
		if !ok {
			continue
		}

		statusIcon := ""
		statusColor := func(s string) string { return s }
		switch phaseState.Status {
		case StatusCompleted:
			statusIcon = "✓"
			statusColor = Green
		case StatusInProgress:
			statusIcon = "◷"
			statusColor = Yellow
		case StatusFailed:
			statusIcon = "✗"
			statusColor = Red
		case StatusSkipped:
			statusIcon = "○"
			statusColor = func(s string) string { return s }
		default:
			statusIcon = "○"
		}

		phaseName := getPhaseName(phase)
		b.WriteString(fmt.Sprintf("  %s %s",
			statusColor(statusIcon),
			phaseName))

		if phaseState.Attempts > 0 {
			b.WriteString(fmt.Sprintf(" (attempts: %d)", phaseState.Attempts))
		}
		b.WriteString("\n")
	}

	return b.String()
}
