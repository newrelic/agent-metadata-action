package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnv_Success(t *testing.T) {
	// Set up environment (mimics what would be set from calling workflow when checkout precedes this action)
	t.Setenv("GITHUB_WORKSPACE", "/tmp/workspace")

	cfg, err := LoadEnv()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/workspace", cfg)
}

func TestLoadEnv_MissingWorkspace(t *testing.T) {
	// Don't set up environment (mimics what would NOT be set from calling workflow if checkout does NOT precede this action)

	cfg, err := LoadEnv()
	assert.Error(t, err)
	assert.Empty(t, cfg)
	assert.Contains(t, err.Error(), "GITHUB_WORKSPACE")
}

func TestReadConfigurationDefinitions_Success(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create test config file
	configFile := filepath.Join(configDir, "configurationDefinitions.yml")
	testYAML := `configurationDefinitions:
  - name: test-config
    slug: test-config
    platform: linux
    description: Test configuration
    type: test-config
    version: 1.0.0
    format: yaml
    schema: ./schemas/myschema.json`

	err = os.WriteFile(configFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Test reading the config
	configs, err := ReadConfigurationDefinitions(tmpDir)
	require.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.Equal(t, "test-config", configs[0].Name)
	assert.Equal(t, "Test configuration", configs[0].Description)
}

func TestReadConfigurationDefinitions_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	configs, err := ReadConfigurationDefinitions(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, configs)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestReadConfigurationDefinitions_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, "configurationDefinitions.yml")
	invalidYAML := `invalid: yaml: content: [unclosed`
	err = os.WriteFile(configFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	configs, err := ReadConfigurationDefinitions(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, configs)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}
