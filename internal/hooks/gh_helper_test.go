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

func TestNewGhHelper(t *testing.T) {
	helper := NewGhHelper()
	assert.NotNil(t, helper)
	assert.IsType(t, &realGhHelper{}, helper)
}

func TestNewGhHelperWithRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := command.NewMockGhRunner(ctrl)
	helper := NewGhHelperWithRunner(mockRunner)

	assert.NotNil(t, helper)
	assert.IsType(t, &realGhHelper{}, helper)
}

func TestRealGhHelper_GetPRBaseBranch(t *testing.T) {
	tests := []struct {
		name        string
		prNumber    string
		setupMock   func(*command.MockGhRunner)
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:     "returns base branch successfully",
			prNumber: "123",
			setupMock: func(m *command.MockGhRunner) {
				m.EXPECT().
					GetPRBaseBranch(gomock.Any(), "", "123").
					Return("main", nil)
			},
			want:    "main",
			wantErr: false,
		},
		{
			name:     "returns develop branch",
			prNumber: "456",
			setupMock: func(m *command.MockGhRunner) {
				m.EXPECT().
					GetPRBaseBranch(gomock.Any(), "", "456").
					Return("develop", nil)
			},
			want:    "develop",
			wantErr: false,
		},
		{
			name:     "fails when gh runner fails",
			prNumber: "789",
			setupMock: func(m *command.MockGhRunner) {
				m.EXPECT().
					GetPRBaseBranch(gomock.Any(), "", "789").
					Return("", fmt.Errorf("pull request not found"))
			},
			wantErr:     true,
			errContains: "pull request not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRunner := command.NewMockGhRunner(ctrl)
			tt.setupMock(mockRunner)

			helper := NewGhHelperWithRunner(mockRunner)
			got, err := helper.GetPRBaseBranch(tt.prNumber)

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

func TestRealGhHelper_GetPRBaseBranch_UsesContextBackground(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := command.NewMockGhRunner(ctrl)
	mockRunner.EXPECT().
		GetPRBaseBranch(gomock.AssignableToTypeOf(context.Background()), "", "123").
		DoAndReturn(func(ctx context.Context, dir string, prNumber string) (string, error) {
			assert.NotNil(t, ctx)
			return "main", nil
		})

	helper := NewGhHelperWithRunner(mockRunner)
	got, err := helper.GetPRBaseBranch("123")

	require.NoError(t, err)
	assert.Equal(t, "main", got)
}
