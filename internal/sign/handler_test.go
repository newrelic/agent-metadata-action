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

func TestSignIndex_Success(t *testing.T) {
	setupTestEnv(t)

	requestCount := 0

	// Create mock signing service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/signing/test-agent/sign", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		body, _ := io.ReadAll(r.Body)
		var request models.SigningRequest
		json.Unmarshal(body, &request)
		assert.Equal(t, "docker.io", request.Registry)
		assert.Equal(t, "newrelic/agents", request.Repository)
		assert.Equal(t, "v1.2.3", request.Tag)          // Index tag (version)
		assert.Equal(t, "sha256:index123", request.Digest) // Index digest

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Override signing URL for testing
	os.Setenv("SIGNING_SERVICE_URL", server.URL)
	defer os.Unsetenv("SIGNING_SERVICE_URL")

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := SignIndex("docker.io/newrelic/agents", "sha256:index123", "v1.2.3", "test-token", "test-agent")

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Equal(t, 1, requestCount, "Should have made 1 signing request")
	assert.Contains(t, outputStr, "Signing manifest index with tag 'v1.2.3'")
	assert.Contains(t, outputStr, "Successfully signed manifest index")
}

func TestSignIndex_EmptyDigest(t *testing.T) {
	setupTestEnv(t)

	getStdout, _ := testutil.CaptureOutput(t)

	err := SignIndex("docker.io/newrelic/agents", "", "v1.2.3", "test-token", "test-agent")

	_ = getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "index digest is required for signing")
}

func TestSignIndex_InvalidRegistryURL(t *testing.T) {
	// Set up test environment
	setupTestEnv(t)

	tests := []struct {
		name          string
		registryURL   string
		errorContains string
	}{
		{
			name:          "empty registry URL",
			registryURL:   "",
			errorContains: "registry URL cannot be empty",
		},
		{
			name:          "no repository path",
			registryURL:   "docker.io",
			errorContains: "must contain both domain and repository path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SignIndex(tt.registryURL, "sha256:index123", "v1.2.3", "test-token", "test-agent")

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
		})
	}
}

func TestSignIndex_RetryOnFailure(t *testing.T) {
	setupTestEnv(t)

	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "temporary error"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		}
	}))
	defer server.Close()

	os.Setenv("SIGNING_SERVICE_URL", server.URL)
	defer os.Unsetenv("SIGNING_SERVICE_URL")

	getStdout, _ := testutil.CaptureOutput(t)

	err := SignIndex("docker.io/newrelic/agents", "sha256:index123", "v1.2.3", "test-token", "test-agent")

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Equal(t, 3, requestCount, "Should have made 3 signing requests (2 retries)")
	assert.Contains(t, outputStr, "Signing attempt 1 failed")
	assert.Contains(t, outputStr, "Signing attempt 2 failed")
	assert.Contains(t, outputStr, "Successfully signed manifest index")
}

func TestSignIndex_FailureAfterAllRetries(t *testing.T) {
	// Set up test environment
	setupTestEnv(t)

	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "persistent error"}`))
	}))
	defer server.Close()

	// Override signing URL for testing
	os.Setenv("SIGNING_SERVICE_URL", server.URL)
	defer os.Unsetenv("SIGNING_SERVICE_URL")

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := SignIndex("docker.io/newrelic/agents", "sha256:index123", "v1.2.3", "test-token", "test-agent")

	outputStr := getStdout()

	require.Error(t, err)
	assert.Equal(t, 3, requestCount, "Should have made 3 signing requests (max retries)")
	assert.Contains(t, err.Error(), "failed to sign manifest index after 3 attempts")
	assert.Contains(t, outputStr, "Signing attempt 1 failed")
	assert.Contains(t, outputStr, "Signing attempt 2 failed")
	assert.Contains(t, outputStr, "Failed to sign manifest index after 3 attempts")
}

func TestSignIndex_RegistryURLParsing(t *testing.T) {
	// Set up test environment
	setupTestEnv(t)

	// Create mock signing service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var request models.SigningRequest
		json.Unmarshal(body, &request)

		w.Header().Set("X-Registry", request.Registry)
		w.Header().Set("X-Repository", request.Repository)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	os.Setenv("SIGNING_SERVICE_URL", server.URL)
	defer os.Unsetenv("SIGNING_SERVICE_URL")

	tests := []struct {
		name             string
		registryURL      string
		expectedRegistry string
		expectedRepo     string
	}{
		{
			name:             "docker.io with path",
			registryURL:      "docker.io/newrelic/agents",
			expectedRegistry: "docker.io",
			expectedRepo:     "newrelic/agents",
		},
		{
			name:             "localhost with port",
			registryURL:      "localhost:5000/test-agents",
			expectedRegistry: "localhost:5000",
			expectedRepo:     "test-agents",
		},
		{
			name:             "ghcr.io with multiple path components",
			registryURL:      "ghcr.io/org/product/agents",
			expectedRegistry: "ghcr.io",
			expectedRepo:     "org/product/agents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, repo, err := ParseRegistryURL(tt.registryURL)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedRegistry, registry)
			assert.Equal(t, tt.expectedRepo, repo)
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
