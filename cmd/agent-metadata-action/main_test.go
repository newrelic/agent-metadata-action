package main

import (
	"agent-metadata-action/internal/loader"
	"context"
	"path/filepath"
	"testing"

	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMetadataClient is a mock implementation for testing
type mockMetadataClient struct {
	sendError error // Optional error to return from SendMetadata
}

func (m *mockMetadataClient) SendMetadata(ctx context.Context, agentType string, agentVersion string, metadata *models.AgentMetadata) error {
	if m.sendError != nil {
		return m.sendError
	}
	return nil
}

func TestRun_AgentRepoFlowSuccess(t *testing.T) {
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
	t.Setenv("INPUT_AGENT_TYPE", "java-agent")
	t.Setenv("INPUT_VERSION", "1.2.3")
	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")

	// Capture stdout and stderr (with display for visibility)
	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

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

func TestRun_AgentRepoFlowSuccess_NoAgentControlFile(t *testing.T) {
	// Override client creation with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() {
		createMetadataClientFunc = originalCreateClient
	}()

	// Mock LoadAndEncodeAgentControl to return an error (simulating missing file)
	originalLoadFunc := loader.LoadAndEncodeAgentControlFunc
	loader.LoadAndEncodeAgentControlFunc = func(workspacePath string) ([]models.AgentControl, error) {
		return nil, assert.AnError
	}
	defer func() {
		loader.LoadAndEncodeAgentControlFunc = originalLoadFunc
	}()

	// Get project root
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	// Set environment variables
	t.Setenv("INPUT_AGENT_TYPE", "java-agent")
	t.Setenv("INPUT_VERSION", "1.2.3")
	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")

	// Capture stdout and stderr (with display for visibility)
	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

	// Call run
	err = run()

	// Retrieve captured output
	outputStr := getStdout()
	stderrStr := getStderr()

	// Verify no error - should succeed even without agent control
	require.NoError(t, err)

	// Verify output contains warning about missing agent control
	assert.Contains(t, outputStr, "::warn::Unable to load agent control file")

	// Verify output still contains configuration definitions
	assert.Contains(t, outputStr, "\"configurationDefinitions\":")
	assert.Contains(t, outputStr, "\"metadata\":")
	assert.Contains(t, outputStr, "\"version\": \"1.2.3\"")

	// Verify agent control is empty (null because the load function returned error)
	assert.Contains(t, outputStr, "\"agentControl\": null")

	// Verify success message
	assert.Contains(t, outputStr, "::notice::Successfully sent metadata for java-agent version 1.2.3")

	// Stderr may contain debug messages but not errors
	if stderrStr != "" {
		assert.NotContains(t, stderrStr, "::error::")
		t.Logf("Stderr: %s", stderrStr)
	}
}

func TestRun_DocsFlowSuccess(t *testing.T) {
	// Override client creation with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() {
		createMetadataClientFunc = originalCreateClient
	}()

	// Get project root and use integration test data
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "docs-flow")
	testMDXFile := filepath.Join(workspace, "src/content/docs/release-notes/agent-release-notes/java-release-notes/java-agent-130.mdx")

	// Mock GetChangedMDXFiles to return integration test MDX file
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

	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

	err = run()

	outputStr := getStdout()
	stderrStr := getStderr()

	// Verify no error
	require.NoError(t, err)

	// Verify docs scenario was triggered
	assert.Contains(t, outputStr, "Running documentation flow")
	assert.Contains(t, outputStr, "::notice::Loaded metadata for 1 out of 1 changed MDX files")

	// Verify output contains agent metadata from integration test file
	assert.Contains(t, outputStr, "java-agent")
	assert.Contains(t, outputStr, "1.3.0")
	assert.Contains(t, outputStr, "Component-based transaction naming")
	assert.Contains(t, outputStr, "ClassCastException setting record_sql: off")

	// Stderr should not contain errors
	if stderrStr != "" {
		assert.NotContains(t, stderrStr, "::error::")
		t.Logf("Stderr: %s", stderrStr)
	}
}

