---
description: Check all open PRs in a repository and create a merge PR that combines them for testing
argument-hint: "[optional: base branch, defaults to main]"
allowed-tools: ["Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task", "WebFetch", "TodoWrite"]
---

# Merge All PRs

Check all open Pull Requests in the current repository and create a single merge PR that combines them all, allowing the user to test all pending changes together.

## Prerequisites

**Required Tools:**
- GitHub CLI (`gh`) - Install via: `brew install gh` or `sudo apt install gh`
- Authenticate with: `gh auth login`

## Arguments

- **$1**: (Optional) Base branch name. Defaults to `main` if not provided.

## Workflow

### 1. List All Open PRs

First, gather information about all open PRs in the repository:

```bash
# Get all open PRs with details
gh pr list --state open --json number,title,headRefName,baseRefName,author,createdAt,updatedAt,mergeable,isDraft

# Get a summary count
gh pr list --state open --json number | jq length
```

Analyze:
- Total number of open PRs
- Which PRs are drafts vs ready for review
- Which PRs are mergeable
- The base branch each PR targets
- Any potential conflicts between PRs

### 2. Filter and Prioritize PRs

Present the list of PRs to the user with:
- PR number and title
- Author
- Target branch
- Draft status
- Mergeable status
- Created/Updated dates

**Ask the user to confirm:**
1. Which PRs should be included in the merge PR (default: all non-draft, mergeable PRs)
2. The order in which PRs should be merged (oldest first by default, or by dependency)
3. Whether to include draft PRs

**IMPORTANT**: Wait for user approval before proceeding.

### 3. Check for Conflicts

Before creating the merge PR, check for potential merge conflicts:

```bash
# For each selected PR, check if it can be merged cleanly
for pr_number in <selected_prs>; do
  gh pr view $pr_number --json mergeable,mergeStateStatus
done
```

If conflicts are detected:
- Report which PRs have conflicts
- Ask the user how to proceed (skip conflicting PRs, or resolve manually)

### 4. Create the Merge Branch

Once the user approves the plan:

1. **Ensure local base branch is up to date:**
   ```bash
   git fetch origin
   git checkout ${1:-main}
   git reset --hard origin/${1:-main}
   ```

2. **Create the merge branch:**
   ```bash
   git checkout -b merge-all-prs-$(date +%Y%m%d-%H%M%S)
   ```

### 5. Merge Each PR Branch

For each selected PR in order:

```bash
# Get the PR branch name
pr_branch=$(gh pr view $pr_number --json headRefName --jq '.headRefName')

# Fetch the branch
git fetch origin $pr_branch

# Merge the branch
git merge origin/$pr_branch --no-edit -m "Merge PR #$pr_number: <pr_title>"
```

**If a merge conflict occurs:**
1. Report the conflict to the user
2. Options:
   - Skip this PR and continue with others
   - Stop and let the user resolve manually
   - Abort the entire operation

### 6. Push and Create the Merge PR

After all selected PRs are merged:

```bash
# Push the merge branch
git push -u origin merge-all-prs-$(date +%Y%m%d-%H%M%S)

# Create the merge PR
gh pr create --title "Merge PR: Combined testing of all open PRs" \
  --body "$(cat <<'EOF'
## Summary

This PR combines all open PRs for integrated testing.

## Included PRs

<List of all merged PRs with numbers and titles>

| PR | Title | Author | Status |
|----|-------|--------|--------|
| #XXX | Title | @author | Merged |
...

## Purpose

This merge PR allows testing all pending changes together before individual PRs are merged.

## Instructions

1. Test the combined changes in this branch
2. If issues are found, identify which PR caused the issue
3. Do NOT merge this PR directly - merge individual PRs instead
4. Delete this branch after testing is complete

---
**Note:** This is a testing branch only. Individual PRs should be reviewed and merged separately.
EOF
)" \
  --base ${1:-main} \
  --draft
```

### 7. Provide Summary to User

Present to the user:
- Merge PR number and URL
- List of all included PRs
- Any PRs that were skipped (with reasons)
- Instructions for testing
- Reminder that this is for testing only

## Guidelines

- Always create the merge PR as a draft to prevent accidental merging
- Preserve the original PR information in the merge PR description
- Skip draft PRs by default unless user explicitly requests them
- Handle merge conflicts gracefully with clear reporting
- The merge PR is for testing only - individual PRs should be merged separately
- Clean up the merge branch after testing is complete
- Consider PR dependencies when determining merge order

## Example

**Repository has 5 open PRs:**
- #101 - "Add user authentication" (ready, mergeable)
- #102 - "Fix database connection" (ready, mergeable)
- #103 - "Update UI components" (draft)
- #104 - "Add API endpoints" (ready, mergeable)
- #105 - "Refactor tests" (ready, has conflicts with #104)

**User selection:** Include #101, #102, #104, skip #103 (draft) and #105 (conflicts)

**Result:**
- Merge branch: `merge-all-prs-20241215-143022`
- Merge PR: `#200` (draft) - "Merge PR: Combined testing of all open PRs"
- Contains merged changes from: #101, #102, #104

**Testing workflow:**
1. Check out the merge PR branch
2. Run all tests
3. Manually test features
4. If issues found, identify the source PR
5. Delete the merge branch when done
