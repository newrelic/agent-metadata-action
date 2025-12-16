package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/models"

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
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Call run
	err = run()

	// Restore stdout/stderr and read captured output
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	outputStr := bufOut.String()
	stderrStr := bufErr.String()

	// Verify no error
	require.NoError(t, err)

	// Verify output
	assert.Contains(t, outputStr, "\"metadata\":")
	assert.NotContains(t, outputStr, "\"version\": null")
	assert.Contains(t, outputStr, "\"features\": null")
	assert.NotContains(t, outputStr, "\"configurationDefinitions\": null")
	assert.NotContains(t, outputStr, "\"agentControl\": null")

	// Stderr may contain debug messages but not errors
	if stderrStr != "" {
		assert.NotContains(t, stderrStr, "::error::")
		t.Logf("Stderr: %s", stderrStr)
	}
}

func TestRun_DocsFlow(t *testing.T) {
	// TODO: Re-enable when docs workflow is implemented
	t.Skip("Docs workflow is currently disabled - skipping test")

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

	workspace := filepath.Join(projectRoot, "integration-test", "docs-flow")

	// Mock GetChangedMDXFiles to return test MDX files
	testMDXFile := filepath.Join(workspace, "src/content/docs/release-notes/agent-release-notes/java-release-notes/java-agent-130.mdx")
	originalFunc := github.GetChangedMDXFilesFunc
	github.GetChangedMDXFilesFunc = func() ([]string, error) {
		return []string{testMDXFile}, nil
	}
	defer func() {
		github.GetChangedMDXFilesFunc = originalFunc
	}()

	// Set environment variables
	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Call run
	err = run()

	// Restore stdout/stderr and read captured output
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	outputStr := bufOut.String()
	stderrStr := bufErr.String()

	// Verify no error
	require.NoError(t, err)

	// Verify output
	assert.Contains(t, outputStr, "\"metadata\":")
	assert.NotContains(t, outputStr, "\"version\": null")
	assert.Contains(t, outputStr, "\"configurationDefinitions\": null")
	assert.Contains(t, outputStr, "\"agentControl\": null")

	// Stderr may have debug message about no PR context
	t.Logf("Stderr: %s", stderrStr)
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
