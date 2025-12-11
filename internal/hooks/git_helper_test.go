package hooks

import (
	"context"
	"fmt"
	"testing"

	"github.com/michael-freling/claude-code-tools/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewGitHelper(t *testing.T) {
	helper := NewGitHelper()
	assert.NotNil(t, helper)
	assert.IsType(t, &realGitHelper{}, helper)
}

func TestNewGitHelperWithRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := command.NewMockGitRunner(ctrl)
	helper := NewGitHelperWithRunner(mockRunner)

	assert.NotNil(t, helper)
	assert.IsType(t, &realGitHelper{}, helper)
}

func TestRealGitHelper_GetCurrentBranch(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*command.MockGitRunner)
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name: "returns current branch successfully",
			setupMock: func(m *command.MockGitRunner) {
				m.EXPECT().
					GetCurrentBranch(gomock.Any(), "").
					Return("main", nil)
			},
			want:    "main",
			wantErr: false,
		},
		{
			name: "returns feature branch",
			setupMock: func(m *command.MockGitRunner) {
				m.EXPECT().
					GetCurrentBranch(gomock.Any(), "").
					Return("feature-branch", nil)
			},
			want:    "feature-branch",
			wantErr: false,
		},
		{
			name: "fails when git runner fails",
			setupMock: func(m *command.MockGitRunner) {
				m.EXPECT().
					GetCurrentBranch(gomock.Any(), "").
					Return("", fmt.Errorf("not a git repository"))
			},
			wantErr:     true,
			errContains: "not a git repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := command.NewMockGitRunner(ctrl)
			tt.setupMock(mockRunner)

			helper := NewGitHelperWithRunner(mockRunner)
			got, err := helper.GetCurrentBranch()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRealGitHelper_GetCurrentBranch_UsesContextBackground(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := command.NewMockGitRunner(ctrl)
	mockRunner.EXPECT().
		GetCurrentBranch(gomock.AssignableToTypeOf(context.Background()), "").
		DoAndReturn(func(ctx context.Context, dir string) (string, error) {
			assert.NotNil(t, ctx)
			return "main", nil
		})

	helper := NewGitHelperWithRunner(mockRunner)
	got, err := helper.GetCurrentBranch()

	require.NoError(t, err)
	assert.Equal(t, "main", got)
}
