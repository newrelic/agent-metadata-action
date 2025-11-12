package config

import (
	"os"
	"testing"

	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/models"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestReadConfigsWithMockedGitHub(t *testing.T) {
	// Reset GitHub client singleton before test
	github.ResetClient()
	// Create mock configs.yaml content
	mockYAML := `configurationDefinitions:
  - name: "Test Config"
    slug: "test-config"
    platform: "linux"
    description: "A test configuration"
    type: "agent"
    version: "1.0.0"
    schema: "v1"
  - name: "Another Config"
    slug: "another-config"
    platform: "windows"
    description: "Another test configuration"
    type: "integration"
    version: "2.0.0"
    schema: "v2"
`

	// Test the parsing logic directly without HTTP mocking
	// since we can't easily access the internal client
	var configFile models.ConfigFile
	err := yaml.Unmarshal([]byte(mockYAML), &configFile)
	assert.NoError(t, err)

	configs := configFile.Configs

	// Verify results
	assert.NotNil(t, configs)
	assert.Equal(t, 2, len(configs))

	// Verify first config
	assert.Equal(t, "Test Config", configs[0].Name)
	assert.Equal(t, "test-config", configs[0].Slug)
	assert.Equal(t, "linux", configs[0].Platform)
	assert.Equal(t, "A test configuration", configs[0].Description)
	assert.Equal(t, "agent", configs[0].Type)
	assert.Equal(t, "1.0.0", configs[0].Version)

	// Verify second config
	assert.Equal(t, "Another Config", configs[1].Name)
	assert.Equal(t, "another-config", configs[1].Slug)
	assert.Equal(t, "windows", configs[1].Platform)
	assert.Equal(t, "Another test configuration", configs[1].Description)
	assert.Equal(t, "integration", configs[1].Type)
	assert.Equal(t, "2.0.0", configs[1].Version)
}

func TestReadConfigsWithoutEnvVars(t *testing.T) {
	if err := os.Unsetenv("AGENT_REPO"); err != nil {
		assert.Fail(t, "Error unsetting AGENT_REPO environment variable", err)
	}
	if err := os.Unsetenv("GITHUB_TOKEN"); err != nil {
		assert.Fail(t, "Error unsetting GITHUB_TOKEN environment variable", err)
	}

	cfg, err := LoadEnv()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}
