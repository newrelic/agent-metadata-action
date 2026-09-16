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

func TestSendMetadata_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		assert.Equal(t, "POST", r.Method)

		// Verify URL path
		assert.True(t, strings.HasPrefix(r.URL.Path, "/v1/agents/NRJavaAgent/versions/"))

		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Verify body can be parsed
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var metadata models.AgentMetadata
		err = json.Unmarshal(body, &metadata)
		require.NoError(t, err)
		assert.Equal(t, "1.2.3", metadata.Metadata["version"])

		// Send success response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControlDefinitions:  []models.AgentControlDefinition{},
	}

	getStdout, getStderr := testutil.CaptureOutput(t)

	// method under test
	err := client.SendMetadata(context.Background(), "NRJavaAgent", "1.2.3", metadata)

	outputStr := getStdout()
	stderrStr := getStderr()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "Sending metadata to instrumentation service")
	assert.Contains(t, outputStr, "Agent type: NRJavaAgent")
	assert.Contains(t, outputStr, "Agent version: 1.2.3")
	assert.Contains(t, outputStr, "HTTP status code: 200")
	assert.Contains(t, outputStr, "Metadata successfully submitted")

	assert.NotContains(t, stderrStr, "::error::")
}

func TestSendMetadata_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		metadata      *models.AgentMetadata
		agentType     string
		agentVersion  string
		expectedInErr string
		expectedInLog string
	}{
		{
			name:          "nil metadata",
			metadata:      nil,
			agentType:     "NRJavaAgent",
			agentVersion:  "1.2.3",
			expectedInErr: "metadata is required",
			expectedInLog: "Metadata is required but was nil",
		},
		{
			name: "empty agent type",
			metadata: &models.AgentMetadata{
				Metadata: models.Metadata{
					"version": "1.2.3",
				},
			},
			agentType:     "",
			agentVersion:  "1.2.3",
			expectedInErr: "agent type is required",
			expectedInLog: "Agent type is required but was empty",
		},
		{
			name: "empty agent version",
			metadata: &models.AgentMetadata{
				Metadata: models.Metadata{
					"version": "1.2.3",
				},
			},
			agentType:     "NRJavaAgent",
			agentVersion:  "",
			expectedInErr: "agent version is required",
			expectedInLog: "Agent version is required but was empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewInstrumentationClient("https://api.example.com", "token")
			getStdout, _ := testutil.CaptureOutput(t)

			// method under test
			err := client.SendMetadata(context.Background(), tt.agentType, tt.agentVersion, tt.metadata)

			outputStr := getStdout()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedInErr)
			assert.Contains(t, outputStr, tt.expectedInLog)
		})
	}
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
					"version": "1.2.3",
				},
				ConfigurationDefinitions: []models.ConfigurationDefinition{},
				AgentControlDefinitions:  []models.AgentControlDefinition{},
			}

			getStdout, _ := testutil.CaptureOutput(t)

			// method under test
			err := client.SendMetadata(context.Background(), "NRJavaAgent", "1.2.3", metadata)

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
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControlDefinitions:  []models.AgentControlDefinition{},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SendMetadata(context.Background(), "NRJavaAgent", "1.2.3", metadata)

	outputStr := getStdout()

	require.Error(t, err)
	// Should contain truncation notice
	assert.Contains(t, outputStr, "(truncated)")
}

func TestSendMetadata_NetworkError(t *testing.T) {
	// Use localhost with a port that's guaranteed not to be listening
	// This fails fast without DNS lookups or long timeouts
	client := NewInstrumentationClient("http://127.0.0.1:1", "token")

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControlDefinitions:  []models.AgentControlDefinition{},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SendMetadata(context.Background(), "NRJavaAgent", "1.2.3", metadata)

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
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControlDefinitions:  []models.AgentControlDefinition{},
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SendMetadata(ctx, "NRJavaAgent", "1.2.3", metadata)

	_ = getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry cancelled")
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
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControlDefinitions:  []models.AgentControlDefinition{},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SendMetadata(context.Background(), "NRJavaAgent", "1.2.3", metadata)

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
		assert.Len(t, metadata.AgentControlDefinitions, 1)

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
		AgentControlDefinitions: []models.AgentControlDefinition{
			{
				"platform": "all",
				"content":  "base64content",
			},
		},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SendMetadata(context.Background(), "NRJavaAgent", "1.2.3", metadata)

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "Configuration definitions count: 2")
	assert.Contains(t, outputStr, "Agent control entries: 1")
}

