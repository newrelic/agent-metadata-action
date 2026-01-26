package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
)

// InstrumentationClient handles instrumentation metadata operations
type InstrumentationClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewInstrumentationClient creates a new instrumentation client
func NewInstrumentationClient(baseURL, token string) *InstrumentationClient {
	return &InstrumentationClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,
	}
}

// SendMetadata sends agent metadata to the instrumentation service
// POST /v1/agents/{agentType}/versions/{agentVersion}
func (c *InstrumentationClient) SendMetadata(ctx context.Context, agentType string, agentVersion string, metadata *models.AgentMetadata) error {
	logging.Log(ctx, "group", "Sending metadata to instrumentation service")
	defer logging.Log(ctx, "endgroup", "")

	// Validate inputs
	logging.Debug(ctx, "Validating inputs...")
	if metadata == nil {
		logging.Error(ctx, "Metadata is required but was nil")
		return fmt.Errorf("metadata is required")
	}
	if agentType == "" {
		logging.Error(ctx, "Agent type is required but was empty")
		return fmt.Errorf("agent type is required")
	}
	if agentVersion == "" {
		logging.Error(ctx, "Agent version is required but was empty")
		return fmt.Errorf("agent version is required")
	}
	logging.Debugf(ctx, "Agent type: %s", agentType)
	logging.Debugf(ctx, "Agent version: %s", agentVersion)

	// Construct URL
	url := fmt.Sprintf("%s/v1/agents/%s/versions/%s", c.baseURL, agentType, agentVersion)
	logging.Debugf(ctx, "Target URL: %s", url)
	logging.Debugf(ctx, "Base URL: %s", c.baseURL)

	// Marshal metadata to JSON
	logging.Debug(ctx, "Marshaling metadata to JSON...")
	jsonBody, err := json.Marshal(metadata)
	if err != nil {
		logging.NoticeErrorWithCategory(ctx, err, "metadata.send", map[string]interface{}{
			"error.operation": "marshal_metadata",
			"agent.type":      agentType,
			"agent.version":   agentVersion,
		})
		logging.Errorf(ctx, "Failed to marshal metadata: %v", err)
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	logging.Debugf(ctx, "JSON payload size: %d bytes", len(jsonBody))
	logging.Debugf(ctx, "Configuration definitions count: %d", len(metadata.ConfigurationDefinitions))
	logging.Debugf(ctx, "Agent control entries: %d", len(metadata.AgentControlDefinitions))

	// Create HTTP request
	logging.Debug(ctx, "Creating HTTP POST request...")
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		logging.NoticeErrorWithCategory(ctx, err, "metadata.send", map[string]interface{}{
			"error.operation": "create_http_request",
			"http.url":        url,
			"agent.type":      agentType,
			"agent.version":   agentVersion,
		})
		logging.Errorf(ctx, "Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	logging.Debug(ctx, "Setting request headers...")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	// SECURITY: Token is in header but not logged

	// Execute request
	logging.Debug(ctx, "Sending HTTP request...")
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		logging.NoticeErrorWithCategory(ctx, err, "metadata.send", map[string]interface{}{
			"error.operation": "execute_http_request",
			"http.url":        url,
			"http.duration":   duration.String(),
			"agent.type":      agentType,
			"agent.version":   agentVersion,
		})
		logging.Errorf(ctx, "HTTP request failed after %s: %v", duration, err)
		return fmt.Errorf("failed to send metadata: %w", err)
	}
	defer resp.Body.Close()

	logging.Debugf(ctx, "Response received in %s", duration)
	logging.Debugf(ctx, "HTTP status code: %d %s", resp.StatusCode, resp.Status)

	// Read response body for error details (with size limit)
	logging.Debug(ctx, "Reading response body...")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.Errorf(ctx, "Failed to read response body: %v", err)
		return fmt.Errorf("failed to read response: %w", err)
	}
	logging.Debugf(ctx, "Response body size: %d bytes", len(body))

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Log response body for debugging, but truncate if too large
		responsePreview := string(body)
		if len(responsePreview) > 500 {
			responsePreview = responsePreview[:500] + "... (truncated)"
		}

		err := fmt.Errorf("metadata submission failed with status %d: %s", resp.StatusCode, string(body))
		logging.NoticeErrorWithCategory(ctx, err, "metadata.send", map[string]interface{}{
			"error.operation":    "http_non_2xx_response",
			"http.status_code":   resp.StatusCode,
			"http.url":           url,
			"http.response_body": responsePreview,
			"agent.type":         agentType,
			"agent.version":      agentVersion,
		})
		logging.Errorf(ctx, "Metadata submission failed with status %d", resp.StatusCode)
		logging.Debugf(ctx, "Error response body: %s", responsePreview)

		return err
	}

	// Success logging
	logging.Notice(ctx, "Metadata successfully submitted to instrumentation service")
	if len(body) > 0 {
		logging.Debugf(ctx, "Success response: %s", string(body))
	}

	return nil
}