func TestRun_ValidationErrors(t *testing.T) {
	// Get project root for workspace path
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	validWorkspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	tests := []struct {
		name          string
		setupEnv      func(*testing.T)
		expectedError string
	}{
		{
			name: "missing workspace",
			setupEnv: func(t *testing.T) {
				t.Setenv("INPUT_AGENT_TYPE", "java-agent")
				t.Setenv("INPUT_VERSION", "1.0.0")
				t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")
				// GITHUB_WORKSPACE not set
			},
			expectedError: "GITHUB_WORKSPACE is required but not set",
		},
		{
			name: "invalid workspace path",
			setupEnv: func(t *testing.T) {
				t.Setenv("INPUT_AGENT_TYPE", "java-agent")
				t.Setenv("INPUT_VERSION", "1.0.0")
				t.Setenv("GITHUB_WORKSPACE", "/nonexistent/path")
				t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")
			},
			expectedError: "workspace directory does not exist",
		},
		{
			name: "missing token",
			setupEnv: func(t *testing.T) {
				t.Setenv("INPUT_AGENT_TYPE", "java-agent")
				t.Setenv("INPUT_VERSION", "1.2.3")
				t.Setenv("GITHUB_WORKSPACE", validWorkspace)
				// NEWRELIC_TOKEN not set
			},
			expectedError: "NEWRELIC_TOKEN is required but not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override client creation with mock
			originalCreateClient := createMetadataClientFunc
			createMetadataClientFunc = func(baseURL, token string) metadataClient {
				return &mockMetadataClient{}
			}
			defer func() {
				createMetadataClientFunc = originalCreateClient
			}()

			// Setup environment
			tt.setupEnv(t)

			// Execute
			err := run()

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestRun_ErrorScenarios(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	tests := []struct {
		name             string
		setupEnv         func(*testing.T)
		setupMocks       func() (restore func())
		expectedErr      string
		additionalChecks func(*testing.T, error)
	}{
		{
			name: "missing fleetControl directory",
			setupEnv: func(t *testing.T) {
				workspace := filepath.Join(projectRoot, "integration-test", "docs-flow")
				t.Setenv("INPUT_AGENT_TYPE", "java-agent")
				t.Setenv("INPUT_VERSION", "1.2.3")
				t.Setenv("GITHUB_WORKSPACE", workspace)
				t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")
			},
			setupMocks: func() func() {
				originalCreateClient := createMetadataClientFunc
				createMetadataClientFunc = func(baseURL, token string) metadataClient {
					return &mockMetadataClient{}
				}
				return func() {
					createMetadataClientFunc = originalCreateClient
				}
			},
			expectedErr: ".fleetControl directory does not exist",
		},
		{
			name: "read configuration definitions error",
			setupEnv: func(t *testing.T) {
				workspace := filepath.Join(projectRoot, "integration-test", "error-cases")
				t.Setenv("INPUT_AGENT_TYPE", "java-agent")
				t.Setenv("INPUT_VERSION", "1.2.3")
				t.Setenv("GITHUB_WORKSPACE", workspace)
				t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")
			},
			setupMocks: func() func() {
				originalCreateClient := createMetadataClientFunc
				createMetadataClientFunc = func(baseURL, token string) metadataClient {
					return &mockMetadataClient{}
				}
				return func() {
					createMetadataClientFunc = originalCreateClient
				}
			},
			expectedErr: "workspace directory does not exist",
		},
		{
			name: "send metadata error - agent flow",
			setupEnv: func(t *testing.T) {
				workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")
				t.Setenv("INPUT_AGENT_TYPE", "java-agent")
				t.Setenv("INPUT_VERSION", "1.2.3")
				t.Setenv("GITHUB_WORKSPACE", workspace)
				t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")
			},
			setupMocks: func() func() {
				originalCreateClient := createMetadataClientFunc
				createMetadataClientFunc = func(baseURL, token string) metadataClient {
					return &mockMetadataClient{
						sendError: assert.AnError,
					}
				}
				return func() {
					createMetadataClientFunc = originalCreateClient
				}
			},
			expectedErr: "failed to send metadata for",
			additionalChecks: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "java-agent")
			},
		},
		{
			name: "load metadata for docs error",
			setupEnv: func(t *testing.T) {
				workspace := filepath.Join(projectRoot, "integration-test", "docs-flow")
				t.Setenv("GITHUB_WORKSPACE", workspace)
				t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")
			},
			setupMocks: func() func() {
				originalCreateClient := createMetadataClientFunc
				createMetadataClientFunc = func(baseURL, token string) metadataClient {
					return &mockMetadataClient{}
				}
				originalFunc := github.GetChangedMDXFilesFunc
				github.GetChangedMDXFilesFunc = func() ([]string, error) {
					return []string{"/nonexistent/file.mdx"}, nil
				}
				return func() {
					createMetadataClientFunc = originalCreateClient
					github.GetChangedMDXFilesFunc = originalFunc
				}
			},
			expectedErr: "failed to load metadata from docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			tt.setupEnv(t)

			// Setup mocks
			restore := tt.setupMocks()
			defer restore()

			// Capture output
			getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

			// Execute
			err := run()

			// Retrieve captured output
			outputStr := getStdout()
			stderrStr := getStderr()

			if outputStr != "" {
				t.Logf("Stdout:\n%s", outputStr)
			}
			if stderrStr != "" {
				t.Logf("Stderr:\n%s", stderrStr)
			}

			// Assert
			require.Error(t, err)
			t.Logf("Error: %v", err)
			assert.Contains(t, err.Error(), tt.expectedErr)

			// Run additional checks if provided
			if tt.additionalChecks != nil {
				tt.additionalChecks(t, err)
			}
		})
	}
}

