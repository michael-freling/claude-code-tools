package workflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/michael-freling/claude-code-tools/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGatherSummaryData(t *testing.T) {
	tests := []struct {
		name               string
		workflowName       string
		implData           *ImplementationSummary
		implLoadErr        error
		splitData          *PRSplitResult
		splitLoadErr       error
		singlePRData       *PRInfo
		singlePRErr        error
		setupGitRunner     func(*MockGitRunner)
		setupGhRunner      func(*MockGhRunner)
		want               *WorkflowSummary
		wantErr            bool
		skipSinglePRLookup bool
	}{
		{
			name:         "all data available with split PR",
			workflowName: "test-workflow",
			implData: &ImplementationSummary{
				FilesChanged: []string{"file1.go", "file2.go"},
				LinesAdded:   100,
				LinesRemoved: 50,
				TestsAdded:   10,
				Summary:      "Implementation complete",
			},
			splitData: &PRSplitResult{
				ParentPR: PRInfo{
					Number: 1,
					URL:    "https://github.com/test/repo/pull/1",
					Title:  "Parent PR",
					Branch: "main-branch",
				},
				ChildPRs: []PRInfo{
					{
						Number: 2,
						URL:    "https://github.com/test/repo/pull/2",
						Title:  "Child PR 1",
						Branch: "child-1",
					},
					{
						Number: 3,
						URL:    "https://github.com/test/repo/pull/3",
						Title:  "Child PR 2",
						Branch: "child-2",
					},
				},
				Summary: "Split complete",
			},
			want: &WorkflowSummary{
				WorkflowName: "test-workflow",
				PRType:       PRSummaryTypeSplit,
				MainPR: &PRInfo{
					Number: 1,
					URL:    "https://github.com/test/repo/pull/1",
					Title:  "Parent PR",
					Branch: "main-branch",
				},
				ChildPRs: []PRInfo{
					{
						Number: 2,
						URL:    "https://github.com/test/repo/pull/2",
						Title:  "Child PR 1",
						Branch: "child-1",
					},
					{
						Number: 3,
						URL:    "https://github.com/test/repo/pull/3",
						Title:  "Child PR 2",
						Branch: "child-2",
					},
				},
				FilesChanged: []string{"file1.go", "file2.go"},
				LinesAdded:   100,
				LinesRemoved: 50,
				TestsAdded:   10,
				Phases:       []PhaseStats{},
			},
			skipSinglePRLookup: true,
		},
		{
			name:         "single PR workflow",
			workflowName: "test-workflow",
			implData: &ImplementationSummary{
				FilesChanged: []string{"file1.go"},
				LinesAdded:   50,
				LinesRemoved: 25,
				TestsAdded:   5,
				Summary:      "Implementation complete",
			},
			splitLoadErr: os.ErrNotExist,
			singlePRData: &PRInfo{
				Number: 1,
				URL:    "https://github.com/test/repo/pull/1",
				Title:  "Test PR",
				Branch: "test-branch",
			},
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("test-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "test-branch").
					Return([]command.PRInfo{
						{
							Number:      1,
							URL:         "https://github.com/test/repo/pull/1",
							Title:       "Test PR",
							HeadRefName: "test-branch",
						},
					}, nil)
			},
			want: &WorkflowSummary{
				WorkflowName: "test-workflow",
				PRType:       PRSummaryTypeSingle,
				MainPR: &PRInfo{
					Number: 1,
					URL:    "https://github.com/test/repo/pull/1",
					Title:  "Test PR",
					Branch: "test-branch",
				},
				ChildPRs:     []PRInfo{},
				FilesChanged: []string{"file1.go"},
				LinesAdded:   50,
				LinesRemoved: 25,
				TestsAdded:   5,
				Phases:       []PhaseStats{},
			},
		},
		{
			name:         "no PR data available",
			workflowName: "test-workflow",
			implData: &ImplementationSummary{
				FilesChanged: []string{"file1.go"},
				LinesAdded:   50,
				LinesRemoved: 25,
				TestsAdded:   5,
			},
			splitLoadErr: os.ErrNotExist,
			singlePRData: nil,
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("test-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "test-branch").
					Return([]command.PRInfo{}, nil)
			},
			want: &WorkflowSummary{
				WorkflowName: "test-workflow",
				PRType:       PRSummaryTypeNone,
				MainPR:       nil,
				ChildPRs:     []PRInfo{},
				FilesChanged: []string{"file1.go"},
				LinesAdded:   50,
				LinesRemoved: 25,
				TestsAdded:   5,
				Phases:       []PhaseStats{},
			},
		},
		{
			name:         "missing implementation data",
			workflowName: "test-workflow",
			implLoadErr:  os.ErrNotExist,
			splitLoadErr: os.ErrNotExist,
			singlePRData: &PRInfo{
				Number: 1,
				URL:    "https://github.com/test/repo/pull/1",
				Title:  "Test PR",
				Branch: "test-branch",
			},
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("test-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "test-branch").
					Return([]command.PRInfo{
						{
							Number:      1,
							URL:         "https://github.com/test/repo/pull/1",
							Title:       "Test PR",
							HeadRefName: "test-branch",
						},
					}, nil)
			},
			want: &WorkflowSummary{
				WorkflowName: "test-workflow",
				PRType:       PRSummaryTypeSingle,
				MainPR: &PRInfo{
					Number: 1,
					URL:    "https://github.com/test/repo/pull/1",
					Title:  "Test PR",
					Branch: "test-branch",
				},
				ChildPRs:     []PRInfo{},
				FilesChanged: []string{},
				LinesAdded:   0,
				LinesRemoved: 0,
				TestsAdded:   0,
				Phases:       []PhaseStats{},
			},
		},
		{
			name:         "no data available at all",
			workflowName: "test-workflow",
			implLoadErr:  os.ErrNotExist,
			splitLoadErr: os.ErrNotExist,
			singlePRData: nil,
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("test-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "test-branch").
					Return([]command.PRInfo{}, nil)
			},
			want: &WorkflowSummary{
				WorkflowName: "test-workflow",
				PRType:       PRSummaryTypeNone,
				MainPR:       nil,
				ChildPRs:     []PRInfo{},
				FilesChanged: []string{},
				LinesAdded:   0,
				LinesRemoved: 0,
				TestsAdded:   0,
				Phases:       []PhaseStats{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			baseDir := filepath.Join(tmpDir, "test")
			require.NoError(t, os.MkdirAll(baseDir, 0755))

			mockStateManager := &MockStateManager{}
			mockGitRunner := &MockGitRunner{}
			mockGhRunner := &MockGhRunner{}
			mockLogger := NewLogger(LogLevelNormal)

			if tt.implLoadErr != nil {
				mockStateManager.On("LoadPhaseOutput", tt.workflowName, PhaseImplementation, &ImplementationSummary{}).
					Return(tt.implLoadErr)
			} else if tt.implData != nil {
				mockStateManager.On("LoadPhaseOutput", tt.workflowName, PhaseImplementation, &ImplementationSummary{}).
					Run(func(args mock.Arguments) {
						target := args.Get(2).(*ImplementationSummary)
						*target = *tt.implData
					}).
					Return(nil)
			}

			if tt.splitLoadErr != nil {
				mockStateManager.On("LoadPhaseOutput", tt.workflowName, PhasePRSplit, &PRSplitResult{}).
					Return(tt.splitLoadErr)
			} else if tt.splitData != nil {
				mockStateManager.On("LoadPhaseOutput", tt.workflowName, PhasePRSplit, &PRSplitResult{}).
					Run(func(args mock.Arguments) {
						target := args.Get(2).(*PRSplitResult)
						*target = *tt.splitData
					}).
					Return(nil)
			}

			if tt.setupGitRunner != nil {
				tt.setupGitRunner(mockGitRunner)
			}

			if tt.setupGhRunner != nil {
				tt.setupGhRunner(mockGhRunner)
			}

			config := DefaultConfig(baseDir)
			o := &Orchestrator{
				stateManager: mockStateManager,
				gitRunner:    mockGitRunner,
				ghRunner:     mockGhRunner,
				logger:       mockLogger,
				config:       config,
			}

			got, err := gatherSummaryData(context.Background(), o, tt.workflowName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			mockStateManager.AssertExpectations(t)
			if tt.setupGitRunner != nil {
				mockGitRunner.AssertExpectations(t)
			}
			if tt.setupGhRunner != nil {
				mockGhRunner.AssertExpectations(t)
			}
		})
	}
}

func TestGetSinglePRInfo(t *testing.T) {
	tests := []struct {
		name           string
		setupGitRunner func(*MockGitRunner)
		setupGhRunner  func(*MockGhRunner)
		want           *PRInfo
		wantErr        bool
	}{
		{
			name: "PR found for current branch",
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("feature-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "feature-branch").
					Return([]command.PRInfo{
						{
							Number:      123,
							URL:         "https://github.com/test/repo/pull/123",
							Title:       "Feature PR",
							HeadRefName: "feature-branch",
						},
					}, nil)
			},
			want: &PRInfo{
				Number: 123,
				URL:    "https://github.com/test/repo/pull/123",
				Title:  "Feature PR",
				Branch: "feature-branch",
			},
		},
		{
			name: "no PR found for current branch",
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("feature-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "feature-branch").
					Return([]command.PRInfo{}, nil)
			},
			want: nil,
		},
		{
			name: "error getting current branch",
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("", errors.New("git error"))
			},
			wantErr: true,
		},
		{
			name: "error listing PRs",
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("feature-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "feature-branch").
					Return(nil, errors.New("gh error"))
			},
			wantErr: true,
		},
		{
			name: "multiple PRs found returns first",
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("feature-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "feature-branch").
					Return([]command.PRInfo{
						{
							Number:      123,
							URL:         "https://github.com/test/repo/pull/123",
							Title:       "First PR",
							HeadRefName: "feature-branch",
						},
						{
							Number:      124,
							URL:         "https://github.com/test/repo/pull/124",
							Title:       "Second PR",
							HeadRefName: "feature-branch",
						},
					}, nil)
			},
			want: &PRInfo{
				Number: 123,
				URL:    "https://github.com/test/repo/pull/123",
				Title:  "First PR",
				Branch: "feature-branch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			baseDir := filepath.Join(tmpDir, "test")
			require.NoError(t, os.MkdirAll(baseDir, 0755))

			mockGitRunner := &MockGitRunner{}
			mockGhRunner := &MockGhRunner{}
			mockLogger := NewLogger(LogLevelNormal)

			if tt.setupGitRunner != nil {
				tt.setupGitRunner(mockGitRunner)
			}

			if tt.setupGhRunner != nil {
				tt.setupGhRunner(mockGhRunner)
			}

			config := DefaultConfig(baseDir)
			o := &Orchestrator{
				gitRunner: mockGitRunner,
				ghRunner:  mockGhRunner,
				logger:    mockLogger,
				config:    config,
			}

			got, err := getSinglePRInfo(context.Background(), o)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)

			if tt.setupGitRunner != nil {
				mockGitRunner.AssertExpectations(t)
			}
			if tt.setupGhRunner != nil {
				mockGhRunner.AssertExpectations(t)
			}
		})
	}
}

