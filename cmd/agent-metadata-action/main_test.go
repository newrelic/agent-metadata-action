validateConfigDirectorypackage main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/loader"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMetadataClient is a mock implementation for testing
type mockMetadataClient struct{}

func (m *mockMetadataClient) SendMetadata(ctx context.Context, agentType string, agentVersion string, metadata *models.AgentMetadata) error {
	// Mock implementation - does nothing, returns success
	return nil
}

type mockFailingMetadataClient struct{}

func (m *mockFailingMetadataClient) SendMetadata(ctx context.Context, agentType string, agentVersion string, metadata *models.AgentMetadata) error {
	return assert.AnError
}

type mockSelectiveFailClient struct {
	callCount *int
}

func (m *mockSelectiveFailClient) SendMetadata(ctx context.Context, agentType string, agentVersion string, metadata *models.AgentMetadata) error {
	*m.callCount++
	if *m.callCount == 1 {
		return assert.AnError
	}
	return nil
}

// createSuccessfulUploadResult creates a mock successful upload result
func createSuccessfulUploadResult(name, digest, tag string) models.ArtifactUploadResult {
	return models.ArtifactUploadResult{
		Name:     name,
		Path:     "./dist/agent.tar.gz",
		OS:       "linux",
		Arch:     "amd64",
		Format:   "tar+gzip",
		Digest:   digest,
		Size:     1024,
		Tag:      tag,
		Uploaded: true,
		Signed:   false,
	}
}

// createFailedUploadResult creates a mock failed upload result
func createFailedUploadResult(name string) models.ArtifactUploadResult {
	return models.ArtifactUploadResult{
		Name:     name,
		Path:     "./dist/agent.tar.gz",
		OS:       "windows",
		Arch:     "amd64",
		Format:   "zip",
		Digest:   "",
		Size:     0,
		Tag:      "",
		Uploaded: false,
		Error:    "upload failed",
		Signed:   false,
	}
}

func TestMain_AgentRepoFlow(t *testing.T) {
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
	t.Setenv("INPUT_OCI_REGISTRY", "") // Disable OCI for this test

	getStdout, getStderr := testutil.CaptureOutput(t)

	// Method under test
	main()

	outputStr := getStdout()
	stderrStr := getStderr()

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

func TestMain_DocsFlow(t *testing.T) {
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
	github.GetChangedMDXFilesFunc = func(ctx context.Context) ([]string, error) {
		return []string{testMDXFile}, nil
	}
	defer func() {
		github.GetChangedMDXFilesFunc = originalFunc
	}()

	// Set environment variables - omit INPUT_AGENT_TYPE to trigger docs flow
	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")

	getStdout, getStderr := testutil.CaptureOutput(t)

	// Method under test
	main()

	outputStr := getStdout()
	_ = getStderr()

	// Verify docs scenario was triggered
	assert.Contains(t, outputStr, "Running documentation flow")
	assert.Contains(t, outputStr, "::notice::Loaded metadata for 1 out of 1 changed MDX files")

	// Verify output contains agent metadata
	assert.Contains(t, outputStr, "NRJavaAgent")
	assert.Contains(t, outputStr, "1.3.0")
	assert.Contains(t, outputStr, "New feature 1")
	assert.Contains(t, outputStr, "Bug fix 1")
	assert.Contains(t, outputStr, "Security fix 1")
}

func TestRun_InvalidEnvironment(t *testing.T) {
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

	// Method under test
	err := run(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace directory does not exist")
}

func TestValidateEnvironment(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		token         string
		setupFunc     func(t *testing.T) string // returns actual workspace path
		wantErr       bool
		errContains   string
		wantWorkspace string
		wantToken     string
	}{
		{
			name:        "missing workspace",
			workspace:   "",
			token:       "mock-token",
			wantErr:     true,
			errContains: "GITHUB_WORKSPACE is required",
		},
		{
			name:        "workspace directory does not exist",
			workspace:   "/nonexistent/path",
			token:       "mock-token",
			wantErr:     true,
			errContains: "workspace directory does not exist",
		},
		{
			name: "missing token",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			token:       "",
			wantErr:     true,
			errContains: "NEWRELIC_TOKEN is required",
		},
		{
			name: "success",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			token:     "test-token",
			wantErr:   false,
			wantToken: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := tt.workspace
			if tt.setupFunc != nil {
				workspace = tt.setupFunc(t)
			}

			t.Setenv("GITHUB_WORKSPACE", workspace)
			t.Setenv("NEWRELIC_TOKEN", tt.token)

			// Method under test
			gotWorkspace, gotToken, err := validateEnvironment(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, workspace, gotWorkspace)
				assert.Equal(t, tt.wantToken, gotToken)
			}
		})
	}
}

