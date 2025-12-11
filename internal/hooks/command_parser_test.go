package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGhApiCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "gh api command",
			command: "gh api /repos/owner/repo",
			want:    true,
		},
		{
			name:    "gh api with method flag",
			command: "gh api -X DELETE /repos/owner/repo/branches/main/protection",
			want:    true,
		},
		{
			name:    "gh api with --method flag",
			command: "gh api --method PUT /repos/owner/repo/branches/main/protection",
			want:    true,
		},
		{
			name:    "gh api with multiple flags",
			command: "gh api -X PUT /repos/owner/repo/rulesets -f data='test'",
			want:    true,
		},
		{
			name:    "gh pr command",
			command: "gh pr list",
			want:    false,
		},
		{
			name:    "gh issue command",
			command: "gh issue create",
			want:    false,
		},
		{
			name:    "git command",
			command: "git status",
			want:    false,
		},
		{
			name:    "only gh",
			command: "gh",
			want:    false,
		},
		{
			name:    "empty command",
			command: "",
			want:    false,
		},
		{
			name:    "non-gh command",
			command: "echo gh api",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGhApiCommand(tt.command)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractHTTPMethod(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{
			name:    "DELETE with -X flag",
			command: "gh api -X DELETE /repos/owner/repo/branches/main/protection",
			want:    "DELETE",
		},
		{
			name:    "PUT with -X flag",
			command: "gh api -X PUT /repos/owner/repo/branches/main/protection",
			want:    "PUT",
		},
		{
			name:    "PATCH with -X flag",
			command: "gh api -X PATCH /repos/owner/repo/branches/main/protection",
			want:    "PATCH",
		},
		{
			name:    "GET with -X flag",
			command: "gh api -X GET /repos/owner/repo/branches/main/protection",
			want:    "GET",
		},
		{
			name:    "DELETE with --method flag",
			command: "gh api --method DELETE /repos/owner/repo/branches/main/protection",
			want:    "DELETE",
		},
		{
			name:    "PUT with --method flag",
			command: "gh api --method PUT /repos/owner/repo/branches/main/protection",
			want:    "PUT",
		},
		{
			name:    "lowercase delete",
			command: "gh api -X delete /repos/owner/repo/branches/main/protection",
			want:    "DELETE",
		},
		{
			name:    "lowercase put",
			command: "gh api -X put /repos/owner/repo/branches/main/protection",
			want:    "PUT",
		},
		{
			name:    "mixed case Delete",
			command: "gh api -X Delete /repos/owner/repo/branches/main/protection",
			want:    "DELETE",
		},
		{
			name:    "method flag after endpoint",
			command: "gh api /repos/owner/repo/branches/main/protection -X DELETE",
			want:    "DELETE",
		},
		{
			name:    "no method flag (defaults to GET)",
			command: "gh api /repos/owner/repo/branches/main/protection",
			want:    "",
		},
		{
			name:    "method flag without value",
			command: "gh api -X",
			want:    "",
		},
		{
			name:    "empty command",
			command: "",
			want:    "",
		},
		{
			name:    "multiple method flags - first one wins",
			command: "gh api -X DELETE -X PUT /repos/owner/repo",
			want:    "DELETE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHTTPMethod(tt.command)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsProtectedBranch(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{
			name:   "main is protected",
			branch: "main",
			want:   true,
		},
		{
			name:   "master is protected",
			branch: "master",
			want:   true,
		},
		{
			name:   "main with leading spaces is protected",
			branch: "  main",
			want:   true,
		},
		{
			name:   "main with trailing spaces is protected",
			branch: "main  ",
			want:   true,
		},
		{
			name:   "main with spaces is protected",
			branch: "  main  ",
			want:   true,
		},
		{
			name:   "master with spaces is protected",
			branch: "  master  ",
			want:   true,
		},
		{
			name:   "feature branch is not protected",
			branch: "feature-branch",
			want:   false,
		},
		{
			name:   "main-feature is not protected",
			branch: "main-feature",
			want:   false,
		},
		{
			name:   "feature-main is not protected",
			branch: "feature-main",
			want:   false,
		},
		{
			name:   "master-copy is not protected",
			branch: "master-copy",
			want:   false,
		},
		{
			name:   "develop is not protected",
			branch: "develop",
			want:   false,
		},
		{
			name:   "staging is not protected",
			branch: "staging",
			want:   false,
		},
		{
			name:   "empty string is not protected",
			branch: "",
			want:   false,
		},
		{
			name:   "spaces only is not protected",
			branch: "   ",
			want:   false,
		},
		{
			name:   "Main with capital M is not protected",
			branch: "Main",
			want:   false,
		},
		{
			name:   "MAIN all caps is not protected",
			branch: "MAIN",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isProtectedBranch(tt.branch)
			assert.Equal(t, tt.want, got)
		})
	}
}
