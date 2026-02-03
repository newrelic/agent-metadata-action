package sign

import (
	"context"
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
	// Set up test environment
	setupTestEnv(t)

	// Create mock signing service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/signing/test-agent/sign", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Verify body
		body, _ := io.ReadAll(r.Body)
		var request models.SigningRequest
		json.Unmarshal(body, &request)
		assert.Equal(t, "docker.io", request.Registry)
		assert.Equal(t, "newrelic/agents", request.Repository)
		assert.Equal(t, "1.2.3", request.Tag)
		assert.Equal(t, "sha256:abc123", request.Digest)

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Override signing URL for testing
	os.Setenv("SIGNING_SERVICE_URL", server.URL)
	defer os.Unsetenv("SIGNING_SERVICE_URL")

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := SignIndex(context.Background(), "docker.io/newrelic/agents", "sha256:abc123", "1.2.3", "test-token", "test-agent")

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "Starting manifest index signing")
	assert.Contains(t, outputStr, "Successfully signed manifest index")
}

func TestSignIndex_RetryOnFailure(t *testing.T) {
	// Set up test environment
	setupTestEnv(t)

	attemptCount := 0

	// Create mock signing service that fails first 2 attempts, succeeds on 3rd
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "temporary failure"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		}
	}))
	defer server.Close()

	// Override signing URL for testing
	os.Setenv("SIGNING_SERVICE_URL", server.URL)
	defer os.Unsetenv("SIGNING_SERVICE_URL")

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := SignIndex(context.Background(), "docker.io/newrelic/agents", "sha256:abc123", "1.2.3", "test-token", "test-agent")

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Equal(t, 3, attemptCount, "Should have made 3 attempts (2 failures + 1 success)")
	assert.Contains(t, outputStr, "Signing attempt 1 failed")
	assert.Contains(t, outputStr, "Signing attempt 2 failed")
	assert.Contains(t, outputStr, "Successfully signed manifest index")
}

func TestSignIndex_FailsAfterMaxRetries(t *testing.T) {
	// Set up test environment
	setupTestEnv(t)

	attemptCount := 0

	// Create mock signing service that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "persistent failure"}`))
	}))
	defer server.Close()

	// Override signing URL for testing
	os.Setenv("SIGNING_SERVICE_URL", server.URL)
	defer os.Unsetenv("SIGNING_SERVICE_URL")

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := SignIndex(context.Background(), "docker.io/newrelic/agents", "sha256:abc123", "1.2.3", "test-token", "test-agent")

	outputStr := getStdout()

	require.Error(t, err)
	assert.Equal(t, 3, attemptCount, "Should have made 3 attempts")
	assert.Contains(t, err.Error(), "failed Signing after 3 attempts")
	assert.Contains(t, outputStr, "Failed to sign manifest index")
}

func TestSignIndex_RegistryURLParsing(t *testing.T) {
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
				err := SignIndex(context.Background(), tt.registryURL, "sha256:abc123", "1.2.3", "test-token", "test-agent")

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
