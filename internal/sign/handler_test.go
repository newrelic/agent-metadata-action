package sign

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignArtifacts_Success_SingleArtifact(t *testing.T) {
	// Set up test environment
	setupTestEnv(t)

	// Create mock signing service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/signing/newrelic/test-agent/sign", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Verify body
		body, _ := io.ReadAll(r.Body)
		var request models.SigningRequest
		json.Unmarshal(body, &request)
		assert.Equal(t, "docker.io", request.Registry)
		assert.Equal(t, "newrelic/agents", request.Repository)
		assert.Equal(t, "v1.2.3-linux-amd64", request.Tag)
		assert.Equal(t, "sha256:abc123", request.Digest)

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Override signing URL for testing
	os.Setenv("SIGNING_SERVICE_URL", server.URL)
	defer os.Unsetenv("SIGNING_SERVICE_URL")

	results := []models.ArtifactUploadResult{
		{
			Name:     "linux-tar",
			Path:     "./dist/agent.tar.gz",
			OS:       "linux",
			Arch:     "amd64",
			Format:   "tar+gzip",
			Digest:   "sha256:abc123",
			Size:     1024,
			Tag:      "v1.2.3-linux-amd64",
			Uploaded: true,
		},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := SignArtifacts(results, "docker.io/newrelic/agents", "test-token", "newrelic/test-agent", "v1.2.3")

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "Starting artifact signing for 1 artifacts")
	assert.Contains(t, outputStr, "Successfully signed artifact linux-tar")
	assert.Contains(t, outputStr, "Artifact signing complete: 1/1 signed successfully")

	// Verify result was updated
	assert.True(t, results[0].Signed)
	assert.Empty(t, results[0].SigningError)
}

func TestSignArtifacts_SkipsFailedUploads(t *testing.T) {
	// Set up test environment
	setupTestEnv(t)

	requestCount := 0

	// Create mock signing service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Override signing URL for testing
	os.Setenv("SIGNING_SERVICE_URL", server.URL)
	defer os.Unsetenv("SIGNING_SERVICE_URL")

	results := []models.ArtifactUploadResult{
		{
			Name:     "linux-tar",
			Digest:   "sha256:abc123",
			Tag:      "v1.2.3-linux-amd64",
			Uploaded: true,
		},
		{
			Name:     "windows-zip",
			Digest:   "",
			Tag:      "",
			Uploaded: false, // This one failed to upload
			Error:    "upload failed",
		},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := SignArtifacts(results, "docker.io/newrelic/agents", "test-token", "newrelic/test-agent", "v1.2.3")

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Equal(t, 1, requestCount, "Should have made 1 signing request (skipped failed upload)")
	assert.Contains(t, outputStr, "Skipping signing for windows-zip - upload failed")
	assert.Contains(t, outputStr, "Successfully signed artifact linux-tar")

	// Verify signing status
	assert.True(t, results[0].Signed)
	assert.False(t, results[1].Signed) // Skipped, not signed
}


func TestSignArtifacts_RegistryURLParsing(t *testing.T) {
	// Set up test environment
	setupTestEnv(t)

	tests := []struct {
		name             string
		registryURL      string
		expectedRegistry string
		expectedRepo     string
		expectError      bool
		errorContains    string
	}{
		{
			name:             "docker.io with path",
			registryURL:      "docker.io/newrelic/agents",
			expectedRegistry: "docker.io",
			expectedRepo:     "newrelic/agents",
			expectError:      false,
		},
		{
			name:             "localhost with port",
			registryURL:      "localhost:5000/test-agents",
			expectedRegistry: "localhost:5000",
			expectedRepo:     "test-agents",
			expectError:      false,
		},
		{
			name:             "ghcr.io with multiple path components",
			registryURL:      "ghcr.io/org/product/agents",
			expectedRegistry: "ghcr.io",
			expectedRepo:     "org/product/agents",
			expectError:      false,
		},
		{
			name:          "invalid - no repository path",
			registryURL:   "docker.io",
			expectError:   true,
			errorContains: "must contain both domain and repository path",
		},
		{
			name:          "invalid - empty string",
			registryURL:   "",
			expectError:   true,
			errorContains: "registry URL cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectError {
				// Only test error cases for URL parsing
				results := []models.ArtifactUploadResult{
					{Name: "test", Digest: "sha256:abc123", Uploaded: true},
				}

				err := SignArtifacts(results, tt.registryURL, "test-token", "newrelic/test-agent", "v1.2.3")

				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				// For success cases, just verify URL parsing works
				// (full integration tested in other tests)
				registry, repo, err := ParseRegistryURL(tt.registryURL)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRegistry, registry)
				assert.Equal(t, tt.expectedRepo, repo)
			}
		})
	}
}

// setupTestEnv sets up test environment variables
func setupTestEnv(t *testing.T) {
	t.Helper()

	// Set GITHUB_REPOSITORY to allow URL override for testing
	os.Setenv("GITHUB_REPOSITORY", "newrelic/agent-metadata-action")

	// Clean up after test
	t.Cleanup(func() {
		os.Unsetenv("GITHUB_REPOSITORY")
		os.Unsetenv("SIGNING_SERVICE_URL")
	})
}
