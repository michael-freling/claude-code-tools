package claudecode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildContainerConfig_FullOptions(t *testing.T) {
	homeDir := t.TempDir()
	configDir := t.TempDir()
	projectDir := t.TempDir()
	sessionDir := t.TempDir()

	// Create all conditional paths
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, "CLAUDE.md"), []byte("# Home"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".claude"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, ".claude", "CLAUDE.md"), []byte("# DotClaude"), 0o644))
	for _, subdir := range []string{"rules", "agents", "commands", "skills"} {
		require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".claude", subdir), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "gitconfig"), []byte("[user]\n"), 0o644))

	opts := Options{
		SkipPermissions: true,
		Worktree:        true,
		Prompt:          "hello world",
		ProjectDir:      projectDir,
		HomeDir:         homeDir,
		ConfigDir:       configDir,
		SessionDir:      sessionDir,
		AuthToken:       "sk-ant-test",
		AuthType:        "api_key",
		ProjectID:       "my-project",
		Owner:           "owner",
		Repo:            "repo",
		GitUserName:     "Test User",
		GitUserEmail:    "test@example.com",
	}

	cfg, err := BuildContainerConfig(opts)

	require.NoError(t, err)

	// Env vars
	assert.Equal(t, "/home/user", cfg.Env["HOME"])
	assert.Equal(t, "0", cfg.Env["GIT_TERMINAL_PROMPT"])
	assert.Equal(t, "sk-ant-test", cfg.Env["ANTHROPIC_API_KEY"])
	assert.Empty(t, cfg.Env["CLAUDE_CODE_OAUTH_TOKEN"])

	// Cmd args
	assert.Contains(t, cfg.Cmd, "--dangerously-skip-permissions")
	assert.Contains(t, cfg.Cmd, "--worktree")
	assert.Contains(t, cfg.Cmd, "-p")
	assert.Contains(t, cfg.Cmd, "hello world")

	// Mounts: project dir + session dir + home CLAUDE.md + .claude/CLAUDE.md +
	// 4 subdirs + settings.json + gitconfig = 10
	assert.Len(t, cfg.Mounts, 10)

	// Verify project dir mount
	assert.Equal(t, projectDir, cfg.Mounts[0].Source)
	assert.Equal(t, "/work", cfg.Mounts[0].Target)
	assert.False(t, cfg.Mounts[0].ReadOnly)

	// Verify session dir mount
	assert.Equal(t, sessionDir, cfg.Mounts[1].Source)
	assert.Equal(t, "/home/user/.claude/projects/my-project/", cfg.Mounts[1].Target)
	assert.False(t, cfg.Mounts[1].ReadOnly)

	// Verify home CLAUDE.md mount
	assert.Equal(t, filepath.Join(homeDir, "CLAUDE.md"), cfg.Mounts[2].Source)
	assert.Equal(t, "/home/user/CLAUDE.md", cfg.Mounts[2].Target)
	assert.True(t, cfg.Mounts[2].ReadOnly)

	// Verify .claude/CLAUDE.md mount
	assert.Equal(t, filepath.Join(homeDir, ".claude", "CLAUDE.md"), cfg.Mounts[3].Source)
	assert.Equal(t, "/home/user/.claude/CLAUDE.md", cfg.Mounts[3].Target)
	assert.True(t, cfg.Mounts[3].ReadOnly)

	// Verify subdirectory mounts (rules, agents, commands, skills)
	subdirs := []string{"rules", "agents", "commands", "skills"}
	for i, subdir := range subdirs {
		idx := 4 + i
		assert.Equal(t, filepath.Join(homeDir, ".claude", subdir), cfg.Mounts[idx].Source)
		assert.Equal(t, "/home/user/.claude/"+subdir+"/", cfg.Mounts[idx].Target)
		assert.True(t, cfg.Mounts[idx].ReadOnly)
	}

	// Verify settings.json mount
	assert.Equal(t, filepath.Join(configDir, "settings.json"), cfg.Mounts[8].Source)
	assert.Equal(t, "/home/user/.claude/settings.json", cfg.Mounts[8].Target)
	assert.True(t, cfg.Mounts[8].ReadOnly)

	// Verify gitconfig mount
	assert.Equal(t, filepath.Join(configDir, "gitconfig"), cfg.Mounts[9].Source)
	assert.Equal(t, "/home/user/.gitconfig", cfg.Mounts[9].Target)
	assert.True(t, cfg.Mounts[9].ReadOnly)

	// Gitconfig content
	assert.Contains(t, cfg.Gitconfig, "Test User")
	assert.Contains(t, cfg.Gitconfig, "test@example.com")
	assert.Contains(t, cfg.Gitconfig, "proxy = http://gateway:8080")
}

