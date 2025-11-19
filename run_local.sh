#!/bin/bash

set -e

# Build the action
echo "Building agent-metadata-action..."
go build -o agent-metadata-action ./cmd/agent-metadata-action

echo ""
echo "=========================================="
echo "Test 1: Agent Repo Example"
echo "Running from agent repo with GITHUB_REF (version parsed from git tag)"
echo "=========================================="
export GITHUB_WORKSPACE="$(pwd)"
export GITHUB_REF="refs/tags/v2.0.0"
export INPUT_AGENT_TYPE="myagent"
export INPUT_VERSION="1.2.3"
unset INPUT_FEATURES
unset INPUT_BUGS
unset INPUT_SECURITY
unset INPUT_DEPRECATIONS
unset INPUT_SUPPORTEDOPERATINGSYSTEMS
unset INPUT_EOL

echo "GITHUB_WORKSPACE: $GITHUB_WORKSPACE"
echo "GITHUB_REF: $GITHUB_REF (version will be parsed as 2.0.0)"
echo ""

./agent-metadata-action

echo ""
echo "=========================================="
echo "Test 2: Docs Workflow Example"
echo "Running from docs workflow with explicit inputs (no workspace)"
echo "=========================================="
unset GITHUB_WORKSPACE
export INPUT_AGENT_TYPE="myagent"
export INPUT_VERSION="1.2.3"
export INPUT_FEATURES="feature1,feature2"
export INPUT_BUGS="bug-123,bug-456"
export INPUT_SECURITY="CVE-2024-1234"
export INPUT_DEPRECATIONS="deprecated-feature1,deprecated-feature2"
export INPUT_SUPPORTEDOPERATINGSYSTEMS="linux,windows,darwin"
export INPUT_EOL="2025-12-31"
unset GITHUB_REF

echo "GITHUB_WORKSPACE: (not set - metadata-only mode)"
echo "INPUT_VERSION: $INPUT_VERSION"
echo "INPUT_FEATURES: $INPUT_FEATURES"
echo "INPUT_BUGS: $INPUT_BUGS"
echo "INPUT_SECURITY: $INPUT_SECURITY"
echo "INPUT_DEPRECATIONS: $INPUT_DEPRECATIONS"
echo "INPUT_SUPPORTEDOPERATINGSYSTEMS: $INPUT_SUPPORTEDOPERATINGSYSTEMS"
echo "INPUT_EOL: $INPUT_EOL"
echo ""

./agent-metadata-action

echo ""
echo "=========================================="
echo "All tests completed successfully!"
echo "=========================================="
