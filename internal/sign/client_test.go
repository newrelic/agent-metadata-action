package sign

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errorReader is a custom reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (e *errorReader) Close() error {
	return nil
}

// errorTransport is a custom transport that returns responses with error readers
type errorTransport struct{}

func (t *errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       &errorReader{},
		Header:     make(http.Header),
	}, nil
}

func TestSignArtifact_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		assert.Equal(t, "POST", r.Method)

		// Verify URL path
		assert.Equal(t, "/v1/signing/test-agent/sign", r.URL.Path)

		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Verify body can be parsed
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var request models.SigningRequest
		err = json.Unmarshal(body, &request)
		require.NoError(t, err)
		assert.Equal(t, "docker.io", request.Registry)
		assert.Equal(t, "newrelic/agents", request.Repository)
		assert.Equal(t, "v1.2.3", request.Tag)
		assert.Equal(t, "sha256:abc123", request.Digest)

		// Send success response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	request := &models.SigningRequest{
		Registry:   "docker.io",
		Repository: "newrelic/agents",
		Tag:        "v1.2.3",
		Digest:     "sha256:abc123",
	}

	getStdout, getStderr := testutil.CaptureOutput(t)

	// method under test
	err := client.SignArtifact(context.Background(), "test-agent", request)

	outputStr := getStdout()
	stderrStr := getStderr()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "Signing artifact")
	assert.Contains(t, outputStr, "Signing client ID: test-agent")
	assert.Contains(t, outputStr, "Registry: docker.io")
	assert.Contains(t, outputStr, "Repository: newrelic/agents")
	assert.Contains(t, outputStr, "HTTP status code: 200")
	assert.Contains(t, outputStr, "Artifact signed successfully")

	assert.NotContains(t, stderrStr, "::error::")
}

func TestSignArtifact_Created(t *testing.T) {
	// Create test server that returns 201 Created
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": "12345", "signed": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	request := &models.SigningRequest{
		Registry:   "docker.io",
		Repository: "newrelic/agents",
		Tag:        "v1.2.3",
		Digest:     "sha256:abc123",
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SignArtifact(context.Background(), "test-agent", request)

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "HTTP status code: 201")
	assert.Contains(t, outputStr, "Artifact signed successfully")
	assert.Contains(t, outputStr, "Success response")
}

func TestSignArtifact_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		clientId      string
		request       *models.SigningRequest
		expectedInErr string
		expectedInLog string
	}{
		{
			name:          "empty client ID",
			clientId:      "",
			request:       &models.SigningRequest{Registry: "docker.io", Repository: "test", Tag: "v1.0.0", Digest: "sha256:abc"},
			expectedInErr: "client ID is required",
			expectedInLog: "Signing client ID is required but was empty",
		},
		{
			name:          "nil request",
			clientId:      "test-repo",
			request:       nil,
			expectedInErr: "signing request is required",
			expectedInLog: "Signing request is required but was nil",
		},
		{
			name:          "empty registry",
			clientId:      "test-repo",
			request:       &models.SigningRequest{Registry: "", Repository: "test", Tag: "v1.0.0", Digest: "sha256:abc"},
			expectedInErr: "registry is required",
			expectedInLog: "Invalid signing request",
		},
		{
			name:          "empty repository",
			clientId:      "test-repo",
			request:       &models.SigningRequest{Registry: "docker.io", Repository: "", Tag: "v1.0.0", Digest: "sha256:abc"},
			expectedInErr: "repository is required",
			expectedInLog: "Invalid signing request",
		},
		{
			name:          "empty tag",
			clientId:      "test-repo",
			request:       &models.SigningRequest{Registry: "docker.io", Repository: "test", Tag: "", Digest: "sha256:abc"},
			expectedInErr: "tag is required",
			expectedInLog: "Invalid signing request",
		},
		{
			name:          "empty digest",
			clientId:      "test-repo",
			request:       &models.SigningRequest{Registry: "docker.io", Repository: "test", Tag: "v1.0.0", Digest: ""},
			expectedInErr: "digest is required",
			expectedInLog: "Invalid signing request",
		},
		{
			name:          "invalid digest format",
			clientId:      "test-repo",
			request:       &models.SigningRequest{Registry: "docker.io", Repository: "test", Tag: "v1.0.0", Digest: "abc123"},
			expectedInErr: "digest must be in format sha256",
			expectedInLog: "Invalid signing request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("https://api.example.com", "token")
			getStdout, _ := testutil.CaptureOutput(t)

			// method under test
			err := client.SignArtifact(context.Background(), tt.clientId, tt.request)

			outputStr := getStdout()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedInErr)
			assert.Contains(t, outputStr, tt.expectedInLog)
		})
	}
}

