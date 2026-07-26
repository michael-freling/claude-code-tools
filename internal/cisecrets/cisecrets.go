// Package cisecrets updates the GitHub Actions secrets used by CI, in
// particular the Claude Code OAuth token consumed by the PR review workflow.
package cisecrets

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// SecretName is the GitHub Actions secret holding the Claude Code OAuth token.
const SecretName = "CLAUDE_CODE_OAUTH_TOKEN"

// secretSetter sets a GitHub Actions secret. It is overridable in tests.
type secretSetter func(ctx context.Context, repo, name, value string) error

// tokenMinter mints a long-lived Claude Code OAuth token (via
// `claude setup-token`). It is overridable in tests.
type tokenMinter func(ctx context.Context) (string, error)

// Updater sets the Claude Code OAuth token secret on a GitHub repository.
type Updater struct {
	// Repo is the target repository (owner/name). When empty, gh resolves the
	// repository from the current working directory's git remote.
	Repo   string
	setter secretSetter
	minter tokenMinter
}

// NewUpdater returns an Updater that uses the gh CLI to set secrets and
// `claude setup-token` to mint long-lived tokens. An empty repo lets gh
// resolve the repository from the current directory.
func NewUpdater(repo string) *Updater {
	return &Updater{Repo: repo, setter: ghSecretSet, minter: claudeSetupToken}
}

// repoLabel describes the target repository for log/error messages.
func (u *Updater) repoLabel() string {
	if u.Repo == "" {
		return "the current repository"
	}
	return u.Repo
}

// Update sets SecretName to token and returns the masked value that was set.
func (u *Updater) Update(ctx context.Context, token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("empty token")
	}
	if err := u.setter(ctx, u.Repo, SecretName, token); err != nil {
		return "", fmt.Errorf("failed to set %s on %s: %w", SecretName, u.repoLabel(), err)
	}
	return mask(token), nil
}

// UpdateFromSetupToken mints a long-lived token with `claude setup-token` and
// sets it as the secret. Unlike the local subscription access token (which
// Claude Code rotates roughly hourly), a setup-token is valid for ~1 year, so
// the CI secret does not go stale within the hour.
//
// `claude setup-token` runs an interactive OAuth authorization flow (it needs a
// terminal and a browser), so this is meant to be run by a human — it cannot
// mint a token unattended.
func (u *Updater) UpdateFromSetupToken(ctx context.Context) (string, error) {
	token, err := u.minter(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to mint a long-lived token with 'claude setup-token': %w", err)
	}
	return u.Update(ctx, token)
}

// mask hides all but the first 8 and last 4 characters of a token.
func mask(token string) string {
	if len(token) <= 12 {
		return "***"
	}
	return token[:8] + "..." + token[len(token)-4:]
}

// setupTokenPattern matches a Claude Code OAuth token as printed by
// `claude setup-token` (e.g. "sk-ant-oat01-...").
var setupTokenPattern = regexp.MustCompile(`sk-ant-oat[0-9]*-[A-Za-z0-9_-]{16,}`)

// extractToken returns the last OAuth token found in text, or "" if none is
// present. `claude setup-token` prints the token last, after any authorization
// URL and prompts, so the final match is the token.
func extractToken(text string) string {
	matches := setupTokenPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1]
}

// claudeSetupToken runs `claude setup-token` and returns the long-lived token
// it prints. The child inherits stdin and stderr so the user can complete the
// interactive browser authorization, while stdout is both shown to the user and
// captured so the token can be extracted from it.
func claudeSetupToken(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "claude", "setup-token")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude setup-token failed: %w", err)
	}
	token := extractToken(buf.String())
	if token == "" {
		return "", fmt.Errorf("could not find an OAuth token in 'claude setup-token' output; run it manually and pass the value via --oauth-token")
	}
	return token, nil
}

// secretSetArgs builds the gh argument list. An empty repo omits --repo so gh
// resolves the repository from the current directory.
func secretSetArgs(repo, name string) []string {
	args := []string{"secret", "set", name}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	return args
}

// ghSecretSet sets a repository secret via the gh CLI, passing the value on
// stdin so it never appears in the process argument list.
func ghSecretSet(ctx context.Context, repo, name, value string) error {
	cmd := exec.CommandContext(ctx, "gh", secretSetArgs(repo, name)...)
	cmd.Stdin = strings.NewReader(value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
			return fmt.Errorf("%w: %s", err, trimmed)
		}
		return err
	}
	return nil
}
