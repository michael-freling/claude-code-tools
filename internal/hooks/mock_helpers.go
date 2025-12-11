package hooks

import "github.com/stretchr/testify/mock"

// MockGitHelper is a mock implementation of GitHelper for testing.
type MockGitHelper struct {
	mock.Mock
}

// GetCurrentBranch is a mock implementation of GitHelper.GetCurrentBranch.
func (m *MockGitHelper) GetCurrentBranch() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// MockGhHelper is a mock implementation of GhHelper for testing.
type MockGhHelper struct {
	mock.Mock
}

// GetPRBaseBranch is a mock implementation of GhHelper.GetPRBaseBranch.
func (m *MockGhHelper) GetPRBaseBranch(prNumber string) (string, error) {
	args := m.Called(prNumber)
	return args.String(0), args.Error(1)
}
