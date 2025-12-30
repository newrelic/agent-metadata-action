package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstrumentationClient(t *testing.T) {
	baseURL := "https://api.example.com"
	token := "test-token-123"

	client := NewInstrumentationClient(baseURL, token)

	assert.NotNil(t, client)
	assert.Equal(t, baseURL, client.baseURL)
	assert.Equal(t, token, client.token)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, client.httpClient.Timeout.Seconds(), float64(30))
}

func TestSendMetadata_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		assert.Equal(t, "POST", r.Method)

		// Verify URL path
		assert.True(t, strings.HasPrefix(r.URL.Path, "/v1/agents/TestAgent/versions/"))

		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Verify body can be parsed
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var metadata models.AgentMetadata
		err = json.Unmarshal(body, &metadata)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", metadata.Metadata.Version)

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			Version: "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControl:             []models.AgentControl{},
	}

	getStdout, getStderr := testutil.CaptureOutput(t)

	err := client.SendMetadata(context.Background(), "java", metadata)

	outputStr := getStdout()
	stderrStr := getStderr()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "Sending metadata to instrumentation service")
	assert.Contains(t, outputStr, "Agent type: java")
	assert.Contains(t, outputStr, "Agent version: 1.2.3")
	assert.Contains(t, outputStr, "HTTP status code: 200")
	assert.Contains(t, outputStr, "Metadata successfully submitted")

	assert.NotContains(t, stderrStr, "::error::")
}

func TestSendMetadata_NilMetadata(t *testing.T) {
	client := NewInstrumentationClient("https://api.example.com", "token")

	getStdout, _ := testutil.CaptureOutput(t)

	err := client.SendMetadata(context.Background(), "java", nil)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata is required")
	assert.Contains(t, outputStr, "Metadata is required but was nil")
}

func TestSendMetadata_EmptyAgentType(t *testing.T) {
	client := NewInstrumentationClient("https://api.example.com", "token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			Version: "1.2.3",
		},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	err := client.SendMetadata(context.Background(), "", metadata)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent type is required")
	assert.Contains(t, outputStr, "Agent type is required but was empty")
}

func TestSendMetadata_EmptyAgentVersion(t *testing.T) {
	client := NewInstrumentationClient("https://api.example.com", "token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			Version: "", // Empty version
		},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	err := client.SendMetadata(context.Background(), "java", metadata)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent version is required")
	assert.Contains(t, outputStr, "Agent version is required but was empty")
}

func TestSendMetadata_HTTPErrors(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedInErr string
		expectedInLog string
	}{
		{
			name:          "internal server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"error": "Internal server error"}`,
			expectedInErr: "metadata submission failed with status 500",
			expectedInLog: "HTTP status code: 500",
		},
		{
			name:          "bad request",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error": "Invalid metadata format"}`,
			expectedInErr: "metadata submission failed with status 400",
			expectedInLog: "Metadata submission failed with status 400",
		},
		{
			name:          "unauthorized",
			statusCode:    http.StatusUnauthorized,
			responseBody:  `{"error": "Invalid token"}`,
			expectedInErr: "metadata submission failed with status 401",
			expectedInLog: "HTTP status code: 401",
		},
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			responseBody:  `{"error": "Endpoint not found"}`,
			expectedInErr: "metadata submission failed with status 404",
			expectedInLog: "HTTP status code: 404",
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

			client := NewInstrumentationClient(server.URL, "test-token")

			metadata := &models.AgentMetadata{
				Metadata: models.Metadata{
					Version: "1.2.3",
				},
				ConfigurationDefinitions: []models.ConfigurationDefinition{},
				AgentControl:             []models.AgentControl{},
			}

			getStdout, _ := testutil.CaptureOutput(t)

			err := client.SendMetadata(context.Background(), "java", metadata)

			outputStr := getStdout()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedInErr)
			assert.Contains(t, outputStr, tt.expectedInLog)
		})
	}
}

func TestSendMetadata_LargeResponseBodyTruncation(t *testing.T) {
	// Create test server that returns large error response
	largeResponse := strings.Repeat("error message ", 100) // Over 500 chars
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(largeResponse))
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			Version: "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControl:             []models.AgentControl{},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	err := client.SendMetadata(context.Background(), "java", metadata)

	outputStr := getStdout()

	require.Error(t, err)
	// Should contain truncation notice
	assert.Contains(t, outputStr, "(truncated)")
}

func TestSendMetadata_NetworkError(t *testing.T) {
	// Use an invalid URL that will cause network error
	client := NewInstrumentationClient("http://invalid-host-that-does-not-exist.local", "token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			Version: "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControl:             []models.AgentControl{},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	err := client.SendMetadata(context.Background(), "java", metadata)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send metadata")
	assert.Contains(t, outputStr, "HTTP request failed")
}

func TestSendMetadata_ContextCancellation(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if context is cancelled
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			Version: "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControl:             []models.AgentControl{},
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	getStdout, _ := testutil.CaptureOutput(t)

	err := client.SendMetadata(ctx, "java", metadata)

	_ = getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send metadata")
}

func TestSendMetadata_SuccessWithResponseBody(t *testing.T) {
	// Create test server that returns response body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": "12345", "message": "Created successfully"}`))
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			Version: "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControl:             []models.AgentControl{},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	err := client.SendMetadata(context.Background(), "java", metadata)

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "HTTP status code: 201")
	assert.Contains(t, outputStr, "Success response")
	assert.Contains(t, outputStr, "Created successfully")
}

func TestSendMetadata_WithConfigurationDefinitionsAndAgentControl(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify body structure
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var metadata models.AgentMetadata
		err = json.Unmarshal(body, &metadata)
		require.NoError(t, err)

		// Verify data is present
		assert.Len(t, metadata.ConfigurationDefinitions, 2)
		assert.Len(t, metadata.AgentControl, 1)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			Version: "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{
			{
				Version:     "1.0.0",
				Platform:    "linux",
				Description: "Config 1",
				Type:        "string",
				Format:      "text",
				Schema:      "schema1",
			},
			{
				Version:     "1.0.0",
				Platform:    "windows",
				Description: "Config 2",
				Type:        "string",
				Format:      "text",
				Schema:      "schema2",
			},
		},
		AgentControl: []models.AgentControl{
			{
				Platform: "all",
				Content:  "base64content",
			},
		},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	err := client.SendMetadata(context.Background(), "java", metadata)

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "Configuration definitions count: 2")
	assert.Contains(t, outputStr, "Agent control entries: 1")
}