func TestRunAgentFlow_MissingFleetControl(t *testing.T) {
	workspace := t.TempDir()
	ctx := context.Background()
	mockClient := &mockMetadataClient{}

	// Method under test
	err := runAgentFlow(ctx, mockClient, workspace, "java", "1.0.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config directory does not exist")
}

func TestRunAgentFlow_InvalidConfigDefinitions(t *testing.T) {
	workspace := t.TempDir()
	fleetControlPath := filepath.Join(workspace, ".fleetControl")
	require.NoError(t, os.MkdirAll(fleetControlPath, 0755))

	configFile := filepath.Join(fleetControlPath, "configurationDefinitions.yml")
	require.NoError(t, os.WriteFile(configFile, []byte("invalid: yaml: ["), 0644))

	ctx := context.Background()
	mockClient := &mockMetadataClient{}

	// Method under test
	err := runAgentFlow(ctx, mockClient, workspace, "java", "1.0.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read configuration definitions")
}

func TestRunAgentFlow_SendMetadataFails(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	ctx := context.Background()
	mockClient := &mockFailingMetadataClient{}

	// method under test
	err = runAgentFlow(ctx, mockClient, workspace, "java", "1.0.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send metadata")
}

func TestRunDocsFlow_LoadMetadataError(t *testing.T) {
	originalFunc := github.GetChangedMDXFilesFunc
	github.GetChangedMDXFilesFunc = func(ctx context.Context) ([]string, error) {
		return nil, assert.AnError
	}
	defer func() {
		github.GetChangedMDXFilesFunc = originalFunc
	}()

	ctx := context.Background()
	mockClient := &mockMetadataClient{}

	// method under test
	err := runDocsFlow(ctx, mockClient)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load metadata from docs")
}

func TestRunDocsFlow_NoMetadataChanges(t *testing.T) {
	originalFunc := github.GetChangedMDXFilesFunc
	github.GetChangedMDXFilesFunc = func(ctx context.Context) ([]string, error) {
		return []string{}, nil
	}
	defer func() {
		github.GetChangedMDXFilesFunc = originalFunc
	}()

	getStdout, _ := testutil.CaptureOutput(t)

	ctx := context.Background()
	mockClient := &mockMetadataClient{}

	// method under test
	err := runDocsFlow(ctx, mockClient)

	assert.NoError(t, err)

	outputStr := getStdout()
	assert.Contains(t, outputStr, "No metadata changes detected")
}

func TestRunDocsFlow_PartialFailure(t *testing.T) {
	workspace := t.TempDir()
	mdxDir := filepath.Join(workspace, "src/content/docs/release-notes/agent-release-notes/java-release-notes")
	require.NoError(t, os.MkdirAll(mdxDir, 0755))

	testMDXFile1 := filepath.Join(mdxDir, "java-agent-130.mdx")
	mdxContent1 := `---
subject: Java agent
releaseDate: '2024-01-15'
version: 1.3.0
---

# Java Agent 1.3.0
`
	require.NoError(t, os.WriteFile(testMDXFile1, []byte(mdxContent1), 0644))

	testMDXFile2 := filepath.Join(mdxDir, "java-agent-131.mdx")
	mdxContent2 := `---
subject: Java agent
releaseDate: '2024-01-16'
version: 1.3.1
---

# Java Agent 1.3.1
`
	require.NoError(t, os.WriteFile(testMDXFile2, []byte(mdxContent2), 0644))

	originalFunc := github.GetChangedMDXFilesFunc
	github.GetChangedMDXFilesFunc = func(ctx context.Context) ([]string, error) {
		return []string{testMDXFile1, testMDXFile2}, nil
	}
	defer func() {
		github.GetChangedMDXFilesFunc = originalFunc
	}()

	t.Setenv("GITHUB_WORKSPACE", workspace)

	callCount := 0
	ctx := context.Background()
	mockClient := &mockSelectiveFailClient{callCount: &callCount}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := runDocsFlow(ctx, mockClient)

	assert.NoError(t, err)

	outputStr := getStdout()

	assert.Contains(t, outputStr, "Successfully sent 1 of 2 metadata entries")
	assert.Contains(t, outputStr, "::error::Failed to send metadata")
}

