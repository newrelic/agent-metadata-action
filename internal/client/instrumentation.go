package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-metadata-action/internal/models"
)

// InstrumentationClient handles instrumentation metadata operations
type InstrumentationClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

const TIMEOUT = 30 * time.Second

// NewInstrumentationClient creates a new instrumentation client
func NewInstrumentationClient(baseURL, token string) *InstrumentationClient {
	return &InstrumentationClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: TIMEOUT,
		},
		token: token,
	}
}

// SendMetadata sends agent metadata to the instrumentation service
// POST /v1/agents/{agentType}/versions/{agentVersion}
func (c *InstrumentationClient) SendMetadata(ctx context.Context, agentType string, agentVersion string, metadata *models.AgentMetadata) error {
	fmt.Println("::group::Sending metadata to instrumentation service")
	defer fmt.Println("::endgroup::")

	if metadata == nil {
		fmt.Println("::error::Metadata is required but was nil")
		return fmt.Errorf("metadata is required")
	}
	if agentType == "" {
		fmt.Println("::error::Agent type is required but was empty")
		return fmt.Errorf("agent type is required")
	}

	// If agent version not in input variables, get version from metadata map (if present)
	if agentVersion == "" {
		agentVersion, _ = metadata.Metadata["version"].(string)
	}
	if agentVersion == "" {
		fmt.Println("::error::Agent version is required but was empty")
		return fmt.Errorf("agent version is required")
	}

	fmt.Printf("::debug::Agent type: %s\n", agentType)
	fmt.Printf("::debug::Agent version: %s\n", agentVersion)

	// Construct URL
	url := fmt.Sprintf("%s/v1/agents/%s/versions/%s", c.baseURL, "TestAgent", agentVersion) // @todo update TestAgent after testing
	fmt.Printf("::debug::Target URL: %s\n", url)
	fmt.Printf("::debug::Base URL: %s\n", c.baseURL)

	// Marshal metadata to JSON
	fmt.Println("::debug::Marshaling metadata to JSON...")
	jsonBody, err := json.Marshal(metadata)
	if err != nil {
		fmt.Printf("::error::Failed to marshal metadata: %v\n", err)
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	fmt.Printf("::debug::JSON payload size: %d bytes\n", len(jsonBody))
	fmt.Printf("::debug::Configuration definitions count: %d\n", len(metadata.ConfigurationDefinitions))
	fmt.Printf("::debug::Agent control entries: %d\n", len(metadata.AgentControl))

	// Create HTTP request
	fmt.Println("::debug::Creating HTTP POST request...")
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("::error::Failed to create request: %v\n", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	fmt.Println("::debug::Setting request headers...")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	// SECURITY: Token is in header but not logged

	// Execute request
	fmt.Println("::debug::Sending HTTP request...")
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("::error::HTTP request failed after %s: %v\n", duration, err)
		return fmt.Errorf("failed to send metadata: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("::debug::Response received in %s\n", duration)
	fmt.Printf("::debug::HTTP status code: %d %s\n", resp.StatusCode, resp.Status)

	// Read response body for error details
	fmt.Println("::debug::Reading response body...")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("::error::Failed to read response body: %v\n", err)
		return fmt.Errorf("failed to read response: %w", err)
	}
	fmt.Printf("::debug::Response body size: %d bytes\n", len(body))

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Log response body for debugging, but truncate if too large
		responsePreview := string(body)
		if len(responsePreview) > 500 {
			responsePreview = responsePreview[:500] + "... (truncated)"
		}
		fmt.Printf("::debug::Error response body: %s\n", responsePreview)

		return fmt.Errorf("metadata submission failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Success logging
	fmt.Println("::notice::Metadata successfully submitted to instrumentation service")
	if len(body) > 0 {
		fmt.Printf("::debug::Success response: %s\n", string(body))
	}

	return nil
}
