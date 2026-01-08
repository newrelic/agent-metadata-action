package loader

import (
	"agent-metadata-action/internal/testutil"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfigurationDefinitions_Success(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	schemasDir := filepath.Join(configDir, "schemas")
	err := os.MkdirAll(schemasDir, 0755)
	require.NoError(t, err)

	// Create test schema file
	schemaContent := `{"type": "object", "properties": {"test": {"type": "string"}}}`
	schemaFile := filepath.Join(schemasDir, "myschema.json")
	err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Create test config file
	configFile := filepath.Join(configDir, "configurationDefinitions.yml")
	testYAML := `configurationDefinitions:
  - platform: linux
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
	assert.Equal(t, "linux", configs[0]["platform"])
	assert.Equal(t, "Test configuration", configs[0]["description"])

	// Verify schema was base64 encoded
	expectedEncoded := base64.StdEncoding.EncodeToString([]byte(schemaContent))
	assert.Equal(t, expectedEncoded, configs[0]["schema"])

	// Verify we can decode it back
	decoded, err := base64.StdEncoding.DecodeString(configs[0]["schema"].(string))
	require.NoError(t, err)
	assert.Equal(t, schemaContent, string(decoded))
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

func TestReadConfigurationDefinitions_SchemaFileNotFound(t *testing.T) {
	// Create temporary directory structure without schema file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create test config file that references non-existent schema
	configFile := filepath.Join(configDir, "configurationDefinitions.yml")
	testYAML := `configurationDefinitions:
  - platform: linux
    description: Test configuration
    type: test-config
    version: 1.0.0
    format: yaml
    schema: ./schemas/nonexistent.json`

	err = os.WriteFile(configFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	getStdout, _ := testutil.CaptureOutput(t)

	// Test reading the config - should not fail if schema can't be loaded
	configs, err := ReadConfigurationDefinitions(tmpDir)

	outputStr := getStdout()

	require.NoError(t, err)
	assert.NotNil(t, configs)
	assert.Contains(t, outputStr, "failed to load schema")
}

func TestReadConfigurationDefinitions_EmptySchemaFile(t *testing.T) {
	// Create temporary directory structure with empty schema file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	schemasDir := filepath.Join(configDir, "schemas")
	err := os.MkdirAll(schemasDir, 0755)
	require.NoError(t, err)

	// Create empty schema file
	schemaFile := filepath.Join(schemasDir, "empty.json")
	err = os.WriteFile(schemaFile, []byte(""), 0644)
	require.NoError(t, err)

	// Create test config file that references empty schema
	configFile := filepath.Join(configDir, "configurationDefinitions.yml")
	testYAML := `configurationDefinitions:
  - platform: linux
    description: Test configuration
    type: test-config
    version: 1.0.0
    format: yaml
    schema: ./schemas/empty.json`

	err = os.WriteFile(configFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	getStdout, _ := testutil.CaptureOutput(t)

	// Test reading the config - should not fail if schema can't be loaded
	configs, err := ReadConfigurationDefinitions(tmpDir)

	outputStr := getStdout()

	require.NoError(t, err)
	assert.NotNil(t, configs)
	assert.Contains(t, outputStr, "failed to load schema")
}

func TestReadConfigurationDefinitions_InvalidJSONSchema(t *testing.T) {
	// Create temporary directory structure with invalid JSON schema file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	schemasDir := filepath.Join(configDir, "schemas")
	err := os.MkdirAll(schemasDir, 0755)
	require.NoError(t, err)

	// Create schema file with invalid JSON
	schemaFile := filepath.Join(schemasDir, "invalid.json")
	err = os.WriteFile(schemaFile, []byte(`{invalid json content`), 0644)
	require.NoError(t, err)

	// Create test config file that references invalid schema
	configFile := filepath.Join(configDir, "configurationDefinitions.yml")
	testYAML := `configurationDefinitions:
  - platform: linux
    description: Test configuration
    type: test-config
    version: 1.0.0
    format: yaml
    schema: ./schemas/invalid.json`

	err = os.WriteFile(configFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	getStdout, _ := testutil.CaptureOutput(t)

	// Test reading the config - should not fail if schema can't be loaded
	configs, err := ReadConfigurationDefinitions(tmpDir)

	outputStr := getStdout()

	require.NoError(t, err)
	assert.NotNil(t, configs)
	assert.Contains(t, outputStr, "failed to load schema")
}

func TestReadConfigurationDefinitions_MultipleConfigs(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	schemasDir := filepath.Join(configDir, "schemas")
	err := os.MkdirAll(schemasDir, 0755)
	require.NoError(t, err)

	// Create test schema files
	schema1Content := `{"type": "object", "properties": {"field1": {"type": "string"}}}`
	schema1File := filepath.Join(schemasDir, "schema1.json")
	err = os.WriteFile(schema1File, []byte(schema1Content), 0644)
	require.NoError(t, err)

	schema2Content := `{"type": "object", "properties": {"field2": {"type": "number"}}}`
	schema2File := filepath.Join(schemasDir, "schema2.json")
	err = os.WriteFile(schema2File, []byte(schema2Content), 0644)
	require.NoError(t, err)

	schema3Content := `{"type": "object", "properties": {"field3": {"type": "boolean"}}}`
	schema3File := filepath.Join(schemasDir, "schema3.json")
	err = os.WriteFile(schema3File, []byte(schema3Content), 0644)
	require.NoError(t, err)

	// Create test config file with multiple configs
	configFile := filepath.Join(configDir, "configurationDefinitions.yml")
	testYAML := `configurationDefinitions:
  - platform: linux
    description: First configuration
    type: config-1
    version: 1.0.0
    format: json
    schema: ./schemas/schema1.json
  - platform: kubernetes
    description: Second configuration
    type: config-2
    version: 2.0.0
    format: json
    schema: ./schemas/schema2.json
  - platform: host
    description: Third configuration
    type: config-3
    version: 3.0.0
    format: yaml
    schema: ./schemas/schema3.json`

	err = os.WriteFile(configFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Test reading the configs
	configs, err := ReadConfigurationDefinitions(tmpDir)
	require.NoError(t, err)
	assert.Len(t, configs, 3)

	// Verify first config
	assert.Equal(t, "linux", configs[0]["platform"])
	expectedEncoded1 := base64.StdEncoding.EncodeToString([]byte(schema1Content))
	assert.Equal(t, expectedEncoded1, configs[0]["schema"])

	// Verify second config
	assert.Equal(t, "kubernetes", configs[1]["platform"])
	expectedEncoded2 := base64.StdEncoding.EncodeToString([]byte(schema2Content))
	assert.Equal(t, expectedEncoded2, configs[1]["schema"])

	// Verify third config
	assert.Equal(t, "host", configs[2]["platform"])
	expectedEncoded3 := base64.StdEncoding.EncodeToString([]byte(schema3Content))
	assert.Equal(t, expectedEncoded3, configs[2]["schema"])
}

func TestReadConfigurationDefinitions_ValidationIntegration(t *testing.T) {
	// This is an integration test to verify that model validation
	// is properly wired up when reading config files.
	// Comprehensive field validation is tested in models_test.go
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Test that schema is optional (will be required in the future)
	configFile := filepath.Join(configDir, "configurationDefinitions.yml")
	yamlContent := `configurationDefinitions:
  - version: 1.2.3
    platform: linux
    description: Test configuration
    type: test-config
    format: yaml`

	err = os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	configs, err := ReadConfigurationDefinitions(tmpDir)
	require.NoError(t, err)
	assert.Len(t, configs, 1)
	// Schema is nil when not provided
	schema, _ := configs[0]["schema"]
	assert.Nil(t, schema)
}

func TestReadConfigurationDefinitions_EmptyArray(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create test config file with empty array
	configFile := filepath.Join(configDir, "configurationDefinitions.yml")
	testYAML := `configurationDefinitions: []`

	err = os.WriteFile(configFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Test reading the config - should error
	configs, err := ReadConfigurationDefinitions(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, configs)
	assert.Contains(t, err.Error(), "configurationDefinitions cannot be empty")
}

func TestReadConfigurationDefinitions_DirectoryTraversal(t *testing.T) {
	tests := []struct {
		name       string
		schemaPath string
	}{
		{
			name:       "parent directory traversal with ../",
			schemaPath: "../../../etc/passwd",
		},
		{
			name:       "relative parent traversal",
			schemaPath: "schemas/../../secrets.json",
		},
		{
			name:       "multiple parent traversals",
			schemaPath: "./../.././../../../sensitive.json",
		},
		{
			name:       "hidden parent in path",
			schemaPath: "schemas/../../../config.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, ".fleetControl")
			err := os.MkdirAll(configDir, 0755)
			require.NoError(t, err)

			// Create config file with malicious schema path
			configFile := filepath.Join(configDir, "configurationDefinitions.yml")
			testYAML := fmt.Sprintf(`configurationDefinitions:
  - version: 1.0.0
    platform: linux
    description: Test configuration
    type: test-config
    format: yaml
    schema: %s`, tt.schemaPath)

			err = os.WriteFile(configFile, []byte(testYAML), 0644)
			require.NoError(t, err)

			getStdout, _ := testutil.CaptureOutput(t)

			// Test reading the config - should not fail if schema can't be loaded
			configs, err := ReadConfigurationDefinitions(tmpDir)

			outputStr := getStdout()

			require.NoError(t, err)
			assert.NotNil(t, configs)
			assert.Contains(t, outputStr, "directory traversal")
		})
	}
}

func TestReadAgentControl_Success(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	agentControlDir := filepath.Join(configDir, "agentControl")
	err := os.MkdirAll(agentControlDir, 0755)
	require.NoError(t, err)

	// Create test agent control file
	agentControlContent := `schema:
  type: object
  properties:
    setting1:
      type: string
    setting2:
      type: boolean`
	agentControlFile := filepath.Join(agentControlDir, "agent-schema-for-agent-control.yml")
	err = os.WriteFile(agentControlFile, []byte(agentControlContent), 0644)
	require.NoError(t, err)

	// Test reading the agent control
	agentControl, err := LoadAndEncodeAgentControl(tmpDir)
	require.NoError(t, err)
	assert.Len(t, agentControl, 1)
	assert.Equal(t, AGENT_CONTROL_PLATFORM, agentControl[0].Platform)
	assert.NotEmpty(t, agentControl[0].Content)

	// Verify content was base64 encoded
	expectedEncoded := base64.StdEncoding.EncodeToString([]byte(agentControlContent))
	assert.Equal(t, expectedEncoded, agentControl[0].Content)

	// Verify we can decode it back
	decoded, err := base64.StdEncoding.DecodeString(agentControl[0].Content)
	require.NoError(t, err)
	assert.Equal(t, agentControlContent, string(decoded))
}

func TestReadAgentControl_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	agentControl, err := LoadAndEncodeAgentControl(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, agentControl)
	assert.Contains(t, err.Error(), "failed to read agent control file")
}

func TestReadAgentControl_EmptyFile(t *testing.T) {
	// Create temporary directory structure with empty agent control file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".fleetControl")
	agentControlDir := filepath.Join(configDir, "agentControl")
	err := os.MkdirAll(agentControlDir, 0755)
	require.NoError(t, err)

	// Create empty agent control file
	agentControlFile := filepath.Join(agentControlDir, "agent-schema-for-agent-control.yml")
	err = os.WriteFile(agentControlFile, []byte(""), 0644)
	require.NoError(t, err)

	// Test reading the agent control - should fail
	agentControl, err := LoadAndEncodeAgentControl(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, agentControl)
	assert.Contains(t, err.Error(), "agent control file")
	assert.Contains(t, err.Error(), "is empty")
}
