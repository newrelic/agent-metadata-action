package loader

import (
	"agent-metadata-action/internal/config"
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
	configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
	schemasDir := filepath.Join(configDir, "schemas")
	err := os.MkdirAll(schemasDir, 0755)
	require.NoError(t, err)

	// Create test schema file
	schemaContent := `{"type": "object", "properties": {"test": {"type": "string"}}}`
	schemaFile := filepath.Join(schemasDir, "myschema.json")
	err = os.WriteFile(schemaFile, []byte(schemaContent), 0644)
	require.NoError(t, err)

	// Create test config file
	configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
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

func TestReadAgentControlDefinitions_Success(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
	agentControlDir := filepath.Join(configDir, "agentControl")
	err := os.MkdirAll(agentControlDir, 0755)
	require.NoError(t, err)

	// Create test agent control content file
	contentData := `agent:
    name: test-agent
    version: 1.0.0
  controls:
    - name: control-1
      enabled: true`
	contentFile := filepath.Join(agentControlDir, "test-control.yml")
	err = os.WriteFile(contentFile, []byte(contentData), 0644)
	require.NoError(t, err)

	// Create test agent control definitions file
	agentControlFile := filepath.Join(configDir, config.GetAgentControlDefinitionsFilename())
	testYAML := `agentControlDefinitions:
    - platform: KUBERNETES
      supportFromAgent: 1.0.0
      supportFromAgentControl: 1.0.0
      content: ./agentControl/test-control.yml`

	err = os.WriteFile(agentControlFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Test reading the agent control definitions
	agentControls, err := ReadAgentControlDefinitions(tmpDir)
	require.NoError(t, err)
	assert.Len(t, agentControls, 1)
	assert.Equal(t, "KUBERNETES", agentControls[0]["platform"])
	assert.Equal(t, "1.0.0", agentControls[0]["supportFromAgent"])
	assert.Equal(t, "1.0.0", agentControls[0]["supportFromAgentControl"])

	// Verify content was base64 encoded
	expectedEncoded := base64.StdEncoding.EncodeToString([]byte(contentData))
	assert.Equal(t, expectedEncoded, agentControls[0]["content"])

	// Verify we can decode it back
	decoded, err := base64.StdEncoding.DecodeString(agentControls[0]["content"].(string))
	require.NoError(t, err)
	assert.Equal(t, contentData, string(decoded))
}

func TestReadConfigurationDefinitions_ErrorCases(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, tmpDir string)
		expectedErrMsg string
	}{
		{
			name: "file not found",
			setupFunc: func(t *testing.T, tmpDir string) {
				// Don't create the config file
			},
			expectedErrMsg: "failed to read file",
		},
		{
			name: "invalid YAML",
			setupFunc: func(t *testing.T, tmpDir string) {
				configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(configDir, 0755))

				configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
				invalidYAML := `invalid: yaml: content: [unclosed`
				require.NoError(t, os.WriteFile(configFile, []byte(invalidYAML), 0644))
			},
			expectedErrMsg: "failed to parse YAML",
		},
		{
			name: "empty array",
			setupFunc: func(t *testing.T, tmpDir string) {
				configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(configDir, 0755))

				configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
				testYAML := `configurationDefinitions: []`
				require.NoError(t, os.WriteFile(configFile, []byte(testYAML), 0644))
			},
			expectedErrMsg: "configurationDefinitions cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFunc(t, tmpDir)

			// method under test
			configs, err := ReadConfigurationDefinitions(tmpDir)

			require.Error(t, err)
			assert.Nil(t, configs)
			assert.Contains(t, err.Error(), tt.expectedErrMsg)
		})
	}
}