func TestSignArtifact_HTTPErrors(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedInErr string
		expectedInLog string
	}{
		{
			name:          "bad request",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error": "Invalid request format"}`,
			expectedInErr: "artifact signing failed with status 400",
			expectedInLog: "Artifact signing failed with status 400",
		},
		{
			name:          "unauthorized",
			statusCode:    http.StatusUnauthorized,
			responseBody:  `{"error": "Invalid token"}`,
			expectedInErr: "artifact signing failed with status 401",
			expectedInLog: "HTTP status code: 401",
		},
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			responseBody:  `{"error": "Endpoint not found"}`,
			expectedInErr: "artifact signing failed with status 404",
			expectedInLog: "HTTP status code: 404",
		},
		{
			name:          "internal server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"error": "Internal server error"}`,
			expectedInErr: "artifact signing failed with status 500",
			expectedInLog: "HTTP status code: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server that returns error
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")

			request := &models.SigningRequest{
				Registry:   "docker.io",
				Repository: "newrelic/agents",
				Tag:        "v1.2.3",
				Digest:     "sha256:abc123",
			}

			getStdout, _ := testutil.CaptureOutput(t)

			// method under test
			err := client.SignArtifact(context.Background(), "newrelic/test-agent", request)

			outputStr := getStdout()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedInErr)
			assert.Contains(t, outputStr, tt.expectedInLog)
		})
	}
}

func TestSignArtifact_NetworkError(t *testing.T) {
	// Use localhost with a port that's guaranteed not to be listening
	client := NewClient("http://127.0.0.1:1", "token")

	request := &models.SigningRequest{
		Registry:   "docker.io",
		Repository: "newrelic/agents",
		Tag:        "v1.2.3",
		Digest:     "sha256:abc123",
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SignArtifact(context.Background(), "test-agent", request)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send signing request")
	assert.Contains(t, outputStr, "HTTP request failed")
}

func TestSignArtifact_ContextCancellation(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if context is cancelled
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	request := &models.SigningRequest{
		Registry:   "docker.io",
		Repository: "newrelic/agents",
		Tag:        "v1.2.3",
		Digest:     "sha256:abc123",
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SignArtifact(ctx, "test-agent", request)

	_ = getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send signing request")
}

func TestSignArtifact_ResponseBodyReadError(t *testing.T) {
	// Create test server with custom response that fails to read
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client with custom HTTP client that returns error reader
	client := &Client{
		baseURL: server.URL,
		token:   "test-token",
		httpClient: &http.Client{
			Transport: &errorTransport{},
		},
	}

	request := &models.SigningRequest{
		Registry:   "docker.io",
		Repository: "newrelic/agents",
		Tag:        "v1.2.3",
		Digest:     "sha256:abc123",
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SignArtifact(context.Background(), "test-agent", request)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read response")
	assert.Contains(t, outputStr, "Failed to read response body")
}

func TestParseRegistryURL_Success(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedRegistry string
		expectedRepo     string
	}{
		{
			name:             "docker.io with path",
			input:            "docker.io/newrelic/agents",
			expectedRegistry: "docker.io",
			expectedRepo:     "newrelic/agents",
		},
		{
			name:             "localhost with port",
			input:            "localhost:5000/test-agents",
			expectedRegistry: "localhost:5000",
			expectedRepo:     "test-agents",
		},
		{
			name:             "ghcr.io with multiple path components",
			input:            "ghcr.io/org/product/agents",
			expectedRegistry: "ghcr.io",
			expectedRepo:     "org/product/agents",
		},
		{
			name:             "with https scheme",
			input:            "https://registry.example.com/path/to/repo",
			expectedRegistry: "registry.example.com",
			expectedRepo:     "path/to/repo",
		},
		{
			name:             "with http scheme",
			input:            "http://localhost:5000/test",
			expectedRegistry: "localhost:5000",
			expectedRepo:     "test",
		},
		{
			name:             "trailing slash",
			input:            "docker.io/newrelic/",
			expectedRegistry: "docker.io",
			expectedRepo:     "newrelic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, repo, err := ParseRegistryURL(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedRegistry, registry)
			assert.Equal(t, tt.expectedRepo, repo)
		})
	}
}

func TestParseRegistryURL_Errors(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name:          "empty string",
			input:         "",
			expectedError: "registry URL cannot be empty",
		},
		{
			name:          "no repository path",
			input:         "docker.io",
			expectedError: "must contain both domain and repository path",
		},
		{
			name:          "only slash",
			input:         "docker.io/",
			expectedError: "repository path cannot be empty",
		},
		{
			name:          "with scheme but no path",
			input:         "https://registry.example.com",
			expectedError: "repository path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseRegistryURL(tt.input)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
