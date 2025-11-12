#!/bin/bash

# Set your repository (format: owner/repo)
export AGENT_REPO="newrelic-experimental/k8s-apm-agent-health-sidecar"

# Set your GitHub token
export GITHUB_TOKEN="YOUR_TOKEN_HERE"

# Set the branch you want to fetch from
export BRANCH="mvick/test-agent-gh-action"

# Build and run
go build -o agent-metadata-action
./agent-metadata-action