func TestSendMetadata_MarshalError(t *testing.T) {
	client := NewInstrumentationClient("https://api.example.com", "token")

	// Create metadata with unmarshalable value (channel)
	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			"version":       "1.2.3",
			"unmarshalable": make(chan int), // Channels cannot be marshaled to JSON
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControlDefinitions:  []models.AgentControlDefinition{},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SendMetadata(context.Background(), "NRJavaAgent", "1.2.3", metadata)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal metadata")
	assert.Contains(t, outputStr, "Failed to marshal metadata")
}

func TestPromoteToReleaseChannel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/agents/NRJavaAgent/versions/1.2.3/release-channel", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req models.SetReleaseChannelRequest
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)
		assert.Equal(t, "HOST", req.Platform)
		assert.Equal(t, "LINUX", req.OperatingSystem)
		assert.Equal(t, "REGULAR", req.Channel)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"agentType":"NRJavaAgent","platform":"HOST","operatingSystem":"LINUX","channel":"REGULAR","currentVersion":"1.2.3","previousVersion":"1.2.2","promotedAt":"2026-08-19T00:00:00Z","promotedBy":"test-principal"}`))
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")

	req := &models.SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "REGULAR"}

	getStdout, getStderr := testutil.CaptureOutput(t)

	// method under test
	promotion, err := client.PromoteToReleaseChannel(context.Background(), "NRJavaAgent", "1.2.3", req)

	outputStr := getStdout()
	stderrStr := getStderr()

	require.NoError(t, err)
	require.NotNil(t, promotion)
	assert.Equal(t, "NRJavaAgent", promotion.AgentType)
	assert.Equal(t, "1.2.3", promotion.CurrentVersion)
	require.NotNil(t, promotion.PreviousVersion)
	assert.Equal(t, "1.2.2", *promotion.PreviousVersion)
	assert.Contains(t, outputStr, "Promoting version to release channel")
	assert.Contains(t, outputStr, "HTTP status code: 200")
	assert.Contains(t, outputStr, "Version successfully promoted to release channel")
	assert.NotContains(t, stderrStr, "::error::")
}

func TestPromoteToReleaseChannel_NoPreviousVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"agentType":"NRJavaAgent","platform":"HOST","operatingSystem":"LINUX","channel":"REGULAR","currentVersion":"1.2.3","promotedAt":"2026-08-19T00:00:00Z","promotedBy":"test-principal"}`))
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")
	req := &models.SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "REGULAR"}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	promotion, err := client.PromoteToReleaseChannel(context.Background(), "NRJavaAgent", "1.2.3", req)

	_ = getStdout()

	require.NoError(t, err)
	require.NotNil(t, promotion)
	assert.Nil(t, promotion.PreviousVersion)
}

func TestPromoteToReleaseChannel_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.SetReleaseChannelRequest
		agentType     string
		agentVersion  string
		expectedInErr string
		expectedInLog string
	}{
		{
			name:          "nil request",
			request:       nil,
			agentType:     "NRJavaAgent",
			agentVersion:  "1.2.3",
			expectedInErr: "release channel request is required",
			expectedInLog: "Release channel request is required but was nil",
		},
		{
			name:          "empty agent type",
			request:       &models.SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "REGULAR"},
			agentType:     "",
			agentVersion:  "1.2.3",
			expectedInErr: "agent type is required",
			expectedInLog: "Agent type is required but was empty",
		},
		{
			name:          "empty agent version",
			request:       &models.SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "REGULAR"},
			agentType:     "NRJavaAgent",
			agentVersion:  "",
			expectedInErr: "agent version is required",
			expectedInLog: "Agent version is required but was empty",
		},
		{
			name:          "invalid request",
			request:       &models.SetReleaseChannelRequest{Platform: "BAREMETAL", Channel: "REGULAR"},
			agentType:     "NRJavaAgent",
			agentVersion:  "1.2.3",
			expectedInErr: "invalid platform",
			expectedInLog: "Release channel request is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewInstrumentationClient("https://api.example.com", "token")
			getStdout, _ := testutil.CaptureOutput(t)

			// method under test
			promotion, err := client.PromoteToReleaseChannel(context.Background(), tt.agentType, tt.agentVersion, tt.request)

			outputStr := getStdout()

			require.Error(t, err)
			assert.Nil(t, promotion)
			assert.Contains(t, err.Error(), tt.expectedInErr)
			assert.Contains(t, outputStr, tt.expectedInLog)
		})
	}
}