func TestRunAgentFlow_AgentControlDefinitionsError(t *testing.T) {
	workspace := t.TempDir()
	fleetControlPath := filepath.Join(workspace, ".fleetControl")
	require.NoError(t, os.MkdirAll(fleetControlPath, 0755))

	// Create valid configurationDefinitions.yml
	configFile := filepath.Join(fleetControlPath, "configurationDefinitions.yml")
	configContent := `configurationDefinitions:
  - name: test-config
    type: string
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	// Create invalid agentControlDefinitions.yml (invalid YAML)
	agentControlFile := filepath.Join(fleetControlPath, "agentControlDefinitions.yml")
	require.NoError(t, os.WriteFile(agentControlFile, []byte("invalid: yaml: ["), 0644))

	ctx := context.Background()
	mockClient := &mockMetadataClient{}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := runAgentFlow(ctx, mockClient, workspace, "java", "1.0.0")

	// Should succeed despite agentControlDefinitions error
	assert.NoError(t, err)

	// Verify warning was logged
	outputStr := getStdout()
	assert.Contains(t, outputStr, "::warn::Unable to load agent control definitions")
	assert.Contains(t, outputStr, "continuing without them")
}

func TestSendDocsMetadata(t *testing.T) {
	tests := []struct {
		name    string
		entry   loader.MetadataForDocs
		client  metadataClient
		wantErr bool
	}{
		{
			name: "success",
			entry: loader.MetadataForDocs{
				AgentType: "NRJavaAgent",
				AgentMetadataFromDocs: map[string]any{
					"version":     "1.2.3",
					"releaseDate": "2024-01-15",
				},
			},
			client:  &mockMetadataClient{},
			wantErr: false,
		},
		{
			name: "send error",
			entry: loader.MetadataForDocs{
				AgentType: "NRJavaAgent",
				AgentMetadataFromDocs: map[string]any{
					"version": "1.2.3",
				},
			},
			client:  &mockFailingMetadataClient{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// method under test
			err := sendDocsMetadata(ctx, tt.client, tt.entry)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunAgentFlow_OCIDisabled(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	t.Setenv("INPUT_OCI_REGISTRY", "")

	ctx := context.Background()
	mockClient := &mockMetadataClient{}

	// method under test
	err = runAgentFlow(ctx, mockClient, workspace, "java", "1.0.0")

	assert.NoError(t, err, "OCI should be skipped when registry is not configured")
}

func TestRunAgentFlow_OCIInvalidConfig(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	t.Setenv("INPUT_OCI_REGISTRY", "ghcr.io/newrelic/agents")
	t.Setenv("INPUT_BINARIES", "") // Empty binaries when registry is set = invalid

	ctx := context.Background()
	mockClient := &mockMetadataClient{}

	// method under test
	err = runAgentFlow(ctx, mockClient, workspace, "java", "1.0.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error loading OCI config")
}

func TestRunAgentFlow_OCIInvalidBinariesJSON(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	t.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
	t.Setenv("INPUT_BINARIES", "not valid json")

	ctx := context.Background()
	mockClient := &mockMetadataClient{}

	// method under test
	err = runAgentFlow(ctx, mockClient, workspace, "java", "1.0.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error loading OCI config")
}

func TestRunAgentFlow_OCIMissingBinaryFile(t *testing.T) {
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	t.Setenv("INPUT_OCI_REGISTRY", "ghcr.io/newrelic/agents")
	t.Setenv("INPUT_BINARIES", `[{"name":"test","path":"./nonexistent.tar.gz","os":"linux","arch":"amd64","format":"tar+gzip"}]`)

	ctx := context.Background()
	mockClient := &mockMetadataClient{}

	// method under test
	err = runAgentFlow(ctx, mockClient, workspace, "java", "1.0.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binary upload failed")
}

func TestRunAgentFlow_SigningSuccess_SingleArtifact(t *testing.T) {
	// Override metadata client with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() { createMetadataClientFunc = originalCreateClient }()

	// Mock OCI handler to return index digest
	originalOCIHandler := ociHandleUploadsFunc
	ociHandleUploadsFunc = func(ctx context.Context, cfg *models.OCIConfig, workspace, version string) (string, error) {
		return "sha256:index123", nil
	}
	defer func() { ociHandleUploadsFunc = originalOCIHandler }()

	// Create mock signing service
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Validate request structure
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/signing/agent-metadata-action/sign", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Validate request body - should be signing the manifest index
		body, _ := io.ReadAll(r.Body)
		var signingReq models.SigningRequest
		json.Unmarshal(body, &signingReq)
		assert.Equal(t, "docker.io", signingReq.Registry)
		assert.Equal(t, "newrelic/agents", signingReq.Repository)
		assert.Equal(t, "1.2.3", signingReq.Tag)
		assert.Equal(t, "sha256:index123", signingReq.Digest)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Setup workspace and environment variables
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	t.Setenv("INPUT_AGENT_TYPE", "java")
	t.Setenv("INPUT_VERSION", "1.2.3")
	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "test-token")
	t.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
	t.Setenv("INPUT_BINARIES", `[{"name":"linux-tar","path":"./dist/agent.tar.gz","os":"linux","arch":"amd64","format":"tar+gzip"}]`)
	t.Setenv("GITHUB_REPOSITORY", "newrelic/agent-metadata-action")
	t.Setenv("SIGNING_SERVICE_URL", server.URL)

	// Capture output
	getStdout, getStderr := testutil.CaptureOutput(t)

	// Execute
	ctx := context.Background()
	mockClient := &mockMetadataClient{}
	err = runAgentFlow(ctx, mockClient, workspace, "java", "1.2.3")

	// Verify success
	require.NoError(t, err)

	outputStr := getStdout()
	stderrStr := getStderr()

	// Verify signing requests
	assert.Equal(t, 1, requestCount, "Should have made 1 signing request")

	// Verify logging - now signing the manifest index
	assert.Contains(t, outputStr, "Starting manifest index signing")
	assert.Contains(t, outputStr, "Successfully signed manifest index")
	assert.NotContains(t, stderrStr, "::error::")
}

func TestRunAgentFlow_SigningDisabled_OCINotEnabled(t *testing.T) {
	// Override metadata client with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() { createMetadataClientFunc = originalCreateClient }()

	// Mock OCI handler should not be called since OCI is disabled
	originalOCIHandler := ociHandleUploadsFunc
	ociHandleUploadsFunc = func(ctx context.Context, cfg *models.OCIConfig, workspace, version string) (string, error) {
		t.Fatal("OCI handler should not be called when OCI is disabled")
		return "", nil
	}
	defer func() { ociHandleUploadsFunc = originalOCIHandler }()

	// Create mock signing service that should NOT be called
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Setup workspace and environment variables
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "test-token")
	t.Setenv("INPUT_OCI_REGISTRY", "") // OCI disabled
	t.Setenv("GITHUB_REPOSITORY", "newrelic/agent-metadata-action")
	t.Setenv("SIGNING_SERVICE_URL", server.URL)

	// Capture output
	getStdout, getStderr := testutil.CaptureOutput(t)

	// Execute
	ctx := context.Background()
	mockClient := &mockMetadataClient{}
	err = runAgentFlow(ctx, mockClient, workspace, "java", "1.2.3")

	// Verify success
	require.NoError(t, err)

	outputStr := getStdout()
	stderrStr := getStderr()

	// Verify NO signing requests were made
	assert.Equal(t, 0, requestCount, "Should have made 0 signing requests")
	assert.NotContains(t, outputStr, "Starting manifest index signing")
	assert.NotContains(t, stderrStr, "::error::")
}

func TestRunAgentFlow_SigningSkipped_AllUploadsFailed(t *testing.T) {
	// Override metadata client with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() { createMetadataClientFunc = originalCreateClient }()

	// Mock OCI handler to return error (fail-fast behavior)
	originalOCIHandler := ociHandleUploadsFunc
	ociHandleUploadsFunc = func(ctx context.Context, cfg *models.OCIConfig, workspace, version string) (string, error) {
		return "", fmt.Errorf("artifact upload failed for linux-tar: upload error")
	}
	defer func() { ociHandleUploadsFunc = originalOCIHandler }()

	// Create mock signing service that should NOT be called
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Setup workspace and environment variables
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "test-token")
	t.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
	t.Setenv("INPUT_BINARIES", `[{"name":"linux-tar","path":"./dist/agent.tar.gz","os":"linux","arch":"amd64","format":"tar+gzip"}]`)
	t.Setenv("GITHUB_REPOSITORY", "newrelic/agent-metadata-action")
	t.Setenv("SIGNING_SERVICE_URL", server.URL)

	// Capture output
	getStdout, _ := testutil.CaptureOutput(t)

	// Execute
	ctx := context.Background()
	mockClient := &mockMetadataClient{}
	err = runAgentFlow(ctx, mockClient, workspace, "java", "1.2.3")

	// Verify error (fail-fast behavior)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "binary upload failed")

	outputStr := getStdout()

	// Verify NO signing requests were made
	assert.Equal(t, 0, requestCount, "Should have made 0 signing requests")
	assert.NotContains(t, outputStr, "Starting manifest index signing")
}

func TestRunAgentFlow_SigningError_ServiceFailure(t *testing.T) {
	// Override metadata client with mock
	originalCreateClient := createMetadataClientFunc
	createMetadataClientFunc = func(baseURL, token string) metadataClient {
		return &mockMetadataClient{}
	}
	defer func() { createMetadataClientFunc = originalCreateClient }()

	// Mock OCI handler to return index digest
	originalOCIHandler := ociHandleUploadsFunc
	ociHandleUploadsFunc = func(ctx context.Context, cfg *models.OCIConfig, workspace, version string) (string, error) {
		return "sha256:index123", nil
	}
	defer func() { ociHandleUploadsFunc = originalOCIHandler }()

	// Create mock signing service that always returns 500
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	// Setup workspace and environment variables
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	t.Setenv("GITHUB_WORKSPACE", workspace)
	t.Setenv("NEWRELIC_TOKEN", "test-token")
	t.Setenv("INPUT_OCI_REGISTRY", "docker.io/newrelic/agents")
	t.Setenv("INPUT_BINARIES", `[{"name":"linux-tar","path":"./dist/agent.tar.gz","os":"linux","arch":"amd64","format":"tar+gzip"}]`)
	t.Setenv("GITHUB_REPOSITORY", "newrelic/agent-metadata-action")
	t.Setenv("SIGNING_SERVICE_URL", server.URL)

	// Capture output
	getStdout, _ := testutil.CaptureOutput(t)

	// Execute
	ctx := context.Background()
	mockClient := &mockMetadataClient{}
	err = runAgentFlow(ctx, mockClient, workspace, "java", "1.2.3")

	// Verify error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "artifact signing failed")

	outputStr := getStdout()

	// Verify retries occurred (3 attempts from retry package)
	assert.Equal(t, 3, requestCount, "Should have made 3 signing requests (retries)")
	assert.Contains(t, outputStr, "Signing attempt 1 failed")
	assert.Contains(t, outputStr, "Signing attempt 2 failed")
	assert.Contains(t, outputStr, "Failed to sign manifest index")
}
