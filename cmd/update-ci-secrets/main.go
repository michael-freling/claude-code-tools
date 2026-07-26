// Command update-ci-secrets sets the CLAUDE_CODE_OAUTH_TOKEN GitHub Actions
// secret so the Claude Code PR review workflow can authenticate.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/michael-freling/claude-code-tools/internal/cisecrets"
	"github.com/spf13/cobra"
)

// updater is the subset of *cisecrets.Updater used by the command. Tests
// override newUpdater to inject a fake.
type updater interface {
	Update(ctx context.Context, token string) (string, error)
	UpdateFromSetupToken(ctx context.Context) (string, error)
}

// newUpdater builds the updater for the given repo. Overridable in tests.
var newUpdater = func(repo string) updater { return cisecrets.NewUpdater(repo) }

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		repo       string
		oauthToken string
	)

	cmd := &cobra.Command{
		Use:   "update-ci-secrets",
		Short: "Update the CLAUDE_CODE_OAUTH_TOKEN GitHub Actions secret",
		Long: `update-ci-secrets sets the CLAUDE_CODE_OAUTH_TOKEN secret on the GitHub
repository so the Claude Code PR review workflow (.github/workflows/claude-review.yml)
can authenticate.

By default it mints a long-lived token by running 'claude setup-token' (valid
~1 year) and uploads that, so the CI secret does not go stale. 'claude
setup-token' runs an interactive OAuth authorization flow, so run this from a
terminal where you can complete the browser login. Pass --oauth-token to upload
a token you already have instead.

The secret is set on the repository in the current directory unless --repo is
given. Requires the gh CLI (installed and authenticated with repo admin access)
and, for the default path, the claude CLI.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := newUpdater(repo)
			ctx := cmd.Context()

			var (
				masked string
				err    error
			)
			// Mint a long-lived token via 'claude setup-token' unless an
			// explicit --oauth-token is given.
			if oauthToken != "" {
				masked, err = u.Update(ctx, oauthToken)
			} else {
				masked, err = u.UpdateFromSetupToken(ctx)
			}
			if err != nil {
				return err
			}

			target := repo
			if target == "" {
				target = "the current repository"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set %s (%s) on %s\n", cisecrets.SecretName, masked, target)
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repository (owner/name); defaults to the repository in the current directory")
	cmd.Flags().StringVar(&oauthToken, "oauth-token", "", "OAuth token to upload directly; when omitted, 'claude setup-token' mints a long-lived one")

	return cmd
}
