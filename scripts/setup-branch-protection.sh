#!/bin/bash

# Script to set up branch protection rules for the main branch
# Usage: ./scripts/setup-branch-protection.sh

set -e

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed"
    echo "Install from: https://cli.github.com/"
    exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Get repository info
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
if [ -z "$REPO" ]; then
    echo "Error: Could not determine repository. Make sure you're authenticated with gh."
    exit 1
fi

echo "Setting up branch protection for: $REPO"
echo "Branch: main"

# Set branch protection rules
gh api \
  --method PUT \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  /repos/$REPO/branches/main/protection \
  --input - <<EOF
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "Lint",
      "Test (1.21.x)",
      "Test (1.22.x)",
      "Build",
      "Security Scan"
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 0
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true,
  "lock_branch": false,
  "allow_fork_syncing": true
}
EOF

echo "Branch protection rules set successfully!"
echo ""
echo "Protection enabled:"
echo "✓ Status checks must pass: Lint, Test (1.21.x), Test (1.22.x), Build, Security Scan"
echo "✓ Branches must be up to date before merging"
echo "✓ Dismiss stale reviews on new commits"
echo "✓ Force pushes disabled"
echo "✓ Branch deletion disabled"
echo "✓ Require conversation resolution"