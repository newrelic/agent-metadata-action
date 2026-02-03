package sign

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/retry"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewClient creates a new signing client
// baseURL: signing service base URL (e.g., "https://oci-signer.service.newrelic.com")
// token: Bearer token for authentication
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,
	}
}

// SignArtifact signs an uploaded artifact
// POST /v1/signing/{clientId}/sign
// clientId: GitHub repository name (e.g., "dotnet-agent")
// request: signing request with registry, repository, tag, digest
// Returns error on failure (non-2xx or network error)
func (c *Client) SignArtifact(ctx context.Context, clientId string, request *models.SigningRequest) error {
	logging.Log(ctx, "group", "Signing artifact")
	defer logging.Log(ctx, "endgroup", "")

	// Validate inputs
	logging.Debug(ctx, "Validating inputs...")
	if clientId == "" {
		logging.Error(ctx, "Signing client ID is required but was empty")
		return retry.NewNonRetryableError(fmt.Errorf("signing client ID is required"))
	}
	if request == nil {
		logging.Error(ctx, "Signing request is required but was nil")
		return retry.NewNonRetryableError(fmt.Errorf("signing request is required"))
	}

	// Validate request fields
	if err := request.Validate(); err != nil {
		logging.Errorf(ctx, "Invalid signing request: %v", err)
		return retry.NewNonRetryableError(fmt.Errorf("invalid signing request: %w", err))
	}

	logging.Debugf(ctx, "Signing client ID: %s", clientId)
	logging.Debugf(ctx, "Registry: %s", request.Registry)
	logging.Debugf(ctx, "Repository: %s", request.Repository)
	logging.Debugf(ctx, "Tag: %s", request.Tag)
	logging.Debugf(ctx, "Digest: %s", request.Digest)

	// Construct URL
	requestURL := fmt.Sprintf("%s/v1/signing/%s/sign", c.baseURL, clientId)
	logging.Debugf(ctx, "Target URL: %s", requestURL)
	logging.Debugf(ctx, "Base URL: %s", c.baseURL)

	// Marshal request to JSON
	logging.Debug(ctx, "Marshaling request to JSON...")
	jsonBody, err := json.Marshal(request)
	if err != nil {
		logging.Errorf(ctx, "Failed to marshal request: %v", err)
		return retry.NewNonRetryableError(fmt.Errorf("failed to marshal request: %w", err))
	}
	logging.Debugf(ctx, "JSON payload size: %d bytes", len(jsonBody))

	// Create HTTP request
	logging.Debug(ctx, "Creating HTTP POST request...")
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		logging.Errorf(ctx, "Failed to create request: %v", err)
		return retry.NewNonRetryableError(fmt.Errorf("failed to create request: %w", err))
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
		logging.Errorf(ctx, "HTTP request failed after %s: %v", duration, err)
		return fmt.Errorf("failed to send signing request: %w", err)
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
		logging.Errorf(ctx, "Artifact signing failed with status %d", resp.StatusCode)

		// Log response body for debugging, but truncate if too large
		responsePreview := string(body)
		if len(responsePreview) > 500 {
			responsePreview = responsePreview[:500] + "... (truncated)"
		}
		logging.Debugf(ctx, "Error response body: %s", responsePreview)

		err := fmt.Errorf("artifact signing failed with status %d: %s", resp.StatusCode, string(body))

		// Determine if error is retryable
		// Retry on: 5xx (server errors), 408 (timeout), 429 (rate limit)
		// Don't retry: 4xx (client errors, except 408 and 429)
		isRetryable := resp.StatusCode >= 500 || resp.StatusCode == 408 || resp.StatusCode == 429
		if !isRetryable {
			return retry.NewNonRetryableError(err)
		}
		return err
	}

	// Success logging
	logging.Notice(ctx, "Artifact signed successfully")
	if len(body) > 0 {
		logging.Debugf(ctx, "Success response: %s", string(body))
	}

	return nil
}

// ParseRegistryURL extracts registry domain and repository path from OCI registry URL
// Input formats:
//   - "docker.io/newrelic/agents"
//   - "localhost:5000/test-agents"
//   - "ghcr.io/company/product/agents"
//   - "https://registry.example.com/path/to/repo"
//
// Returns:
//   - registry: domain with optional port (e.g., "docker.io", "localhost:5000")
//   - repository: path without domain (e.g., "newrelic/agents", "test-agents")
//   - error: if URL cannot be parsed
func ParseRegistryURL(registryURL string) (registry string, repository string, err error) {
	if registryURL == "" {
		return "", "", fmt.Errorf("registry URL cannot be empty")
	}

	// Check if URL has explicit scheme (http://, https://)
	if strings.HasPrefix(registryURL, "http://") || strings.HasPrefix(registryURL, "https://") {
		// Parse as URL
		parsedURL, parseErr := url.Parse(registryURL)
		if parseErr != nil {
			return "", "", fmt.Errorf("failed to parse registry URL: %w", parseErr)
		}

		registry = parsedURL.Host
		repository = strings.Trim(parsedURL.Path, "/")
	} else {
		// No scheme - split on first '/'
		parts := strings.SplitN(registryURL, "/", 2)
		if len(parts) < 2 {
			return "", "", fmt.Errorf("registry URL must contain both domain and repository path, got: %s", registryURL)
		}

		registry = parts[0]
		repository = parts[1]
	}

	// Validate both parts are non-empty
	if registry == "" {
		return "", "", fmt.Errorf("registry domain cannot be empty")
	}
	if repository == "" {
		return "", "", fmt.Errorf("repository path cannot be empty")
	}

	// Trim any trailing slashes from repository
	repository = strings.Trim(repository, "/")

	return registry, repository, nil
}