func TestBuildContainerConfig_MinimalOptions(t *testing.T) {
	projectDir := t.TempDir()

	opts := Options{
		ProjectDir: projectDir,
		AuthToken:  "sk-ant-test",
		AuthType:   "api_key",
	}

	cfg, err := BuildContainerConfig(opts)

	require.NoError(t, err)

	// Only the required project dir mount
	assert.Len(t, cfg.Mounts, 1)
	assert.Equal(t, projectDir, cfg.Mounts[0].Source)
	assert.Equal(t, "/work", cfg.Mounts[0].Target)

	// No command args (SkipPermissions is false by default)
	assert.Empty(t, cfg.Cmd)
}

func TestBuildContainerConfig_OAuthAuth(t *testing.T) {
	projectDir := t.TempDir()

	opts := Options{
		ProjectDir: projectDir,
		AuthToken:  "oauth-token-123",
		AuthType:   "oauth",
	}

	cfg, err := BuildContainerConfig(opts)

	require.NoError(t, err)
	assert.Equal(t, "oauth-token-123", cfg.Env["CLAUDE_CODE_OAUTH_TOKEN"])
	assert.Empty(t, cfg.Env["ANTHROPIC_API_KEY"])
}

func TestBuildContainerConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		opts        Options
		errContains string
	}{
		{
			name: "missing project dir",
			opts: Options{
				AuthToken: "sk-test",
				AuthType:  "api_key",
			},
			errContains: "project directory is required",
		},
		{
			name: "missing auth token",
			opts: Options{
				ProjectDir: "/some/dir",
				AuthType:   "api_key",
			},
			errContains: "auth token is required",
		},
		{
			name: "invalid auth type",
			opts: Options{
				ProjectDir: "/some/dir",
				AuthToken:  "sk-test",
				AuthType:   "bearer",
			},
			errContains: `auth type must be "api_key" or "oauth"`,
		},
		{
			name: "empty auth type",
			opts: Options{
				ProjectDir: "/some/dir",
				AuthToken:  "sk-test",
				AuthType:   "",
			},
			errContains: `auth type must be "api_key" or "oauth"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildContainerConfig(tt.opts)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestBuildContainerConfig_CommandCombinations(t *testing.T) {
	tests := []struct {
		name     string
		opts     Options
		wantCmd  []string
		notInCmd []string
	}{
		{
			name: "skip permissions only",
			opts: Options{
				SkipPermissions: true,
			},
			wantCmd: []string{"--dangerously-skip-permissions"},
		},
		{
			name: "worktree only",
			opts: Options{
				Worktree: true,
			},
			wantCmd: []string{"--worktree"},
		},
		{
			name: "resume session",
			opts: Options{
				Resume: "session-123",
			},
			wantCmd: []string{"--resume", "session-123"},
		},
		{
			name: "continue most recent",
			opts: Options{
				Continue: true,
			},
			wantCmd: []string{"--continue"},
		},
		{
			name: "resume takes precedence over continue",
			opts: Options{
				Resume:   "session-123",
				Continue: true,
			},
			wantCmd:  []string{"--resume", "session-123"},
			notInCmd: []string{"--continue"},
		},
		{
			name: "prompt only",
			opts: Options{
				Prompt: "do something",
			},
			wantCmd: []string{"-p", "do something"},
		},
		{
			name: "skip permissions false",
			opts: Options{
				SkipPermissions: false,
			},
			wantCmd:  nil,
			notInCmd: []string{"--dangerously-skip-permissions"},
		},
		{
			name: "all flags together",
			opts: Options{
				SkipPermissions: true,
				Worktree:        true,
				Resume:          "sess-1",
				Prompt:          "build it",
			},
			wantCmd: []string{"--dangerously-skip-permissions", "--worktree", "--resume", "sess-1", "-p", "build it"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set required fields
			tt.opts.ProjectDir = t.TempDir()
			tt.opts.AuthToken = "sk-test"
			tt.opts.AuthType = "api_key"

			cfg, err := BuildContainerConfig(tt.opts)

			require.NoError(t, err)

			if tt.wantCmd == nil {
				assert.Empty(t, cfg.Cmd)
			} else {
				assert.Equal(t, tt.wantCmd, cfg.Cmd)
			}

			for _, arg := range tt.notInCmd {
				assert.NotContains(t, cfg.Cmd, arg)
			}
		})
	}
}

func TestBuildContainerConfig_ConditionalMounts(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()

	// Only create some paths
	require.NoError(t, os.WriteFile(filepath.Join(homeDir, "CLAUDE.md"), []byte("# Home"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".claude", "rules"), 0o755))
	// Don't create agents, commands, skills, .claude/CLAUDE.md

	opts := Options{
		ProjectDir: projectDir,
		HomeDir:    homeDir,
		AuthToken:  "sk-test",
		AuthType:   "api_key",
	}

	cfg, err := BuildContainerConfig(opts)

	require.NoError(t, err)

	// project dir + home CLAUDE.md + rules = 3
	assert.Len(t, cfg.Mounts, 3)
	assert.Equal(t, "/work", cfg.Mounts[0].Target)
	assert.Equal(t, "/home/user/CLAUDE.md", cfg.Mounts[1].Target)
	assert.Equal(t, "/home/user/.claude/rules/", cfg.Mounts[2].Target)
}

func TestGenerateGitconfig(t *testing.T) {
	opts := Options{
		GitUserName:  "Jane Doe",
		GitUserEmail: "jane@example.com",
	}

	result := generateGitconfig(opts)

	assert.Contains(t, result, `[http "https://github.com"]`)
	assert.Contains(t, result, `proxy = http://gateway:8080`)
	assert.Contains(t, result, `[http "https://api.github.com"]`)
	assert.Contains(t, result, `[user]`)
	assert.Contains(t, result, `name = Jane Doe`)
	assert.Contains(t, result, `email = jane@example.com`)
}

func TestGenerateGitconfig_EmptyUserInfo(t *testing.T) {
	opts := Options{}

	result := generateGitconfig(opts)

	assert.Contains(t, result, "name = \n")
	assert.Contains(t, result, "email = \n")
	// Proxy settings should still be present
	assert.Contains(t, result, `proxy = http://gateway:8080`)
}

func TestWriteGitconfig(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "nested", "config")

	opts := Options{
		GitUserName:  "John Doe",
		GitUserEmail: "john@example.com",
	}

	err := WriteGitconfig(configDir, opts)

	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(filepath.Join(configDir, "gitconfig"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "name = John Doe")
	assert.Contains(t, content, "email = john@example.com")
	assert.Contains(t, content, `proxy = http://gateway:8080`)
}

func TestWriteGitconfig_CreatesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	configDir := filepath.Join(baseDir, "does", "not", "exist")

	opts := Options{
		GitUserName:  "User",
		GitUserEmail: "user@test.com",
	}

	err := WriteGitconfig(configDir, opts)

	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify file exists
	_, err = os.Stat(filepath.Join(configDir, "gitconfig"))
	require.NoError(t, err)
}

func TestEnsureSettings_CreatesIfMissing(t *testing.T) {
	configDir := t.TempDir()

	err := EnsureSettings(configDir)

	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(configDir, "settings.json"))
	require.NoError(t, err)
	assert.Equal(t, DefaultSettings(), string(data))
}

func TestEnsureSettings_DoesNotOverwrite(t *testing.T) {
	configDir := t.TempDir()

	existingContent := `{"custom": "settings"}`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "settings.json"), []byte(existingContent), 0o644))

	err := EnsureSettings(configDir)

	require.NoError(t, err)

	// Verify content was NOT overwritten
	data, err := os.ReadFile(filepath.Join(configDir, "settings.json"))
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(data))
}

func TestEnsureSettings_CreatesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	configDir := filepath.Join(baseDir, "new", "dir")

	err := EnsureSettings(configDir)

	require.NoError(t, err)

	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	_, err = os.Stat(filepath.Join(configDir, "settings.json"))
	require.NoError(t, err)
}

func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()

	assert.Contains(t, settings, `"hasCompletedOnboarding": true`)
	assert.Contains(t, settings, `"autoUpdaterStatus": "disabled"`)
}