func TestFormatPRSection(t *testing.T) {
	tests := []struct {
		name    string
		summary *WorkflowSummary
		want    string
	}{
		{
			name:    "nil summary",
			summary: nil,
			want:    "",
		},
		{
			name: "no PR data",
			summary: &WorkflowSummary{
				PRType: PRSummaryTypeNone,
			},
			want: "",
		},
		{
			name: "single PR",
			summary: &WorkflowSummary{
				PRType: PRSummaryTypeSingle,
				MainPR: &PRInfo{
					Number: 123,
					Title:  "Add feature X",
					URL:    "https://github.com/owner/repo/pull/123",
				},
			},
			want: Bold("Pull Requests:") + "\n" +
				"  Main PR: " + Cyan("#123") + " - Add feature X\n" +
				"          " + Cyan("https://github.com/owner/repo/pull/123") + "\n",
		},
		{
			name: "split PR with children",
			summary: &WorkflowSummary{
				PRType: PRSummaryTypeSplit,
				MainPR: &PRInfo{
					Number: 100,
					Title:  "Parent PR",
					URL:    "https://github.com/owner/repo/pull/100",
				},
				ChildPRs: []PRInfo{
					{
						Number: 101,
						Title:  "Part 1: Database changes",
						URL:    "https://github.com/owner/repo/pull/101",
					},
					{
						Number: 102,
						Title:  "Part 2: API changes",
						URL:    "https://github.com/owner/repo/pull/102",
					},
				},
			},
			want: Bold("Pull Requests:") + "\n" +
				"  Main PR: " + Cyan("#100") + " - Parent PR\n" +
				"          " + Cyan("https://github.com/owner/repo/pull/100") + "\n\n" +
				"  Child PRs:\n" +
				"    • " + Cyan("#101") + " - Part 1: Database changes\n" +
				"      " + Cyan("https://github.com/owner/repo/pull/101") + "\n" +
				"    • " + Cyan("#102") + " - Part 2: API changes\n" +
				"      " + Cyan("https://github.com/owner/repo/pull/102") + "\n",
		},
		{
			name: "split PR without children",
			summary: &WorkflowSummary{
				PRType: PRSummaryTypeSplit,
				MainPR: &PRInfo{
					Number: 100,
					Title:  "Parent PR",
					URL:    "https://github.com/owner/repo/pull/100",
				},
				ChildPRs: []PRInfo{},
			},
			want: Bold("Pull Requests:") + "\n" +
				"  Main PR: " + Cyan("#100") + " - Parent PR\n" +
				"          " + Cyan("https://github.com/owner/repo/pull/100") + "\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPRSection(tt.summary)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatStatsSection(t *testing.T) {
	tests := []struct {
		name    string
		summary *WorkflowSummary
		want    string
	}{
		{
			name:    "nil summary",
			summary: nil,
			want:    "",
		},
		{
			name: "no stats data",
			summary: &WorkflowSummary{
				FilesChanged: []string{},
				LinesAdded:   0,
				LinesRemoved: 0,
				TestsAdded:   0,
			},
			want: "",
		},
		{
			name: "with stats data",
			summary: &WorkflowSummary{
				FilesChanged: []string{"file1.go", "file2.go", "file3.go"},
				LinesAdded:   350,
				LinesRemoved: 42,
				TestsAdded:   8,
			},
			want: Bold("Implementation Stats:") + "\n" +
				"  Files Changed: " + Green("3") + "\n" +
				"  Lines Added:   " + Green("+350") + "\n" +
				"  Lines Removed: -42\n" +
				"  Tests Added:   " + Green("8") + "\n",
		},
		{
			name: "only files changed",
			summary: &WorkflowSummary{
				FilesChanged: []string{"file1.go"},
				LinesAdded:   0,
				LinesRemoved: 0,
				TestsAdded:   0,
			},
			want: Bold("Implementation Stats:") + "\n" +
				"  Files Changed: " + Green("1") + "\n" +
				"  Lines Added:   " + Green("+0") + "\n" +
				"  Lines Removed: -0\n" +
				"  Tests Added:   " + Green("0") + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatsSection(tt.summary)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatPhaseTimings(t *testing.T) {
	tests := []struct {
		name    string
		summary *WorkflowSummary
		want    string
	}{
		{
			name:    "nil summary",
			summary: nil,
			want:    "",
		},
		{
			name: "no phase data",
			summary: &WorkflowSummary{
				Phases: []PhaseStats{},
			},
			want: "",
		},
		{
			name: "single successful phase",
			summary: &WorkflowSummary{
				Phases: []PhaseStats{
					{
						Name:     "Implementation",
						Attempts: 1,
						Duration: 15*60*1000000000 + 30*1000000000,
						Success:  true,
					},
				},
			},
			want: Bold("Phase Execution:") + "\n" +
				"  " + Green("✓") + " Implementation    " + Yellow("15m 30s") + " (1 attempt)\n",
		},
		{
			name: "multiple phases with different statuses",
			summary: &WorkflowSummary{
				Phases: []PhaseStats{
					{
						Name:     "Architecture Design",
						Attempts: 1,
						Duration: 2*60*1000000000 + 15*1000000000,
						Success:  true,
					},
					{
						Name:     "Implementation",
						Attempts: 2,
						Duration: 15*60*1000000000 + 30*1000000000,
						Success:  true,
					},
					{
						Name:     "Code Review",
						Attempts: 1,
						Duration: 3*60*1000000000 + 45*1000000000,
						Success:  true,
					},
				},
			},
			want: Bold("Phase Execution:") + "\n" +
				"  " + Green("✓") + " Architecture Design    " + Yellow("2m 15s") + " (1 attempt)\n" +
				"  " + Green("✓") + " Implementation    " + Yellow("15m 30s") + " (2 attempts)\n" +
				"  " + Green("✓") + " Code Review    " + Yellow("3m 45s") + " (1 attempt)\n",
		},
		{
			name: "failed phase",
			summary: &WorkflowSummary{
				Phases: []PhaseStats{
					{
						Name:     "Implementation",
						Attempts: 3,
						Duration: 10 * 60 * 1000000000,
						Success:  false,
					},
				},
			},
			want: Bold("Phase Execution:") + "\n" +
				"  " + Red("✗") + " Implementation    " + Yellow("10m 0s") + " (3 attempts)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPhaseTimings(tt.summary)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatWorkflowSummary(t *testing.T) {
	tests := []struct {
		name    string
		summary *WorkflowSummary
		want    string
	}{
		{
			name:    "nil summary",
			summary: nil,
			want:    "",
		},
		{
			name: "minimal summary",
			summary: &WorkflowSummary{
				WorkflowName:  "test-workflow",
				PRType:        PRSummaryTypeNone,
				FilesChanged:  []string{},
				Phases:        []PhaseStats{},
				TotalDuration: 5 * 60 * 1000000000,
			},
			want: "═══════════════════════════════════════════════════\n" +
				Bold("Workflow Summary: ") + "test-workflow\n" +
				"═══════════════════════════════════════════════════\n" +
				"\n" +
				Bold("Total Duration: ") + Yellow("5m 0s") + "\n",
		},
		{
			name: "complete summary with all sections",
			summary: &WorkflowSummary{
				WorkflowName: "feature-implementation",
				PRType:       PRSummaryTypeSingle,
				MainPR: &PRInfo{
					Number: 123,
					Title:  "Add feature X",
					URL:    "https://github.com/owner/repo/pull/123",
				},
				FilesChanged: []string{"file1.go", "file2.go"},
				LinesAdded:   100,
				LinesRemoved: 50,
				TestsAdded:   5,
				Phases: []PhaseStats{
					{
						Name:     "Implementation",
						Attempts: 1,
						Duration: 10 * 60 * 1000000000,
						Success:  true,
					},
				},
				TotalDuration: 15 * 60 * 1000000000,
			},
			want: "═══════════════════════════════════════════════════\n" +
				Bold("Workflow Summary: ") + "feature-implementation\n" +
				"═══════════════════════════════════════════════════\n" +
				"\n" +
				Bold("Pull Requests:") + "\n" +
				"  Main PR: " + Cyan("#123") + " - Add feature X\n" +
				"          " + Cyan("https://github.com/owner/repo/pull/123") + "\n" +
				"\n" +
				Bold("Implementation Stats:") + "\n" +
				"  Files Changed: " + Green("2") + "\n" +
				"  Lines Added:   " + Green("+100") + "\n" +
				"  Lines Removed: -50\n" +
				"  Tests Added:   " + Green("5") + "\n" +
				"\n" +
				Bold("Phase Execution:") + "\n" +
				"  " + Green("✓") + " Implementation    " + Yellow("10m 0s") + " (1 attempt)\n" +
				"\n" +
				Bold("Total Duration: ") + Yellow("15m 0s") + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatWorkflowSummary(tt.summary)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDisplayWorkflowSummary(t *testing.T) {
	tests := []struct {
		name              string
		workflowName      string
		implData          *ImplementationSummary
		implLoadErr       error
		splitData         *PRSplitResult
		splitLoadErr      error
		setupGitRunner    func(*MockGitRunner)
		setupGhRunner     func(*MockGhRunner)
		shouldPrintOutput bool
	}{
		{
			name:         "successful summary display",
			workflowName: "test-workflow",
			implData: &ImplementationSummary{
				FilesChanged: []string{"file1.go", "file2.go"},
				LinesAdded:   100,
				LinesRemoved: 50,
				TestsAdded:   10,
				Summary:      "Implementation complete",
			},
			splitLoadErr: os.ErrNotExist,
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("test-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "test-branch").
					Return([]command.PRInfo{
						{
							Number:      1,
							URL:         "https://github.com/test/repo/pull/1",
							Title:       "Test PR",
							HeadRefName: "test-branch",
						},
					}, nil)
			},
			shouldPrintOutput: true,
		},
		{
			name:         "summary gathering fails - logs warning and continues",
			workflowName: "test-workflow",
			implLoadErr:  errors.New("failed to load implementation"),
			splitLoadErr: errors.New("failed to load split"),
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("", errors.New("git error"))
			},
			shouldPrintOutput: false,
		},
		{
			name:         "empty summary - no output",
			workflowName: "test-workflow",
			implLoadErr:  os.ErrNotExist,
			splitLoadErr: os.ErrNotExist,
			setupGitRunner: func(m *MockGitRunner) {
				m.On("GetCurrentBranch", context.Background(), mock.Anything).
					Return("test-branch", nil)
			},
			setupGhRunner: func(m *MockGhRunner) {
				m.On("ListPRs", context.Background(), mock.Anything, "test-branch").
					Return([]command.PRInfo{}, nil)
			},
			shouldPrintOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			baseDir := filepath.Join(tmpDir, "test")
			require.NoError(t, os.MkdirAll(baseDir, 0755))

			mockStateManager := &MockStateManager{}
			mockGitRunner := &MockGitRunner{}
			mockGhRunner := &MockGhRunner{}
			mockLogger := NewLogger(LogLevelVerbose)

			if tt.implLoadErr != nil {
				mockStateManager.On("LoadPhaseOutput", tt.workflowName, PhaseImplementation, &ImplementationSummary{}).
					Return(tt.implLoadErr)
			} else if tt.implData != nil {
				mockStateManager.On("LoadPhaseOutput", tt.workflowName, PhaseImplementation, &ImplementationSummary{}).
					Run(func(args mock.Arguments) {
						target := args.Get(2).(*ImplementationSummary)
						*target = *tt.implData
					}).
					Return(nil)
			}

			if tt.splitLoadErr != nil {
				mockStateManager.On("LoadPhaseOutput", tt.workflowName, PhasePRSplit, &PRSplitResult{}).
					Return(tt.splitLoadErr)
			} else if tt.splitData != nil {
				mockStateManager.On("LoadPhaseOutput", tt.workflowName, PhasePRSplit, &PRSplitResult{}).
					Run(func(args mock.Arguments) {
						target := args.Get(2).(*PRSplitResult)
						*target = *tt.splitData
					}).
					Return(nil)
			}

			if tt.setupGitRunner != nil {
				tt.setupGitRunner(mockGitRunner)
			}

			if tt.setupGhRunner != nil {
				tt.setupGhRunner(mockGhRunner)
			}

			config := DefaultConfig(baseDir)
			o := &Orchestrator{
				stateManager: mockStateManager,
				gitRunner:    mockGitRunner,
				ghRunner:     mockGhRunner,
				logger:       mockLogger,
				config:       config,
			}

			o.displayWorkflowSummary(context.Background(), tt.workflowName)

			mockStateManager.AssertExpectations(t)
			if tt.setupGitRunner != nil {
				mockGitRunner.AssertExpectations(t)
			}
			if tt.setupGhRunner != nil {
				mockGhRunner.AssertExpectations(t)
			}
		})
	}
}
