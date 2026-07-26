package cisecrets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpdater_StoresRepoVerbatim(t *testing.T) {
	assert.Equal(t, "", NewUpdater("").Repo)
	assert.Equal(t, "owner/custom", NewUpdater("owner/custom").Repo)
}

func TestSecretSetArgs(t *testing.T) {
	assert.Equal(t, []string{"secret", "set", SecretName}, secretSetArgs("", SecretName))
	assert.Equal(t, []string{"secret", "set", SecretName, "--repo", "owner/repo"}, secretSetArgs("owner/repo", SecretName))
}

func TestRepoLabel(t *testing.T) {
	assert.Equal(t, "the current repository", (&Updater{}).repoLabel())
	assert.Equal(t, "owner/repo", (&Updater{Repo: "owner/repo"}).repoLabel())
}

func TestMask(t *testing.T) {
	assert.Equal(t, "***", mask("short"))
	assert.Equal(t, "***", mask("exactly12chr")) // 12 chars -> masked entirely
	assert.Equal(t, "abcdefgh...wxyz", mask("abcdefghijklmnopqrstuvwxyz"))
}

func TestUpdate(t *testing.T) {
	t.Run("empty token", func(t *testing.T) {
		u := &Updater{Repo: "owner/repo", setter: func(context.Context, string, string, string) error {
			t.Fatal("setter must not run for empty token")
			return nil
		}}
		_, err := u.Update(context.Background(), "   ")
		assert.ErrorContains(t, err, "empty token")
	})

	t.Run("success trims and masks", func(t *testing.T) {
		var gotRepo, gotName, gotValue string
		u := &Updater{Repo: "owner/repo", setter: func(_ context.Context, repo, name, value string) error {
			gotRepo, gotName, gotValue = repo, name, value
			return nil
		}}
		masked, err := u.Update(context.Background(), "  sk-ant-oat-1234567890abcd  ")
		require.NoError(t, err)
		assert.Equal(t, "owner/repo", gotRepo)
		assert.Equal(t, SecretName, gotName)
		assert.Equal(t, "sk-ant-oat-1234567890abcd", gotValue)
		assert.Equal(t, "sk-ant-o...abcd", masked)
	})

	t.Run("setter error is wrapped", func(t *testing.T) {
		sentinel := errors.New("gh failed")
		u := &Updater{Repo: "owner/repo", setter: func(context.Context, string, string, string) error {
			return sentinel
		}}
		_, err := u.Update(context.Background(), "sk-ant-oat-1234567890abcd")
		assert.ErrorIs(t, err, sentinel)
		assert.ErrorContains(t, err, SecretName)
	})
}

func TestUpdateFromSetupToken(t *testing.T) {
	t.Run("mints and uploads the token", func(t *testing.T) {
		var gotValue string
		u := &Updater{
			Repo:   "owner/repo",
			minter: func(context.Context) (string, error) { return "  sk-ant-oat01-longlivedtoken1234  ", nil },
			setter: func(_ context.Context, _, _, value string) error {
				gotValue = value
				return nil
			},
		}
		masked, err := u.UpdateFromSetupToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "sk-ant-oat01-longlivedtoken1234", gotValue)
		assert.Equal(t, "sk-ant-o...1234", masked)
	})

	t.Run("minter error is wrapped", func(t *testing.T) {
		u := &Updater{
			Repo:   "owner/repo",
			minter: func(context.Context) (string, error) { return "", errors.New("claude boom") },
			setter: func(context.Context, string, string, string) error {
				t.Fatal("setter must not run when minting fails")
				return nil
			},
		}
		_, err := u.UpdateFromSetupToken(context.Background())
		assert.ErrorContains(t, err, "claude boom")
		assert.ErrorContains(t, err, "setup-token")
	})
}

func TestExtractToken(t *testing.T) {
	t.Run("bare token", func(t *testing.T) {
		assert.Equal(t, "sk-ant-oat01-abcdefghijklmnop-qrstuv_wxyz", extractToken("sk-ant-oat01-abcdefghijklmnop-qrstuv_wxyz\n"))
	})

	t.Run("token amid surrounding output", func(t *testing.T) {
		out := "Visit https://claude.ai/oauth to authorize.\n" +
			"Paste code: xyz\n" +
			"Your token: sk-ant-oat01-abcdefghijklmnop1234\n" +
			"Done.\n"
		assert.Equal(t, "sk-ant-oat01-abcdefghijklmnop1234", extractToken(out))
	})

	t.Run("returns the last match", func(t *testing.T) {
		out := "old sk-ant-oat01-aaaaaaaaaaaaaaaa11\nnew sk-ant-oat01-bbbbbbbbbbbbbbbb22\n"
		assert.Equal(t, "sk-ant-oat01-bbbbbbbbbbbbbbbb22", extractToken(out))
	})

	t.Run("no token", func(t *testing.T) {
		assert.Equal(t, "", extractToken("nothing to see here"))
	})
}

func TestClaudeSetupToken_CommandError(t *testing.T) {
	// Point PATH at an empty dir so claude cannot be found, exercising the
	// error path without running the real interactive flow.
	t.Setenv("PATH", t.TempDir())
	_, err := claudeSetupToken(context.Background())
	assert.ErrorContains(t, err, "claude setup-token failed")
}

func TestGhSecretSet_CommandError(t *testing.T) {
	// Point PATH at an empty dir so gh cannot be found, exercising the error path.
	t.Setenv("PATH", t.TempDir())
	err := ghSecretSet(context.Background(), "owner/repo", SecretName, "value")
	assert.Error(t, err)
}