func TestPromoteToReleaseChannel_HTTPErrors(t *testing.T) {
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
			expectedInErr: "release channel promotion failed with status 500",
			expectedInLog: "HTTP status code: 500",
		},
		{
			name:          "bad request",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error": "Invalid request"}`,
			expectedInErr: "release channel promotion failed with status 400",
			expectedInLog: "Release channel promotion failed with status 400",
		},
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			responseBody:  `{"error": "Version not found"}`,
			expectedInErr: "release channel promotion failed with status 404",
			expectedInLog: "HTTP status code: 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewInstrumentationClient(server.URL, "test-token")
			req := &models.SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "REGULAR"}

			getStdout, _ := testutil.CaptureOutput(t)

			// method under test
			promotion, err := client.PromoteToReleaseChannel(context.Background(), "NRJavaAgent", "1.2.3", req)

			outputStr := getStdout()

			require.Error(t, err)
			assert.Nil(t, promotion)
			assert.Contains(t, err.Error(), tt.expectedInErr)
			assert.Contains(t, outputStr, tt.expectedInLog)
		})
	}
}

func TestPromoteToReleaseChannel_NetworkError(t *testing.T) {
	client := NewInstrumentationClient("http://127.0.0.1:1", "token")
	req := &models.SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "REGULAR"}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	promotion, err := client.PromoteToReleaseChannel(context.Background(), "NRJavaAgent", "1.2.3", req)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Nil(t, promotion)
	assert.Contains(t, err.Error(), "failed to promote release channel")
	assert.Contains(t, outputStr, "HTTP request failed")
}

func TestPromoteToReleaseChannel_ResponseBodyReadError(t *testing.T) {
	client := &InstrumentationClient{
		baseURL: "https://api.example.com",
		token:   "test-token",
		httpClient: &http.Client{
			Transport: &errorTransport{},
		},
	}

	req := &models.SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "REGULAR"}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	promotion, err := client.PromoteToReleaseChannel(context.Background(), "NRJavaAgent", "1.2.3", req)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Nil(t, promotion)
	assert.Contains(t, err.Error(), "failed to read response")
	assert.Contains(t, outputStr, "Failed to read response body")
}

func TestPromoteToReleaseChannel_MalformedResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	client := NewInstrumentationClient(server.URL, "test-token")
	req := &models.SetReleaseChannelRequest{Platform: "HOST", OperatingSystem: "LINUX", Channel: "REGULAR"}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	promotion, err := client.PromoteToReleaseChannel(context.Background(), "NRJavaAgent", "1.2.3", req)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Nil(t, promotion)
	assert.Contains(t, err.Error(), "failed to parse release channel promotion response")
	assert.Contains(t, outputStr, "Failed to parse response body")
}

func TestSendMetadata_ResponseBodyReadError(t *testing.T) {
	// Create test server with custom response that fails to read
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Response is written but we'll replace the body reader in the client
	}))
	defer server.Close()

	// Create client with custom HTTP client that returns error reader
	client := &InstrumentationClient{
		baseURL: server.URL,
		token:   "test-token",
		httpClient: &http.Client{
			Transport: &errorTransport{},
		},
	}

	metadata := &models.AgentMetadata{
		Metadata: models.Metadata{
			"version": "1.2.3",
		},
		ConfigurationDefinitions: []models.ConfigurationDefinition{},
		AgentControlDefinitions:  []models.AgentControlDefinition{},
	}

	getStdout, _ := testutil.CaptureOutput(t)

	// method under test
	err := client.SendMetadata(context.Background(), "NRJavaAgent", "1.2.3", metadata)

	outputStr := getStdout()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read response")
	assert.Contains(t, outputStr, "Failed to read response body")
}