func TestReadAgentControlDefinitions_ErrorCases(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, tmpDir string)
		expectedErrMsg string
	}{
		{
			name: "file not found",
			setupFunc: func(t *testing.T, tmpDir string) {
				// Don't create the agent control definitions file
			},
			expectedErrMsg: "failed to read file",
		},
		{
			name: "invalid YAML",
			setupFunc: func(t *testing.T, tmpDir string) {
				configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(configDir, 0755))

				agentControlFile := filepath.Join(configDir, config.GetAgentControlDefinitionsFilename())
				invalidYAML := `invalid: yaml: content: [unclosed`
				require.NoError(t, os.WriteFile(agentControlFile, []byte(invalidYAML), 0644))
			},
			expectedErrMsg: "failed to parse YAML",
		},
		{
			name: "empty array",
			setupFunc: func(t *testing.T, tmpDir string) {
				configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(configDir, 0755))

				agentControlFile := filepath.Join(configDir, config.GetAgentControlDefinitionsFilename())
				testYAML := `agentControlDefinitions: []`
				require.NoError(t, os.WriteFile(agentControlFile, []byte(testYAML), 0644))
			},
			expectedErrMsg: "agentControlDefinitions cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFunc(t, tmpDir)

			// method under test
			agentControls, err := ReadAgentControlDefinitions(tmpDir)

			require.Error(t, err)
			assert.Nil(t, agentControls)
			assert.Contains(t, err.Error(), tt.expectedErrMsg)
		})
	}
}

func TestReadConfigurationDefinitions_SchemaLoadingWarnings(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func(t *testing.T, tmpDir string) string // returns schema path for config
		expectedWarning string
	}{
		{
			name: "schema file not found",
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Don't create schema file
				return "./schemas/nonexistent.json"
			},
			expectedWarning: "failed to load schema",
		},
		{
			name: "empty schema file",
			setupFunc: func(t *testing.T, tmpDir string) string {
				configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
				schemasDir := filepath.Join(configDir, "schemas")
				require.NoError(t, os.MkdirAll(schemasDir, 0755))

				schemaFile := filepath.Join(schemasDir, "empty.json")
				require.NoError(t, os.WriteFile(schemaFile, []byte(""), 0644))
				return "./schemas/empty.json"
			},
			expectedWarning: "failed to load schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
			require.NoError(t, os.MkdirAll(configDir, 0755))

			schemaPath := tt.setupFunc(t, tmpDir)

			// Create test config file that references the schema
			configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
			testYAML := fmt.Sprintf(`configurationDefinitions:
  - platform: linux
    description: Test configuration
    type: test-config
    version: 1.0.0
    format: yaml
    schema: %s`, schemaPath)

			require.NoError(t, os.WriteFile(configFile, []byte(testYAML), 0644))

			getStdout, _ := testutil.CaptureOutput(t)

			// method under test - should not fail if schema can't be loaded
			configs, err := ReadConfigurationDefinitions(tmpDir)

			outputStr := getStdout()

			require.NoError(t, err)
			assert.NotNil(t, configs)
			assert.Contains(t, outputStr, tt.expectedWarning)
		})
	}
}

