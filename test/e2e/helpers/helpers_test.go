package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTempRepo(t *testing.T) {
	RequireGit(t)

	repo := NewTempRepo(t)

	require.NotEmpty(t, repo.Dir)
	assert.DirExists(t, repo.Dir)

	gitDir := filepath.Join(repo.Dir, ".git")
	assert.DirExists(t, gitDir)
}

func TestTempRepo_CreateFile(t *testing.T) {
	RequireGit(t)

	tests := []struct {
		name    string
		path    string
		content string
		wantErr bool
	}{
		{
			name:    "creates file in root directory",
			path:    "test.txt",
			content: "hello world",
			wantErr: false,
		},
		{
			name:    "creates file in subdirectory",
			path:    "subdir/test.txt",
			content: "nested file",
			wantErr: false,
		},
		{
			name:    "creates file with empty content",
			path:    "empty.txt",
			content: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewTempRepo(t)

			err := repo.CreateFile(tt.path, tt.content)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			fullPath := filepath.Join(repo.Dir, tt.path)
			assert.FileExists(t, fullPath)

			got, err := os.ReadFile(fullPath)
			require.NoError(t, err)
			assert.Equal(t, tt.content, string(got))
		})
	}
}

func TestTempRepo_Commit(t *testing.T) {
	RequireGit(t)

	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{
			name:    "creates commit successfully",
			message: "Test commit",
			wantErr: false,
		},
		{
			name:    "creates commit with multiline message",
			message: "Test commit\n\nWith details",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewTempRepo(t)

			err := repo.CreateFile("test.txt", "content")
			require.NoError(t, err)

			err = repo.Commit(tt.message)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			output, err := repo.RunGit("log", "--oneline", "-1")
			require.NoError(t, err)
			assert.Contains(t, output, "Test commit")
		})
	}
}

func TestTempRepo_CreateBranch(t *testing.T) {
	RequireGit(t)

	tests := []struct {
		name       string
		branchName string
		wantErr    bool
	}{
		{
			name:       "creates branch successfully",
			branchName: "feature-branch",
			wantErr:    false,
		},
		{
			name:       "creates branch with slashes",
			branchName: "feature/test",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewTempRepo(t)

			err := repo.CreateFile("test.txt", "content")
			require.NoError(t, err)

			err = repo.Commit("Initial commit")
			require.NoError(t, err)

			err = repo.CreateBranch(tt.branchName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			output, err := repo.RunGit("rev-parse", "--abbrev-ref", "HEAD")
			require.NoError(t, err)
			assert.Contains(t, output, tt.branchName)
		})
	}
}

func TestTempRepo_RunGit(t *testing.T) {
	RequireGit(t)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "runs git status",
			args:    []string{"status"},
			wantErr: false,
		},
		{
			name:    "runs git branch",
			args:    []string{"branch"},
			wantErr: false,
		},
		{
			name:    "fails with invalid command",
			args:    []string{"invalid-command"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewTempRepo(t)

			_, err := repo.RunGit(tt.args...)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestTempRepo_Cleanup(t *testing.T) {
	RequireGit(t)

	repo := NewTempRepo(t)
	dir := repo.Dir

	assert.DirExists(t, dir)

	repo.Cleanup()

	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}

func TestRequireGit(t *testing.T) {
	RequireGit(t)
}

func TestGitVersion(t *testing.T) {
	RequireGit(t)

	version := GitVersion(t)

	require.NotEmpty(t, version)
	assert.Contains(t, version, "git version")
}

func TestRequireGH(t *testing.T) {
	RequireGH(t)
}

func TestGHVersion(t *testing.T) {
	RequireGH(t)

	version := GHVersion(t)

	require.NotEmpty(t, version)
	assert.Contains(t, version, "gh version")
}

func TestIsCLIAvailable(t *testing.T) {
	available := IsCLIAvailable()

	assert.IsType(t, true, available)
}

func TestCleanupDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "cleanup-test-*")
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "test.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	CleanupDir(t, dir)

	_, err = os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}

func TestCleanupOnFailure(t *testing.T) {
	cleaned := false

	CleanupOnFailure(t, func() {
		cleaned = true
	})

	assert.False(t, cleaned, "cleanup should not run immediately")
}
