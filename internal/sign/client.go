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

	"agent-metadata-action/internal/models"
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
	fmt.Println("::group::Signing artifact")
	defer fmt.Println("::endgroup::")

	// Validate inputs
	fmt.Println("::debug::Validating inputs...")
	if clientId == "" {
		fmt.Println("::error::Client ID is required but was empty")
		return fmt.Errorf("client ID is required")
	}
	if request == nil {
		fmt.Println("::error::Signing request is required but was nil")
		return fmt.Errorf("signing request is required")
	}

	// Validate request fields
	if err := request.Validate(); err != nil {
		fmt.Printf("::error::Invalid signing request: %v\n", err)
		return fmt.Errorf("invalid signing request: %w", err)
	}

	fmt.Printf("::debug::Client ID: %s\n", clientId)
	fmt.Printf("::debug::Registry: %s\n", request.Registry)
	fmt.Printf("::debug::Repository: %s\n", request.Repository)
	fmt.Printf("::debug::Tag: %s\n", request.Tag)
	fmt.Printf("::debug::Digest: %s\n", request.Digest)

	// Construct URL
	requestURL := fmt.Sprintf("%s/v1/signing/%s/sign", c.baseURL, clientId)
	fmt.Printf("::debug::Target URL: %s\n", requestURL)
	fmt.Printf("::debug::Base URL: %s\n", c.baseURL)

	// Marshal request to JSON
	fmt.Println("::debug::Marshaling request to JSON...")
	jsonBody, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("::error::Failed to marshal request: %v\n", err)
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	fmt.Printf("::debug::JSON payload size: %d bytes\n", len(jsonBody))

	// Create HTTP request
	fmt.Println("::debug::Creating HTTP POST request...")
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(jsonBody))
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
		return fmt.Errorf("failed to send signing request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("::debug::Response received in %s\n", duration)
	fmt.Printf("::debug::HTTP status code: %d %s\n", resp.StatusCode, resp.Status)

	// Read response body for error details (with size limit)
	fmt.Println("::debug::Reading response body...")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("::error::Failed to read response body: %v\n", err)
		return fmt.Errorf("failed to read response: %w", err)
	}
	fmt.Printf("::debug::Response body size: %d bytes\n", len(body))

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("::error::Artifact signing failed with status %d\n", resp.StatusCode)

		// Log response body for debugging, but truncate if too large
		responsePreview := string(body)
		if len(responsePreview) > 500 {
			responsePreview = responsePreview[:500] + "... (truncated)"
		}
		fmt.Printf("::debug::Error response body: %s\n", responsePreview)

		return fmt.Errorf("artifact signing failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Success logging
	fmt.Println("::notice::Artifact signed successfully")
	if len(body) > 0 {
		fmt.Printf("::debug::Success response: %s\n", string(body))
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