func TestReadConfigurationDefinitions_MultipleConfigs(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
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
	configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
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
	configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Test that schema is optional (will be required in the future)
	configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
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
			configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
			err := os.MkdirAll(configDir, 0755)
			require.NoError(t, err)

			// Create config file with malicious schema path
			configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
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

func TestReadConfigurationDefinitions_EmptyArray(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create test config file with empty array
	configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
	testYAML := `configurationDefinitions: []`

	err = os.WriteFile(configFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Test reading the config - should error
	configs, err := ReadConfigurationDefinitions(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, configs)
	assert.Contains(t, err.Error(), "configurationDefinitions cannot be empty")
}

func TestReadDefinitionsFile_ItemNotMap(t *testing.T) {
	// Test for error path: item in array is not a map (line 118 in definitions.go)
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create config file where array contains non-map items
	configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
	testYAML := `configurationDefinitions:
    - platform: linux
    - "this is a string, not a map"
    - 123`

	err = os.WriteFile(configFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Test reading the config - should error
	configs, err := ReadConfigurationDefinitions(tmpDir)
	require.Error(t, err)
	assert.Nil(t, configs)
	assert.Contains(t, err.Error(), "is not a map")
}

func TestReadDefinitionsFile_NoArrayFound(t *testing.T) {
	// Test for error path: no array found in YAML file (line 130 in definitions.go)
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create config file with only scalar values (no arrays) - valid YAML
	configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
	testYAML := `name: test-config
version: 1.0.0
description: This file has no arrays`

	err = os.WriteFile(configFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Test reading the config - should error with "no array found"
	configs, err := ReadConfigurationDefinitions(tmpDir)
	require.Error(t, err)
	assert.Nil(t, configs)
	assert.Contains(t, err.Error(), "no array found in YAML file")
}

func TestReadAgentControlDefinitions_ContentLoadingWarnings(t *testing.T) {
	// Test warning paths in loadAndEncodeFile when loading agent control content
	tests := []struct {
		name            string
		setupFunc       func(t *testing.T, tmpDir string) string // returns content path for config
		expectedWarning string
	}{
		{
			name: "content file not found",
			setupFunc: func(t *testing.T, tmpDir string) string {
				// Don't create content file
				return "./agentControl/nonexistent.yml"
			},
			expectedWarning: "failed to load content",
		},
		{
			name: "empty content file",
			setupFunc: func(t *testing.T, tmpDir string) string {
				configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
				agentControlDir := filepath.Join(configDir, "agentControl")
				require.NoError(t, os.MkdirAll(agentControlDir, 0755))

				contentFile := filepath.Join(agentControlDir, "empty.yml")
				require.NoError(t, os.WriteFile(contentFile, []byte(""), 0644))
				return "./agentControl/empty.yml"
			},
			expectedWarning: "failed to load content",
		},
		{
			name: "directory traversal in content path",
			setupFunc: func(t *testing.T, tmpDir string) string {
				return "../../../etc/passwd"
			},
			expectedWarning: "directory traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
			require.NoError(t, os.MkdirAll(configDir, 0755))

			contentPath := tt.setupFunc(t, tmpDir)

			// Create test agent control definitions file that references the content
			agentControlFile := filepath.Join(configDir, config.GetAgentControlDefinitionsFilename())
			testYAML := fmt.Sprintf(`agentControlDefinitions:
    - platform: KUBERNETES
      supportFromAgent: 1.0.0
      supportFromAgentControl: 1.0.0
      content: %s`, contentPath)

			require.NoError(t, os.WriteFile(agentControlFile, []byte(testYAML), 0644))

			getStdout, _ := testutil.CaptureOutput(t)

			// method under test - should not fail if content can't be loaded
			agentControls, err := ReadAgentControlDefinitions(tmpDir)

			outputStr := getStdout()

			require.NoError(t, err)
			assert.NotNil(t, agentControls)
			assert.Contains(t, outputStr, tt.expectedWarning)
		})
	}
}

func TestReadAgentControlDefinitions_MultipleDefinitions(t *testing.T) {
	// Test loading multiple agent control definitions with different platforms
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
	agentControlDir := filepath.Join(configDir, "agentControl")
	err := os.MkdirAll(agentControlDir, 0755)
	require.NoError(t, err)

	// Create test content files
	content1 := `control1: data`
	contentFile1 := filepath.Join(agentControlDir, "k8s-control.yml")
	err = os.WriteFile(contentFile1, []byte(content1), 0644)
	require.NoError(t, err)

	content2 := `control2: data`
	contentFile2 := filepath.Join(agentControlDir, "host-control.yml")
	err = os.WriteFile(contentFile2, []byte(content2), 0644)
	require.NoError(t, err)

	content3 := `control3: data`
	contentFile3 := filepath.Join(agentControlDir, "linux-control.yml")
	err = os.WriteFile(contentFile3, []byte(content3), 0644)
	require.NoError(t, err)

	// Create agent control definitions file with multiple entries
	agentControlFile := filepath.Join(configDir, config.GetAgentControlDefinitionsFilename())
	testYAML := `agentControlDefinitions:
    - platform: KUBERNETES
      supportFromAgent: 1.0.0
      supportFromAgentControl: 1.0.0
      content: ./agentControl/k8s-control.yml
    - platform: HOST
      supportFromAgent: 1.1.0
      supportFromAgentControl: 1.1.0
      content: ./agentControl/host-control.yml
    - platform: LINUX
      supportFromAgent: 1.2.0
      supportFromAgentControl: 1.2.0
      content: ./agentControl/linux-control.yml`

	err = os.WriteFile(agentControlFile, []byte(testYAML), 0644)
	require.NoError(t, err)

	// Test reading the agent control definitions
	agentControls, err := ReadAgentControlDefinitions(tmpDir)
	require.NoError(t, err)
	assert.Len(t, agentControls, 3)

	// Verify first definition
	assert.Equal(t, "KUBERNETES", agentControls[0]["platform"])
	expectedEncoded1 := base64.StdEncoding.EncodeToString([]byte(content1))
	assert.Equal(t, expectedEncoded1, agentControls[0]["content"])

	// Verify second definition
	assert.Equal(t, "HOST", agentControls[1]["platform"])
	expectedEncoded2 := base64.StdEncoding.EncodeToString([]byte(content2))
	assert.Equal(t, expectedEncoded2, agentControls[1]["content"])

	// Verify third definition
	assert.Equal(t, "LINUX", agentControls[2]["platform"])
	expectedEncoded3 := base64.StdEncoding.EncodeToString([]byte(content3))
	assert.Equal(t, expectedEncoded3, agentControls[2]["content"])
}

func TestReadConfigurationDefinitions_InvalidFieldTypes(t *testing.T) {
	tests := []struct {
		name            string
		yamlContent     string
		expectedWarning string
		expectedDebug   string
	}{
		{
			name: "schema field is not a string (number)",
			yamlContent: `configurationDefinitions:
  - platform: linux
    description: Test config
    type: test-config
    version: 1.0.0
    format: yaml
    schema: 12345`,
			expectedWarning: "schema field is not a string",
		},
		{
			name: "schema field is not a string (boolean)",
			yamlContent: `configurationDefinitions:
  - platform: linux
    description: Test config
    type: test-config
    version: 1.0.0
    format: yaml
    schema: true`,
			expectedWarning: "schema field is not a string",
		},
		{
			name: "schema field is not a string (object)",
			yamlContent: `configurationDefinitions:
  - platform: linux
    description: Test config
    type: test-config
    version: 1.0.0
    format: yaml
    schema:
      type: object`,
			expectedWarning: "schema field is not a string",
		},
		{
			name: "schema field is nil",
			yamlContent: `configurationDefinitions:
  - platform: linux
    description: Test config
    type: test-config
    version: 1.0.0
    format: yaml
    schema: null`,
			expectedDebug: "no schema provided",
		},
		{
			name: "schema field is empty string",
			yamlContent: `configurationDefinitions:
  - platform: linux
    description: Test config
    type: test-config
    version: 1.0.0
    format: yaml
    schema: ""`,
			expectedDebug: "no schema provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
			require.NoError(t, os.MkdirAll(configDir, 0755))

			configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
			require.NoError(t, os.WriteFile(configFile, []byte(tt.yamlContent), 0644))

			getStdout, _ := testutil.CaptureOutput(t)

			// method under test
			configs, err := ReadConfigurationDefinitions(tmpDir)

			outputStr := getStdout()

			require.NoError(t, err)
			assert.NotNil(t, configs)
			assert.Len(t, configs, 1)

			if tt.expectedWarning != "" {
				assert.Contains(t, outputStr, tt.expectedWarning)
			}
			if tt.expectedDebug != "" {
				assert.Contains(t, outputStr, tt.expectedDebug)
			}
		})
	}
}

func TestReadAgentControlDefinitions_InvalidFieldTypes(t *testing.T) {
	tests := []struct {
		name            string
		yamlContent     string
		expectedWarning string
		expectedDebug   string
	}{
		{
			name: "content field is not a string (number)",
			yamlContent: `agentControlDefinitions:
  - platform: KUBERNETES
    supportFromAgent: 1.0.0
    supportFromAgentControl: 1.0.0
    content: 12345`,
			expectedWarning: "content field is not a string",
		},
		{
			name: "content field is not a string (boolean)",
			yamlContent: `agentControlDefinitions:
  - platform: KUBERNETES
    supportFromAgent: 1.0.0
    supportFromAgentControl: 1.0.0
    content: false`,
			expectedWarning: "content field is not a string",
		},
		{
			name: "content field is not a string (array)",
			yamlContent: `agentControlDefinitions:
  - platform: KUBERNETES
    supportFromAgent: 1.0.0
    supportFromAgentControl: 1.0.0
    content: [file1.yml, file2.yml]`,
			expectedWarning: "content field is not a string",
		},
		{
			name: "content field is nil",
			yamlContent: `agentControlDefinitions:
  - platform: KUBERNETES
    supportFromAgent: 1.0.0
    supportFromAgentControl: 1.0.0
    content: null`,
			expectedDebug: "no content provided",
		},
		{
			name: "content field is empty string",
			yamlContent: `agentControlDefinitions:
  - platform: KUBERNETES
    supportFromAgent: 1.0.0
    supportFromAgentControl: 1.0.0
    content: ""`,
			expectedDebug: "no content provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, config.GetRootFolderForAgentRepo())
			require.NoError(t, os.MkdirAll(configDir, 0755))

			agentControlFile := filepath.Join(configDir, config.GetAgentControlDefinitionsFilename())
			require.NoError(t, os.WriteFile(agentControlFile, []byte(tt.yamlContent), 0644))

			getStdout, _ := testutil.CaptureOutput(t)

			// method under test
			agentControls, err := ReadAgentControlDefinitions(tmpDir)

			outputStr := getStdout()

			require.NoError(t, err)
			assert.NotNil(t, agentControls)
			assert.Len(t, agentControls, 1)

			if tt.expectedWarning != "" {
				assert.Contains(t, outputStr, tt.expectedWarning)
			}
			if tt.expectedDebug != "" {
				assert.Contains(t, outputStr, tt.expectedDebug)
			}
		})
	}
}

func TestLoadAndEncodeFile_PathValidation(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T) (workspace string, filePath string)
		expectedErrMsg string
	}{
		{
			name: "path with directory traversal using ../",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				workspace := filepath.Join(tmpDir, "workspace", config.GetRootFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(workspace, 0755))

				// Path that would escape using ../
				return filepath.Join(tmpDir, "workspace"), "../../../etc/passwd"
			},
			expectedErrMsg: "directory traversal",
		},
		{
			name: "path with multiple directory traversals",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				workspace := filepath.Join(tmpDir, "workspace", config.GetRootFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(workspace, 0755))

				return filepath.Join(tmpDir, "workspace"), "schemas/../../sensitive.json"
			},
			expectedErrMsg: "directory traversal",
		},
		{
			name: "path attempting to read outside .fleetControl via symlink-like path",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				workspace := filepath.Join(tmpDir, "workspace", config.GetRootFolderForAgentRepo())
				require.NoError(t, os.MkdirAll(workspace, 0755))

				// Create a file at workspace level (outside .fleetControl)
				outsideFile := filepath.Join(tmpDir, "workspace", "outside.json")
				require.NoError(t, os.WriteFile(outsideFile, []byte(`{"test": "data"}`), 0644))

				// Try to access it from .fleetControl
				return filepath.Join(tmpDir, "workspace"), "../outside.json"
			},
			expectedErrMsg: "directory traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace, filePath := tt.setupFunc(t)

			getStdout, _ := testutil.CaptureOutput(t)

			// Create a config file that references this path
			configDir := filepath.Join(workspace, config.GetRootFolderForAgentRepo())
			configFile := filepath.Join(configDir, config.GetConfigurationDefinitionsFilename())
			testYAML := fmt.Sprintf(`configurationDefinitions:
  - platform: linux
    description: Test config
    type: test-config
    version: 1.0.0
    format: yaml
    schema: %s`, filePath)

			require.NoError(t, os.WriteFile(configFile, []byte(testYAML), 0644))

			// method under test
			configs, err := ReadConfigurationDefinitions(workspace)

			outputStr := getStdout()

			require.NoError(t, err)
			assert.NotNil(t, configs)
			assert.Contains(t, outputStr, tt.expectedErrMsg)
		})
	}
}