func TestRun_DocsFlowMixedResults(t *testing.T) {
	// Override client creation with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() {
		createMetadataClientFunc = originalCreateClient
	}()

	// Get project root and use integration test data
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "docs-flow")
	validMDXFile := filepath.Join(workspace, "src/content/docs/release-notes/agent-release-notes/java-release-notes/java-agent-130.mdx")
	invalidMDXFile := "/nonexistent/file.mdx"

	// Mock GetChangedMDXFiles to return both valid and invalid files
	originalFunc := github.GetChangedMDXFilesFunc
	github.GetChangedMDXFilesFunc = func() ([]string, error) {
		return []string{validMDXFile, invalidMDXFile}, nil
	}
	defer func() {
		github.GetChangedMDXFilesFunc = originalFunc
	}()

	// Set environment variables - omit INPUT_AGENT_TYPE to trigger docs flow
	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")

	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

	err = run()

	outputStr := getStdout()
	stderrStr := getStderr()

	if outputStr != "" {
		t.Logf("Stdout:\n%s", outputStr)
	}
	if stderrStr != "" {
		t.Logf("Stderr:\n%s", stderrStr)
	}

	// Verify no error - the action should succeed with warnings
	require.NoError(t, err)

	// Verify docs scenario was triggered
	assert.Contains(t, outputStr, "Running documentation flow")

	// Verify we got a warning about the failed file
	assert.Contains(t, outputStr, "::warn::Failed to parse MDX file")
	assert.Contains(t, outputStr, "/nonexistent/file.mdx")

	// Verify we successfully loaded metadata for 1 out of 2 files
	assert.Contains(t, outputStr, "Loaded metadata for 1 out of 2 changed MDX files")

	// Verify output contains valid agent metadata
	assert.Contains(t, outputStr, "java-agent")
	assert.Contains(t, outputStr, "1.3.0")
	assert.Contains(t, outputStr, "Component-based transaction naming")

	// Verify successful send message
	assert.Contains(t, outputStr, "::notice::Sent metadata for java-agent version")
}
