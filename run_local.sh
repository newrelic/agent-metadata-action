#!/bin/bash

# Build the action
echo "Building agent-metadata-action..."
go build -o agent-metadata-action ./cmd/agent-metadata-action

if [ $? -ne 0 ]; then
    echo "::error::Build failed"
    exit 1
fi

# Set GITHUB_WORKSPACE to current directory (simulates GitHub Actions environment)
export GITHUB_WORKSPACE="$(pwd)"

# Run the action (it will read from the current directory)
echo "Running agent-metadata-action in: $GITHUB_WORKSPACE"
./agent-metadata-action
