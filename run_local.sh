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

# Create mock PR event to test MDX file parsing
cat > /tmp/pr-event.json <<EOF
{
  "pull_request": {
    "base": {"sha": "$(git rev-parse main)"},
    "head": {"sha": "$(git rev-parse HEAD)"}
  }
}
EOF

export GITHUB_EVENT_PATH="/tmp/pr-event.json"
unset GITHUB_WORKSPACE
export INPUT_AGENT_TYPE="myagent"
export INPUT_VERSION="1.3.0"

echo "GITHUB_EVENT_PATH: $GITHUB_EVENT_PATH"
echo "Testing MDX file parsing from changed files in PR"
echo ""

./agent-metadata-action

echo ""
echo "=========================================="
echo "All tests completed successfully!"
echo "=========================================="
