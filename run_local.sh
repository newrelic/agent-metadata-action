#!/bin/bash

set -e

# Build the action
echo "Building agent-metadata-action..."
go build -o agent-metadata-action ./cmd/agent-metadata-action

echo ""
echo "=========================================="
echo "Test 1: Agent Repo Example"
echo "=========================================="
unset GITHUB_EVENT_PATH
export GITHUB_WORKSPACE="$(pwd)/integration-test/agent-flow"
export INPUT_AGENT_TYPE="myagent"
export INPUT_VERSION="1.2.3"

echo "GITHUB_WORKSPACE: $GITHUB_WORKSPACE"
echo ""

./agent-metadata-action

echo ""
echo "=========================================="
echo "Test 2: Docs Workflow Example (MDX Parsing)"
echo "=========================================="

# Set up a temporary git repo with integration-test files to simulate PR changes
ORIGINAL_DIR=$(pwd)
TEMP_WORKSPACE=$(mktemp -d)
echo "Creating temporary workspace: $TEMP_WORKSPACE"

# Initialize git repo in temp workspace
cd "$TEMP_WORKSPACE"
git init -q
git config user.email "test@example.com"
git config user.name "Test User"

# Create base commit (without MDX files - just directory structure)
mkdir -p src/content/docs/release-notes
git commit -q --allow-empty -m "Initial commit"
BASE_SHA=$(git rev-parse HEAD)

# Copy integration-test MDX files and commit them
cp -r "$ORIGINAL_DIR/integration-test/docs-flow/"* .
git add .
git commit -q -m "Add release notes"
HEAD_SHA=$(git rev-parse HEAD)

cd "$ORIGINAL_DIR"

# Create mock PR event
cat > /tmp/pr-event.json <<EOF
{
  "pull_request": {
    "base": {"sha": "$BASE_SHA"},
    "head": {"sha": "$HEAD_SHA"}
  }
}
EOF

export GITHUB_EVENT_PATH="/tmp/pr-event.json"
export GITHUB_WORKSPACE="$TEMP_WORKSPACE"
unset INPUT_AGENT_TYPE
unset INPUT_VERSION

echo "GITHUB_WORKSPACE: $GITHUB_WORKSPACE"
echo "GITHUB_EVENT_PATH: $GITHUB_EVENT_PATH"
echo "Testing MDX file parsing from changed files in PR"
echo ""

./agent-metadata-action

# Cleanup
rm -rf "$TEMP_WORKSPACE"

echo ""
echo "=========================================="
echo "All tests completed successfully!"
echo "=========================================="
