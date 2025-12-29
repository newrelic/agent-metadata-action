package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMetadataClient is a mock implementation for testing
type mockMetadataClient struct{}

func (m *mockMetadataClient) SendMetadata(ctx context.Context, agentType string, metadata *models.AgentMetadata) error {
	// Mock implementation - does nothing, returns success
	return nil
}

func TestRun_AgentRepoFlow(t *testing.T) {
	// Override client creation with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() {
		createMetadataClientFunc = originalCreateClient
	}()
	// Get project root
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	// Set environment variables
	t.Setenv("INPUT_AGENT_TYPE", "java")
	t.Setenv("INPUT_VERSION", "1.2.3")
	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")

	// Capture stdout and stderr
	getStdout, getStderr := testutil.CaptureOutput(t)

	// Call run
	err = run()

	// Retrieve captured output
	outputStr := getStdout()
	stderrStr := getStderr()

	// Verify no error
	require.NoError(t, err)

	// Verify output
	assert.Contains(t, outputStr, "\"metadata\":")
	assert.Contains(t, outputStr, "\"version\": \"1.2.3\"")
	assert.NotContains(t, outputStr, "\"configurationDefinitions\": null")
	assert.NotContains(t, outputStr, "\"agentControl\": null")

	// Stderr may contain debug messages but not errors
	if stderrStr != "" {
		assert.NotContains(t, stderrStr, "::error::")
		t.Logf("Stderr: %s", stderrStr)
	}
}

func TestRun_DocsFlow(t *testing.T) {
	// Override client creation with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() {
		createMetadataClientFunc = originalCreateClient
	}()

	// Create temporary workspace with test MDX files
	workspace := t.TempDir()
	mdxDir := filepath.Join(workspace, "src/content/docs/release-notes/agent-release-notes/java-release-notes")
	require.NoError(t, os.MkdirAll(mdxDir, 0755))

	// Create test MDX file with frontmatter
	testMDXFile := filepath.Join(mdxDir, "java-agent-130.mdx")
	mdxContent := `---
subject: Java agent
releaseDate: '2024-01-15'
version: 1.3.0
features: ["New feature 1", "New feature 2"]
bugs: ["Bug fix 1"]
security: ["Security fix 1"]
deprecations: ["Deprecated API"]
supportedOperatingSystems: ["linux", "windows", "macos"]
eol: '2025-12-31'
---

# Java Agent 1.3.0

Release notes content here.
`
	require.NoError(t, os.WriteFile(testMDXFile, []byte(mdxContent), 0644))

	// Mock GetChangedMDXFiles to return test MDX files
	originalFunc := github.GetChangedMDXFilesFunc
	github.GetChangedMDXFilesFunc = func() ([]string, error) {
		return []string{testMDXFile}, nil
	}
	defer func() {
		github.GetChangedMDXFilesFunc = originalFunc
	}()

	// Set environment variables - omit INPUT_AGENT_TYPE to trigger docs flow
	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")

	getStdout, getStderr := testutil.CaptureOutput(t)

	err := run()

	outputStr := getStdout()
	stderrStr := getStderr()

	// Verify no error
	require.NoError(t, err)

	// Verify docs scenario was triggered
	assert.Contains(t, outputStr, "Docs scenario")
	assert.Contains(t, stderrStr, "::notice::Loaded metadata for 1 out of 1 changed MDX files")

	// Verify output contains agent metadata
	assert.Contains(t, outputStr, "JavaAgent")
	assert.Contains(t, outputStr, "1.3.0")
	assert.Contains(t, outputStr, "New feature 1")
	assert.Contains(t, outputStr, "Bug fix 1")
	assert.Contains(t, outputStr, "Security fix 1")
}

func TestRun_InvalidWorkspace(t *testing.T) {
	// Override client creation with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() {
		createMetadataClientFunc = originalCreateClient
	}()

	// Set invalid workspace
	t.Setenv("INPUT_AGENT_TYPE", "java")
	t.Setenv("INPUT_VERSION", "1.0.0")
	t.Setenv("GITHUB_WORKSPACE", "/nonexistent/path")
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")

	err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace directory does not exist")
}
