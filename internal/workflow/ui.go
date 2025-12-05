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

// Spinner provides a simple spinner for long-running operations
type Spinner struct {
	message string
	done    chan bool
	running bool
	mu      sync.Mutex
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
		running: false,
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
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				fmt.Printf("\r%s %s", frames[i%len(frames)], s.message)
				i++
				time.Sleep(100 * time.Millisecond)
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

// FormatPlanSummary formats a plan in a nice box
func FormatPlanSummary(plan *Plan) string {
	var b strings.Builder

	b.WriteString("┌─────────────────────────────────────────────────────┐\n")
	b.WriteString("│ " + Bold("Plan Summary") + strings.Repeat(" ", 48-len("Plan Summary")) + "│\n")
	b.WriteString("├─────────────────────────────────────────────────────┤\n")

	summary := plan.Summary
	if len(summary) > 50 {
		summary = summary[:47] + "..."
	}
	b.WriteString("│ " + summary + strings.Repeat(" ", 52-len(summary)) + "│\n")
	b.WriteString("│" + strings.Repeat(" ", 54) + "│\n")

	b.WriteString("│ " + Bold("Phases:") + strings.Repeat(" ", 48) + "│\n")
	for i, phase := range plan.Phases {
		if i >= 5 {
			b.WriteString("│   ..." + strings.Repeat(" ", 50) + "│\n")
			break
		}
		line := fmt.Sprintf("%d. %s (%d files, ~%d lines)",
			i+1, phase.Name, phase.EstimatedFiles, phase.EstimatedLines)
		if len(line) > 50 {
			line = line[:47] + "..."
		}
		b.WriteString("│   " + line + strings.Repeat(" ", 52-len(line)-2) + "│\n")
	}
	b.WriteString("│" + strings.Repeat(" ", 54) + "│\n")

	complexityLine := fmt.Sprintf("Complexity: %s", plan.Complexity)
	b.WriteString("│ " + complexityLine + strings.Repeat(" ", 52-len(complexityLine)) + "│\n")

	totalLine := fmt.Sprintf("Total: ~%d lines across %d files",
		plan.EstimatedTotalLines, plan.EstimatedTotalFiles)
	b.WriteString("│ " + totalLine + strings.Repeat(" ", 52-len(totalLine)) + "│\n")

	b.WriteString("└─────────────────────────────────────────────────────┘")

	return b.String()
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
