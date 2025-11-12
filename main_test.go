package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestMainWithoutAgentRepo(t *testing.T) {
	if err := os.Unsetenv("AGENT_REPO"); err != nil {
		assert.Fail(t, "Error unsetting AGENT_REPO environment variable", err)
	}

	_, err := loadGitHubConfig()
	if err == nil {
		assert.Fail(t, "Expected error reading AGENT_REPO env var")
	}

	if err := os.Unsetenv("GITHUB_TOKEN"); err != nil {
		assert.Fail(t, "Error unsetting GITHUB_TOKEN environment variable", err)
	}

	_, err = loadGitHubConfig()
	if err == nil {
		assert.Fail(t, "Expected error reading GITHUB_TOKEN env var")
	}
}

func TestMainWithoutConfigFile(t *testing.T) {
	if err := os.Setenv("AGENT_REPO", "newrelic/newrelic-java-agent"); err != nil {
		assert.Fail(t, "Error setting AGENT_REPO environment variable", err)
	}
	if err := os.Setenv("GITHUB_TOKEN", "test-token"); err != nil {
		assert.Fail(t, "Error setting GITHUB_TOKEN environment variable", err)
	}

	_, err := readConfigsFileToJson("newrelic/newrelic-java-agent", "test-token")
	if err == nil {
		assert.Fail(t, "Expected error reading config file, got nil")
	}
}

func TestReadConfigsWithMockedFile(t *testing.T) {
	// Create mock configs.yaml content
	mockYAML := `configs:
  - name: "Test GitHubConfig"
    slug: "test-config"
    platform: "linux"
    description: "A test configuration"
    type: "agent"
    version: "1.0.0"
    schema: "v1"
  - name: "Another GitHubConfig"
    slug: "another-config"
    platform: "windows"
    description: "Another test configuration"
    type: "integration"
    version: "2.0.0"
    schema: "v2"
`

	// Create a mock HTTP server to simulate GitHub API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Encode the mock YAML as base64 (as GitHub API does)
		encodedContent := base64.StdEncoding.EncodeToString([]byte(mockYAML))

		// Return GitHub API response format
		response := GitHubContent{
			Content:  encodedContent,
			Encoding: "base64",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Override the GitHub API URL for testing
	// Note: You'll need to modify the FetchFile method to accept a base URL
	// For now, we'll test with a custom client
	client := NewGitHubClient("test-token")

	// Create a custom request to the mock server
	req, err := http.NewRequest("GET", mockServer.URL, nil)
	if err != nil {
		assert.Fail(t, "Failed to create request", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.client.Do(req)
	if err != nil {
		assert.Fail(t, "Failed to make request", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		assert.Fail(t, "Failed to read response", err)
	}

	var content GitHubContent
	if err := json.Unmarshal(body, &content); err != nil {
		assert.Fail(t, "Failed to parse response", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		assert.Fail(t, "Failed to decode content", err)
	}

	// Parse the YAML
	var configFile ConfigFile
	if err := yaml.Unmarshal(decoded, &configFile); err != nil {
		assert.Fail(t, "Failed to parse YAML", err)
	}

	configs := convertToConfigJson(configFile.Configs)

	// Verify results
	assert.NotNil(t, configs, "Expected configs to not be nil")
	assert.Equal(t, 2, len(configs), "Expected 2 configs")

	// Verify first config
	assert.Equal(t, "Test GitHubConfig", configs[0].Name)
	assert.Equal(t, "test-config", configs[0].Slug)
	assert.Equal(t, "linux", configs[0].Platform)
	assert.Equal(t, "A test configuration", configs[0].Description)
	assert.Equal(t, "agent", configs[0].Type)
	assert.Equal(t, "1.0.0", configs[0].Version)

	// Verify second config
	assert.Equal(t, "Another GitHubConfig", configs[1].Name)
	assert.Equal(t, "another-config", configs[1].Slug)
	assert.Equal(t, "windows", configs[1].Platform)
	assert.Equal(t, "Another test configuration", configs[1].Description)
	assert.Equal(t, "integration", configs[1].Type)
	assert.Equal(t, "2.0.0", configs[1].Version)
}
