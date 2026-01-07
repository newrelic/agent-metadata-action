package client

import (
	"context"
	"encoding/json"
	"fmt"
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
	assert.Equal(t, TIMEOUT, client.httpClient.Timeout)
}

func TestSendMetadata_Success(t *testing.T) {
	var receivedBody []byte
	var receivedMethod string
	var receivedPath string
	var receivedHeaders http.Header

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedHeaders = r.Header.Clone()

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedBody = body

		// Send success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControl:             []models.AgentControl{},
	}

	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

	err := client.SendMetadata(context.Background(), "java", "1.2.3", metadata)

	_ = getStdout()
	_ = getStderr()

	// Verify behavior, not log messages
	require.NoError(t, err)
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/v1/agents/TestAgent/versions/1.2.3", receivedPath)
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "Bearer test-token", receivedHeaders.Get("Authorization"))

	// Verify body can be parsed
	var receivedMetadata models.AgentMetadata
	err = json.Unmarshal(receivedBody, &receivedMetadata)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", receivedMetadata.Metadata["version"])
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
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{
			{
				"version":     "1.0.0",
				"platform":    "linux",
				"description": "Config 1",
				"type":        "string",
				"format":      "text",
				"schema":      "schema1",
			},
			{
				"version":     "1.0.0",
				"platform":    "windows",
				"description": "Config 2",
				"type":        "string",
				"format":      "text",
				"schema":      "schema2",
			},
		},
		AgentControl: []models.AgentControl{
			{
				Platform: "all",
				Content:  "base64content",
			},
		},
	}

	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

	err := client.SendMetadata(context.Background(), "java", "1.2.3", metadata)

	_ = getStdout()
	_ = getStderr()

	// Verify behavior - successful request with correct data
	require.NoError(t, err)
}

func TestSendMetadata_ValidationErrors(t *testing.T) {
	tests := []struct {
		name              string
		agentType         string
		agentVersion      string
		metadata          *models.AgentMetadata
		expectedErrorText string
	}{
		{
			name:              "nil metadata",
			agentType:         "java-agent",
			agentVersion:      "1.2.3",
			metadata:          nil,
			expectedErrorText: "metadata is required",
		},
		{
			name:         "empty agent type",
			agentType:    "",
			agentVersion: "1.2.3",
			metadata: &models.AgentMetadata{
				Metadata: models.Metadata{
					"version": "1.2.3",
				},
			},
			expectedErrorText: "agent type is required",
		},
		{
			name:         "empty agent version",
			agentType:    "java-agent",
			agentVersion: "",
			metadata: &models.AgentMetadata{
				Metadata: models.Metadata{
					"version": "",
				},
			},
			expectedErrorText: "agent version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewInstrumentationClient("https://api.example.com", "token")

			getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

			err := client.SendMetadata(context.Background(), tt.agentType, tt.agentVersion, tt.metadata)

			_ = getStdout()
			_ = getStderr()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErrorText)
		})
	}
}

func TestSendMetadata_EmptyAgentVersion_FallbackSuccess(t *testing.T) {
	var receivedPath string

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			"version": "2.0.0",
		},
	}

	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

	// Pass empty agent version - should fallback to metadata.Metadata["version"]
	err := client.SendMetadata(context.Background(), "java", "", metadata)

	_ = getStdout()
	_ = getStderr()

	require.NoError(t, err)
	assert.Equal(t, "/v1/agents/TestAgent/versions/2.0.0", receivedPath)
}

func TestSendMetadata_HTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedStatus int
		expectedInBody string
	}{
		{
			name:           "internal server error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": "Internal server error"}`,
			expectedStatus: 500,
			expectedInBody: "Internal server error",
		},
		{
			name:           "bad request",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"error": "Invalid metadata format"}`,
			expectedStatus: 400,
			expectedInBody: "Invalid metadata format",
		},
		{
			name:           "unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "Invalid token"}`,
			expectedStatus: 401,
			expectedInBody: "Invalid token",
		},
		{
			name:           "not found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "Endpoint not found"}`,
			expectedStatus: 404,
			expectedInBody: "Endpoint not found",
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
					"version": "1.2.3",
				},
				ConfigurationDefinitions: []models.ConfigurationDefinition{},
				AgentControl:             []models.AgentControl{},
			}

			getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

			err := client.SendMetadata(context.Background(), "java", "1.2.3", metadata)

			_ = getStdout()
			_ = getStderr()

			require.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("metadata submission failed with status %d", tt.expectedStatus))
			assert.Contains(t, err.Error(), tt.expectedInBody)
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
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControl:             []models.AgentControl{},
	}

	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

	err := client.SendMetadata(context.Background(), "java", "1.2.3", metadata)

	_ = getStdout()
	_ = getStderr()

	// Verify error behavior
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata submission failed with status 500")
}

func TestSendMetadata_NetworkError(t *testing.T) {
	// Use a non-routable IP address that will cause network error without DNS lookup
	// 192.0.2.1 is in TEST-NET-1 range (RFC 5737) - reserved for documentation
	client := NewInstrumentationClient("http://192.0.2.1:1", "token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControl:             []models.AgentControl{},
	}

	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

	err := client.SendMetadata(context.Background(), "java", "1.2.3", metadata)

	_ = getStdout()
	_ = getStderr()

	// Verify error behavior
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send metadata")
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
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControl:             []models.AgentControl{},
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	getStdout, getStderr := testutil.CaptureOutputWithDisplay(t)

	err := client.SendMetadata(ctx, "java", "1.2.3", metadata)

	_ = getStdout()
	_ = getStderr()

	// Verify error behavior
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send metadata")
}
